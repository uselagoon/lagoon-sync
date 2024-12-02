package synchers

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
	"github.com/uselagoon/lagoon-sync/utils"
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

type MariadbSyncLocal struct {
	Config BaseMariaDbSync
}

type MariadbSyncRoot struct {
	Config                   BaseMariaDbSync
	LocalOverrides           MariadbSyncLocal `yaml:"local"`
	TransferId               string
	TransferResourceOverride string
}

func (mariadbConfig *BaseMariaDbSync) setDefaults() {
	if mariadbConfig.DbHostname == "" {
		mariadbConfig.DbHostname = "${MARIADB_HOST:-mariadb}"
	}
	if mariadbConfig.DbUsername == "" {
		mariadbConfig.DbUsername = "${MARIADB_USERNAME:-drupal}"
	}
	if mariadbConfig.DbPassword == "" {
		mariadbConfig.DbPassword = "${MARIADB_PASSWORD:-drupal}"
	}
	if mariadbConfig.DbPort == "" {
		mariadbConfig.DbPort = "${MARIADB_PORT:-3306}"
	}
	if mariadbConfig.DbDatabase == "" {
		mariadbConfig.DbDatabase = "${MARIADB_DATABASE:-drupal}"
	}
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

func (m MariadbSyncPlugin) GetPluginAliases() []string {
	return []string{"mysql"}
}

func (m MariadbSyncPlugin) UnmarshallYaml(root SyncherConfigRoot) (Syncer, error) {
	mariadb := MariadbSyncRoot{}
	mariadb.Config.setDefaults()
	mariadb.LocalOverrides.Config.setDefaults()

	syncherConfig := root.LagoonSync[m.GetPluginId()]

	// If yaml config is there then unmarshall into struct and override default values if there are any
	if syncherConfig != nil {
		_ = UnmarshalIntoStruct(syncherConfig, &mariadb)
		utils.LogDebugInfo("Config that will be used for sync", mariadb)
	} else {
		// If config from active config file is empty, then use defaults
		if syncherConfig == nil {
			utils.LogDebugInfo("Active syncer config is empty, so using defaults", mariadb)
		}
	}
	if mariadb.Config.IsBaseMariaDbStructureEmpty() && &mariadb == nil {
		m.isConfigEmpty = true
		utils.LogFatalError("No syncer configuration could be found in", viper.GetViper().ConfigFileUsed())
	}

	lagoonSyncer, _ := mariadb.PrepareSyncer()
	return lagoonSyncer, nil
}

func init() {
	RegisterSyncer(MariadbSyncPlugin{})
}

func (m *MariadbSyncRoot) IsInitialized() (bool, error) {

	var missingEnvvars []string

	if m.Config.DbHostname == "" {
		missingEnvvars = append(missingEnvvars, "hostname")
	}
	if m.Config.DbUsername == "" {
		missingEnvvars = append(missingEnvvars, "username")
	}
	if m.Config.DbPassword == "" {
		missingEnvvars = append(missingEnvvars, "password")
	}
	if m.Config.DbPort == "" {
		missingEnvvars = append(missingEnvvars, "port")
	}
	if m.Config.DbDatabase == "" {
		missingEnvvars = append(missingEnvvars, "database")
	}

	if len(missingEnvvars) > 0 {
		return false, errors.New(fmt.Sprintf("Missing configuration values: %v", strings.Join(missingEnvvars, ",")))
	}

	return true, nil
}

func (root *MariadbSyncRoot) PrepareSyncer() (Syncer, error) {
	root.TransferId = strconv.FormatInt(time.Now().UnixNano(), 10)
	return root, nil
}

func (root *MariadbSyncRoot) GetPrerequisiteCommand(environment Environment, command string) SyncCommand {
	lagoonSyncBin, _ := utils.FindLagoonSyncOnEnv()

	return SyncCommand{
		command: fmt.Sprintf("{{ .bin }} {{ .command }} || true"),
		substitutions: map[string]interface{}{
			"bin":     lagoonSyncBin,
			"command": command,
		},
	}
}

func (root *MariadbSyncRoot) GetRemoteCommand(sourceEnvironment Environment) []SyncCommand {
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

	//We remove the `.gz` from the transfer resource name for because we _first_ generate a plain `.sql` file
	//and _then_ gzip it
	resourceNameWithoutGz := strings.TrimSuffix(transferResource.Name, filepath.Ext(transferResource.Name))
	substitutions := map[string]interface{}{
		"dumpOptions":      "--max-allowed-packet=500M --quick --add-locks --no-autocommit --single-transaction",
		"hostname":         m.DbHostname,
		"username":         m.DbUsername,
		"password":         m.DbPassword,
		"port":             m.DbPort,
		"tablesToIgnore":   tablesWhoseDataToIgnore,
		"database":         m.DbDatabase,
		"transferResource": resourceNameWithoutGz,
	}
	return []SyncCommand{
		{
			command:       fmt.Sprintf("mysqldump {{ .dumpOptions }} -h{{ .hostname }} -u{{ .username }} -p{{ .password }} -P{{ .port }} {{ .tablesToIgnore }} {{ .database }} > {{ .transferResource }}"),
			substitutions: substitutions,
		},
		{
			command:       fmt.Sprintf("gzip {{ .transferResource }}"),
			substitutions: substitutions,
		},
	}
}

func (m *MariadbSyncRoot) GetLocalCommand(targetEnvironment Environment) []SyncCommand {
	l := m.Config
	if targetEnvironment.EnvironmentName == LOCAL_ENVIRONMENT_NAME {
		l = m.getEffectiveLocalDetails()
	}
	transferResource := m.GetTransferResource(targetEnvironment)
	resourceNameWithoutGz := strings.TrimSuffix(transferResource.Name, filepath.Ext(transferResource.Name))
	return []SyncCommand{
		generateSyncCommand("gunzip {{ .transferResource }}",
			map[string]interface{}{
				"hostname":         l.DbHostname,
				"username":         l.DbUsername,
				"password":         l.DbPassword,
				"port":             l.DbPort,
				"database":         l.DbDatabase,
				"transferResource": transferResource.Name,
			}),
		generateSyncCommand("mysql -h{{ .hostname }} -u{{ .username }} -p{{ .password }} -P{{ .port }} {{ .database }} < {{ .resourceNameWithoutGz }}",
			map[string]interface{}{
				"hostname":              l.DbHostname,
				"username":              l.DbUsername,
				"password":              l.DbPassword,
				"port":                  l.DbPort,
				"database":              l.DbDatabase,
				"resourceNameWithoutGz": resourceNameWithoutGz,
			}),
	}
}

func (m *MariadbSyncRoot) GetFilesToCleanup(environment Environment) []string {
	transferResource := m.GetTransferResource(environment)
	resourceNameWithoutGz := strings.TrimSuffix(transferResource.Name, filepath.Ext(transferResource.Name))
	return []string{
		transferResource.Name,
		resourceNameWithoutGz,
	}
}

func (m *MariadbSyncRoot) GetTransferResource(environment Environment) SyncerTransferResource {
	resourceName := fmt.Sprintf("%vlagoon_sync_mariadb_%v.sql.gz", m.GetOutputDirectory(), m.TransferId)
	if m.TransferResourceOverride != "" {
		resourceName = m.TransferResourceOverride
	}
	return SyncerTransferResource{
		Name:        resourceName,
		IsDirectory: false}
}

func (m *MariadbSyncRoot) SetTransferResource(transferResourceName string) error {
	m.TransferResourceOverride = transferResourceName
	return nil
}

func (root *MariadbSyncRoot) GetOutputDirectory() string {
	m := root.Config
	if len(m.OutputDirectory) == 0 {
		return "/tmp/"
	}
	return m.OutputDirectory
}

func (syncConfig *MariadbSyncRoot) getEffectiveLocalDetails() BaseMariaDbSync {
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
