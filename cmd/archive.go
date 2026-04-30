package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	synchers "github.com/uselagoon/lagoon-sync/synchers"
	"github.com/uselagoon/lagoon-sync/utils"
)

var archiveFile, archiveInputFile string
var extractionRoot string
var overrideVolumes []string
var useServiceApi bool

// fileExtractionIgnoreList holds file/directory names (matched against
// ExtractError.Name) that are acceptable to skip during file extraction.
var fileExtractionIgnoreList = []string{
	".lagoon-rootless-migration-complete",
}

var archiveCmd = &cobra.Command{
	Use:   "archive",
	Short: "Archive resources from an environment",
	Long: `Archive resources from a Lagoon environment.

This command allows you to create archives of databases, files, 
or other resources from a specified environment.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// NOTE - we run PersistenPreRunE here to explicitly override the
		// config run. Since we don't use or need any of it, it partcularly
		// on archive we don't want it to force the creation of a lagoon.yml
		// file.
		fmt.Println("Running archive")
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {

		var services map[string]utils.Service

		if useServiceApi {
			// TODO: implement service API path
			return fmt.Errorf("--use-service-api is not yet implemented")
		} else {
			serviceMap, err := utils.GetServices(dockerComposeFile)
			if err != nil {
				utils.LogFatalError(err.Error(), nil)
			}
			services = serviceMap
		}

		if len(services) == 0 {
			utils.LogFatalError(fmt.Sprint("No Lagoon Services found"), nil)
		}

		// let's begin by doing the mariadb and postgres services only
		// so a nice simple case
		skipVolumeAutodiscovery := len(overrideVolumes) > 0
		runService := utils.Service{
			Name:    "",
			Volumes: map[string]string{},
		}

		if !skipVolumeAutodiscovery { // we're actually not going to look at the defined items anyways, so skip
			rs, ok := services[ServiceName]
			if !ok {
				utils.LogFatalError(fmt.Sprintf("Unable to locate run service in dockerfile: %v", ServiceName), nil)
			}
			runService = rs
		}

		tasks, err := discoverSyncTasks(services, runService, skipVolumeAutodiscovery, false)

		if err != nil {
			utils.LogFatalError(err.Error(), nil)
		}

		// This will _only_ run locally - and so we set up the environments thusly
		environment := synchers.Environment{
			ProjectName:     "",
			EnvironmentName: synchers.LOCAL_ENVIRONMENT_NAME,
			ServiceName:     ServiceName,
		}

		// okay - we got here, we may need a temporary directory
		dirname, err := os.MkdirTemp(os.TempDir(), "lagoon-sync-archive-*")
		if err != nil {
			utils.LogFatalError(err.Error(), nil)
		}
		defer os.RemoveAll(dirname)

		archive, err := utils.InitArchive(archiveFile, rootCmd.Version)

		if err != nil {
			utils.LogFatalError(err.Error(), nil)
		}

		areVolumesOverridden := false
		if len(overrideVolumes) > 0 {
			areVolumesOverridden = true
			for _, volumePath := range overrideVolumes {
				err = archive.AddItem("files", volumePath, nil)
				if err != nil {
					utils.LogFatalError(err.Error(), nil)
				}
			}
		}

		for _, task := range tasks {
			switch task.Type {
			case "mariadb":
				// let's do the dump here and then

				s, err := synchers.NewBaseMariaDbSyncRootFromService(task.Service)
				if err != nil {
					utils.LogFatalError(err.Error(), nil)
				}

				transferResourceName := fmt.Sprintf("mysql-%v.sql.gz", task.Service.Name)
				s.SetTransferResource(filepath.Join(dirname, transferResourceName))
				// We can simply run the source command directly.
				err = synchers.SyncRunSourceCommand(environment, s, false, nil)
				if err != nil {
					utils.LogFatalError(err.Error(), nil)
				}
				//we'll save the syncer detail for reloading on the other side
				syncerJson, err := json.Marshal(s)
				if err != nil {
					utils.LogFatalError(err.Error(), nil)
				}
				err = archive.AddItem("mariadb", s.GetTransferResource(environment).Name, map[string]string{
					"syncer": string(syncerJson),
				})
				if err != nil {
					utils.LogFatalError(err.Error(), nil)
				}
			case "postgres":
				s, err := synchers.NewBasePostgresSyncRootFromService(task.Service)
				if err != nil {
					utils.LogFatalError(err.Error(), nil)
				}

				transferResourceName := fmt.Sprintf("postgres-%v.sql.gz", task.Service.Name)
				s.SetTransferResource(filepath.Join(dirname, transferResourceName))
				// We can simply run the source command directly.
				err = synchers.SyncRunSourceCommand(environment, s, false, nil)
				if err != nil {
					utils.LogFatalError(err.Error(), nil)
				}
				//we'll save the syncer detail for reloading on the other side
				syncerJson, err := json.Marshal(s)
				if err != nil {
					utils.LogFatalError(err.Error(), nil)
				}
				err = archive.AddItem("postgres", s.GetTransferResource(environment).Name, map[string]string{
					"syncer": string(syncerJson),
				})
				if err != nil {
					utils.LogFatalError(err.Error(), nil)
				}
			case "files":
				// this should be the simplest, we just add it to the archive
				if !areVolumesOverridden {
					err = archive.AddItem("files", task.VolumePath, nil)
					if err != nil {
						utils.LogFatalError(err.Error(), nil)
					}
				}
			}

		}

		err = archive.WriteArchive()

		if err != nil {
			utils.LogFatalError(err.Error(), nil)
		}

		return nil
	},
}

var extractCmd = &cobra.Command{
	Use:   "extract",
	Short: "Extracts archive resources from an environment",
	Long: `Extracts archive resources from a Lagoon environment.

This command allows you to extract archives of databases, files, 
or other resources from a specified environment.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// NOTE - we run PersistenPreRunE here to explicitly override the
		// config run. Since we don't use or need any of it, it parcularly
		// on archive we don't want it to force the creation of a lagoon.yml
		// file.
		fmt.Println("Running extract")
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if archiveInputFile == "" {
			cmd.Help()
			return fmt.Errorf("--archive-input is required")
		}

		if useServiceApi {
			// TODO: implement service API path
			return fmt.Errorf("--use-service-api is not yet implemented")
		}

		// let's pull the manifest out of this thing.
		manifest, err := utils.ExtractManifest(archiveInputFile)

		if err != nil {
			utils.LogFatalError(err.Error(), nil)
		}

		// we'll set up a temp dir for extraction / working
		tmpdir, err := os.MkdirTemp(os.TempDir(), "lagoon-sync-extract-*")

		if err != nil {
			utils.LogFatalError("Unable to create a temporary directory", nil)
		}

		defer os.RemoveAll(tmpdir)

		environment := synchers.Environment{
			ProjectName:     "",
			EnvironmentName: synchers.LOCAL_ENVIRONMENT_NAME,
			ServiceName:     "",
		}

		for _, item := range manifest.Items {
			switch item.Syncer {
			case "mariadb":
				var s synchers.MariadbSyncRoot
				var data string
				var ok bool
				// grab the syncer data from the manifest
				if data, ok = item.Data["syncer"]; !ok {
					if data, ok = item.Data["syncher"]; !ok {
						utils.LogFatalError("Unable to find syncer data for mariadb service", nil)
					}
				}

				err = json.Unmarshal([]byte(data), &s)
				if err != nil {
					utils.LogFatalError(err.Error(), nil)
				}

				// We'll want to remove the leading `/` from this

				err = utils.ExtractFromArchive(archiveInputFile, item.Filename, tmpdir, true, nil)
				if err != nil {
					utils.LogFatalError(err.Error(), nil)
				}

				s.TransferResourceOverride = filepath.Join(tmpdir, s.TransferResourceOverride)
				err = synchers.SyncRunTargetCommand(environment, &s, dryRun, nil)
				if err != nil {
					utils.LogFatalError(err.Error(), nil)
				}
			case "postgres":
				var s synchers.PostgresSyncRoot
				var data string
				var ok bool
				if data, ok = item.Data["syncer"]; !ok {
					if data, ok = item.Data["syncher"]; !ok {
						utils.LogFatalError("Unable to find syncer data for postgres service", nil)
					}
				}

				err = json.Unmarshal([]byte(data), &s)
				if err != nil {
					utils.LogFatalError(err.Error(), nil)
				}

				err = utils.ExtractFromArchive(archiveInputFile, item.Filename, tmpdir, true, nil)
				if err != nil {
					utils.LogFatalError(err.Error(), nil)
				}

				s.TransferResourceOverride = filepath.Join(tmpdir, s.TransferResourceOverride)
				err = synchers.SyncRunTargetCommand(environment, &s, dryRun, nil)
				if err != nil {
					utils.LogFatalError(err.Error(), nil)
				}
			case "files":

				err = utils.ExtractFromArchive(archiveInputFile, item.Filename, extractionRoot, true, fileExtractionIgnoreList)

				if err != nil {
					utils.LogFatalError(err.Error(), nil)
				}
			}
		}

		return nil
	},
}

