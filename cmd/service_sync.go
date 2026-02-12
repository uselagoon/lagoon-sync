package cmd

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	synchers "github.com/uselagoon/lagoon-sync/synchers"
	"github.com/uselagoon/lagoon-sync/utils"
)

var dockerComposeFile string
var sersyncListOnly bool

// We use this to filter the standard service types we can sync.
var supportedSynchableServicetypes = []string{
	"mariadb",
	"mariadb-single",
	"mariadb-dbaas",
	"postgres",
	"postgres-single",
	"postgres-dbaas",
}

var serviceCmd = &cobra.Command{
	Use:   "service-sync",
	Short: "Automated service based sync tool",
	Long:  `List or sync all services and their volumes from a docker-compose.yml file or services api`,
	Run:   servicesCommandRun,
}

func servicesCommandRun(cmd *cobra.Command, args []string) {
	// Default to docker-compose.yml in current directory if not specified
	path := dockerComposeFile
	if path == "" {
		path = "docker-compose.yml"
	}

	services, err := utils.LoadDockerCompose(path)
	if err != nil {
		utils.LogFatalError(fmt.Sprintf("Failed to load docker-compose file: %v", err), nil)
	}

	if sersyncListOnly {
		prettyPrintServiceOutput(services)
		return
	}

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

	runService, err := selectServiceFromList(services, "Select service to use to do the transfer (typically your 'cli' service)", []string{})
	if err != nil {
		utils.LogFatalError(err.Error(), nil)
	}

	// Build source and target environments
	sourceEnvironment, targetEnvironment := buildEnvironments(ProjectName, runService.Name, sourceEnvironmentName, targetEnvironmentName)

	sshOptions := buildSSHOptions(configRoot, SSHHost, SSHPort, SSHKey, SSHVerbose, SSHSkipAgent, RsyncArguments)
	// Build SSH option wrapper with optional SSH portal integration
	sshOptionWrapper, err := buildSSHOptionWrapper(ProjectName, sshOptions, configRoot, APIEndpoint, useSshPortal)
	if err != nil {
		utils.LogFatalError(fmt.Sprintf("Failed to configure SSH options: %v", err), nil)
	}
	_ = sshOptionWrapper
	// Add assertion - we no longer support Remote to Remote syncs
	if sourceEnvironment.EnvironmentName != synchers.LOCAL_ENVIRONMENT_NAME && targetEnvironment.EnvironmentName != synchers.LOCAL_ENVIRONMENT_NAME {
		utils.LogFatalError("Remote to Remote transfers are not supported", nil)
	}

	// ask whether the user wants to sync files or databases
	syncType := "databases"
	if len(runService.Volumes) > 0 {
		syncType, err = selectSyncType()
		if err != nil {
			utils.LogFatalError(err.Error(), nil)
		}
	}

	// Now we offer the final menu of services
	var lagoonSyncer synchers.Syncer
	switch syncType {
	case ("files"):
		// let's select the volume to move
		selectedVolume, err := selectVolume(runService.Volumes)
		if err != nil {
			utils.LogFatalError(err.Error(), nil)
		}
		fmt.Printf("Gonna sync %v \n", selectedVolume)

		// okay, now we can actually invoke the synch
		lagoonSyncer, err = synchers.NewBaseFilesSyncRootFromService(runService, selectedVolume)
		if err != nil {
			utils.LogFatalError(err.Error(), nil)
		}
		fmt.Print(lagoonSyncer)

	default:
		// let's select a DB service to transfer
		var err error
		syncService, err := selectServiceFromList(services, "Select service to sync", supportedSynchableServicetypes)
		if err != nil {
			utils.LogFatalError(err.Error(), nil)
		}
		fmt.Printf("Gonna sync %v\n", syncService.Name)

		switch {
		case strings.Contains(syncService.Type, "mariadb"):
			lagoonSyncer, err = synchers.NewBaseMariaDbSyncRootFromService(syncService)
			fmt.Print(lagoonSyncer)
			if err != nil {
				utils.LogFatalError(err.Error(), nil)
			}
		case strings.Contains(syncService.Type, "postgresql"):
			lagoonSyncer, err = synchers.NewBasePostgresSyncRootFromService(syncService)
			fmt.Print(lagoonSyncer)
			if err != nil {
				utils.LogFatalError(err.Error(), nil)
			}
		}
	}

	if !noCliInteraction {

		// We'll set the spinner utility to show
		utils.SetShowSpinner(true)

		// Ask for confirmation
		confirmationResult, err := confirmPrompt(fmt.Sprintf("Project: %s - you are about to sync %s from %s to %s, is this correct",
			ProjectName,
			"TODO SET SERVICE TYPE",
			sourceEnvironment.EnvironmentName, targetEnvironment.EnvironmentName))
		utils.SetColour(true)
		if err != nil || !confirmationResult {
			utils.LogFatalError("User cancelled sync - exiting", nil)
		}
	}

	debugPrintSyncArgs(synchers.RunSyncProcessFunctionTypeArguments{
		SourceEnvironment:    sourceEnvironment,
		TargetEnvironment:    targetEnvironment,
		LagoonSyncer:         lagoonSyncer,
		SyncerType:           SyncerType,
		DryRun:               dryRun,
		SshOptionWrapper:     sshOptionWrapper,
		SkipTargetCleanup:    skipTargetCleanup,
		SkipSourceCleanup:    skipSourceCleanup,
		SkipTargetImport:     skipTargetImport,
		TransferResourceName: namedTransferResource,
	})

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

func selectServiceFromList(services map[string]utils.Service, title string, filterList []string) (utils.Service, error) {

	options := []huh.Option[string]{}

	// build an ordered list of service names so selection is deterministic
	names := make([]string, 0, len(services))

	for name, _ := range services {
		names = append(names, name)
	}
	sort.Strings(names)

	// Collect `cli` options first (preserving alphabetical order), then others.
	cliOpts := make([]huh.Option[string], 0, 1)
	otherOpts := make([]huh.Option[string], 0, len(names))
	for _, name := range names {
		svc := services[name]
		opt := huh.NewOption(fmt.Sprintf("%v - %v", name, svc.Type), name)

		if len(filterList) > 0 {
			if !utils.SliceContains(filterList, svc.Type) {
				continue
			}
		}

		if svc.Type == "cli" {
			cliOpts = append(cliOpts, opt)
		} else {
			// right now the only other types we support are mariadb and postgres
			otherOpts = append(otherOpts, opt)
		}
	}

	options = append(cliOpts, otherOpts...)

	var selected string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().Title(title).Options(options...).Value(&selected),
		),
	)

	err := form.Run()
	if err != nil {
		return utils.Service{}, err
	}

	return services[selected], nil

}

