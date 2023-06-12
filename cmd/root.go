package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

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

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Path to the file used to set lagoon-sync configuration")
	rootCmd.PersistentFlags().BoolVar(&ShowDebug, "show-debug", false, "Shows debug information")
	viper.BindPFlag("show-debug", rootCmd.PersistentFlags().Lookup("show-debug"))

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func initConfig() {
	err := processConfig(cfgFile)
	if err != nil {
		utils.LogFatalError("Unable to read in config file", err)
		os.Exit(1)
	}
}

func processConfig(cfgFile string) error {
	// If cfgFile arg given, return early
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
		viper.AutomaticEnv()

		if err := viper.ReadInConfig(); err == nil {
			utils.LogDebugInfo("Using config file", viper.ConfigFileUsed())
		} else {
			return fmt.Errorf("failed to read config file: %v", err)
		}

		return nil
	}

	home, err := homedir.Dir()
	if err != nil {
		return fmt.Errorf("unable to find home directory: %v", err)
	}

	paths := []string{".", "./lagoon", "/tmp", home}
	for _, path := range paths {
		viper.AddConfigPath(path)
	}
	viper.SetConfigName(cfgFile)
	viper.SetConfigType("yaml")

	// Find config file from env vars (e.g., 'LAGOON_SYNC_DEFAULTS_PATH' and 'LAGOON_SYNC_PATH')
	defaultFiles := map[string]string{
		"LAGOON_SYNC_DEFAULTS_PATH": "/lagoon/.lagoon-sync-defaults",
		"LAGOON_SYNC_PATH":          "/lagoon/.lagoon-sync",
	}

	for envVar, defaultFile := range defaultFiles {
		filePath, exists := os.LookupEnv(envVar)
		if exists {
			utils.LogDebugInfo(envVar+" env var found", filePath)
			if utils.FileExists(filePath) {
				viper.SetConfigFile(filePath)
				cfgFile = filePath
				break
			}
		} else {
			if utils.FileExists(defaultFile) {
				viper.SetConfigFile(defaultFile)
				cfgFile = defaultFile
				break
			}
		}
	}

	// Next, check for 'lagoon.yml' files in the default locations and override.
	for _, path := range paths {
		filePath := filepath.Join(path, ".lagoon.yml")
		if utils.FileExists(filePath) {
			cfgFile = filePath
			break
		}
	}

	// Set the config file if found
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
		viper.AutomaticEnv()

		// If a config file is found, read it in.
		if err := viper.ReadInConfig(); err == nil {
			utils.LogDebugInfo("Using config file", viper.ConfigFileUsed())
		} else {
			return fmt.Errorf("failed to read config file: %v", err)
		}
	} else {
		// If no config file is found, load the default config
		defaultConfigData, err := assets.GetDefaultConfig()
		if err != nil {
			return fmt.Errorf("failed to load default config: %v", err)
		}

		viper.SetConfigType("yaml")
		viper.SetConfigName("default")

		err = viper.ReadConfig(bytes.NewBuffer(defaultConfigData))
		if err != nil {
			return fmt.Errorf("failed to read default config: %v", err)
		}

		// Then safe-write config to '.lagoon.yml' when it doesn't exist
		viper.SafeWriteConfigAs(".lagoon.yml")
		viper.SetConfigFile(".lagoon.yml")
	}

	return nil
}
