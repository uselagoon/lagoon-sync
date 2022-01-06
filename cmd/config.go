package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/uselagoon/lagoon-sync/prerequisite"
	"github.com/uselagoon/lagoon-sync/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Configuration struct {
	Version           string                              `json:"version"`
	LagoonSyncPath    string                              `json:"lagoon-sync-path"`
	EnvPrerequisite   []prerequisite.GatheredPrerequisite `json:"env-config"`
	RysncPrerequisite []prerequisite.GatheredPrerequisite `json:"rsync-config"`
	OtherPrerequisite []prerequisite.GatheredPrerequisite `json:"other-config"`
	SyncConfigFiles   SyncConfigFiles                     `json:"sync-config-files"`
}

type SyncConfigFiles struct {
	ConfigFileActive             string `json:"config-file-active"`
	LagoonSyncConfigFile         string `json:"lagoon-sync-path"`
	LagoonSyncDefaultsConfigFile string `json:"lagoon-sync-defaults-path"`
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Print the config that is being used by lagoon-sync",
	Run: func(v *cobra.Command, args []string) {
		configJSON := PrintConfigOut()
		fmt.Println(string(configJSON))
	},
}

func PrintConfigOut() []byte {
	lagoonSyncDefaultsFile, exists := os.LookupEnv("LAGOON_SYNC_DEFAULTS_PATH")
	if !exists {
		lagoonSyncDefaultsFile = "false"
	}
	lagoonSyncCfgFile, exists := os.LookupEnv("LAGOON_SYNC_PATH")
	if !exists {
		lagoonSyncCfgFile = "false"
	}

	lagoonSyncPath, exists := utils.FindLagoonSyncOnEnv()

	// Run the prerequsite gatherers
	prerequisiteConfig := prerequisite.GetPrerequisiteGatherer()
	var RsyncPrerequisites []prerequisite.GatheredPrerequisite
	var envVarPrerequisites []prerequisite.GatheredPrerequisite
	var otherPrerequisites []prerequisite.GatheredPrerequisite

	for _, c := range prerequisiteConfig {
		if c.GetValue() {
			gatheredConfig, err := c.GatherPrerequisites()
			if err != nil {
				log.Println(err.Error())
				continue
			}

			switch c.GetName() {
			case "rsync_path":
				RsyncPrerequisites = append(RsyncPrerequisites, gatheredConfig...)
			case "env-vars":
				envVarPrerequisites = append(envVarPrerequisites, gatheredConfig...)
			default:
				otherPrerequisites = append(otherPrerequisites, gatheredConfig...)
			}
		}
	}

	config := Configuration{
		Version:           rootCmd.Version,
		LagoonSyncPath:    lagoonSyncPath,
		RysncPrerequisite: RsyncPrerequisites,
		EnvPrerequisite:   envVarPrerequisites,
		OtherPrerequisite: otherPrerequisites,
		SyncConfigFiles: SyncConfigFiles{
			ConfigFileActive:             viper.ConfigFileUsed(),
			LagoonSyncConfigFile:         lagoonSyncCfgFile,
			LagoonSyncDefaultsConfigFile: lagoonSyncDefaultsFile,
		},
	}
	configUnmarshalled, err := json.MarshalIndent(config, "", " ")
	if err != nil {
		log.Fatalf(err.Error())
	}

	return configUnmarshalled
}

func init() {
	rootCmd.AddCommand(configCmd)
}
