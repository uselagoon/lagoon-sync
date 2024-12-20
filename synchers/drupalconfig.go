package synchers

import (
	"fmt"
	"strconv"
	"time"

	"github.com/uselagoon/lagoon-sync/utils"
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

func (m DrupalConfigSyncPlugin) UnmarshallYaml(syncerConfigRoot SyncherConfigRoot, targetService string) (Syncer, error) {
	drupalconfig := DrupalconfigSyncRoot{}
	drupalconfig.Config.OutputDirectory = drupalconfig.GetOutputDirectory()

	configMap := syncerConfigRoot.LagoonSync[targetService]
	_ = UnmarshalIntoStruct(configMap, &drupalconfig)

	// If yaml config is there then unmarshall into struct and override default values if there are any
	if len(syncerConfigRoot.LagoonSync) != 0 {
		_ = UnmarshalIntoStruct(configMap, &drupalconfig)
	}

	// If config from active config file is empty, then use defaults
	if configMap == nil {
		utils.LogDebugInfo("Active syncer config is empty, so using defaults", &drupalconfig)
	}

	lagoonSyncer, _ := drupalconfig.PrepareSyncer()
	return lagoonSyncer, nil
}

func init() {
	RegisterSyncer(DrupalConfigSyncPlugin{})
}

func (root DrupalconfigSyncRoot) PrepareSyncer() (Syncer, error) {
	root.TransferId = strconv.FormatInt(time.Now().UnixNano(), 10)
	return &root, nil
}

func (m DrupalconfigSyncRoot) IsInitialized() (bool, error) {
	return true, nil
}

func (root DrupalconfigSyncRoot) GetPrerequisiteCommand(environment Environment, command string) SyncCommand {
	return SyncCommand{}
}

func (root DrupalconfigSyncRoot) GetRemoteCommand(environment Environment) []SyncCommand {
	transferResource := root.GetTransferResource(environment)
	return []SyncCommand{
		{
			command: fmt.Sprintf("drush config-export --destination=%s || true", transferResource.Name),
		},
	}
}

func (m DrupalconfigSyncRoot) GetLocalCommand(environment Environment) []SyncCommand {
	// l := m.getEffectiveLocalDetails()
	transferResource := m.GetTransferResource(environment)

	return []SyncCommand{
		{
			command: fmt.Sprintf("drush -y config-import --source=%s || true", transferResource.Name),
		},
	}

}

func (m DrupalconfigSyncRoot) GetFilesToCleanup(environment Environment) []string {
	transferResource := m.GetTransferResource(environment)
	return []string{
		transferResource.Name,
	}
}

func (m DrupalconfigSyncRoot) GetTransferResource(environment Environment) SyncerTransferResource {
	return SyncerTransferResource{
		Name:        fmt.Sprintf("%vdrupalconfig-sync-%v", m.GetOutputDirectory(), m.TransferId),
		IsDirectory: true}
}

func (m *DrupalconfigSyncRoot) SetTransferResource(transferResourceName string) error {
	return fmt.Errorf("Setting the transfer resource is not supported for drupal config")
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
