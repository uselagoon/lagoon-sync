package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/amazeeio/lagoon-sync/prerequisite"
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
	DefaultConfigFile            string `json:"default-config-path"`
	LagoonSyncDefaultsConfigFile string `json:"lagoon-sync-defaults-path"`
}

func init() {
	rootCmd.AddCommand(configCmd)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Print the config that is being used by lagoon-sync",
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

	lagoonSyncPath, exists := FindLagoonSyncOnEnv()

	config := Configuration{
		Version:           rootCmd.Version,
		LagoonSyncPath:    lagoonSyncPath,
		RysncPrerequisite: RsyncPrerequisites,
		EnvPrerequisite:   envVarPrerequisites,
		OtherPrerequisite: otherPrerequisites,
		SyncConfigFiles: SyncConfigFiles{
			ConfigFileActive:             viper.ConfigFileUsed(),
			DefaultConfigFile:            defaultCfgFile,
			LagoonSyncDefaultsConfigFile: lagoonSyncCfgFile,
		},
	}
	configJSON, err := json.MarshalIndent(config, "", " ")
	if err != nil {
		log.Fatalf(err.Error())
	}
	fmt.Println(string(configJSON))
}

func FindLagoonSyncOnEnv() (string, bool) {
	cmd := exec.Command("sh", "-c", "which ./lagoon-sync || which /tmp/lagoon-sync* || which lagoon-sync || true")
	stdoutStderr, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
		log.Fatal(string(stdoutStderr))
	}

	lagoonPath := strings.TrimSuffix(string(stdoutStderr), "\n")
	if lagoonPath != "" {
		return lagoonPath, true
	}
	return "", false
}
