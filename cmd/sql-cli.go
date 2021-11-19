package cmd

import (
	"fmt"
	"github.com/amazeeio/lagoon-sync/preflight"
	"github.com/amazeeio/lagoon-sync/synchers"
	"github.com/amazeeio/lagoon-sync/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"os"
	"os/exec"
	"strings"
)

var sqlCliCmd = &cobra.Command{
	Use:   "sql-cli",
	Short: "Connect to mysql cli",
	Long:  "Connect to mysql cli if available ",
	Run: func(v *cobra.Command, args []string) {
		if len(args) == 0 {
			log.Fatalln("No syncer type given. For example, pass 'lagoon-sync sql-cli mariadb' as a command.")
		}

		SyncerType := args[0]
		if SyncerType != "mariadb" {
			log.Fatalln("Only 'mariadb' syncer is supported currently.")
		}

		lagoonConfigBytestream, err := LoadLagoonConfig(cfgFile)
		if err != nil {
			utils.LogFatalError("Couldn't load lagoon config file - ", err.Error())
		}

		configRoot, err := synchers.UnmarshallLagoonYamlToLagoonSyncStructure(lagoonConfigBytestream)
		if err != nil {
			log.Fatalf("There was an issue unmarshalling the sync configuration from %v: %v", viper.ConfigFileUsed(), err)
		}

		var dbConfig map[interface{}]interface{}
		if lagoonSyncConfig, ok := configRoot.LagoonSync[SyncerType].(map[interface{}]interface{}); ok {
			for k, config := range lagoonSyncConfig {
				// only interested in remote config for now - passing in a local flag to access local config would be a good addition here.
				if k == "config" {
					dbConfig = config.(map[interface{}]interface{})
				}
			}
		}

		var sqlConnectionCommand = preflight.MysqlConnectionCommand(
			fmt.Sprintf("%s", dbConfig["database"]),
			fmt.Sprintf("%s", dbConfig["hostname"]),
			fmt.Sprintf("%s", dbConfig["port"]),
			fmt.Sprintf("%s", dbConfig["username"]),
			fmt.Sprintf("%s", dbConfig["password"]),
		)
		if ProjectName == "" {
			project, exists := os.LookupEnv("LAGOON_PROJECT")
			if exists {
				ProjectName = strings.Replace(project, "_", "-", -1)
			}
			if configRoot.Project != "" {
				ProjectName = configRoot.Project
			}
		}

		sourceEnvironment := synchers.Environment{
			ProjectName:     ProjectName,
			EnvironmentName: sourceEnvironmentName,
			ServiceName:     ServiceName,
		}

		remoteCommand := synchers.GenerateRemoteCommand(sourceEnvironment, sqlConnectionCommand, verboseSSH)
		utils.LogExecutionStep("Running the following against source", remoteCommand)

		cmd := exec.Command("bash", "-c", remoteCommand)
		err = cmd.Run()
		if err != nil {
			log.Fatalln(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(sqlCliCmd)
	rootCmd.PersistentFlags().StringVarP(&ProjectName, "project-name", "p", "", "The Lagoon project name of the remote system")
	rootCmd.PersistentFlags().StringVarP(&sourceEnvironmentName, "source-environment-name", "e", "", "The Lagoon environment name of the source system")
	rootCmd.MarkPersistentFlagRequired("source-environment-name")
	rootCmd.PersistentFlags().BoolVar(&noCliInteraction, "no-interaction", false, "Disallow interaction")
	rootCmd.PersistentFlags().BoolVar(&verboseSSH, "verbose", false, "Run ssh commands in verbose (useful for debugging)")
}