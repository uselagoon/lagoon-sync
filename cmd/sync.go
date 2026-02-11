package cmd

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/mitchellh/mapstructure"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	synchers "github.com/uselagoon/lagoon-sync/synchers"
	"github.com/uselagoon/lagoon-sync/utils"
)

var ProjectName string
var sourceEnvironmentName string
var targetEnvironmentName string
var SyncerType string
var ServiceName string
var configurationFile string
var SSHHost string
var SSHPort string
var SSHKey string
var SSHVerbose bool
var SSHSkipAgent bool
var CmdSSHKey string
var noCliInteraction bool
var dryRun bool
var verboseSSH bool
var RsyncArguments string
var runSyncProcess synchers.RunSyncProcessFunctionType
var skipSourceCleanup bool
var skipTargetCleanup bool
var skipTargetImport bool
var localTransferResourceName string
var namedTransferResource string

var APIEndpoint string
var useSshPortal bool // This is our feature flag for now. With the major version, we change the ssh config details for lagoon-sync files

var syncCmd = &cobra.Command{
	Use:   "sync [mariadb|files|mongodb|postgres|etc.]",
	Short: "Sync a resource type",
	Long:  `Use Lagoon-Sync to sync an external environments resources with the local environment`,
	Args:  cobra.MinimumNArgs(1),
	Run:   syncCommandRun,
}

func syncCommandRun(cmd *cobra.Command, args []string) {

	// SyncerType can be one of two things
	// 1. a direct reference to a syncer - i.e. mariadb, postgres, files
	// 2. a reference to an alias in the configuration file (.lagoon.yml/.lagoon-sync.yml)
	// 3. a reference to a custom syncer, also defined in the config file.
	SyncerType := args[0]

	viper.Set("syncer-type", args[0])

	// Load configuration
	configRoot, err := loadConfigRoot()
	if err != nil {
		utils.LogFatalError(fmt.Sprintf("Failed to load configuration: %v", err), nil)
	}
	if viper.ConfigFileUsed() == "" {
		utils.LogWarning("No configuration has been given/found for syncer: ", SyncerType)
	}

	// Resolve project name from multiple sources
	ProjectName = resolveProjectName(ProjectName, configRoot)

	// Set service default to 'cli'
	if ServiceName == "" {
		ServiceName = getServiceName(SyncerType)
	}

	// Build source and target environments
	sourceEnvironment, targetEnvironment := buildEnvironments(ProjectName, ServiceName, sourceEnvironmentName, targetEnvironmentName)

	// Resolve the appropriate syncer
	lagoonSyncer, err := resolveSyncer(SyncerType, configRoot)
	if err != nil {
		utils.LogFatalError(err.Error(), nil)
	}

	if ProjectName == "" {
		utils.LogFatalError("No Project name given", nil)
	}

	if !noCliInteraction {

		// We'll set the spinner utility to show
		utils.SetShowSpinner(true)

		// Ask for confirmation
		confirmationResult, err := confirmPrompt(fmt.Sprintf("Project: %s - you are about to sync %s from %s to %s, is this correct",
			ProjectName,
			SyncerType,
			sourceEnvironment.EnvironmentName, targetEnvironment.EnvironmentName))
		utils.SetColour(true)
		if err != nil || !confirmationResult {
			utils.LogFatalError("User cancelled sync - exiting", nil)
		}
	}

	// Build SSH options from config, env vars, and flags
	sshOptions := buildSSHOptions(configRoot, SSHHost, SSHPort, SSHKey, SSHVerbose, SSHSkipAgent, RsyncArguments)

	// Build SSH option wrapper with optional SSH portal integration
	sshOptionWrapper, err := buildSSHOptionWrapper(ProjectName, sshOptions, configRoot, APIEndpoint, useSshPortal)
	if err != nil {
		utils.LogFatalError(fmt.Sprintf("Failed to configure SSH options: %v", err), nil)
	}

	// let's update the named transfer resource if it is set
	if namedTransferResource != "" {
		err = lagoonSyncer.SetTransferResource(namedTransferResource)
		if err != nil {
			utils.LogFatalError(err.Error(), nil)
		}
	}

	utils.LogDebugInfo("Config that is used for SSH", sshOptions)

	// Add assertion - we no longer support Remote to Remote syncs
	if sourceEnvironment.EnvironmentName != synchers.LOCAL_ENVIRONMENT_NAME && targetEnvironment.EnvironmentName != synchers.LOCAL_ENVIRONMENT_NAME {
		utils.LogFatalError("Remote to Remote transfers are not supported", nil)
	}

	err = runSyncProcess(synchers.RunSyncProcessFunctionTypeArguments{
		SourceEnvironment: sourceEnvironment,
		TargetEnvironment: targetEnvironment,
		LagoonSyncer:      lagoonSyncer,
		SyncerType:        SyncerType,
		DryRun:            dryRun,
		//SshOptions:           sshOptions,
		SshOptionWrapper:     sshOptionWrapper,
		SkipTargetCleanup:    skipTargetCleanup,
		SkipSourceCleanup:    skipSourceCleanup,
		SkipTargetImport:     skipTargetImport,
		TransferResourceName: namedTransferResource,
	})

	if err != nil {
		utils.LogFatalError("There was an error running the sync process:", err)
	}

	if !dryRun {
		log.Printf("\n------\nSuccessful sync of %s from %s to %s\n------", SyncerType, sourceEnvironment.GetOpenshiftProjectName(), targetEnvironment.GetOpenshiftProjectName())
	}
}

