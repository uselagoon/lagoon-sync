package cmd

import (
	"bufio"
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
var Api string
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
var namedTransferResource string

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
				configRoot = loadedConfigRoot
			}
		}
	}
	Sync.Config = configRoot

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
	lagoonSyncer, err := synchers.GetSyncerForTypeFromConfigRoot(SyncerType, Sync.Config)
	if err != nil {
		utils.LogFatalError(err.Error(), nil)
	}

	if ProjectName == "" {
		utils.LogFatalError("No Project name given", nil)
	}

	// get api endpoint from config if found
	if configRoot.Api != "" {
		Sync.Config.Api = configRoot.Api
	}

	// if no api endpoint found in config, ask the user if they want to set it
	if !noCliInteraction {
		if configRoot.Api == "" {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("\033[32mWe couldn't find a Lagoon API endpoint in your config.\n\033[0m")
			fmt.Print("\033[32mDo you want to define that now, or use the default ('https://api.lagoon.amazeeio.cloud/graphql')? (yes/no): \033[0m")
			input, err := reader.ReadString('\n')
			if err != nil {
				log.Fatalf("Error reading user input: %v", err)
			}
			input = strings.TrimSpace(strings.ToLower(input))
			if input == "yes" {

				fmt.Print("Enter Lagoon API Endpoint: ")
				endpoint, err := reader.ReadString('\n')
				if err != nil {
					log.Fatalf("Error reading user input: %v", err)
				}
				Sync.Config.Api = strings.TrimSpace(endpoint)
			}
		}
	}

	// use cli arg to override api endpoint if given
	if Api != "" {
		Sync.Config.Api = Api
	}

	sourceEnvironment.SSH = Sync.GetSSHOptions(ProjectName, sourceEnvironment, Sync.Config)
	utils.LogDebugInfo("Config that is used for source SSH", sourceEnvironment.SSH)
	targetEnvironment.SSH = Sync.GetSSHOptions(ProjectName, targetEnvironment, Sync.Config)
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

	// let's update the named transfer resource if it is set
	if namedTransferResource != "" {
		err = lagoonSyncer.SetTransferResource(namedTransferResource)
		if err != nil {
			utils.LogFatalError(err.Error(), nil)
		}
	}

	utils.LogDebugInfo("Config that is used for SSH", sshOptions)

	err = runSyncProcess(synchers.RunSyncProcessFunctionTypeArguments{
		SourceEnvironment:    sourceEnvironment,
		TargetEnvironment:    targetEnvironment,
		LagoonSyncer:         lagoonSyncer,
		SyncerType:           SyncerType,
		DryRun:               dryRun,
		SshOptions:           sshOptions,
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

// Get SSH options based on the following prority:
// 1. CLI arguments given (--ssh-host, --ssh-port, for examples)
// 2. Cluster config variables such as 'LAGOON_CONFIG_SSH_HOST' and 'LAGOON_CONFIG_SSH_HOST'
// 3. Deploy Targets set for the environment from the Lagoon API
// 4. SSH defined fields in any config files (lagoon-sync.ssh, ssh)
// 5. User prompted input if none of the above is found
// 6. CLI defaults as fallback
func (s *Sync) GetSSHOptions(project string, environment synchers.Environment, configRoot synchers.SyncherConfigRoot) synchers.SSHOptions {
	sshConfig := &synchers.SSHOptions{}

	// SSH Config from yaml
	if s.Config.Ssh != "" {
		sshString := strings.Split(s.Config.Ssh, ":")
		if len(sshString) != 2 {
			utils.LogFatalError("Invalid ssh host input format - should match 'host:port'.", nil)
		}

		host := strings.TrimSpace(sshString[0])
		port := strings.TrimSpace(sshString[1])

		sshConfig.Host = host
		sshConfig.Port = port
	}
	if configRoot.LagoonSync["ssh"] != nil {
		mapstructure.Decode(configRoot.LagoonSync["ssh"], &sshConfig)
	}

	// if no ssh config is found, then ask user if they want to set it now or the defaults will be used
	if !noCliInteraction {
		if sshConfig.Host == "" || sshConfig.Port == "" {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("\033[32mWe couldn't find any Lagoon SSH config.\n\033[0m")
			fmt.Print("\033[32mDo you want to define that now, or use the defaults? (yes/no): \033[0m")
			input, err := reader.ReadString('\n')
			if err != nil {
				log.Fatalf("Error reading user input: %v", err)
			}
			input = strings.TrimSpace(strings.ToLower(input))
			if input == "yes" {

				fmt.Print("\033[32mEnter custom LagoonAPI SSHHost: \033[0m")
				sshHost, err := reader.ReadString('\n')
				if err != nil {
					log.Fatalf("Error reading user input: %v", err)
				}
				sshConfig.Host = strings.TrimSpace(sshHost)

				fmt.Print("\033[32mEnter custom LagoonAPI SSHPort: \033[0m")
				sshPort, err := reader.ReadString('\n')
				if err != nil {
					log.Fatalf("Error reading user input: %v", err)
				}
				sshConfig.Port = strings.TrimSpace(sshPort)

				if sshConfig.PrivateKey == "" {
					fmt.Print("\033[32mEnter custom LagoonAPI SSHKey: \033[0m")
					sshKey, err := reader.ReadString('\n')
					if err != nil {
						log.Fatalf("Error reading user input: %v", err)
					}
					sshConfig.PrivateKey = strings.TrimSpace(sshKey)
				}
			}
		}
	}

	// Check lagoon api for ssh config based on deploy targets
	sshConfig, err := s.fetchSSHPortalConfigFromAPI(project, environment.EnvironmentName, &synchers.SSHOptions{
		Host:       sshConfig.Host,
		PrivateKey: sshConfig.PrivateKey,
		Port:       sshConfig.Port,
		Verbose:    sshConfig.Verbose,
		RsyncArgs:  sshConfig.RsyncArgs,
	})
	if err != nil {
		utils.LogFatalError(err.Error(), nil)
	}

	// Check LAGOOON_CONFIG_X env vars
	lagoonSSHHost := os.Getenv("LAGOON_CONFIG_SSH_HOST")
	if lagoonSSHHost != "" {
		sshConfig.Host = lagoonSSHHost
	}

	lagoonSSHPort := os.Getenv("LAGOON_CONFIG_SSH_PORT")
	if lagoonSSHPort != "" {
		sshConfig.Port = lagoonSSHPort
	}

	// cli argument overrides config
	if SSHHost != "" && SSHHost != "ssh.lagoon.amazeeio.cloud" {
		sshConfig.Host = SSHHost
	}
	if SSHPort != "" && SSHPort != "32222" {
		sshConfig.Port = SSHPort
	}
	if SSHKey != "" {
		sshConfig.PrivateKey = SSHKey
	}
	if SSHVerbose {
		sshConfig.Verbose = SSHVerbose
	}
	if RsyncArguments != "" && RsyncArguments != "--omit-dir-times --no-perms --no-group --no-owner --chmod=ugo=rwX --recursive --compress" {
		sshConfig.RsyncArgs = RsyncArguments
	}

	return *sshConfig
}

func (s *Sync) fetchSSHPortalConfigFromAPI(project string, environment string, sshConfig *synchers.SSHOptions) (*synchers.SSHOptions, error) {
	if SkipAPI {
		return sshConfig, nil
	}

	// Grab a lagoon token
	token, err := sshtoken.RetrieveToken(sshConfig.PrivateKey, sshConfig.Host, sshConfig.Port)
	if err != nil {
		log.Println(fmt.Sprintf("ERROR: unable to generate token: %v", err))
		return nil, err
	}

	lc := lclient.New(s.Config.Api, "lagoon-sync", &token, false)
	ctx := context.TODO()
	p, err := lagoon.GetSSHEndpointsByProject(ctx, project, lc)
	if err != nil {
		errMessage := fmt.Sprintf("Failed to get ssh config for '%s' at '%s': ", project, s.Config.Api)
		utils.LogFatalError(errMessage, err.Error())
		return nil, err
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
	syncCmd.PersistentFlags().StringVarP(&Api, "api", "A", "https://api.lagoon.amazeeio.cloud/graphql", "Specify your lagoon api endpoint")
	syncCmd.PersistentFlags().StringVarP(&SSHHost, "ssh-host", "H", "ssh.lagoon.amazeeio.cloud", "Specify your lagoon ssh host")
	syncCmd.PersistentFlags().StringVarP(&SSHPort, "ssh-port", "P", "32222", "Specify your ssh port")
	syncCmd.PersistentFlags().StringVarP(&SSHKey, "ssh-key", "i", "", "Specify path to a specific SSH key to use for authentication")
	syncCmd.PersistentFlags().BoolVar(&SSHVerbose, "verbose", false, "Run ssh commands in verbose (useful for debugging)")
	syncCmd.PersistentFlags().BoolVar(&noCliInteraction, "no-interaction", false, "Disallow interaction")
	syncCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Don't run the commands, just preview what will be run")
	syncCmd.PersistentFlags().StringVarP(&RsyncArguments, "rsync-args", "r", rsyncArgDefaults, "Pass through arguments to change the behaviour of rsync")
	syncCmd.PersistentFlags().BoolVar(&skipSourceCleanup, "skip-source-cleanup", false, "Don't clean up any of the files generated on the source")
	syncCmd.PersistentFlags().BoolVar(&skipTargetCleanup, "skip-target-cleanup", false, "Don't clean up any of the files generated on the target")
	syncCmd.PersistentFlags().BoolVar(&skipTargetImport, "skip-target-import", false, "This will skip the import step on the target, in combination with 'no-target-cleanup' this essentially produces a resource dump")
	syncCmd.PersistentFlags().BoolVar(&SkipAPI, "skip-api", false, "This will skip checking the api for configuration and instead use the defaults")
	syncCmd.PersistentFlags().StringVarP(&namedTransferResource, "transfer-resource-name", "", "", "The name of the temporary file to be used to transfer generated resources (db dumps, etc) - random /tmp file otherwise")

	// By default, we hook up the syncers.RunSyncProcess function to the runSyncProcess variable
	// by doing this, it lets us easily override it for testing the command - but for most of the time
	// this should be okay.
	runSyncProcess = synchers.RunSyncProcess
}
