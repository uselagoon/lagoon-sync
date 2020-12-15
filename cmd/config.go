package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type configuration struct {
	Version             string `json:"version"`
	ConfigFile          string `json:"config-file-active"`
	DefaultConfigFile   string `json:"default-config-path"`
	LagoonSynConfigFile string `json:"lagoon-sync-path"`
}

func init() {
	rootCmd.AddCommand(configCmd)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Print the onfig that is being used by lagoon-sync",
	Run: func(v *cobra.Command, args []string) {
		PrintConfigOut()
	},
}

func PrintConfigOut() {
	defaultCfgFile, exists := os.LookupEnv("LAGOON_SYNC_DEFAULTS_PATH")
	if !exists {
		defaultCfgFile = "false"
	}
	lagoonSyncCfgFile, exists := os.LookupEnv("LAGOON_SYNC_PATH")
	if !exists {
		lagoonSyncCfgFile = "false"
	}

	config := configuration{Version: rootCmd.Version, ConfigFile: viper.ConfigFileUsed(), DefaultConfigFile: defaultCfgFile, LagoonSynConfigFile: lagoonSyncCfgFile}
	configJSON, err := json.MarshalIndent(config, "", " ")
	if err != nil {
		log.Fatalf(err.Error())
	}
	fmt.Println(string(configJSON))
}
