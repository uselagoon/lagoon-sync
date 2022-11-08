package cmd

import (
	"encoding/json"
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
var CmdSSHKey string
var noCliInteraction bool
var dryRun bool
var verboseSSH bool
var RsyncArguments string
var runSyncProcess synchers.RunSyncProcessFunctionType
var skipSourceCleanup bool
var skipTargetCleanup bool
var skipTargetImport bool
var skipApi bool
var localTransferResourceName string
var rsyncArgDefaults = "--omit-dir-times --no-perms --no-group --no-owner --chmod=ugo=rwX --recursive --compress"

var syncCmd = &cobra.Command{
	Use:   "sync [mariadb|files|mongodb|postgres|etc.]",
	Short: "Sync a resource type",
	Long:  `Use Lagoon-Sync to sync an external environments resources with the local environment`,
	Args:  cobra.MinimumNArgs(1),
	Run:   syncCommandRun,
}

func syncCommandRun(cmd *cobra.Command, args []string) {

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

	// SSH options will be set from lagoon.yaml config, or overridden by given cli arguments.
	sshOptions := SetSSHOptions(configRoot)
	utils.LogDebugInfo("Config that is used for SSH", *sshOptions)

	err = runSyncProcess(synchers.RunSyncProcessFunctionTypeArguments{
		SourceEnvironment: sourceEnvironment,
		TargetEnvironment: targetEnvironment,
		LagoonSyncer:      lagoonSyncer,
		SyncerType:        SyncerType,
		DryRun:            dryRun,
		SSHOptions:        *sshOptions,
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

func SetSSHOptions(configRoot synchers.SyncherConfigRoot) *synchers.SSHOptions {
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
	sshOptions, err := fetchSSHPortalConfigFromAPI(&synchers.SSHOptions{
		Host:       sshHost,
		PrivateKey: sshKey,
		Port:       sshPort,
		Verbose:    sshVerbose,
		RsyncArgs:  rsyncArgs,
	})
	if err != nil {
		utils.LogFatalError(err.Error(), nil)
	}

	return sshOptions
}

func fetchSSHPortalConfigFromAPI(sshConfig *synchers.SSHOptions) (*synchers.SSHOptions, error) {
	if skipApi == true {
		return sshConfig, nil
	}

	// Lagoon client

	// Call projectByName query to retrieve openshift object
	response, err := projectByName(ProjectName)
	if err != nil {
		utils.LogFatalError(err.Error(), nil)
	}

	var unmarshalJSONResponse ProjectByNameResponse
	err = json.Unmarshal(response, &unmarshalJSONResponse)
	if err != nil {
		fmt.Println("Can't unmarshal api response: ", err.Error())
	}

	if unmarshalJSONResponse.Data.ProjectByName.Openshift.SSHHost != "" {
		return &synchers.SSHOptions{
			Host: unmarshalJSONResponse.Data.ProjectByName.Openshift.SSHHost,
			Port: unmarshalJSONResponse.Data.ProjectByName.Openshift.SSHPort,
		}, nil
	}

	return sshConfig, nil
}

type ProjectByNameResponse struct {
	Data struct {
		ProjectByName struct {
			Openshift struct {
				ID      int    `json:"id"`
				Name    string `json:"name"`
				SSHHost string `json:"sshHost"`
				SSHPort string `json:"sshPort"`
			} `json:"openshift"`
		} `json:"projectByName"`
	} `json:"data"`
}

func projectByName(name string) ([]byte, error) {
	// run api query to get ssh portal config
	return []byte(`{
	  "data": {
		"projectByName": {
		  "openshift": {
			"id": 1,
			"name": "test6.amazee.io",
			"sshHost": "ssh.test6.amazee.io",
			"sshPort": "22"
		  }
		}
	  }
	}`), nil
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
	syncCmd.PersistentFlags().BoolVar(&skipApi, "skip-api", false, "This will skip checking the api for configuration and instead use the defaults")

	// By default, we hook up the syncers.RunSyncProcess function to the runSyncProcess variable
	// by doing this, it lets us easily override it for testing the command - but for most of the time
	// this should be okay.
	runSyncProcess = synchers.RunSyncProcess
}
