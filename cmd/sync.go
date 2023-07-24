package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	synchers "github.com/uselagoon/lagoon-sync/synchers"
	"github.com/uselagoon/lagoon-sync/utils"
	"github.com/uselagoon/machinery/api/lagoon"
	lclient "github.com/uselagoon/machinery/api/lagoon/client"
	"github.com/uselagoon/machinery/utils/sshtoken"
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
var CmdSSHKey string
var noCliInteraction bool
var dryRun bool
var verboseSSH bool
var RsyncArguments string
var runSyncProcess synchers.RunSyncProcessFunctionType
var skipSourceCleanup bool
var skipTargetCleanup bool
var skipTargetImport bool
var SkipAPI bool
var localTransferResourceName string
var rsyncArgDefaults = "--omit-dir-times --no-perms --no-group --no-owner --chmod=ugo=rwX --recursive --compress"

var syncCmd = &cobra.Command{
	Use:   "sync [mariadb|files|mongodb|postgres|etc.]",
	Short: "Sync a resource type",
	Long:  `Use Lagoon-Sync to sync an external environments resources with the local environment`,
	Args:  cobra.MinimumNArgs(1),
	Run:   syncCommandRun,
}

type Sync struct {
	Source      synchers.Environment
	Target      synchers.Environment
	Type        string
	Config      synchers.SyncherConfigRoot
	EnableDebug bool
}

func syncCommandRun(cmd *cobra.Command, args []string) {
	Sync := Sync{}
	SyncerType := args[0]
	viper.Set("syncer-type", args[0])
	Sync.Type = SyncerType

	lagoonConfigBytestream, err := LoadLagoonConfig(cfgFile)
	if err != nil {
		utils.LogFatalError("Couldn't load lagoon config file - ", err.Error())
	}

	configRoot, err := synchers.UnmarshallLagoonYamlToLagoonSyncStructure(lagoonConfigBytestream)
	if err != nil {
		log.Fatalf("There was an issue unmarshalling the sync configuration from %v: %v", viper.ConfigFileUsed(), err)
	}
	Sync.Config = configRoot

	// Set LagoonAPI defaults
	Sync.Config.LagoonAPI = synchers.LagoonAPI{
		Endpoint: "https://api.lagoon.amazeeio.cloud/graphql",
		SSHKey:   "~/$HOME/.ssh/id_rsa",
		SSHHost:  "ssh.lagoon.amazeeio.cloud",
		SSHPort:  "32222",
	}

	// Override defaults with config from yaml
	if configRoot.LagoonAPI.Endpoint != "" {
		Sync.Config.LagoonAPI.Endpoint = configRoot.LagoonAPI.Endpoint
	}
	if configRoot.LagoonAPI.SSHKey != "" {
		Sync.Config.LagoonAPI.SSHKey = configRoot.LagoonAPI.SSHKey
	}
	if configRoot.LagoonAPI.SSHHost != "" {
		Sync.Config.LagoonAPI.SSHHost = configRoot.LagoonAPI.SSHHost
	}
	if configRoot.LagoonAPI.SSHPort != "" {
		Sync.Config.LagoonAPI.SSHPort = configRoot.LagoonAPI.SSHPort
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
		SSH:             synchers.SSHOptions{},
	}
	// We assume that the target environment is local if it's not passed as an argument
	if targetEnvironmentName == "" {
		targetEnvironmentName = synchers.LOCAL_ENVIRONMENT_NAME
	}
	targetEnvironment := synchers.Environment{
		ProjectName:     ProjectName,
		EnvironmentName: targetEnvironmentName,
		ServiceName:     ServiceName,
		SSH:             synchers.SSHOptions{},
	}

	var lagoonSyncer synchers.Syncer

	// Syncers are registered in their init() functions - so here we attempt to match
	// the syncer type with the argument passed through to this command
	// (e.g. if we're running `lagoon-sync sync mariadb --...options follow` the function
	// GetSyncersForTypeFromConfigRoot will return a prepared mariadb syncher object)
	lagoonSyncer, err = synchers.GetSyncerForTypeFromConfigRoot(SyncerType, Sync.Config)
	if err != nil {
		utils.LogFatalError(err.Error(), nil)
	}

	if ProjectName == "" {
		utils.LogFatalError("No Project name given", nil)
	}

	// SSH options will be set from lagoon.yaml config, or overridden by given cli arguments.
	sourceEnvironment.SSH = Sync.GetSSHOptions(ProjectName, sourceEnvironment.EnvironmentName, Sync.Config)
	utils.LogDebugInfo("Config that is used for source SSH", sourceEnvironment.SSH)
	targetEnvironment.SSH = Sync.GetSSHOptions(ProjectName, targetEnvironment.EnvironmentName, Sync.Config)
	utils.LogDebugInfo("Config that is used for target SSH", targetEnvironment.SSH)

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

	err = runSyncProcess(synchers.RunSyncProcessFunctionTypeArguments{
		SourceEnvironment: sourceEnvironment,
		TargetEnvironment: targetEnvironment,
		LagoonSyncer:      lagoonSyncer,
		SyncerType:        SyncerType,
		DryRun:            dryRun,
		SkipTargetCleanup: skipTargetCleanup,
		SkipSourceCleanup: skipSourceCleanup,
		SkipTargetImport:  skipTargetImport,
	})

	if err != nil {
		utils.LogFatalError("There was an error running the sync process", err)
	}

	if !dryRun {
		log.Printf("\n------\nSuccessful sync of %s from %s to %s\n------", SyncerType, sourceEnvironment.GetOpenshiftProjectName(), targetEnvironment.GetOpenshiftProjectName())
	}
}