// getServiceName will return the name of the service in which we run the commands themselves. This is typically
// the cli pod in a project
// TODO: this needs to be expanded to be dynamic in the future.
func getServiceName(SyncerType string) string {
	if SyncerType == "mongodb" {
		return SyncerType
	}
	return "cli"
}

func getEnvironmentSshDetails(conn utils.ApiConn, projectName string, defaultSshOptions synchers.SSHOptions) (synchers.SSHOptions, map[string]synchers.SSHOptions, error) {
	environments, err := conn.GetProjectEnvironmentDeployTargets(projectName)
	retMap := map[string]synchers.SSHOptions{}

	if err != nil {
		return synchers.SSHOptions{}, retMap, err
	}

	var defaultOptions synchers.SSHOptions
	defaultSet := false

	for _, environment := range *environments {
		retMap[environment.Name] = synchers.SSHOptions{
			Host:       environment.DeployTarget.SSHHost,
			Port:       environment.DeployTarget.SSHPort,
			Verbose:    defaultSshOptions.Verbose,
			PrivateKey: "",
			SkipAgent:  defaultSshOptions.SkipAgent,
			RsyncArgs:  defaultSshOptions.RsyncArgs,
		}

		if environment.EnvironmentType == "production" {
			defaultOptions = retMap[environment.Name]
			defaultSet = true
		}
	}
	if defaultSet == false {
		return synchers.SSHOptions{}, retMap, errors.New("COULD NOT FIND DEFAULT OPTION SET")
	}
	return defaultOptions, retMap, nil
}

func confirmPrompt(message string) (bool, error) {
	prompt := promptui.Prompt{
		Label:     message,
		IsConfirm: true,
	}
	result, err := prompt.Run()
	if result == "y" {
		return true, err
	}
	return false, err
}

// loadConfigRoot loads and unmarshals the lagoon config file if present
func loadConfigRoot() (synchers.SyncherConfigRoot, error) {
	var configRoot synchers.SyncherConfigRoot

	if viper.ConfigFileUsed() == "" {
		return configRoot, nil
	}

	lagoonConfigBytestream, err := LoadLagoonConfig(viper.ConfigFileUsed())
	if err != nil {
		return configRoot, fmt.Errorf("couldn't load lagoon config file: %w", err)
	}

	loadedConfigRoot, err := synchers.UnmarshallLagoonYamlToLagoonSyncStructure(lagoonConfigBytestream)
	if err != nil {
		return configRoot, fmt.Errorf("issue unmarshalling sync configuration from %v: %w", viper.ConfigFileUsed(), err)
	}

	return loadedConfigRoot, nil
}

// resolveProjectName determines the project name from flags, env vars, or config
// Priority: flagValue -> LAGOON_PROJECT env var -> configRoot.Project
func resolveProjectName(flagValue string, configRoot synchers.SyncherConfigRoot) string {
	if flagValue != "" {
		return flagValue
	}

	project, exists := os.LookupEnv("LAGOON_PROJECT")
	if exists {
		return strings.Replace(project, "_", "-", -1)
	}

	if configRoot.Project != "" {
		return configRoot.Project
	}

	return ""
}

