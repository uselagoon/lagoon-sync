package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/amazeeio/lagoon-sync/prerequisite"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Configuration struct {
	Version           string                              `json:"version"`
	EnvPrerequisite   []prerequisite.GatheredPrerequisite `json:"env-config"`
	RysncPrequisite   []prerequisite.GatheredPrerequisite `json:"rsync-config"`
	OtherPrerequisite []prerequisite.GatheredPrerequisite `json:"other-config"`
	SyncConfigFiles   SyncConfigFiles                     `json:"sync-config-files"`
}

type SyncConfigFiles struct {
	ConfigFileActive    string `json:"config-file-active"`
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

	// run the prerequsite gatherers
	prerequisiteConfig := prerequisite.GetConfigPrerequisite()
	var rsyncPrerequisites []prerequisite.GatheredPrerequisite
	var envVarPrerequisites []prerequisite.GatheredPrerequisite
	var otherPrerequisites []prerequisite.GatheredPrerequisite

	for _, c := range prerequisiteConfig {
		if c.GetValue() {
			gatheredConfig, err := c.GatherValue()
			if err != nil {
				log.Println(err.Error())
				continue
			}

			switch c.GetName() {
			case "rsync_path":
				rsyncPrerequisites = append(rsyncPrerequisites, gatheredConfig...)
			case "env-vars":
				envVarPrerequisites = append(envVarPrerequisites, gatheredConfig...)
			default:
				otherPrerequisites = append(otherPrerequisites, gatheredConfig...)
			}
		}
	}

	config := Configuration{
		Version:           rootCmd.Version,
		RysncPrequisite:   rsyncPrerequisites,
		EnvPrerequisite:   envVarPrerequisites,
		OtherPrerequisite: otherPrerequisites,
		SyncConfigFiles: SyncConfigFiles{
			ConfigFileActive:    viper.ConfigFileUsed(),
			DefaultConfigFile:   defaultCfgFile,
			LagoonSynConfigFile: lagoonSyncCfgFile,
		},
	}
	configJSON, err := json.MarshalIndent(config, "", " ")
	if err != nil {
		log.Fatalf(err.Error())
	}
	fmt.Println(string(configJSON))
}