func selectSyncType() (string, error) {
	options := []huh.Option[string]{
		huh.NewOption("Files", "files"),
		huh.NewOption("Databases", "databases"),
	}

	var selected string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select what to sync").
				Options(options...).
				Value(&selected),
		),
	)

	if err := form.Run(); err != nil {
		return "", err
	}
	return selected, nil
}

func selectVolume(volumeMap map[string]string) (string, error) {
	// volumes := make([]string, len(volumeMap))
	volumes := []string{}
	for _, v := range volumeMap {
		volumes = append(volumes, v)
	}
	sort.Strings(volumes)
	options := []huh.Option[string]{}
	for _, vol := range volumes {
		options = append(options, huh.NewOption(vol, vol))
	}

	var selected string
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select which volume to sync").
				Options(options...).
				Value(&selected),
		),
	).Run()
	if err != nil {
		return "", err
	}
	return selected, nil
}

func prettyPrintServiceOutput(services map[string]utils.Service) {
	// Output the services
	for name, svc := range services {
		fmt.Printf("Service: %s\n", name)
		fmt.Printf("  Type: %s\n", svc.Type)
		if len(svc.Volumes) > 0 {
			fmt.Printf("  Volumes:\n")
			for vol, path := range svc.Volumes {
				fmt.Printf("    %s: %s\n", vol, path)
			}
		}
	}
}

