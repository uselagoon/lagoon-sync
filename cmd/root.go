package cmd

import (
	"fmt"
	"os"

	"github.com/uselagoon/lagoon-sync/assets"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/uselagoon/lagoon-sync/utils"
)

var cfgFile string
var lagoonSyncDefaultsFile string
var lagoonSyncCfgFile string
var ShowDebug bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "lagoon-sync",
	Short:   "Sync resources between Lagoon hosted environment",
	Long:    `lagoon-sync is a tool for syncing resources between environments in Lagoon hosted applications. This includes files, databases, and configurations.`,
	Version: Version(),
}

// Version Read version from /assets/.version, this will get updated automatically on release.
func Version() string {
	version := assets.GetVersion()
	return version
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.SetVersionTemplate(Version())

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "./.lagoon.yml", "config file (default is .lagoon.yaml)")
	rootCmd.PersistentFlags().BoolVar(&ShowDebug, "show-debug", false, "Shows debug information")
	viper.BindPFlag("show-debug", rootCmd.PersistentFlags().Lookup("show-debug"))

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func initConfig() {
	// Find home directory.
	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	paths := []string{home, "/lagoon", "/tmp"}

	cfgFile, err = processConfigEnv(paths, cfgFile)
	if err != nil {
		utils.LogFatalError("Unable to read in config file", err)
		os.Exit(1)
	}
}

// initConfig reads in config file and ENV variables if set.
func processConfigEnv(paths []string, DefaultConfigFileName string) (string, error) {

	// Search config in home directory with name ".lagoon-sync" (without extension).
	for _, path := range paths {
		viper.AddConfigPath(path)
	}
	viper.SetConfigName(DefaultConfigFileName)
	viper.SetConfigType("yaml")

	// Find default config file for env vars (e.g. 'lagoon-sync-defaults')
	lagoonSyncDefaultsFile, exists := os.LookupEnv("LAGOON_SYNC_DEFAULTS_PATH")
	if exists {
		utils.LogDebugInfo("LAGOON_SYNC_DEFAULTS_PATH env var found", lagoonSyncDefaultsFile)
	} else {
		lagoonSyncDefaultsFile = "/lagoon/.lagoon-sync-defaults"
	}

	// Find lagoon-sync config file (e.g. 'lagoon-sync')
	lagoonSyncCfgFile, exists := os.LookupEnv("LAGOON_SYNC_PATH")
	if exists {
		utils.LogDebugInfo("LAGOON_SYNC_PATH env var found", lagoonSyncCfgFile)
	} else {
		lagoonSyncCfgFile = "/lagoon/.lagoon-sync"
	}

	if DefaultConfigFileName != "" {
		// Use config file from the flag, default for this is '.lagoon.yml'
		if utils.FileExists(DefaultConfigFileName) {
			viper.SetConfigName(DefaultConfigFileName)
			viper.SetConfigFile(DefaultConfigFileName)
		}

		// Set '.lagoon-sync-defaults' as config file is it exists.
		if utils.FileExists(lagoonSyncDefaultsFile) {
			viper.SetConfigName(lagoonSyncDefaultsFile)
			viper.SetConfigFile(lagoonSyncDefaultsFile)
			DefaultConfigFileName = lagoonSyncDefaultsFile
		}

		// Set '.lagoon-sync' as config file is it exists.
		if utils.FileExists(lagoonSyncCfgFile) {
			viper.SetConfigName(lagoonSyncCfgFile)
			viper.SetConfigFile(lagoonSyncCfgFile)
			DefaultConfigFileName = lagoonSyncCfgFile
		}
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		utils.LogDebugInfo("Using config file", viper.ConfigFileUsed())
	} else {
		return "", err
	}
	return DefaultConfigFileName, nil
}
