package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/bomoko/lagoon-sync/synchers"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var ProjectName string
var sourceEnvironmentName string
var targetEnvironmentName string
var configurationFile string
var noCliInteraction bool
var dryRun bool


// syncCmd represents the sync command
var syncCmd = &cobra.Command{
	Use:   "sync [mariadb|files|mongodb|postgres|etc.]",
	Short: "Sync a resource type",
	Long:  `Use Lagoon-Sync to sync an external environments resources with the local environment`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		moduleName := args[0]

		//For now, let's just try write up a command that generates the strings ...

		//Perhaps we should refactor this into some generic thing ...
		lagoonConfigBytestream, err := LoadLagoonConfig(configurationFile)
		if err != nil {
			log.Println("Couldn't load lagoon config file")
			os.Exit(1)
		}

		sourceEnvironment := synchers.Environment{
			ProjectName:     ProjectName,
			EnvironmentName: sourceEnvironmentName,
		}

		//We assume that the target environment is local if it's not passed as an argument
		if targetEnvironmentName == "" {
			targetEnvironmentName = synchers.LOCAL_ENVIRONMENT_NAME
		}
		targetEnvironment := synchers.Environment{
			ProjectName:     ProjectName,
			EnvironmentName: targetEnvironmentName,
		}

		//TODO: we need some standard way of extracting the project name
		// For now, let's just pull it straight from the .lagoon.yml

		configRoot, err := synchers.UnmarshallLagoonYamlToLagoonSyncStructure(lagoonConfigBytestream)
		if err != nil {
			log.Printf("There was an issue unmarshalling the sync configuration: %v", err)
			return
		}

		var lagoonSyncer synchers.Syncer
		//TODO: perhaps there's a more dynamic way of doing this match?
		switch moduleName {
		case "mariadb":
			lagoonSyncer = configRoot.LagoonSync.Mariadb.PrepareSyncer()
			break
		case "postgres":
			lagoonSyncer = configRoot.LagoonSync.Postgres.PrepareSyncer()
			break
		case "drupalconfig":
			lagoonSyncer = configRoot.LagoonSync.Drupalconfig.PrepareSyncer()
			break
		case "files":
			lagoonSyncer = configRoot.LagoonSync.Filesconfig.PrepareSyncer()
			break
		default:
			log.Print("Could not match type : %v", moduleName)
			return
			break
		}

		if noCliInteraction == false {
			confirmationResult, err := confirmPrompt(fmt.Sprintf("Project: %s - you are about to sync %s from %s to %s, is this correct?",
				ProjectName,
				moduleName,
				sourceEnvironmentName, targetEnvironmentName))
			if err != nil || confirmationResult == false {
				log.Printf("User cancelled sync - exiting")
				os.Exit(1)
			}
		}

		err = synchers.RunSyncProcess(sourceEnvironment, targetEnvironment, lagoonSyncer, dryRun)

		if err != nil {
			log.Printf("There was an error running the sync process: %v", err)
			return
		}
	},
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

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// syncCmd.PersistentFlags().String("foo", "", "A help for foo")
	syncCmd.PersistentFlags().StringVar(&ProjectName, "project-name", "", "The Lagoon project name of the remote system")
	syncCmd.MarkPersistentFlagRequired("project-name")
	syncCmd.PersistentFlags().StringVar(&sourceEnvironmentName, "source-environment-name", "", "The Lagoon environment name of the source system")
	syncCmd.MarkPersistentFlagRequired("source-environment-name")
	syncCmd.PersistentFlags().StringVar(&targetEnvironmentName, "target-environment-name", "", "The Lagoon environment name of the source system (defaults to local)")
	syncCmd.PersistentFlags().StringVar(&configurationFile, "configuration-file", "./.lagoon.yml", "File containing sync configuration. Defaults to ./.lagoon.yml")
	syncCmd.MarkPersistentFlagRequired("remote-environment-name")

	syncCmd.PersistentFlags().BoolVar(&noCliInteraction, "no-interaction", false, "Disallow interaction")

	syncCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Don't run the commands, just preview what will be run")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// syncCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
