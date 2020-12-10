package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/spf13/cobra"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var version string
var cfgFile string
var defaultCfgFile string
var lagoonSyncCfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "lagoon-sync",
	Short:   "Sync resources between Lagoon hosted environment",
	Long:    `lagoon-sync is a tool for syncing resources between environments in Lagoon hosted applications. This includes files, databases, and configurations.`,
	Version: Version(),
}

// Read version from .version, this will get updated automatically on release.
func Version() string {
	pwd, _ := os.Getwd()
	version, err := ioutil.ReadFile(pwd + "/.version")
	if err != nil {
		log.Print(err)
	}

	return string(version)
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

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Find home directory.
	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	// Search config in home directory with name ".lagoon-sync" (without extension).
	viper.AddConfigPath(home)

	viper.SetConfigType("yaml")

	// Find default config file for env vars (e.g. 'lagoon-sync-defaults')
	defaultCfgFile, exists := os.LookupEnv("LAGOON_SYNC_DEFAULTS_PATH")
	if exists {
		log.Println("Default config file path set: ", defaultCfgFile)
	} else {
		defaultCfgFile = "/lagoon/.lagoon-sync-defaults"
	}

	// Find lagoon-sync config file (e.g. 'lagoon-sync')
	lagoonSyncCfgFile, exists := os.LookupEnv("LAGOON_SYNC_PATH")
	if exists {
		log.Println("Lagoon sync config file path set: ", lagoonSyncCfgFile)
	} else {
		lagoonSyncCfgFile = "/lagoon/.lagoon-sync"
	}

	if cfgFile != "" {
		if _, err := os.Stat(cfgFile); err == nil {
			viper.SetConfigName(cfgFile)
			// Use config file from the flag, default for this is '.lagoon.yml'
			viper.SetConfigFile(cfgFile)
		} else if os.IsNotExist(err) {
			if _, err := os.Stat(defaultCfgFile); err == nil {
				viper.SetConfigName(".lagoon-sync-defaults")
				viper.SetConfigFile(defaultCfgFile)
				cfgFile = defaultCfgFile
			}
			if _, err := os.Stat(lagoonSyncCfgFile); err == nil {
				viper.SetConfigName(".lagoon-sync")
				viper.SetConfigFile(lagoonSyncCfgFile)
				cfgFile = lagoonSyncCfgFile
			}
		}
	}
	if defaultCfgFile != "" {
		if _, err := os.Stat(defaultCfgFile); err == nil {
			viper.SetConfigName(".lagoon-sync-defaults")
			// Use the default config file.
			viper.SetConfigFile(defaultCfgFile)
			cfgFile = defaultCfgFile
		}
		if err != nil {
			fmt.Println(err)
		}
	}
	if lagoonSyncCfgFile != "" {
		if _, err := os.Stat(lagoonSyncCfgFile); err == nil {
			viper.SetConfigName(".lagoon-sync")
			viper.SetConfigFile(lagoonSyncCfgFile)
			cfgFile = lagoonSyncCfgFile
		}
		if err != nil {
			fmt.Println(err)
		}
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		log.Println("Using config file:", viper.ConfigFileUsed())
	}
}
