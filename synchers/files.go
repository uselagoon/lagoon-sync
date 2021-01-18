package synchers

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/spf13/viper"
)

type BaseFilesSync struct {
	SyncPath string `yaml:"sync-directory"`
	Exclude  []string
}

func (filesConfig *BaseFilesSync) setDefaults() {
	if filesConfig.SyncPath == "" {
		filesConfig.SyncPath = "/app/web/sites/default/files"
	}
}

type FilesSyncRoot struct {
	Config         BaseFilesSync
	LocalOverrides FilesSyncLocal `yaml:"local"`
	TransferId     string
}

type FilesSyncLocal struct {
	Config BaseFilesSync
}

// Init related types and functions follow

type FilesSyncPlugin struct {
}

func (m FilesSyncPlugin) GetPluginId() string {
	return "files"
}

func (m FilesSyncPlugin) UnmarshallYaml(root SyncherConfigRoot) (Syncer, error) {
	filesroot := FilesSyncRoot{}
	filesroot.Config.setDefaults()

	// Use 'environment-defaults' if present
	envVars := root.Prerequisites
	var configMap interface{}
	if envVars == nil {
		// Use 'lagoon-sync' yaml as override if env vars are not available
		configMap = root.LagoonSync[m.GetPluginId()]
	}

	// if still missing, then exit out
	if configMap == nil {
		log.Fatalf("Syncer config is missing and unable to proceed in %v: %v", viper.GetViper().ConfigFileUsed(), configMap)
	}

	// unmarshal environment variables as defaults
	_ = UnmarshalIntoStruct(configMap, &filesroot)

	// if yaml config is there then unmarshall into struct and override default values if there are any
	if len(root.LagoonSync) != 0 {
		_ = UnmarshalIntoStruct(configMap, &filesroot)
	}

	lagoonSyncer, _ := filesroot.PrepareSyncer()
	return lagoonSyncer, nil
}

func init() {
	RegisterSyncer(FilesSyncPlugin{})
}

func (root FilesSyncRoot) PrepareSyncer() (Syncer, error) {
	root.TransferId = strconv.FormatInt(time.Now().UnixNano(), 10)
	return root, nil
}

func (root FilesSyncRoot) GetPrerequisiteCommand(environment Environment, command string) SyncCommand {
	return SyncCommand{}
}

func (root FilesSyncRoot) GetRemoteCommand(environment Environment) SyncCommand {
	return generateNoOpSyncCommand()
}

func (m FilesSyncRoot) GetLocalCommand(environment Environment) SyncCommand {
	return generateNoOpSyncCommand()
}

func (m FilesSyncRoot) GetTransferResource(environment Environment) SyncerTransferResource {
	config := m.Config
	if environment.EnvironmentName == LOCAL_ENVIRONMENT_NAME {
		config = m.getEffectiveLocalDetails()
	}
	return SyncerTransferResource{
		Name:             fmt.Sprintf(config.SyncPath),
		IsDirectory:      true,
		SkipCleanup:      true,
		ExcludeResources: m.Config.Exclude,
	}
}

func (syncConfig FilesSyncRoot) getEffectiveLocalDetails() BaseFilesSync {
	returnDetails := BaseFilesSync{
		SyncPath: syncConfig.Config.SyncPath,
	}

	assignLocalOverride := func(target *string, override *string) {
		if len(*override) > 0 {
			*target = *override
		}
	}

	//TODO: can this be replaced with reflection?
	assignLocalOverride(&returnDetails.SyncPath, &syncConfig.LocalOverrides.Config.SyncPath)
	return returnDetails
}