func preRunSetSSHDetailsFromEnvars(cmd *cobra.Command, args []string) {
	if v, exists := os.LookupEnv("LAGOON_CONFIG_API_HOST"); exists {
		fmt.Print("Setting endpoint to " + APIEndpoint)
		APIEndpoint = v + "/graphql"
		fmt.Print("Setting endpoint to " + APIEndpoint)
	}
	if v, exists := os.LookupEnv("LAGOON_CONFIG_SSH_HOST"); exists {
		SSHHost = v
	}
	if v, exists := os.LookupEnv("LAGOON_CONFIG_SSH_PORT"); exists {
		SSHPort = v
	}
}

func init() {
	rootCmd.AddCommand(archiveCmd)
	rootCmd.AddCommand(extractCmd)

	// Add flags for archive
	archiveCmd.Flags().StringVarP(&dockerComposeFile, "docker-compose-file", "f", "", "Path to docker-compose.yml (defaults to docker-compose.yml in current directory)")
	archiveCmd.Flags().StringVarP(&archiveFile, "archive-output", "", "archive.tar.gz", "Name of output archive")
	archiveCmd.Flags().BoolVar(&useServiceApi, "use-service-api", false, "Use the Lagoon service API for lookups")
	archiveCmd.Flags().StringArrayVar(&overrideVolumes, "override-volume", []string{}, "Override volume paths (repeatable)")
	archiveCmd.Flags().StringVarP(&ServiceName, "service-name", "s", "cli", "The service name to run archive commands in (default is 'cli')")
	archiveCmd.PersistentFlags().StringVarP(&SSHHost, "ssh-host", "H", "ssh.lagoon.amazeeio.cloud", "Specify your lagoon ssh host, defaults to 'ssh.lagoon.amazeeio.cloud'")
	archiveCmd.PersistentFlags().StringVarP(&SSHPort, "ssh-port", "P", "32222", "Specify your ssh port, defaults to '32222'")
	archiveCmd.PersistentFlags().StringVarP(&SSHKey, "ssh-key", "i", "", "Specify path to a specific SSH key to use for authentication")
	archiveCmd.PersistentFlags().StringVarP(&APIEndpoint, "api", "A", "https://api.lagoon.amazeeio.cloud/graphql", "Specify your lagoon api endpoint - required for ssh-portal integration")

	// Add flags for extract
	extractCmd.Flags().StringVarP(&archiveInputFile, "archive-input", "", "", "Name of input archive")
	extractCmd.Flags().BoolVar(&useServiceApi, "use-service-api", false, "Use the Lagoon service API for lookups")
	extractCmd.Flags().StringVarP(&extractionRoot, "extraction-root", "", "/", "Root path for file extraction")
	extractCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Don't run the commands, just preview what will be run")
	extractCmd.PersistentFlags().StringVarP(&SSHHost, "ssh-host", "H", "ssh.lagoon.amazeeio.cloud", "Specify your lagoon ssh host, defaults to 'ssh.lagoon.amazeeio.cloud'")
	extractCmd.PersistentFlags().StringVarP(&SSHPort, "ssh-port", "P", "32222", "Specify your ssh port, defaults to '32222'")
	extractCmd.PersistentFlags().StringVarP(&SSHKey, "ssh-key", "i", "", "Specify path to a specific SSH key to use for authentication")
	extractCmd.PersistentFlags().StringVarP(&APIEndpoint, "api", "A", "https://api.lagoon.amazeeio.cloud/graphql", "Specify your lagoon api endpoint - required for ssh-portal integration")
}