func (s *Sync) GetSSHOptions(project string, environment string, configRoot synchers.SyncherConfigRoot) synchers.SSHOptions {
	sshConfig := synchers.SSHOptions{}

	// SSH Config from file
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
	rsyncArgs := RsyncArguments
	if sshConfig.RsyncArgs != "" && RsyncArguments == rsyncArgDefaults {
		rsyncArgs = sshConfig.RsyncArgs
	}

	// Check lagoon api for ssh config
	sshOptions, err := s.fetchSSHPortalConfigFromAPI(project, environment, &synchers.SSHOptions{
		Host:       sshHost,
		PrivateKey: sshKey,
		Port:       sshPort,
		Verbose:    sshVerbose,
		RsyncArgs:  rsyncArgs,
	})
	if err != nil {
		utils.LogFatalError(err.Error(), nil)
	}

	return *sshOptions
}

func (s *Sync) fetchSSHPortalConfigFromAPI(project string, environment string, sshConfig *synchers.SSHOptions) (*synchers.SSHOptions, error) {
	if SkipAPI {
		return sshConfig, nil
	}

	// Grab a lagoon token
	token, err := sshtoken.RetrieveToken(s.Config.LagoonAPI.SSHKey, s.Config.LagoonAPI.SSHHost, s.Config.LagoonAPI.SSHPort)
	if err != nil {
		log.Println(fmt.Sprintf("ERROR: unable to generate token: %v", err))
	}

	lc := lclient.New(s.Config.LagoonAPI.Endpoint, "lagoon-sync", &token, false)
	ctx := context.TODO()
	p, err := lagoon.GetSSHEndpointsByProject(ctx, project, lc)
	if err != nil {
		utils.LogFatalError(err.Error(), nil)
	}

	if p.Environments != nil {
		for _, e := range p.Environments {
			if e.Name == environment {
				return &synchers.SSHOptions{
					Host: e.DeployTarget.SSHHost,
					Port: e.DeployTarget.SSHPort,
				}, nil
			}
		}
	}

	return sshConfig, nil
}

func getServiceName(SyncerType string) string {
	if SyncerType == "mongodb" {
		return SyncerType
	}
	return "cli"
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
	syncCmd.PersistentFlags().BoolVar(&SSHVerbose, "verbose", false, "Run ssh commands in verbose (useful for debugging)")
	syncCmd.PersistentFlags().BoolVar(&noCliInteraction, "no-interaction", false, "Disallow interaction")
	syncCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Don't run the commands, just preview what will be run")
	syncCmd.PersistentFlags().StringVarP(&RsyncArguments, "rsync-args", "r", rsyncArgDefaults, "Pass through arguments to change the behaviour of rsync")
	syncCmd.PersistentFlags().BoolVar(&skipSourceCleanup, "skip-source-cleanup", false, "Don't clean up any of the files generated on the source")
	syncCmd.PersistentFlags().BoolVar(&skipTargetCleanup, "skip-target-cleanup", false, "Don't clean up any of the files generated on the target")
	syncCmd.PersistentFlags().BoolVar(&skipTargetImport, "skip-target-import", false, "This will skip the import step on the target, in combination with 'no-target-cleanup' this essentially produces a resource dump")
	syncCmd.PersistentFlags().BoolVar(&SkipAPI, "skip-api", false, "This will skip checking the api for configuration and instead use the defaults")

	// By default, we hook up the syncers.RunSyncProcess function to the runSyncProcess variable
	// by doing this, it lets us easily override it for testing the command - but for most of the time
	// this should be okay.
	runSyncProcess = synchers.RunSyncProcess
}
