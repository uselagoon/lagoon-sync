package cmd

import (
	// "fmt"
	// "os"

	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	synchers "github.com/uselagoon/lagoon-sync/synchers"
	"github.com/uselagoon/lagoon-sync/utils"
	// "github.com/uselagoon/lagoon-sync/utils"
)

var archiveFile string

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
		dirname, err := os.MkdirTemp("./", "")
		if err != nil {
			utils.LogFatalError(err.Error(), nil)
		}
		defer os.RemoveAll(dirname)

		archive, err := utils.InitArchive(archiveFile)

		if err != nil {
			utils.LogFatalError(err.Error(), nil)
		}

		for _, task := range tasks {
			// fmt.Println(task)
			switch task.Type {
			case "mariadb":
				// let's do the dump here and then

				s, err := synchers.NewBaseMariaDbSyncRootFromService(task.Service)
				if err != nil {
					utils.LogFatalError(err.Error(), nil)
				}

				s.SetTransferResource(filepath.Join(dirname, "mysql.sql.gz"))
				// We can simply run the source command directly.
				err = synchers.SyncRunSourceCommand(environment, s, false, nil)
				archive.AddItem("mariadb", s.GetTransferResource(environment).Name, nil)
			case "postgres":
				s, err := synchers.NewBasePostgresSyncRootFromService(task.Service)
				if err != nil {
					utils.LogFatalError(err.Error(), nil)
				}

				s.SetTransferResource(filepath.Join(dirname, "postgres.sql.gz"))
				// We can simply run the source command directly.
				err = synchers.SyncRunSourceCommand(environment, s, false, nil)
				archive.AddItem("postgres", s.GetTransferResource(environment).Name, nil)
			case "files":
				// this should be the simplest, we just add it to the archive
				archive.AddItem("files", task.VolumePath, nil)
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

		for _, item := range manifest.Items {
			switch item.Syncher {
			case "mariadb":
				// we extract the file into a temp location, then do a restore
				err = utils.ExtractFromArchive(archiveFile, item.Filename, tmpdir)
				if err != nil {
					utils.LogFatalError(err.Error(), nil)
				}
				archiveName := filepath.Join(tmpdir, item.Filename)

				// now we go ahead and do the restore side of the synhcer

			}
		}

	},
}

func init() {
	rootCmd.AddCommand(archiveCmd)
	rootCmd.AddCommand(extractCmd)
	archiveCmd.Flags().StringVarP(&dockerComposeFile, "docker-compose-file", "f", "", "Path to docker-compose.yml (defaults to docker-compose.yml in current directory)")
	archiveCmd.Flags().StringVarP(&archiveFile, "archive-output", "", "archive.tar.gz", "Name of output archive")
	extractCmd.Flags().StringVarP(&archiveFile, "archive-input", "", "", "Name of input archive")
}
