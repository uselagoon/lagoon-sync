package synchers

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/uselagoon/lagoon-sync/utils"
)

type BaseCustomSyncCommands struct {
	Commands []string `yaml:"commands"`
}

type BaseCustomSync struct {
}

func (customConfig *BaseCustomSync) setDefaults() {
	// Defaults don't make sense here, so noop
}

type CustomSyncRoot struct {
	TransferResource string                 `yaml:"transfer-resource"`
	Source           BaseCustomSyncCommands `yaml:"source"`
	Target           BaseCustomSyncCommands `yaml:"target"`
}

func (m CustomSyncRoot) SetTransferResource(transferResourceName string) error {
	m.TransferResource = transferResourceName
	return nil
}

// Init related types and functions follow

type CustomSyncPlugin struct {
	isConfigEmpty bool
	CustomRoot    string
}

func (m BaseCustomSync) IsBaseCustomStructureEmpty() bool {
	return reflect.DeepEqual(m, BaseCustomSync{})
}

func (m CustomSyncPlugin) GetPluginId() string {
	if m.CustomRoot != "" {
		return m.CustomRoot
	}
	return "custom"
}

func (m CustomSyncPlugin) GetPluginAliases() []string {
	return []string{}
}

func GetCustomSync(configRoot SyncherConfigRoot, syncerName string) (Syncer, error) {

	m := CustomSyncPlugin{
		CustomRoot: syncerName,
	}

	ret, err := m.UnmarshallYaml(configRoot)
	if err != nil {
		return CustomSyncRoot{}, err
	}

	return ret, nil
}

func (m CustomSyncPlugin) UnmarshallYaml(root SyncherConfigRoot) (Syncer, error) {
	custom := CustomSyncRoot{}

	// Use 'environment-defaults' if present
	envVars := root.Prerequisites
	var configMap interface{}

	configMap = root.LagoonSync[m.GetPluginId()]

	if envVars == nil {
		// Use 'lagoon-sync' yaml as override if env vars are not available
		configMap = root.LagoonSync[m.GetPluginId()]
	}

	// If config from active config file is empty, then use defaults
	if configMap == nil {
		utils.LogDebugInfo("Active syncer config is empty, so using defaults", custom)
	}

	// unmarshal environment variables as defaults
	err := UnmarshalIntoStruct(configMap, &custom)
	if err != nil {

	}

	if len(root.LagoonSync) != 0 {
		_ = UnmarshalIntoStruct(configMap, &custom)
		utils.LogDebugInfo("Config that will be used for sync", custom)
	}

	lagoonSyncer, _ := custom.PrepareSyncer()

	if custom.TransferResource == "" {
		return lagoonSyncer, errors.New("Transfer resource MUST be set on custom syncher definition")
	}

	return lagoonSyncer, nil
}

func init() {
	RegisterSyncer(CustomSyncPlugin{})
}

func (m CustomSyncRoot) IsInitialized() (bool, error) {
	return true, nil
}

// Sync related functions follow
func (root CustomSyncRoot) PrepareSyncer() (Syncer, error) {
	//root.TransferId = strconv.FormatInt(time.Now().UnixNano(), 10)
	return root, nil
}

func (root CustomSyncRoot) GetPrerequisiteCommand(environment Environment, command string) SyncCommand {
	lagoonSyncBin, _ := utils.FindLagoonSyncOnEnv()

	return SyncCommand{
		command: fmt.Sprintf("{{ .bin }} {{ .command }} || true"),
		substitutions: map[string]interface{}{
			"bin":     lagoonSyncBin,
			"command": command,
		},
	}
}

func (root CustomSyncRoot) GetRemoteCommand(sourceEnvironment Environment) []SyncCommand {

	transferResource := root.GetTransferResource(sourceEnvironment)

	ret := []SyncCommand{}

	substitutions := map[string]interface{}{
		"transferResource": transferResource.Name,
	}

	for _, c := range root.Source.Commands {
		ret = append(ret, generateSyncCommand(c, substitutions))
	}

	return ret
}

func (m CustomSyncRoot) GetLocalCommand(targetEnvironment Environment) []SyncCommand {
	transferResource := m.GetTransferResource(targetEnvironment)

	ret := []SyncCommand{}

	substitutions := map[string]interface{}{
		"transferResource": transferResource.Name,
	}

	for _, c := range m.Target.Commands {
		ret = append(ret, generateSyncCommand(c, substitutions))
	}

	return ret
}

func (m CustomSyncRoot) GetFilesToCleanup(environment Environment) []string {
	transferResource := m.GetTransferResource(environment)
	return []string{
		transferResource.Name,
	}
}

func (m CustomSyncRoot) GetTransferResource(environment Environment) SyncerTransferResource {
	return SyncerTransferResource{
		Name:        m.TransferResource,
		IsDirectory: false}
}
