package synchers

import (
	"fmt"
	"strings"

	"github.com/amazeeio/lagoon-sync/prerequisite"
	"gopkg.in/yaml.v2"
)

const LOCAL_ENVIRONMENT_NAME = "local"

type Syncer interface {
	// GetPrerequisiteCommand(environmnt Environment, command string) SyncCommand
	// GetRemoteCommand will return the command to be run on the source system
	GetRemoteCommand(environment Environment) SyncCommand
	// GetLocalCommand will return the command to be run on the target system
	GetLocalCommand(environment Environment) SyncCommand
	GetTransferResource(environment Environment) SyncerTransferResource
	// PrepareSyncer does any preparations required on a Syncer before it is used
	PrepareSyncer() (Syncer, error)
}

type SyncCommand struct {
	command       string
	substitutions map[string]interface{}
	NoOp          bool // NoOp can be set to true if this command performs no operation (in situations like file transfers)
}

// SyncerTransferResource describes what it is the is produced by the actions of GetRemoteCommand()
type SyncerTransferResource struct {
	Name             string
	IsDirectory      bool
	ExcludeResources []string // ExcludeResources is a string list of any resources that aren't to be included in the transfer
	SkipCleanup      bool
}

type Environment struct {
	ProjectName     string
	EnvironmentName string
	ServiceName     string // This is used to determine which Lagoon service we need to rsync
	RsyncAvailable  bool
	RsyncPath       string
	RsyncLocalPath  string
}

func (r Environment) getOpenshiftProjectName() string {
	return fmt.Sprintf("%s-%s", strings.ToLower(r.ProjectName), strings.ToLower(r.EnvironmentName))
}

// SyncherConfigRoot is used to unmarshall yaml config details generally
type SyncherConfigRoot struct {
	Project       string                 `yaml:"project"`
	LagoonSync    map[string]interface{} `yaml:"lagoon-sync"`
	Prerequisites []prerequisite.GatheredPrerequisite
}

// takes interface, marshals back to []byte, then unmarshals to desired struct
// from https://github.com/go-yaml/yaml/issues/13#issuecomment-428952604
func UnmarshalIntoStruct(pluginIn interface{}, pluginOut interface{}) error {
	b, err := yaml.Marshal(pluginIn)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(b, pluginOut)
}
