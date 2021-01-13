package synchers

import (
	"fmt"
	"log"
	"reflect"
	"strconv"
	"time"

	"github.com/spf13/viper"
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
	if mariaConfig.IgnoreTable == nil {
		mariaConfig.IgnoreTable = []string{}
	}
	if mariaConfig.IgnoreTableData == nil {
		mariaConfig.IgnoreTable = []string{}
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
	isConfigEmpty bool
}

func (m BaseMariaDbSync) IsBaseMariaDbStructureEmpty() bool {
	return reflect.DeepEqual(m, BaseMariaDbSync{})
}

func (m MariadbSyncPlugin) GetPluginId() string {
	return "mariadb"
}

func (m MariadbSyncPlugin) UnmarshallYaml(root SyncherConfigRoot) (Syncer, error) {
	mariadb := MariadbSyncRoot{}
	mariadb.Config.setDefaults()
	mariadb.LocalOverrides.Config.setDefaults()

	// Use 'lagoon-sync' yaml as default
	configMap := root.LagoonSync[m.GetPluginId()]

	// if yaml config is there then unmarshall into struct and override default values if there are any
	if len(root.LagoonSync) != 0 {
		_ = UnmarshalIntoStruct(configMap, &mariadb)
	}

	// if envVars := root.Prerequisites; envVars != nil {
	// 	// Use prerequisites if present
	// 	log.Println(envVars)
	// 	log.Println(configMap)

	// 	for k, g := range envVars {
	// 		fmt.Println("name: ", envVars[k].Name)
	// 		fmt.Println("status: ", g.Status)
	// 		fmt.Println("value: ", g.Value)

	// 		//cast configMap to map
	// 		configMap := configMap.(map[interface{}]interface{})
	// 		for j, c := range configMap {
	// 			configMap = c.(map[interface{}]interface{})

	// 			fmt.Println("envVar name: ", envVars[k].Name)
	// 			fmt.Println("map name: ", configMap[envVars[k].Name])

	// 			if j == "config" {
	// 				switch envVars[k].Name {
	// 				case configMap[envVars[k].Name]:
	// 					configMap["hostname"] = "New"
	// 				}
	// 			}
	// 		}
	// 	}
	// }

	// if still missing, then exit out
	if configMap == nil {
		log.Fatalf("Config missing in %v: %v", viper.GetViper().ConfigFileUsed(), configMap)
	}

	if mariadb.Config.IsBaseMariaDbStructureEmpty() {
		m.isConfigEmpty = true
		log.Fatalf("No configuration could be found for %v in %v", m.GetPluginId(), viper.GetViper().ConfigFileUsed())
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

func (root MariadbSyncRoot) GetPrerequisiteCommand(environment Environment, command string) SyncCommand {
	lagoonSyncBin := "lagoon_sync=$(which ./lagoon-sync* || which /tmp/lagoon-sync || false) && $lagoon_sync"

	return SyncCommand{
		command: fmt.Sprintf("{{ .bin }} {{ .command }} || true"),
		substitutions: map[string]interface{}{
			"bin":     lagoonSyncBin,
			"command": command,
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
