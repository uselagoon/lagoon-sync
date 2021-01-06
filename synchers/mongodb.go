package synchers

import (
	"fmt"
	"strconv"
	"time"
)

type BaseMongoDbSync struct {
	DbHostname      string   `yaml:"hostname"`
	DbUsername      string   `yaml:"username"`
	DbPassword      string   `yaml:"password"`
	DbPort          string   `yaml:"port"`
	DbDatabase      string   `yaml:"database"`
	IgnoreTable     []string `yaml:"ignore-table"`
	IgnoreTableData []string `yaml:"ignore-table-data"`
	OutputDirectory string
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
}

func (m MongoDbSyncPlugin) GetPluginId() string {
	return "mongodb"
}

func (m MongoDbSyncPlugin) UnmarshallYaml(root SyncherConfigRoot) (Syncer, error) {
	mongodb := MongoDbSyncRoot{}
	_ = UnmarshalIntoStruct(root.LagoonSync[m.GetPluginId()], &mongodb)
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
	lagoonSyncBin := "$(which ./lagoon-sync || which /tmp/lagoon-sync* || which lagoon-sync)"

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

	// var tablesToIgnore string
	// for _, s := range m.IgnoreTable {
	// 	tablesToIgnore += fmt.Sprintf("--ignore-table=%s.%s ", m.DbDatabase, s)
	// }

	// var tablesWhoseDataToIgnore string
	// for _, s := range m.IgnoreTableData {
	// 	tablesWhoseDataToIgnore += fmt.Sprintf("--ignore-table-data=%s.%s ", m.DbDatabase, s)
	// }

	return SyncCommand{
		command: fmt.Sprintf("mongodump --archive={{ .transferResource }}"),
		substitutions: map[string]interface{}{
			"hostname":         m.DbHostname,
			"username":         m.DbUsername,
			"password":         m.DbPassword,
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
	return generateSyncCommand("mongorestore --archive={{ .transferResource }}",
		map[string]interface{}{
			"hostname":         l.DbHostname,
			"username":         l.DbUsername,
			"password":         l.DbPassword,
			"port":             l.DbPort,
			"database":         l.DbDatabase,
			"transferResource": transferResource.Name,
		})
}

func (m MongoDbSyncRoot) GetTransferResource(environment Environment) SyncerTransferResource {
	return SyncerTransferResource{
		Name:        fmt.Sprintf("%vlagoon_sync_mongodb_%v.archive", m.GetOutputDirectory(), m.TransferId),
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
		DbUsername:      syncConfig.Config.DbUsername,
		DbPassword:      syncConfig.Config.DbPassword,
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
	assignLocalOverride(&returnDetails.DbUsername, &syncConfig.LocalOverrides.Config.DbUsername)
	assignLocalOverride(&returnDetails.DbPassword, &syncConfig.LocalOverrides.Config.DbPassword)
	assignLocalOverride(&returnDetails.DbPort, &syncConfig.LocalOverrides.Config.DbPort)
	assignLocalOverride(&returnDetails.DbDatabase, &syncConfig.LocalOverrides.Config.DbDatabase)
	assignLocalOverride(&returnDetails.OutputDirectory, &syncConfig.LocalOverrides.Config.OutputDirectory)
	return returnDetails
}
