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
var skipSourceRun bool
var skipSourceCleanup bool
var skipTargetCleanup bool
var skipTargetImport bool
var localTransferResourceName string
var namedTransferResource string

var apiEndpoint string
var useSshPortal bool // This is our feature flag for now. With the major version, we change the ssh config details for lagoon-sync files
const fallbackApi = "https://api.lagoon.amazeeio.cloud/graphql"

var syncCmd = &cobra.Command{
	Use:   "sync [mariadb|files|mongodb|postgres|etc.]",
	Short: "Sync a resource type",
	Long:  `Use Lagoon-Sync to sync an external environments resources with the local environment`,
	Args:  cobra.MinimumNArgs(1),
	Run:   syncCommandRun,
}

func syncCommandRun(cmd *cobra.Command, args []string) {
	SyncerType := ""
	if len(args) > 0 {
		SyncerType = args[0]
		viper.Set("syncer-type", args[0])
	} else {
		if cmd.Name() == "to-file" || cmd.Name() == "from-file" {
			// set default as mariadb for now if no arg is given
			SyncerType = "mariadb"
		} else {
			fmt.Println("Error: No SyncerType provided.")
			return
		}
	}

	var configRoot synchers.SyncherConfigRoot

	if viper.ConfigFileUsed() == "" {
		utils.LogWarning("No configuration has been given/found for syncer: ", SyncerType)
	}

	if viper.ConfigFileUsed() != "" {
		lagoonConfigBytestream, err := LoadLagoonConfig(viper.ConfigFileUsed())
		if err != nil {
			utils.LogDebugInfo("Couldn't load lagoon config file - ", err.Error())
		} else {
			loadedConfigRoot, err := synchers.UnmarshallLagoonYamlToLagoonSyncStructure(lagoonConfigBytestream)
			if err != nil {
				log.Fatalf("There was an issue unmarshalling the sync configuration from %v: %v", viper.ConfigFileUsed(), err)
			} else {
				// Update configRoot with loaded
				configRoot = loadedConfigRoot
			}
		}
	}

	// If no project flag is given, find project from env var.
	if ProjectName == "" {
		project, exists := os.LookupEnv("LAGOON_PROJECT")
		if exists {
			ProjectName = strings.Replace(project, "_", "-", -1)
		}
		if configRoot.Project != "" {
			ProjectName = configRoot.Project
		}
	}

	// Set service default to 'cli'
	if ServiceName == "" {
		ServiceName = getServiceName(SyncerType)
	}

	sourceEnvironment := synchers.Environment{
		ProjectName:     ProjectName,
		EnvironmentName: sourceEnvironmentName,
		ServiceName:     ServiceName,
	}

	// We assume that the target environment is local if it's not passed as an argument
	if targetEnvironmentName == "" {
		targetEnvironmentName = synchers.LOCAL_ENVIRONMENT_NAME
	}
	targetEnvironment := synchers.Environment{
		ProjectName:     ProjectName,
		EnvironmentName: targetEnvironmentName,
		ServiceName:     ServiceName,
	}

	var lagoonSyncer synchers.Syncer
	// Syncers are registered in their init() functions - so here we attempt to match
	// the syncer type with the argument passed through to this command
	// (e.g. if we're running `lagoon-sync sync mariadb --...options follow` the function
	// GetSyncersForTypeFromConfigRoot will return a prepared mariadb syncher object)
	lagoonSyncer, err := synchers.GetSyncerForTypeFromConfigRoot(SyncerType, configRoot)
	if err != nil {
		// Let's ask the custom syncer if this will work, if so, we fall back on it ...
		lagoonSyncer, err = synchers.GetCustomSync(configRoot, SyncerType)
		if err != nil {
			utils.LogFatalError(err.Error(), nil)
		}
	}

	if ProjectName == "" {
		utils.LogFatalError("No Project name given", nil)
	}

	if !noCliInteraction {
		confirmationResult, err := confirmPrompt(fmt.Sprintf("Project: %s - you are about to sync %s from %s to %s, is this correct",
			ProjectName,
			SyncerType,
			sourceEnvironmentName, targetEnvironmentName))
		utils.SetColour(true)
		if err != nil || !confirmationResult {
			utils.LogFatalError("User cancelled sync - exiting", nil)
		}
	}

	// SSH Config from file
	sshConfig := synchers.SSHOptions{}
	if configRoot.LagoonSync["ssh"] != nil {
		mapstructure.Decode(configRoot.LagoonSync["ssh"], &sshConfig)
	}
	sshHost := SSHHost
	if sshConfig.Host != "" && SSHHost == "ssh.lagoon.amazeeio.cloud" {
		sshHost = sshConfig.Host
	}
	sshPort := SSHPort
	if sshConfig.Port != "" && SSHPort == "32222" {
		sshPort = sshConfig.Port
	}

	sshKey := SSHKey
	if sshConfig.PrivateKey != "" && SSHKey == "" {
		sshKey = sshConfig.PrivateKey
	}

	sshVerbose := SSHVerbose
	if sshConfig.Verbose && !sshVerbose {
		sshVerbose = sshConfig.Verbose
	}

	// Here we have the default - let's add it to a wrapper
	sshOptions := synchers.SSHOptions{
		Host:       sshHost,
		PrivateKey: sshKey,
		Port:       sshPort,
		Verbose:    sshVerbose,
		RsyncArgs:  RsyncArguments,
		SkipAgent:  SSHSkipAgent,
	}

	sshOptionWrapper := synchers.NewSshOptionWrapper(ProjectName, sshOptions)

	if useSshPortal {

		//Let's work out the api endpoint
		if configRoot.Api != "" && apiEndpoint != "" {
			apiEndpoint = configRoot.Api
		}

		apiConn := utils.ApiConn{}
		err := apiConn.Init(apiEndpoint, sshKey, "ssh.main.lagoon-core.test6.amazee.io", "22")
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		defaultSshOption, sshopts, err := getEnvironmentSshDetails(apiConn, ProjectName, sshOptions)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		sshOptionWrapper.SetDefaultSshOptions(defaultSshOption)
		for envName, option := range sshopts {
			sshOptionWrapper.AddSsshOptionForEnvironment(envName, option)
		}

	}

	// let's update the named transfer resource if it is set & not syncing to/from a file
	if namedTransferResource != "" && !strings.Contains(cmd.Name(), "-file") {
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
		SkipSourceRun:        skipSourceRun,
		SshOptionWrapper:     sshOptionWrapper,
		SkipTargetCleanup:    skipTargetCleanup,
		SkipSourceCleanup:    skipSourceCleanup,
		SkipTargetImport:     skipTargetImport,
		TransferResourceName: namedTransferResource,
	})

	if err != nil {
		utils.LogFatalError("There was an error running the sync process", err)
	}

	if !dryRun {
		log.Printf("\n------\nSuccessful sync of %s from %s to %s\n------", SyncerType, sourceEnvironment.GetOpenshiftProjectName(), targetEnvironment.GetOpenshiftProjectName())
	}
}

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