// buildEnvironments creates source and target Environment structs
func buildEnvironments(projectName, serviceName, sourceEnvName, targetEnvName string) (synchers.Environment, synchers.Environment) {
	sourceEnvironment := synchers.Environment{
		ProjectName:     projectName,
		EnvironmentName: sourceEnvName,
		ServiceName:     serviceName,
	}

	// Default target to local if not specified
	if targetEnvName == "" {
		targetEnvName = synchers.LOCAL_ENVIRONMENT_NAME
	}

	targetEnvironment := synchers.Environment{
		ProjectName:     projectName,
		EnvironmentName: targetEnvName,
		ServiceName:     serviceName,
	}

	return sourceEnvironment, targetEnvironment
}

// resolveSyncer gets the appropriate syncer from the config, falling back to custom syncer
func resolveSyncer(syncerType string, configRoot synchers.SyncherConfigRoot) (synchers.Syncer, error) {
	lagoonSyncer, err := synchers.GetSyncerForTypeFromConfigRoot(syncerType, configRoot)
	if err != nil {
		// Fall back to custom syncer
		lagoonSyncer, err = synchers.GetCustomSync(configRoot, syncerType)
		if err != nil {
			return nil, fmt.Errorf("could not find syncer for type %s: %w", syncerType, err)
		}
	}
	return lagoonSyncer, nil
}

// buildSSHOptions constructs SSH options from config, env vars, and flags
// Priority for host/port: flag (if not default) -> env var -> config -> flag default
func buildSSHOptions(configRoot synchers.SyncherConfigRoot, flagHost, flagPort, flagKey string, flagVerbose, flagSkipAgent bool, rsyncArgs string) synchers.SSHOptions {
	// Decode SSH config from file if present
	sshConfig := synchers.SSHOptions{}
	if configRoot.LagoonSync["ssh"] != nil {
		mapstructure.Decode(configRoot.LagoonSync["ssh"], &sshConfig)
	}

	// Resolve SSH host
	sshHost := flagHost
	if flagHost == "ssh.lagoon.amazeeio.cloud" { // using default, check for overrides
		envSshHost, exists := os.LookupEnv("LAGOON_CONFIG_SSH_HOST")
		if exists {
			sshHost = envSshHost
		} else if sshConfig.Host != "" {
			sshHost = sshConfig.Host
		}
	}

	// Resolve SSH port
	sshPort := flagPort
	if flagPort == "32222" { // using default, check for overrides
		envSshPort, exists := os.LookupEnv("LAGOON_CONFIG_SSH_PORT")
		if exists {
			sshPort = envSshPort
		} else if sshConfig.Port != "" {
			sshPort = sshConfig.Port
		}
	}

	// Resolve SSH key (config takes priority if flag is empty)
	sshKey := flagKey
	if sshConfig.PrivateKey != "" && flagKey == "" {
		sshKey = sshConfig.PrivateKey
	}

	// Resolve verbose flag (config OR flag)
	sshVerbose := flagVerbose
	if sshConfig.Verbose && !flagVerbose {
		sshVerbose = sshConfig.Verbose
	}

	return synchers.SSHOptions{
		Host:       sshHost,
		PrivateKey: sshKey,
		Port:       sshPort,
		Verbose:    sshVerbose,
		RsyncArgs:  rsyncArgs,
		SkipAgent:  flagSkipAgent,
	}
}

// buildSSHOptionWrapper creates and configures an SSH option wrapper, optionally with SSH portal integration
func buildSSHOptionWrapper(projectName string, baseOptions synchers.SSHOptions, configRoot synchers.SyncherConfigRoot, apiEndpoint string, usePortal bool) (*synchers.SSHOptionWrapper, error) {
	sshOptionWrapper := synchers.NewSshOptionWrapper(projectName, baseOptions)

	if !usePortal {
		return sshOptionWrapper, nil
	}

	// Resolve API endpoint
	apiEndPoint := apiEndpoint
	if apiEndpoint == "https://api.lagoon.amazeeio.cloud/graphql" { // using default, check for overrides
		envApiHost, exists := os.LookupEnv("LAGOON_CONFIG_API_HOST")
		if exists {
			apiEndPoint = envApiHost + "/graphql"
		} else if configRoot.Api != "" {
			apiEndPoint = configRoot.Api
		}
	}

	// Initialize API connection and fetch environment SSH details
	apiConn := utils.ApiConn{}
	err := apiConn.Init(apiEndPoint, baseOptions.PrivateKey, baseOptions.Host, baseOptions.Port)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize API connection: %w", err)
	}

	defaultSshOption, sshopts, err := getEnvironmentSshDetails(apiConn, projectName, baseOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to get environment SSH details: %w", err)
	}

	sshOptionWrapper.SetDefaultSshOptions(defaultSshOption)
	for envName, option := range sshopts {
		sshOptionWrapper.AddSsshOptionForEnvironment(envName, option)
	}

	return sshOptionWrapper, nil
}

