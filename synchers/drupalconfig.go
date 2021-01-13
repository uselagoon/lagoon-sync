package synchers

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/spf13/viper"
)

type BaseDrupalconfigSync struct {
	SyncPath        string `yaml:"syncpath"`
	OutputDirectory string
}

type DrupalconfigSyncRoot struct {
	Config         BaseDrupalconfigSync
	LocalOverrides DrupalconfigSyncLocal `yaml:"local"`
	TransferId     string
}

type DrupalconfigSyncLocal struct {
	Config BaseDrupalconfigSync
}

// Init related types and functions follow

type DrupalConfigSyncPlugin struct {
}

func (m DrupalConfigSyncPlugin) GetPluginId() string {
	return "drupalconfig"
}

func (m DrupalConfigSyncPlugin) UnmarshallYaml(syncerConfigRoot SyncherConfigRoot) (Syncer, error) {
	drupalconfig := DrupalconfigSyncRoot{}

	// Use prerequisites if present
	envVars := syncerConfigRoot.Prerequisites
	var configMap interface{}
	if envVars == nil {
		// Use 'lagoon-sync' yaml as override if source-environment-deaults is not available
		configMap = syncerConfigRoot.LagoonSync[m.GetPluginId()]
	}

	// if still missing, then exit out
	if configMap == nil {
		log.Fatalf("Config missing in %v: %v", viper.GetViper().ConfigFileUsed(), configMap)
	}

	// unmarshal environment variables as defaults
	_ = UnmarshalIntoStruct(configMap, &drupalconfig)

	// if yaml config is there then unmarshall into struct and override default values if there are any
	if len(syncerConfigRoot.LagoonSync) != 0 {
		_ = UnmarshalIntoStruct(configMap, &drupalconfig)
	}

	lagoonSyncer, _ := drupalconfig.PrepareSyncer()
	return lagoonSyncer, nil
}

func init() {
	RegisterSyncer(DrupalConfigSyncPlugin{})
}

// Sync functions below

func (root DrupalconfigSyncRoot) PrepareSyncer() (Syncer, error) {
	root.TransferId = strconv.FormatInt(time.Now().UnixNano(), 10)
	return root, nil
}

func (root DrupalconfigSyncRoot) GetPrerequisiteCommand(environment Environment, command string) SyncCommand {
	return SyncCommand{}
}

func (root DrupalconfigSyncRoot) GetRemoteCommand(environment Environment) SyncCommand {
	transferResource := root.GetTransferResource(environment)
	return SyncCommand{
		command: fmt.Sprintf("drush config-export --destination=%s || true", transferResource.Name),
	}
}

func (m DrupalconfigSyncRoot) GetLocalCommand(environment Environment) SyncCommand {
	// l := m.getEffectiveLocalDetails()
	transferResource := m.GetTransferResource(environment)

	return SyncCommand{
		command: fmt.Sprintf("drush -y config-import --source=%s || true", transferResource.Name),
	}

}

func (m DrupalconfigSyncRoot) GetTransferResource(environment Environment) SyncerTransferResource {
	return SyncerTransferResource{
		Name:        fmt.Sprintf("%vdrupalconfig-sync-%v", m.GetOutputDirectory(), m.TransferId),
		IsDirectory: true}
}

func (root DrupalconfigSyncRoot) GetOutputDirectory() string {
	m := root.Config
	if len(m.OutputDirectory) == 0 {
		return "/tmp/"
	}
	return m.OutputDirectory
}

func (syncConfig DrupalconfigSyncRoot) getEffectiveLocalDetails() BaseDrupalconfigSync {
	returnDetails := BaseDrupalconfigSync{
		SyncPath:        syncConfig.Config.SyncPath,
		OutputDirectory: syncConfig.Config.OutputDirectory,
	}

	assignLocalOverride := func(target *string, override *string) {
		if len(*override) > 0 {
			*target = *override
		}
	}

	//TODO: can this be replaced with reflection?
	assignLocalOverride(&returnDetails.SyncPath, &syncConfig.LocalOverrides.Config.SyncPath)
	assignLocalOverride(&returnDetails.OutputDirectory, &syncConfig.LocalOverrides.Config.OutputDirectory)
	return returnDetails
}
