package cmd

import (
	"fmt"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/uselagoon/lagoon-sync/synchers"
	"github.com/uselagoon/lagoon-sync/utils"
	"log"
	"os"
	"strings"
)

var backupCmd = &cobra.Command{
	Use:   "backup [mariadb|files|mongodb|postgres|etc.]",
	Short: "Backup a resource type",
	Long:  `Use Lagoon-Sync to pull from an external environment to the local environment`,
	Args:  cobra.MinimumNArgs(1),
	Run:   backupCmdRun,
}

var outputFilename string

func backupCmdRun(cmd *cobra.Command, args []string) {

	SyncerType := args[0]
	viper.Set("syncer-type", args[0])

	lagoonConfigBytestream, err := LoadLagoonConfig(cfgFile)
	if err != nil {
		utils.LogFatalError("Couldn't load lagoon config file - ", err.Error())
	}

	configRoot, err := synchers.UnmarshallLagoonYamlToLagoonSyncStructure(lagoonConfigBytestream)
	if err != nil {
		log.Fatalf("There was an issue unmarshalling the sync configuration from %v: %v", viper.ConfigFileUsed(), err)
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
	lagoonSyncer, err = synchers.GetSyncerForTypeFromConfigRoot(SyncerType, configRoot)
	if err != nil {
		utils.LogFatalError(err.Error(), nil)
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
	sshOptions := synchers.SSHOptions{
		Host:       sshHost,
		PrivateKey: sshKey,
		Port:       sshPort,
		Verbose:    sshVerbose,
		RsyncArgs:  RsyncArguments,
	}

	if namedTransferResource == "" {
		utils.LogFatalError("You need to specify a backup file name using `--output-file=<filename>", nil)
	} else {
		err = lagoonSyncer.SetTransferResource(namedTransferResource)
		if err != nil {
			utils.LogFatalError(err.Error(), nil)
		}
	}

	utils.LogDebugInfo("Config that is used for SSH", sshOptions)

	result, err := runSyncProcess(synchers.RunSyncProcessFunctionTypeArguments{
		SourceEnvironment:    sourceEnvironment,
		TargetEnvironment:    targetEnvironment,
		LagoonSyncer:         lagoonSyncer,
		SyncerType:           SyncerType,
		DryRun:               dryRun,
		SshOptions:           sshOptions,
		SkipTargetCleanup:    true,
		SkipSourceCleanup:    skipSourceCleanup,
		SkipTargetImport:     true,
		TransferResourceName: namedTransferResource,
	})

	if err != nil {
		utils.LogFatalError("There was an error running the sync process", err)
	}

	if len(result.RemainingArtifacts) != 1 {
		utils.LogFatalError("No local copy of the transfer file found. Backup failed", nil)
	}

	// finally we copy the temp item to the given location

	r, err := os.Open(result.RemainingArtifacts[0])
	if err != nil {
		panic(err)
	}
	defer r.Close()
	w, err := os.Create(outputFilename)
	if err != nil {
		panic(err)
	}
	defer w.Close()
	_, err = w.ReadFrom(r)

	if err != nil {
		utils.LogFatalError(err.Error(), nil)
	}

	e := os.Remove(result.RemainingArtifacts[0])
	if e != nil {
		utils.LogFatalError(err.Error(), nil)
	}

	if !dryRun {
		log.Printf("\n------\nSuccessful backup of %s from %s to %s\n------", SyncerType, sourceEnvironment.GetOpenshiftProjectName(), targetEnvironment.GetOpenshiftProjectName())
	}
}

func init() {
	rootCmd.AddCommand(backupCmd)
	backupCmd.PersistentFlags().StringVarP(&ProjectName, "project-name", "p", "", "The Lagoon project name of the remote system")
	backupCmd.PersistentFlags().StringVarP(&sourceEnvironmentName, "source-environment-name", "e", "", "The Lagoon environment name of the source system")
	backupCmd.MarkPersistentFlagRequired("source-environment-name")
	backupCmd.PersistentFlags().StringVarP(&targetEnvironmentName, "target-environment-name", "t", "", "The target environment name (defaults to local)")
	backupCmd.PersistentFlags().StringVarP(&ServiceName, "service-name", "s", "", "The service name (default is 'cli'")
	backupCmd.MarkPersistentFlagRequired("remote-environment-name")
	backupCmd.PersistentFlags().StringVarP(&SSHHost, "ssh-host", "H", "ssh.lagoon.amazeeio.cloud", "Specify your lagoon ssh host, defaults to 'ssh.lagoon.amazeeio.cloud'")
	backupCmd.PersistentFlags().StringVarP(&SSHPort, "ssh-port", "P", "32222", "Specify your ssh port, defaults to '32222'")
	backupCmd.PersistentFlags().StringVarP(&SSHKey, "ssh-key", "i", "", "Specify path to a specific SSH key to use for authentication")
	backupCmd.PersistentFlags().BoolVar(&SSHVerbose, "verbose", false, "Run ssh commands in verbose (useful for debugging)")
	backupCmd.PersistentFlags().BoolVar(&noCliInteraction, "no-interaction", false, "Disallow interaction")
	backupCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Don't run the commands, just preview what will be run")
	backupCmd.PersistentFlags().StringVarP(&RsyncArguments, "rsync-args", "r", "--omit-dir-times --no-perms --no-group --no-owner --chmod=ugo=rwX --recursive --compress", "Pass through arguments to change the behaviour of rsync")
	backupCmd.PersistentFlags().StringVarP(&outputFilename, "output-file", "o", "", "Target file")

	// By default, we hook up the syncers.RunSyncProcess function to the runSyncProcess variable
	// by doing this, it lets us easily override it for testing the command - but for most of the time
	// this should be okay.
	runSyncProcess = synchers.RunSyncProcess
}