func init() {
	rootCmd.AddCommand(syncCmd)
	syncCmd.PersistentFlags().StringVarP(&ProjectName, "project-name", "p", "", "The Lagoon project name of the remote system")
	syncCmd.PersistentFlags().StringVarP(&sourceEnvironmentName, "source-environment-name", "e", "", "The Lagoon environment name of the source system")
	syncCmd.MarkPersistentFlagRequired("source-environment-name")
	syncCmd.PersistentFlags().StringVarP(&targetEnvironmentName, "target-environment-name", "t", "", "The target environment name (defaults to local)")
	syncCmd.PersistentFlags().StringVarP(&ServiceName, "service-name", "s", "", "The service name (default is 'cli'")
	syncCmd.MarkPersistentFlagRequired("remote-environment-name")
	syncCmd.PersistentFlags().StringVarP(&SSHHost, "ssh-host", "H", "ssh.lagoon.amazeeio.cloud", "Specify your lagoon ssh host, defaults to 'ssh.lagoon.amazeeio.cloud'")
	syncCmd.PersistentFlags().StringVarP(&SSHPort, "ssh-port", "P", "32222", "Specify your ssh port, defaults to '32222'")
	syncCmd.PersistentFlags().StringVarP(&SSHKey, "ssh-key", "i", "", "Specify path to a specific SSH key to use for authentication")
	syncCmd.PersistentFlags().BoolVar(&SSHSkipAgent, "ssh-skip-agent", false, "Do not attempt to use an ssh-agent for key management")
	syncCmd.PersistentFlags().BoolVar(&SSHVerbose, "verbose", false, "Run ssh commands in verbose (useful for debugging)")
	syncCmd.PersistentFlags().BoolVarP(&noCliInteraction, "no-interaction", "y", false, "Disallow interaction")
	syncCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Don't run the commands, just preview what will be run")
	syncCmd.PersistentFlags().StringVarP(&RsyncArguments, "rsync-args", "r", "--omit-dir-times --no-perms --no-group --no-owner --chmod=ugo=rwX --recursive --compress", "Pass through arguments to change the behaviour of rsync")
	syncCmd.PersistentFlags().BoolVar(&skipSourceCleanup, "skip-source-cleanup", false, "Don't clean up any of the files generated on the source")
	syncCmd.PersistentFlags().BoolVar(&skipTargetCleanup, "skip-target-cleanup", false, "Don't clean up any of the files generated on the target")
	syncCmd.PersistentFlags().BoolVar(&skipTargetImport, "skip-target-import", false, "This will skip the import step on the target, in combination with 'no-target-cleanup' this essentially produces a resource dump")
	syncCmd.PersistentFlags().StringVarP(&namedTransferResource, "transfer-resource-name", "", "", "The name of the temporary file to be used to transfer generated resources (db dumps, etc) - random /tmp file otherwise")
	syncCmd.PersistentFlags().StringVarP(&APIEndpoint, "api", "A", "https://api.lagoon.amazeeio.cloud/graphql", "Specify your lagoon api endpoint - required for ssh-portal integration")
	syncCmd.PersistentFlags().BoolVar(&useSshPortal, "use-ssh-portal", false, "This will use the SSH Portal rather than the (soon to be removed) SSH Service on Lagoon core. Will become default in a future release.")
	// By default, we hook up the syncers.RunSyncProcess function to the runSyncProcess variable
	// by doing this, it lets us easily override it for testing the command - but for most of the time
	// this should be okay.
	runSyncProcess = synchers.RunSyncProcess
}
