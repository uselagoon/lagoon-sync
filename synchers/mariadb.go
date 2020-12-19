package synchers

import (
	"fmt"
	"strconv"
	"time"
)

type BaseMariaDbSync struct {
	DbHostname      string   `yaml:"hostname"`
	DbUsername      string   `yaml:"username"`
	DbPassword      string   `yaml:"password"`
	DbPort          string   `yaml:"port"`
	DbDatabase      string   `yaml:"database"`
	IgnoreTable     []string `yaml:"ignore-table"`
	IgnoreTableData []string `yaml:"ignore-table-data"`
	OutputDirectory string
}

func (mariaConfig *BaseMariaDbSync) setDefaults() {
	// If no values from config files, set some expected defaults
	if mariaConfig.DbHostname == "" {
		mariaConfig.DbHostname = "$MARIADB_HOST"
	}
	if mariaConfig.DbUsername == "" {
		mariaConfig.DbUsername = "$MARIADB_USERNAME"
	}
	if mariaConfig.DbPassword == "" {
		mariaConfig.DbPassword = "$MARIADB_PASSWORD"
	}
	if mariaConfig.DbPort == "" {
		mariaConfig.DbPort = "$MARIADB_PORT"
	}
	if mariaConfig.DbDatabase == "" {
		mariaConfig.DbDatabase = "$MARIADB_DATABASE"
	}
}

type MariadbSyncLocal struct {
	Config BaseMariaDbSync
}

type MariadbSyncRoot struct {
	Config         BaseMariaDbSync
	LocalOverrides MariadbSyncLocal `yaml:"local"`
	TransferId     string
}

// Init related types and functions follow

type MariadbSyncPlugin struct {
}

func (m MariadbSyncPlugin) GetPluginId() string {
	return "mariadb"
}

func (m MariadbSyncPlugin) UnmarshallYaml(root SyncherConfigRoot) (Syncer, error) {
	mariadb := MariadbSyncRoot{}
	mariadb.Config.setDefaults()
	mariadb.LocalOverrides.Config.setDefaults()

	// unmarshal environment variables as defaults
	_ = UnmarshalIntoStruct(root.EnvironmentDefaults[m.GetPluginId()], &mariadb)

	// if yaml config is there then unmarshall into struct and override default values
	if len(root.LagoonSync) != 0 {
		_ = UnmarshalIntoStruct(root.LagoonSync[m.GetPluginId()], &mariadb)
	}

	lagoonSyncer, _ := mariadb.PrepareSyncer()
	return lagoonSyncer, nil
}

func init() {
	RegisterSyncer(MariadbSyncPlugin{})
}

// Sync related functions follow
func (root MariadbSyncRoot) PrepareSyncer() (Syncer, error) {
	root.TransferId = strconv.FormatInt(time.Now().UnixNano(), 10)
	return root, nil
}

func (root MariadbSyncRoot) GetPrerequisiteCommand(environment Environment) SyncCommand {
	return SyncCommand{
		command: fmt.Sprintf("./lagoon-sync {{ .config }}"),
		substitutions: map[string]interface{}{
			"config": "config",
		},
	}
}

func (root MariadbSyncRoot) GetRemoteCommand(sourceEnvironment Environment) SyncCommand {
	m := root.Config

	if sourceEnvironment.EnvironmentName == LOCAL_ENVIRONMENT_NAME {
		m = root.getEffectiveLocalDetails()
	}

	transferResource := root.GetTransferResource(sourceEnvironment)

	var tablesToIgnore string
	for _, s := range m.IgnoreTable {
		tablesToIgnore += fmt.Sprintf("--ignore-table=%s.%s ", m.DbDatabase, s)
	}

	var tablesWhoseDataToIgnore string
	for _, s := range m.IgnoreTableData {
		tablesWhoseDataToIgnore += fmt.Sprintf("--ignore-table-data=%s.%s ", m.DbDatabase, s)
	}

	return SyncCommand{
		command: fmt.Sprintf("mysqldump -h{{ .hostname }} -u{{ .username }} -p{{ .password }} -P{{ .port }} {{ .tablesToIgnore }} {{ .database }} > {{ .transferResource }}"),
		substitutions: map[string]interface{}{
			"hostname":         m.DbHostname,
			"username":         m.DbUsername,
			"password":         m.DbPassword,
			"port":             m.DbPort,
			"tablesToIgnore":   tablesWhoseDataToIgnore,
			"database":         m.DbDatabase,
			"transferResource": transferResource.Name,
		},
	}
}

func (m MariadbSyncRoot) GetLocalCommand(targetEnvironment Environment) SyncCommand {
	l := m.Config
	if targetEnvironment.EnvironmentName == LOCAL_ENVIRONMENT_NAME {
		l = m.getEffectiveLocalDetails()
	}
	transferResource := m.GetTransferResource(targetEnvironment)
	return generateSyncCommand("mysql -h{{ .hostname }} -u{{ .username }} -p{{ .password }} -P{{ .port }} {{ .database }} < {{ .transferResource }}",
		map[string]interface{}{
			"hostname":         l.DbHostname,
			"username":         l.DbUsername,
			"password":         l.DbPassword,
			"port":             l.DbPort,
			"database":         l.DbDatabase,
			"transferResource": transferResource.Name,
		})
}

func (m MariadbSyncRoot) GetTransferResource(environment Environment) SyncerTransferResource {
	return SyncerTransferResource{
		Name:        fmt.Sprintf("%vlagoon_sync_mariadb_%v.sql", m.GetOutputDirectory(), m.TransferId),
		IsDirectory: false}
}

func (root MariadbSyncRoot) GetOutputDirectory() string {
	m := root.Config
	if len(m.OutputDirectory) == 0 {
		return "/tmp/"
	}
	return m.OutputDirectory
}

func (syncConfig MariadbSyncRoot) getEffectiveLocalDetails() BaseMariaDbSync {
	returnDetails := BaseMariaDbSync{
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
