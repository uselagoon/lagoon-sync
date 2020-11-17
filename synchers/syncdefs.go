package synchers

import (
	"fmt"
	"strings"
)

// consts are defined here

const LOCAL_ENVIRONMENT_NAME = "local"

// general interfaces defined below

type Syncer interface {
	// GetRemoteCommand will return the command to be run on the source system
	GetRemoteCommand(environment Environment) SyncCommand
	// GetLocalCommand will return the command to be run on the target system
	GetLocalCommand(environment Environment) SyncCommand
	GetTransferResource(environment Environment) SyncerTransferResource
	// PrepareSyncer does any preparations required on a Syncer before it is used
	PrepareSyncer() (Syncer, error)
}

type SyncCommand struct {
	command string
	substitutions map[string]interface{}
	NoOp bool // NoOp can be set to true if this command performs no operation (in situations like file transfers)
}

// SyncerTransferResource describes what it is the is produced by the actions of GetRemoteCommand()
type SyncerTransferResource struct {
	Name        string
	IsDirectory bool
	ExcludeResources []string // ExcludeResources is a string list of any resources that aren't to be included in the transfer
	SkipCleanup bool
}

type Environment struct {
	ProjectName     string
	EnvironmentName string
}

func (r Environment) getOpenshiftProjectName() string {
	return fmt.Sprintf("%s-%s", strings.ToLower(r.ProjectName), strings.ToLower(r.EnvironmentName))
}

// The following is the root structure for unmarshalling yaml configurations
// Each syncer must register its structure here
type LagoonSync struct {
	Mariadb      MariadbSyncRoot      `yaml:"mariadb"`
	Postgres     PostgresSyncRoot     `yaml:"postgres"`
	Drupalconfig DrupalconfigSyncRoot `yaml:"drupalconfig"`
	Filesconfig  FilesSyncRoot        `yaml:"files"`
	TransferId   string               // a unique id which can be used to identify this entire transaction
}

// SyncherConfigRoot is used to unmarshall yaml config details generally
type SyncherConfigRoot struct {
	Project    string
	LagoonSync LagoonSync `yaml:"lagoon-sync"`
}