// debugPrintSyncArgs prints a formatted debug output of RunSyncProcessFunctionTypeArguments
func debugPrintSyncArgs(args synchers.RunSyncProcessFunctionTypeArguments) {
	fmt.Println("\n==================== SYNC PROCESS DEBUG ====================")

	// Source Environment
	fmt.Println("\n--- Source Environment ---")
	fmt.Printf("  Project Name:      %s\n", args.SourceEnvironment.ProjectName)
	fmt.Printf("  Environment Name:  %s\n", args.SourceEnvironment.EnvironmentName)
	fmt.Printf("  Service Name:      %s\n", args.SourceEnvironment.ServiceName)
	fmt.Printf("  OpenShift Project: %s\n", args.SourceEnvironment.GetOpenshiftProjectName())
	fmt.Printf("  Rsync Available:   %v\n", args.SourceEnvironment.RsyncAvailable)
	fmt.Printf("  Rsync Path:        %s\n", args.SourceEnvironment.RsyncPath)
	fmt.Printf("  Rsync Local Path:  %s\n", args.SourceEnvironment.RsyncLocalPath)

	// Target Environment
	fmt.Println("\n--- Target Environment ---")
	fmt.Printf("  Project Name:      %s\n", args.TargetEnvironment.ProjectName)
	fmt.Printf("  Environment Name:  %s\n", args.TargetEnvironment.EnvironmentName)
	fmt.Printf("  Service Name:      %s\n", args.TargetEnvironment.ServiceName)
	fmt.Printf("  OpenShift Project: %s\n", args.TargetEnvironment.GetOpenshiftProjectName())
	fmt.Printf("  Rsync Available:   %v\n", args.TargetEnvironment.RsyncAvailable)
	fmt.Printf("  Rsync Path:        %s\n", args.TargetEnvironment.RsyncPath)
	fmt.Printf("  Rsync Local Path:  %s\n", args.TargetEnvironment.RsyncLocalPath)

	// SSH Options
	if args.SshOptionWrapper != nil {
		fmt.Println("\n--- SSH Options Wrapper ---")
		fmt.Printf("  Project Name: %s\n", args.SshOptionWrapper.ProjectName)

		fmt.Println("\n  Default SSH Options:")
		printSSHOptions(args.SshOptionWrapper.Default, "    ")

		if len(args.SshOptionWrapper.Options) > 0 {
			fmt.Println("\n  Environment-specific SSH Options:")
			for envName, opts := range args.SshOptionWrapper.Options {
				fmt.Printf("    [%s]:\n", envName)
				printSSHOptions(opts, "      ")
			}
		}
	} else {
		fmt.Println("\n--- SSH Options Wrapper ---")
		fmt.Println("  <nil>")
	}

	// Syncer Info
	fmt.Println("\n--- Syncer Configuration ---")
	fmt.Printf("  Syncer Type:           %s\n", args.SyncerType)
	fmt.Printf("  Dry Run:               %v\n", args.DryRun)
	fmt.Printf("  Skip Source Cleanup:   %v\n", args.SkipSourceCleanup)
	fmt.Printf("  Skip Target Cleanup:   %v\n", args.SkipTargetCleanup)
	fmt.Printf("  Skip Target Import:    %v\n", args.SkipTargetImport)
	fmt.Printf("  Transfer Resource:     %s\n", args.TransferResourceName)

	if args.LagoonSyncer != nil {
		fmt.Printf("  Lagoon Syncer:         %T\n", args.LagoonSyncer)
	} else {
		fmt.Printf("  Lagoon Syncer:         <nil>\n")
	}

	fmt.Println("\n============================================================\n")
}

