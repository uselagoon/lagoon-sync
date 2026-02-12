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
var allServices bool
var serviceToRunSync string

// We use this to filter the standard service types we can sync.
var supportedSynchableServicetypes = []string{
	"mariadb",
	"mariadb-single",
	"mariadb-dbaas",
	"postgres",
	"postgres-single",
	"postgres-dbaas",
}

// SyncTask represents a single resource (DB or volume) to be synced
type SyncTask struct {
	Type       string // "mariadb", "postgres", "files"
	Service    utils.Service
	VolumePath string // only populated for files
	Label      string // human-readable label for display
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

	// Resolve interaction constraints upfront
	requiresInteraction := serviceToRunSync == "" && !allServices
	if noCliInteraction && requiresInteraction {
		utils.LogFatalError("Cannot run non-interactively without --service-name or --all-services", nil)
	}

	if !noCliInteraction {
		// let's set up the spinners and colors, hooray!
		utils.SetColour(true)
		utils.SetShowSpinner(true)
	}

	// Resolve the run service
	var runService utils.Service
	if serviceToRunSync != "" {
		var ok bool
		if runService, ok = services[serviceToRunSync]; !ok {
			utils.LogFatalError(fmt.Sprintf("Unable to find service '%v' - exiting\n", serviceToRunSync), nil)
		}
	} else {
		runService, err = selectServiceFromList(services, "Select service to use to do the transfer (typically your 'cli' service)", []string{})
		if err != nil {
			utils.LogFatalError(err.Error(), nil)
		}
	}

	// Build source and target environments
	sourceEnvironment, targetEnvironment := buildEnvironments(ProjectName, runService.Name, sourceEnvironmentName, targetEnvironmentName)

	configRootTyped := configRoot
	sshOptions := buildSSHOptions(configRootTyped, SSHHost, SSHPort, SSHKey, SSHVerbose, SSHSkipAgent, RsyncArguments)
	sshOptionWrapper, err := buildSSHOptionWrapper(ProjectName, sshOptions, configRootTyped, APIEndpoint, useSshPortal)
	if err != nil {
		utils.LogFatalError(fmt.Sprintf("Failed to configure SSH options: %v", err), nil)
	}

	// Add assertion - we no longer support Remote to Remote syncs
	if sourceEnvironment.EnvironmentName != synchers.LOCAL_ENVIRONMENT_NAME && targetEnvironment.EnvironmentName != synchers.LOCAL_ENVIRONMENT_NAME {
		utils.LogFatalError("Remote to Remote transfers are not supported", nil)
	}

	// Gather tasks: either discover all, or let user pick one interactively
	var tasks []SyncTask
	if allServices {
		tasks, err = discoverSyncTasks(services, runService)
		if err != nil {
			utils.LogFatalError(fmt.Sprintf("Failed to discover sync tasks: %v", err), nil)
		}
	} else {
		task, err := gatherSingleTask(services, runService)
		if err != nil {
			utils.LogFatalError(err.Error(), nil)
		}
		tasks = []SyncTask{task}
	}

	if len(tasks) == 0 {
		utils.LogWarning("No sync tasks to run.", nil)
		return
	}

	// Execute all tasks through the same path
	results := executeSyncTasks(tasks, sourceEnvironment, targetEnvironment, sshOptionWrapper)

	// Report results
	reportSyncResults(results)
}

// gatherSingleTask uses the interactive menus to build one SyncTask from user selection
func gatherSingleTask(services map[string]utils.Service, runService utils.Service) (SyncTask, error) {
	// ask whether the user wants to sync files or databases
	syncType := "databases"
	if len(runService.Volumes) > 0 {
		var err error
		syncType, err = selectSyncType()
		if err != nil {
			return SyncTask{}, err
		}
	}

	switch syncType {
	case "files":
		selectedVolume, err := selectVolume(runService.Volumes)
		if err != nil {
			return SyncTask{}, err
		}
		return SyncTask{
			Type:       "files",
			Service:    runService,
			VolumePath: selectedVolume,
			Label:      fmt.Sprintf("%s (Files)", selectedVolume),
		}, nil

	default:
		syncService, err := selectServiceFromList(services, "Select service to sync", supportedSynchableServicetypes)
		if err != nil {
			return SyncTask{}, err
		}

		serviceType := ""
		if strings.Contains(syncService.Type, "mariadb") {
			serviceType = "mariadb"
		} else if strings.Contains(syncService.Type, "postgres") {
			serviceType = "postgres"
		}

		return SyncTask{
			Type:    serviceType,
			Service: syncService,
			Label:   fmt.Sprintf("%s (%s)", syncService.Name, strings.ToUpper(serviceType)),
		}, nil
	}
}

