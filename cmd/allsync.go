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

var dockerComposeFileAllSync string

// SyncTask represents a single resource (DB or volume) to be synced
type SyncTask struct {
	Type       string // "mariadb", "postgres", "files"
	Service    utils.Service
	VolumePath string // only populated for files
	Label      string // human-readable label for display
}

var allsyncCmd = &cobra.Command{
	Use:   "all-sync",
	Short: "Automated all-in-one sync tool",
	Long:  `Discover and sync all supported database services and volumes from a docker-compose.yml file`,
	Run:   allsyncCommandRun,
}

func allsyncCommandRun(cmd *cobra.Command, args []string) {
	// Default to docker-compose.yml in current directory if not specified
	path := dockerComposeFileAllSync
	if path == "" {
		path = "docker-compose.yml"
	}

	services, err := utils.LoadDockerCompose(path)
	if err != nil {
		utils.LogFatalError(fmt.Sprintf("Failed to load docker-compose file: %v", err), nil)
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

	// Select service to run from (default to 'cli')
	runService, ok := services["cli"]
	if !ok {
		// If no cli service, let user select
		runService, err = selectServiceFromList(services, "Select service to use to do the transfer (typically your 'cli' service)", []string{})
		if err != nil {
			utils.LogFatalError(err.Error(), nil)
		}
	}

	// Build source and target environments
	sourceEnvironment, targetEnvironment := buildEnvironments(ProjectName, runService.Name, sourceEnvironmentName, targetEnvironmentName)

	sshOptions := buildSSHOptions(configRoot, SSHHost, SSHPort, SSHKey, SSHVerbose, SSHSkipAgent, RsyncArguments)
	sshOptionWrapper, err := buildSSHOptionWrapper(ProjectName, sshOptions, configRoot, APIEndpoint, useSshPortal)
	if err != nil {
		utils.LogFatalError(fmt.Sprintf("Failed to configure SSH options: %v", err), nil)
	}

	// Add assertion - we no longer support Remote to Remote syncs
	if sourceEnvironment.EnvironmentName != synchers.LOCAL_ENVIRONMENT_NAME && targetEnvironment.EnvironmentName != synchers.LOCAL_ENVIRONMENT_NAME {
		utils.LogFatalError("Remote to Remote transfers are not supported", nil)
	}

	// Discover all sync tasks
	tasks, err := discoverSyncTasks(services, runService)
	if err != nil {
		utils.LogFatalError(fmt.Sprintf("Failed to discover sync tasks: %v", err), nil)
	}

	if len(tasks) == 0 {
		utils.LogWarning("No sync tasks discovered. Check your services and volumes.", nil)
		return
	}

	if !noCliInteraction {
		// Display and confirm tasks
		if !confirmSyncTasks(tasks) {
			utils.LogWarning("Sync cancelled by user", nil)
			return
		}
	}

	// Execute all syncs sequentially with error collection
	results := executeSyncTasks(tasks, sourceEnvironment, targetEnvironment, sshOptionWrapper)

	// Report results
	reportSyncResults(results)
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

// confirmSyncTasks displays tasks and asks for confirmation
func confirmSyncTasks(tasks []SyncTask) bool {
	fmt.Println("\n==================== SYNC TASKS ====================")
	fmt.Printf("Will sync %d resources:\n\n", len(tasks))

	for i, task := range tasks {
		fmt.Printf("  [%d] %s\n", i+1, task.Label)
	}

	fmt.Println("\n===================================================\n")

	options := []huh.Option[string]{
		huh.NewOption("Proceed with sync", "yes"),
		huh.NewOption("Cancel", "no"),
	}

	var selected string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Continue?").
				Options(options...).
				Value(&selected),
		),
	)

	if err := form.Run(); err != nil {
		return false
	}

	return selected == "yes"
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
		err = synchers.RunSyncProcess(syncArgs)

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

func init() {
	rootCmd.AddCommand(allsyncCmd)
	allsyncCmd.Flags().StringVarP(&dockerComposeFileAllSync, "docker-compose-file", "f", "", "Path to docker-compose.yml (defaults to docker-compose.yml in current directory)")
	allsyncCmd.PersistentFlags().StringVarP(&ProjectName, "project-name", "p", "", "The Lagoon project name of the remote system")
	allsyncCmd.PersistentFlags().StringVarP(&sourceEnvironmentName, "source-environment-name", "e", "", "The Lagoon environment name of the source system")
	allsyncCmd.MarkPersistentFlagRequired("source-environment-name")
	allsyncCmd.PersistentFlags().StringVarP(&targetEnvironmentName, "target-environment-name", "t", "", "The target environment name (defaults to local)")
	allsyncCmd.PersistentFlags().StringVarP(&ServiceName, "service-name", "s", "", "The service name (default is 'cli'")
	allsyncCmd.PersistentFlags().StringVarP(&SSHHost, "ssh-host", "H", "ssh.lagoon.amazeeio.cloud", "Specify your lagoon ssh host, defaults to 'ssh.lagoon.amazeeio.cloud'")
	allsyncCmd.PersistentFlags().StringVarP(&SSHPort, "ssh-port", "P", "32222", "Specify your ssh port, defaults to '32222'")
	allsyncCmd.PersistentFlags().StringVarP(&SSHKey, "ssh-key", "i", "", "Specify path to a specific SSH key to use for authentication")
	allsyncCmd.PersistentFlags().BoolVar(&SSHSkipAgent, "ssh-skip-agent", false, "Do not attempt to use an ssh-agent for key management")
	allsyncCmd.PersistentFlags().BoolVar(&SSHVerbose, "verbose", false, "Run ssh commands in verbose (useful for debugging)")
	allsyncCmd.PersistentFlags().BoolVarP(&noCliInteraction, "no-interaction", "y", false, "Disallow interaction")
	allsyncCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Don't run the commands, just preview what will be run")
	allsyncCmd.PersistentFlags().StringVarP(&RsyncArguments, "rsync-args", "r", "--omit-dir-times --no-perms --no-group --no-owner --chmod=ugo=rwX --recursive --compress", "Pass through arguments to change the behaviour of rsync")
	allsyncCmd.PersistentFlags().BoolVar(&skipSourceCleanup, "skip-source-cleanup", false, "Don't clean up any of the files generated on the source")
	allsyncCmd.PersistentFlags().BoolVar(&skipTargetCleanup, "skip-target-cleanup", false, "Don't clean up any of the files generated on the target")
	allsyncCmd.PersistentFlags().BoolVar(&skipTargetImport, "skip-target-import", false, "This will skip the import step on the target, in combination with 'no-target-cleanup' this essentially produces a resource dump")
	allsyncCmd.PersistentFlags().StringVarP(&namedTransferResource, "transfer-resource-name", "", "", "The name of the temporary file to be used to transfer generated resources (db dumps, etc) - random /tmp file otherwise")
	allsyncCmd.PersistentFlags().StringVarP(&APIEndpoint, "api", "A", "https://api.lagoon.amazeeio.cloud/graphql", "Specify your lagoon api endpoint - required for ssh-portal integration")
	allsyncCmd.PersistentFlags().BoolVar(&useSshPortal, "use-ssh-portal", false, "This will use the SSH Portal rather than the (soon to be removed) SSH Service on Lagoon core. Will become default in a future release.")
}
