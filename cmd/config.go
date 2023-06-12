package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/uselagoon/lagoon-sync/prerequisite"
	synchers "github.com/uselagoon/lagoon-sync/synchers"
	"github.com/uselagoon/lagoon-sync/utils"
)

type Configuration struct {
	Version           string                              `json:"version"`
	LagoonSyncPath    string                              `json:"lagoon-sync-path"`
	EnvPrerequisite   []prerequisite.GatheredPrerequisite `json:"env-config"`
	RysncPrerequisite []prerequisite.GatheredPrerequisite `json:"rsync-config"`
	OtherPrerequisite []prerequisite.GatheredPrerequisite `json:"other-config"`
	SyncConfigFiles   SyncConfigFiles                     `json:"sync-config-files"`
	SSHConfig         synchers.SSHOptions                 `json:"ssh"`
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

func LoadLagoonConfig(lagoonYamlPath string) ([]byte, error) {
	var data, err = ioutil.ReadFile(lagoonYamlPath)
	if err != nil {
		return []byte{}, err
	}
	return data, nil
}

func PrintConfigOut() []byte {
	lagoonSyncDefaultsFile, exists := os.LookupEnv("LAGOON_SYNC_DEFAULTS_PATH")
	if !exists {
		lagoonSyncDefaultsFile = ""
	}
	lagoonSyncCfgFile, exists := os.LookupEnv("LAGOON_SYNC_PATH")
	if !exists {
		lagoonSyncCfgFile = ""
	}

	lagoonSyncPath, exists := utils.FindLagoonSyncOnEnv()
	activeLagoonYmlFile := viper.ConfigFileUsed()

	// Gather lagoon.yml configuration
	lagoonConfigBytestream, err := LoadLagoonConfig(activeLagoonYmlFile)
	if err != nil {
		utils.LogFatalError("Couldn't load lagoon config file - ", err.Error())
	}
	lagoonConfig, err := synchers.UnmarshallLagoonYamlToLagoonSyncStructure(lagoonConfigBytestream)
	if err != nil {
		log.Fatalf("There was an issue unmarshalling the sync configuration from %v: %v", activeLagoonYmlFile, err)
	}

	// Store Lagoon yaml config
	sshConfig := synchers.SSHOptions{}
	if lagoonConfig.LagoonSync["ssh"] != nil {
		mapstructure.Decode(lagoonConfig.LagoonSync["ssh"], &sshConfig)
	}

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
			ConfigFileActive:             activeLagoonYmlFile,
			LagoonSyncConfigFile:         lagoonSyncCfgFile,
			LagoonSyncDefaultsConfigFile: lagoonSyncDefaultsFile,
		},
		SSHConfig: sshConfig,
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