// discoverSyncTasks finds all DB services and volumes to sync
func discoverSyncTasks(services map[string]utils.Service, runService utils.Service) ([]SyncTask, error) {
	var tasks []SyncTask

	// Discover database services (from all services matching supported types)
	for name, svc := range services {
		// Skip the run service itself
		if name == runService.Name {
			continue
		}

		// Check if this is a supported DB service type
		if utils.SliceContains(supportedSynchableServicetypes, svc.Type) {
			serviceType := ""
			if strings.Contains(svc.Type, "mariadb") {
				serviceType = "mariadb"
			} else if strings.Contains(svc.Type, "postgres") {
				serviceType = "postgres"
			}

			if serviceType != "" {
				task := SyncTask{
					Type:    serviceType,
					Service: svc,
					Label:   fmt.Sprintf("%s@%s (%s)", name, name, strings.ToUpper(serviceType)),
				}
				tasks = append(tasks, task)
			}
		}
	}

	// Discover file volumes from run service
	for volName, volPath := range runService.Volumes {
		task := SyncTask{
			Type:       "files",
			Service:    runService,
			VolumePath: volPath,
			Label:      fmt.Sprintf("%s → %s (Files)", volName, volPath),
		}
		tasks = append(tasks, task)
	}

	// Sort tasks for deterministic output (DBs first, then files)
	sort.Slice(tasks, func(i, j int) bool {
		// Files last
		if tasks[i].Type == "files" && tasks[j].Type != "files" {
			return false
		}
		if tasks[i].Type != "files" && tasks[j].Type == "files" {
			return true
		}
		// Otherwise sort by label
		return tasks[i].Label < tasks[j].Label
	})

	return tasks, nil
}

// selectSyncTasks presents a multi-select checklist of tasks
func selectSyncTasks(tasks []SyncTask) ([]SyncTask, error) {
	options := make([]huh.Option[int], len(tasks))
	for i, task := range tasks {
		options[i] = huh.NewOption(task.Label, i)
	}

	var selected []int
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[int]().
				Title("Select tasks to sync").
				Options(options...).
				Value(&selected),
		),
	)

	if err := form.Run(); err != nil {
		return nil, err
	}

	var result []SyncTask
	for _, idx := range selected {
		result = append(result, tasks[idx])
	}
	return result, nil
}

// SyncResult holds the outcome of a single sync task
type SyncResult struct {
	Task     SyncTask
	Success  bool
	Error    error
	Duration string // optional, for future use
}

// executeSyncTasks runs all sync tasks sequentially
func executeSyncTasks(tasks []SyncTask, sourceEnv, targetEnv synchers.Environment, sshWrapper *synchers.SSHOptionWrapper) []SyncResult {
	var results []SyncResult

	for _, task := range tasks {
		result := SyncResult{Task: task}

		// Create syncher based on task type
		var syncher synchers.Syncer
		var err error

		switch task.Type {
		case "mariadb":
			syncher, err = synchers.NewBaseMariaDbSyncRootFromService(task.Service)
		case "postgres":
			syncher, err = synchers.NewBasePostgresSyncRootFromService(task.Service)
		case "files":
			syncher, err = synchers.NewBaseFilesSyncRootFromService(task.Service, task.VolumePath)
		}

		if err != nil {
			result.Error = fmt.Errorf("failed to create syncher for %s: %w", task.Label, err)
			results = append(results, result)
			continue
		}

		// Execute sync process
		syncArgs := synchers.RunSyncProcessFunctionTypeArguments{
			SourceEnvironment:    sourceEnv,
			TargetEnvironment:    targetEnv,
			LagoonSyncer:         syncher,
			SyncerType:           SyncerType,
			DryRun:               dryRun,
			SshOptionWrapper:     sshWrapper,
			SkipSourceCleanup:    skipSourceCleanup,
			SkipTargetCleanup:    skipTargetCleanup,
			SkipTargetImport:     skipTargetImport,
			TransferResourceName: namedTransferResource,
		}

		fmt.Printf("\n[SYNCING] %s...\n", task.Label)
		err = runSyncProcess(syncArgs)

		if err != nil {
			result.Error = err
			fmt.Printf("[FAILED] %s: %v\n", task.Label, err)
		} else {
			result.Success = true
			fmt.Printf("[SUCCESS] %s\n", task.Label)
		}

		results = append(results, result)
	}

	return results
}

// reportSyncResults prints the final summary
func reportSyncResults(results []SyncResult) {
	fmt.Println("\n==================== SYNC SUMMARY ====================")

	successCount := 0
	failureCount := 0
	var failures []SyncResult

	for _, result := range results {
		if result.Success {
			successCount++
			fmt.Printf("✓ %s\n", result.Task.Label)
		} else {
			failureCount++
			fmt.Printf("✗ %s\n", result.Task.Label)
			failures = append(failures, result)
		}
	}

	fmt.Printf("\nTotal: %d succeeded, %d failed\n", successCount, failureCount)

	if failureCount > 0 {
		fmt.Println("\n--- Failures ---")
		for _, failure := range failures {
			fmt.Printf("• %s: %v\n", failure.Task.Label, failure.Error)
		}
		fmt.Println("\nNote: Sync process may succeed partially. Review any remaining cleanup needed.")
	}

	fmt.Println("====================================================\n")

	if failureCount > 0 {
		log.Fatalf("Sync completed with %d errors", failureCount)
	} else if !dryRun {
		log.Printf("✓ All %d syncs completed successfully", successCount)
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
	serviceCmd.Flags().BoolVar(&allServices, "all-services", false, "sync all discovered database services and volumes (default false, enables multi-select)")
	serviceCmd.PersistentFlags().StringVarP(&ProjectName, "project-name", "p", "", "The Lagoon project name of the remote system")
	serviceCmd.PersistentFlags().StringVarP(&serviceToRunSync, "run-service", "", "", "The service to use to run the transfer (typically cli)")
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
	runSyncProcess = synchers.RunSyncProcess
}
