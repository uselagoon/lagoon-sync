package synchers

import (
	"fmt"
	"strconv"
	"time"
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

	// unmarshal environment variables as defaults
	_ = UnmarshalIntoStruct(syncerConfigRoot.EnvironmentDefaults[m.GetPluginId()], &drupalconfig)
	_ = UnmarshalIntoStruct(syncerConfigRoot.EnvironmentDefaults[m.GetPluginId()], &drupalconfig.LocalOverrides)

	if len(syncerConfigRoot.LagoonSync) != 0 {
		_ = UnmarshalIntoStruct(syncerConfigRoot.LagoonSync[m.GetPluginId()], &drupalconfig)
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

func (root DrupalconfigSyncRoot) GetRemoteCommand(environment Environment) SyncCommand {
	transferResource := root.GetTransferResource(environment)
	return SyncCommand{
		command: fmt.Sprintf("drush config-export --destination=%s", transferResource.Name),
	}
}

func (m DrupalconfigSyncRoot) GetLocalCommand(environment Environment) SyncCommand {
	// l := m.getEffectiveLocalDetails()
	transferResource := m.GetTransferResource(environment)

	return SyncCommand{
		command: fmt.Sprintf("drush -y config-import --source=%s", transferResource.Name),
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
