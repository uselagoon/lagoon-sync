package synchers

import (
	"fmt"
	"log"
	"reflect"
	"strconv"
	"time"

	"github.com/spf13/viper"
)

type BasePostgresSync struct {
	DbHostname       string   `yaml:"hostname"`
	DbUsername       string   `yaml:"username"`
	DbPassword       string   `yaml:"password"`
	DbPort           string   `yaml:"port"`
	DbDatabase       string   `yaml:"database"`
	ExcludeTable     []string `yaml:"exclude-table"`
	ExcludeTableData []string `yaml:"exclude-table-data"`
	OutputDirectory  string
}
type PostgresSyncRoot struct {
	Config         BasePostgresSync
	LocalOverrides PostgresSyncLocal `yaml:"local"`
	TransferId     string
}

type PostgresSyncLocal struct {
	Config BasePostgresSync
}

func (postgresConfig *BasePostgresSync) setDefaults() {
	if postgresConfig.DbHostname == "" {
		postgresConfig.DbHostname = "$AMAZEEIO_DB_HOST"
	}
	if postgresConfig.DbUsername == "" {
		postgresConfig.DbUsername = "$AMAZEEIO_DB_USERNAME"
	}
	if postgresConfig.DbPassword == "" {
		postgresConfig.DbPassword = "$AMAZEEIO_DB_PASSWORD"
	}
	if postgresConfig.DbPort == "" {
		postgresConfig.DbPort = "$AMAZEEIO_DB_PORT"
	}
	if postgresConfig.DbDatabase == "" {
		postgresConfig.DbDatabase = "$POSTGRES_DATABASE"
	}
}

// Init related types and functions follow

type PostgresSyncPlugin struct {
	isConfigEmpty bool
}

func (m BasePostgresSync) IsBasePostgresDbStructureEmpty() bool {
	return reflect.DeepEqual(m, BasePostgresSync{})
}

func (m PostgresSyncPlugin) GetPluginId() string {
	return "postgres"
}

func (m PostgresSyncPlugin) UnmarshallYaml(syncerConfigRoot SyncherConfigRoot) (Syncer, error) {
	postgres := PostgresSyncRoot{}
	postgres.Config.setDefaults()
	postgres.LocalOverrides.Config.setDefaults()

	// Use 'source-environment-defaults' yaml if present
	configMap := syncerConfigRoot.EnvironmentDefaults[m.GetPluginId()]
	if configMap == nil {
		// Use 'lagoon-sync' yaml as override if source-environment-deaults is not available
		configMap = syncerConfigRoot.LagoonSync[m.GetPluginId()]
	}

	// if still missing, then exit out
	if configMap == nil {
		log.Fatalf("Config missing in %v: %v", viper.GetViper().ConfigFileUsed(), configMap)
	}

	// unmarshal environment variables as defaults
	_ = UnmarshalIntoStruct(configMap, &postgres)

	if len(syncerConfigRoot.LagoonSync) != 0 {
		_ = UnmarshalIntoStruct(configMap, &postgres)
	}

	// check here if we have any default values - if not we bail out.
	if postgres.Config.IsBasePostgresDbStructureEmpty() {
		m.isConfigEmpty = true
		log.Fatalf("No syncer configuration could be found for %v in %v", m.GetPluginId(), viper.GetViper().ConfigFileUsed())
	}

	lagoonSyncer, _ := postgres.PrepareSyncer()
	return lagoonSyncer, nil
}

func init() {
	RegisterSyncer(PostgresSyncPlugin{})
}

// Sync related functions below

func (root PostgresSyncRoot) PrepareSyncer() (Syncer, error) {
	root.TransferId = strconv.FormatInt(time.Now().UnixNano(), 10)
	return root, nil
}

func (root PostgresSyncRoot) GetPrerequisiteCommand(environment Environment, command string) SyncCommand {
	lagoonSyncBin := "lagoon_sync=$(which ./lagoon-sync* || which /tmp/lagoon-sync || false) && $lagoon_sync"

	return SyncCommand{
		command: fmt.Sprintf("{{ .bin }} {{ .command }}"),
		substitutions: map[string]interface{}{
			"bin":     lagoonSyncBin,
			"command": command,
		},
	}
}

func (root PostgresSyncRoot) GetRemoteCommand(environment Environment) SyncCommand {
	m := root.Config
	transferResource := root.GetTransferResource(environment)

	var tablesToExclude string
	for _, s := range m.ExcludeTable {
		tablesToExclude += fmt.Sprintf("--exclude-table=%s.%s ", m.DbDatabase, s)
	}

	var tablesWhoseDataToExclude string
	for _, s := range m.ExcludeTableData {
		tablesWhoseDataToExclude += fmt.Sprintf("--exclude-table-data=%s.%s ", m.DbDatabase, s)
	}

	return SyncCommand{
		command: fmt.Sprintf("PGPASSWORD=\"%s\" pg_dump -h%s -U%s -p%s -d%s %s %s -Fc -w -f%s", m.DbPassword, m.DbHostname, m.DbUsername, m.DbPort, m.DbDatabase, tablesToExclude, tablesWhoseDataToExclude, transferResource.Name),
	}
}

func (m PostgresSyncRoot) GetLocalCommand(environment Environment) SyncCommand {
	l := m.getEffectiveLocalDetails()
	transferResource := m.GetTransferResource(environment)
	return SyncCommand{
		command: fmt.Sprintf("PGPASSWORD=\"%s\" pg_restore -c -x -w -h%s -d%s -p%s -U%s %s", l.DbPassword, l.DbHostname, l.DbDatabase, l.DbPort, l.DbUsername, transferResource.Name),
	}
}

func (m PostgresSyncRoot) GetTransferResource(environment Environment) SyncerTransferResource {
	return SyncerTransferResource{
		Name:        fmt.Sprintf("%vlagoon_sync_postgres_%v.sql", m.GetOutputDirectory(), m.TransferId),
		IsDirectory: false}
}

func (root PostgresSyncRoot) GetOutputDirectory() string {
	m := root.Config
	if len(m.OutputDirectory) == 0 {
		return "/tmp/"
	}
	return m.OutputDirectory
}

func (syncConfig PostgresSyncRoot) getEffectiveLocalDetails() BasePostgresSync {
	returnDetails := BasePostgresSync{
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
