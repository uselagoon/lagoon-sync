package synchers

import (
	"fmt"
	"log"
	"reflect"
	"strconv"
	"time"

	"github.com/spf13/viper"
)

type BaseMongoDbSync struct {
	DbHostname      string `yaml:"hostname"`
	DbPort          string `yaml:"port"`
	DbDatabase      string `yaml:"database"`
	OutputDirectory string
}

func (mongoConfig *BaseMongoDbSync) setDefaults() {
	// If no values from config files, set some expected defaults
	if mongoConfig.DbHostname == "" {
		mongoConfig.DbHostname = "$HOSTNAME"
	}
	if mongoConfig.DbPort == "" {
		mongoConfig.DbPort = "27017"
	}
	if mongoConfig.DbDatabase == "" {
		mongoConfig.DbDatabase = "local"
	}
}

type MongoDbSyncLocal struct {
	Config BaseMongoDbSync
}

type MongoDbSyncRoot struct {
	Config         BaseMongoDbSync
	LocalOverrides MongoDbSyncLocal `yaml:"local"`
	TransferId     string
}

// Init related types and functions follow

type MongoDbSyncPlugin struct {
	isConfigEmpty bool
}

func (m BaseMongoDbSync) IsBaseMongoDbStructureEmpty() bool {
	return reflect.DeepEqual(m, BaseMongoDbSync{})
}

func (m MongoDbSyncPlugin) GetPluginId() string {
	return "mongodb"
}

func (m MongoDbSyncPlugin) UnmarshallYaml(root SyncherConfigRoot) (Syncer, error) {
	mongodb := MongoDbSyncRoot{}
	mongodb.Config.setDefaults()
	mongodb.LocalOverrides.Config.setDefaults()

	// Use 'source-environment-defaults' yaml if present
	configMap := root.EnvironmentDefaults[m.GetPluginId()]
	if configMap == nil {
		// Use 'lagoon-sync' yaml as override if source-environment-deaults is not available
		configMap = root.LagoonSync[m.GetPluginId()]
	}

	// if still missing, then exit out
	if configMap == nil {
		log.Fatalf("Config missing in %v: %v", viper.GetViper().ConfigFileUsed(), configMap)
	}

	// unmarshal environment variables as defaults
	_ = UnmarshalIntoStruct(configMap, &mongodb)

	if len(root.LagoonSync) != 0 {
		_ = UnmarshalIntoStruct(configMap, &mongodb)
	}

	// check here if we have any default values - if not we bail out.
	if mongodb.Config.IsBaseMongoDbStructureEmpty() {
		m.isConfigEmpty = true
		log.Fatalf("No syncer configuration could be found for %v in %v", m.GetPluginId(), viper.GetViper().ConfigFileUsed())
	}

	lagoonSyncer, _ := mongodb.PrepareSyncer()
	return lagoonSyncer, nil
}

func init() {
	RegisterSyncer(MongoDbSyncPlugin{})
}

// Sync related functions follow
func (root MongoDbSyncRoot) PrepareSyncer() (Syncer, error) {
	root.TransferId = strconv.FormatInt(time.Now().UnixNano(), 10)
	return root, nil
}

func (root MongoDbSyncRoot) GetPrerequisiteCommand(environment Environment, command string) SyncCommand {
	lagoonSyncBin := "lagoon_sync=$(which ./lagoon-sync* || which /tmp/lagoon-sync || false) && $lagoon_sync"

	return SyncCommand{
		command: fmt.Sprintf("{{ .bin }} {{ .command }} || true"),
		substitutions: map[string]interface{}{
			"bin":     lagoonSyncBin,
			"command": command,
		},
	}
}

func (root MongoDbSyncRoot) GetRemoteCommand(sourceEnvironment Environment) SyncCommand {
	m := root.Config

	if sourceEnvironment.EnvironmentName == LOCAL_ENVIRONMENT_NAME {
		m = root.getEffectiveLocalDetails()
	}

	transferResource := root.GetTransferResource(sourceEnvironment)
	return SyncCommand{
		command: fmt.Sprintf("mongodump --host {{ .hostname }} --port {{ .port }} --db {{ .database }} --archive={{ .transferResource }}"),
		substitutions: map[string]interface{}{
			"hostname":         m.DbHostname,
			"port":             m.DbPort,
			"database":         m.DbDatabase,
			"transferResource": transferResource.Name,
		},
	}
}

func (m MongoDbSyncRoot) GetLocalCommand(targetEnvironment Environment) SyncCommand {
	l := m.Config
	if targetEnvironment.EnvironmentName == LOCAL_ENVIRONMENT_NAME {
		l = m.getEffectiveLocalDetails()
	}
	transferResource := m.GetTransferResource(targetEnvironment)
	return generateSyncCommand("mongorestore --drop --host {{ .hostname }} --port {{ .port }} --archive={{ .transferResource }}",
		map[string]interface{}{
			"hostname":         l.DbHostname,
			"port":             l.DbPort,
			"database":         l.DbDatabase,
			"transferResource": transferResource.Name,
		})
}

func (m MongoDbSyncRoot) GetTransferResource(environment Environment) SyncerTransferResource {
	return SyncerTransferResource{
		Name:        fmt.Sprintf("%vlagoon_sync_mongodb_%v.bson", m.GetOutputDirectory(), m.TransferId),
		IsDirectory: false}
}

func (root MongoDbSyncRoot) GetOutputDirectory() string {
	m := root.Config
	if len(m.OutputDirectory) == 0 {
		return "/tmp/"
	}
	return m.OutputDirectory
}

func (syncConfig MongoDbSyncRoot) getEffectiveLocalDetails() BaseMongoDbSync {
	returnDetails := BaseMongoDbSync{
		DbHostname:      syncConfig.Config.DbHostname,
		DbPort:          syncConfig.Config.DbPort,
		DbDatabase:      syncConfig.Config.DbDatabase,
		OutputDirectory: syncConfig.Config.OutputDirectory,
	}

	assignLocalOverride := func(target *string, override *string) {
		if len(*override) > 0 {
			*target = *override
		}
	}

	//TODO: can this be replaced with reflection?
	assignLocalOverride(&returnDetails.DbHostname, &syncConfig.LocalOverrides.Config.DbHostname)
	assignLocalOverride(&returnDetails.DbPort, &syncConfig.LocalOverrides.Config.DbPort)
	assignLocalOverride(&returnDetails.DbDatabase, &syncConfig.LocalOverrides.Config.DbDatabase)
	assignLocalOverride(&returnDetails.OutputDirectory, &syncConfig.LocalOverrides.Config.OutputDirectory)
	return returnDetails
}