var toFileCmd = &cobra.Command{
	Use:   "to-file",
	Short: "Sync to a file",
	Long:  "Sync/Dump to a file to produce a resource dump",
	Run: func(cmd *cobra.Command, args []string) {
		fileFlag, _ := cmd.Flags().GetString("transfer-resource-name")
		fmt.Println("You are about to dump to:", fileFlag)

		// set the skipTargetImport and skipTargetCleanup flags to true
		cmd.Parent().PersistentFlags().Set("skip-target-import", "true")
		cmd.Parent().PersistentFlags().Set("skip-source-cleanup", "true")
		cmd.Parent().PersistentFlags().Set("skip-target-cleanup", "true")

		syncCommandRun(cmd, args)
	},
}

var fromFileCmd = &cobra.Command{
	Use:   "from-file",
	Short: "Sync from a file",
	Long:  "Sync from a file and perform related actions",
	Run: func(cmd *cobra.Command, args []string) {
		fileFlag, _ := cmd.Flags().GetString("transfer-resource-name")
		fmt.Println("You are about to import from:", fileFlag)

		cmd.Parent().PersistentFlags().Set("skip-source-run", "true")
		cmd.Parent().PersistentFlags().Set("skip-source-cleanup", "true")
		cmd.Parent().PersistentFlags().Set("skip-target-cleanup", "true")

		syncCommandRun(cmd, args)
	},
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
	syncCmd.PersistentFlags().BoolVar(&noCliInteraction, "no-interaction", false, "Disallow interaction")
	syncCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Don't run the commands, just preview what will be run")
	syncCmd.PersistentFlags().StringVarP(&RsyncArguments, "rsync-args", "r", "--omit-dir-times --no-perms --no-group --no-owner --chmod=ugo=rwX --recursive --compress", "Pass through arguments to change the behaviour of rsync")
	syncCmd.PersistentFlags().BoolVar(&skipSourceRun, "skip-source-run", false, "Don't run any ops on the source")
	syncCmd.PersistentFlags().BoolVar(&skipSourceCleanup, "skip-source-cleanup", false, "Don't clean up any of the files generated on the source")
	syncCmd.PersistentFlags().BoolVar(&skipTargetCleanup, "skip-target-cleanup", false, "Don't clean up any of the files generated on the target")
	syncCmd.PersistentFlags().BoolVar(&skipTargetImport, "skip-target-import", false, "This will skip the import step on the target, in combination with 'no-target-cleanup' this essentially produces a resource dump")
	syncCmd.PersistentFlags().StringVarP(&apiEndpoint, "api", "A", "", "Specify your lagoon api endpoint - required for ssh-portal integration")
	syncCmd.PersistentFlags().BoolVar(&useSshPortal, "use-ssh-portal", false, "This will use the SSH Portal rather than the (soon to be removed) SSH Service on Lagoon core. Will become default in a future release.")
	syncCmd.PersistentFlags().StringVarP(&namedTransferResource, "transfer-resource-name", "f", "", "The name of the temporary file to be used to transfer generated resources (db dumps, etc) - random /tmp file otherwise")

	syncCmd.AddCommand(toFileCmd)
	syncCmd.AddCommand(fromFileCmd)

	// By default, we hook up the syncers.RunSyncProcess function to the runSyncProcess variable
	// by doing this, it lets us easily override it for testing the command - but for most of the time
	// this should be okay.
	runSyncProcess = synchers.RunSyncProcess
}
