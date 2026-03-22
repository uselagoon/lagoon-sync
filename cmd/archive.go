package cmd

import (
	// "fmt"
	// "os"

	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	synchers "github.com/uselagoon/lagoon-sync/synchers"
	"github.com/uselagoon/lagoon-sync/utils"
)

var archiveFile string
var extractionRoot string
var overrideVolumes []string

var archiveCmd = &cobra.Command{
	Use:   "archive",
	Short: "Archive resources from an environment",
	Long: `Archive resources from a Lagoon environment.

This command allows you to create archives of databases, files, 
or other resources from a specified environment.`,
	Run: func(cmd *cobra.Command, args []string) {

		const serviceName = "cli"

		services, err := utils.GetServices(dockerComposeFile)
		if err != nil {
			utils.LogFatalError(err.Error(), nil)
		}

		if len(services) == 0 {
			utils.LogFatalError(fmt.Sprint("No Lagoon Services found"), nil)
		}

		// let's begin by doing the mariadb and postgres services only
		// so a nice simple case

		tasks, err := discoverSyncTasks(services, services[serviceName])

		if err != nil {
			utils.LogFatalError(err.Error(), nil)
		}

		// This will _only_ run locally - and so we set up the environments thusly
		environment := synchers.Environment{
			ProjectName:     "",
			EnvironmentName: synchers.LOCAL_ENVIRONMENT_NAME,
			ServiceName:     serviceName,
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
				archive.AddItem("files", volumePath, nil)
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
				//we'll save the syncher detail for reloading on the other side
				syncherJson, err := json.Marshal(s)
				if err != nil {
					utils.LogFatalError(err.Error(), nil)
				}
				archive.AddItem("mariadb", s.GetTransferResource(environment).Name, map[string]string{
					"syncher": string(syncherJson),
				})
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
				//we'll save the syncher detail for reloading on the other side
				syncherJson, err := json.Marshal(s)
				if err != nil {
					utils.LogFatalError(err.Error(), nil)
				}
				archive.AddItem("postgres", s.GetTransferResource(environment).Name, map[string]string{
					"syncher": string(syncherJson),
				})
			case "files":
				// this should be the simplest, we just add it to the archive
				if !areVolumesOverridden {
					archive.AddItem("files", task.VolumePath, nil)
				}
			}

		}

		err = archive.WriteArchive()

		if err != nil {
			// utils.LogFatalError(err.Error(), nil)
			fmt.Println(err.Error())
		}
	},
}

var extractCmd = &cobra.Command{
	Use:   "extract",
	Short: "Extracts archive resources from an environment",
	Long: `Extracts archive resources from a Lagoon environment.

This command allows you to extract archives of databases, files, 
or other resources from a specified environment.`,
	Run: func(cmd *cobra.Command, args []string) {
		if archiveFile == "" {
			utils.LogFatalError("--archive-input is required as an argument", nil)
		}

		// let's pull the manifest out of this thing.
		manifest, err := utils.ExtractManifest(archiveFile)

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
			switch item.Syncher {
			case "mariadb":
				var s synchers.MariadbSyncRoot
				var data string
				var ok bool
				// grab the syncher data from the manifest
				if data, ok = item.Data["syncher"]; ok != true {
					utils.LogFatalError("Unable to find syncher for mariadb service", nil)
				}

				err = json.Unmarshal([]byte(data), &s)
				if err != nil {
					utils.LogFatalError(err.Error(), nil)
				}

				err = utils.ExtractFromArchive(archiveFile, item.Filename, tmpdir)
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
				if data, ok = item.Data["syncher"]; ok != true {
					utils.LogFatalError("Unable to find syncher for postgres service", nil)
				}

				err = json.Unmarshal([]byte(data), &s)
				if err != nil {
					utils.LogFatalError(err.Error(), nil)
				}

				err = utils.ExtractFromArchive(archiveFile, item.Filename, tmpdir)
				if err != nil {
					utils.LogFatalError(err.Error(), nil)
				}

				s.TransferResourceOverride = filepath.Join(tmpdir, s.TransferResourceOverride)
				err = synchers.SyncRunTargetCommand(environment, &s, dryRun, nil)
				if err != nil {
					utils.LogFatalError(err.Error(), nil)
				}
			case "files":

				err = utils.ExtractFromArchive(archiveFile, item.Filename, extractionRoot)

				if err != nil {
					utils.LogFatalError(err.Error(), nil)
				}
			}
		}

	},
}

func init() {
	rootCmd.AddCommand(archiveCmd)
	rootCmd.AddCommand(extractCmd)
	archiveCmd.Flags().StringVarP(&dockerComposeFile, "docker-compose-file", "f", "", "Path to docker-compose.yml (defaults to docker-compose.yml in current directory)")
	archiveCmd.Flags().StringVarP(&archiveFile, "archive-output", "", "archive.tar.gz", "Name of output archive")
	archiveCmd.Flags().StringArrayVar(&overrideVolumes, "override-volume", []string{}, "Override volume paths (repeatable)")
	extractCmd.Flags().StringVarP(&archiveFile, "archive-input", "", "", "Name of input archive")
	extractCmd.Flags().StringVarP(&extractionRoot, "extraction-root", "", "/", "Root path for file extraction")
	extractCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Don't run the commands, just preview what will be run")
}
