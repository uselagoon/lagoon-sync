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

		// outputdir := "/tmp/" // for now.

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
			}
		}

		// gonna just set up one of these suckers for testing

		// archive, err := utils.InitArchive("/tmp/test.tar.gz")

		// if err != nil {
		// 	os.Exit(1)
		// }

		// archive.AddItem("files", "/tmp/file1.txt", map[string]string{})
		// archive.AddItem("files", "/tmp/file2.txt", map[string]string{})
		// err = archive.WriteArchive()

		// if err != nil {
		// 	fmt.Print(err.Error())
		// 	os.Exit(1)
		// }

	},
}

func init() {
	rootCmd.AddCommand(archiveCmd)
	archiveCmd.Flags().StringVarP(&dockerComposeFile, "docker-compose-file", "f", "", "Path to docker-compose.yml (defaults to docker-compose.yml in current directory)")
}