// printSSHOptions is a helper to format SSH options with indentation
func printSSHOptions(opts synchers.SSHOptions, indent string) {
	fmt.Printf("%sHost:        %s\n", indent, opts.Host)
	fmt.Printf("%sPort:        %s\n", indent, opts.Port)
	fmt.Printf("%sVerbose:     %v\n", indent, opts.Verbose)
	fmt.Printf("%sPrivate Key: %s\n", indent, opts.PrivateKey)
	fmt.Printf("%sSkip Agent:  %v\n", indent, opts.SkipAgent)
	fmt.Printf("%sRsync Args:  %s\n", indent, opts.RsyncArgs)
}

func init() {
	rootCmd.AddCommand(serviceCmd)
	serviceCmd.Flags().StringVarP(&dockerComposeFile, "docker-compose-file", "f", "", "Path to docker-compose.yml (defaults to docker-compose.yml in current directory)")
	serviceCmd.Flags().BoolVarP(&sersyncListOnly, "list-only", "l", false, "only display service sync options (default false)")
	serviceCmd.PersistentFlags().StringVarP(&ProjectName, "project-name", "p", "", "The Lagoon project name of the remote system")
	serviceCmd.PersistentFlags().StringVarP(&sourceEnvironmentName, "source-environment-name", "e", "", "The Lagoon environment name of the source system")
	serviceCmd.MarkPersistentFlagRequired("source-environment-name")
	serviceCmd.PersistentFlags().StringVarP(&targetEnvironmentName, "target-environment-name", "t", "", "The target environment name (defaults to local)")
	serviceCmd.PersistentFlags().StringVarP(&ServiceName, "service-name", "s", "", "The service name (default is 'cli'")
	serviceCmd.MarkPersistentFlagRequired("remote-environment-name")
	serviceCmd.PersistentFlags().StringVarP(&SSHHost, "ssh-host", "H", "ssh.lagoon.amazeeio.cloud", "Specify your lagoon ssh host, defaults to 'ssh.lagoon.amazeeio.cloud'")
	serviceCmd.PersistentFlags().StringVarP(&SSHPort, "ssh-port", "P", "32222", "Specify your ssh port, defaults to '32222'")
	serviceCmd.PersistentFlags().StringVarP(&SSHKey, "ssh-key", "i", "", "Specify path to a specific SSH key to use for authentication")
	serviceCmd.PersistentFlags().BoolVar(&SSHSkipAgent, "ssh-skip-agent", false, "Do not attempt to use an ssh-agent for key management")
	serviceCmd.PersistentFlags().BoolVar(&SSHVerbose, "verbose", false, "Run ssh commands in verbose (useful for debugging)")
	serviceCmd.PersistentFlags().BoolVarP(&noCliInteraction, "no-interaction", "y", false, "Disallow interaction")
	serviceCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Don't run the commands, just preview what will be run")
	serviceCmd.PersistentFlags().StringVarP(&RsyncArguments, "rsync-args", "r", "--omit-dir-times --no-perms --no-group --no-owner --chmod=ugo=rwX --recursive --compress", "Pass through arguments to change the behaviour of rsync")
	serviceCmd.PersistentFlags().BoolVar(&skipSourceCleanup, "skip-source-cleanup", false, "Don't clean up any of the files generated on the source")
	serviceCmd.PersistentFlags().BoolVar(&skipTargetCleanup, "skip-target-cleanup", false, "Don't clean up any of the files generated on the target")
	serviceCmd.PersistentFlags().BoolVar(&skipTargetImport, "skip-target-import", false, "This will skip the import step on the target, in combination with 'no-target-cleanup' this essentially produces a resource dump")
	serviceCmd.PersistentFlags().StringVarP(&namedTransferResource, "transfer-resource-name", "", "", "The name of the temporary file to be used to transfer generated resources (db dumps, etc) - random /tmp file otherwise")
	serviceCmd.PersistentFlags().StringVarP(&APIEndpoint, "api", "A", "https://api.lagoon.amazeeio.cloud/graphql", "Specify your lagoon api endpoint - required for ssh-portal integration")
	serviceCmd.PersistentFlags().BoolVar(&useSshPortal, "use-ssh-portal", false, "This will use the SSH Portal rather than the (soon to be removed) SSH Service on Lagoon core. Will become default in a future release.")
}
