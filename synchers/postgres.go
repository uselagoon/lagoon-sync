package synchers

import (
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/spf13/viper"
	"github.com/uselagoon/lagoon-sync/utils"
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
	Type                     string `yaml:"type" json:"type"`
	ServiceName              string `yaml:"serviceName"`
	Config                   BasePostgresSync
	LocalOverrides           PostgresSyncLocal `yaml:"local"`
	TransferId               string
	TransferResourceOverride string
}

type PostgresSyncLocal struct {
	Config BasePostgresSync
}

// SetDefaults is a public function that is used to set all defaults for this struct
func (postgresConfig *BasePostgresSync) SetDefaults() {
	postgresConfig.setDefaults()
}

func (postgresConfig *BasePostgresSync) setDefaults() {
	if postgresConfig.DbHostname == "" {
		postgresConfig.DbHostname = "${POSTGRES_HOST:-postgres}"
	}
	if postgresConfig.DbUsername == "" {
		postgresConfig.DbUsername = "${POSTGRES_USERNAME:-drupal}"
	}
	if postgresConfig.DbPassword == "" {
		postgresConfig.DbPassword = "${POSTGRES_PASSWORD:-drupal}"
	}
	if postgresConfig.DbPort == "" {
		postgresConfig.DbPort = "${POSTGRES_PORT:-5432}"
	}
	if postgresConfig.DbDatabase == "" {
		postgresConfig.DbDatabase = "${POSTGRES_DATABASE:-drupal}"
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

func (m PostgresSyncPlugin) UnmarshallYaml(syncerConfigRoot SyncherConfigRoot, targetService string) (Syncer, error) {
	postgres := PostgresSyncRoot{}
	postgres.Type = m.GetPluginId()
	postgres.Config.setDefaults()

	configMap := syncerConfigRoot.LagoonSync[targetService]

	// If yaml config is there then unmarshall into struct and override default values if there are any
	if configMap != nil {
		_ = UnmarshalIntoStruct(configMap, &postgres)
		utils.LogDebugInfo("Config that will be used for sync", postgres)
	} else {
		// If config from active config file is empty, then use defaults
		if configMap == nil {
			utils.LogDebugInfo("Active syncer config is empty, so using defaults", postgres)
		}
	}

	if postgres.Config.IsBasePostgresDbStructureEmpty() && &postgres == nil {
		m.isConfigEmpty = true
		utils.LogFatalError("No syncer configuration could be found in", viper.GetViper().ConfigFileUsed())
	}

	lagoonSyncer, _ := postgres.PrepareSyncer()
	return lagoonSyncer, nil
}

func init() {
	RegisterSyncer(PostgresSyncPlugin{})
}

func (m *PostgresSyncRoot) IsInitialized() (bool, error) {
	return true, nil
}

// Sync related functions below

func (root *PostgresSyncRoot) PrepareSyncer() (Syncer, error) {
	root.TransferId = strconv.FormatInt(time.Now().UnixNano(), 10)
	return root, nil
}

func (root *PostgresSyncRoot) GetPrerequisiteCommand(environment Environment, command string) SyncCommand {
	lagoonSyncBin, _ := utils.FindLagoonSyncOnEnv()

	return SyncCommand{
		command: fmt.Sprintf("{{ .bin }} {{ .command }}"),
		substitutions: map[string]interface{}{
			"bin":     lagoonSyncBin,
			"command": command,
		},
	}
}

func (root *PostgresSyncRoot) GetRemoteCommand(environment Environment) []SyncCommand {
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

	return []SyncCommand{
		{
			command: fmt.Sprintf("PGPASSWORD=\"%s\" pg_dump -h%s -U%s -p%s -d%s %s %s -Fc -w -f%s", m.DbPassword, m.DbHostname, m.DbUsername, m.DbPort, m.DbDatabase, tablesToExclude, tablesWhoseDataToExclude, transferResource.Name),
		},
	}
}

func (m *PostgresSyncRoot) GetLocalCommand(environment Environment) []SyncCommand {
	l := m.getEffectiveLocalDetails()
	transferResource := m.GetTransferResource(environment)
	return []SyncCommand{{
		command: fmt.Sprintf("PGPASSWORD=\"%s\" pg_restore -O -c -x -w -h%s -d%s -p%s -U%s %s", l.DbPassword, l.DbHostname, l.DbDatabase, l.DbPort, l.DbUsername, transferResource.Name),
	},
	}
}

func (m *PostgresSyncRoot) GetFilesToCleanup(environment Environment) []string {
	transferResource := m.GetTransferResource(environment)
	return []string{
		transferResource.Name,
	}
}

func (m *PostgresSyncRoot) GetTransferResource(environment Environment) SyncerTransferResource {
	return SyncerTransferResource{
		Name:        fmt.Sprintf("%vlagoon_sync_postgres_%v.sql", m.GetOutputDirectory(), m.TransferId),
		IsDirectory: false}
}

func (m *PostgresSyncRoot) SetTransferResource(transferResourceName string) error {
	m.TransferResourceOverride = transferResourceName
	return nil
}

func (root *PostgresSyncRoot) GetOutputDirectory() string {
	m := root.Config
	if len(m.OutputDirectory) == 0 {
		return "/tmp/"
	}
	return m.OutputDirectory
}

func (syncConfig *PostgresSyncRoot) getEffectiveLocalDetails() BasePostgresSync {
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
