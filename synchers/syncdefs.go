package synchers

import (
	"fmt"
	"strings"
)

type Syncer interface {
	// GetRemoteCommand will return the command to be run on the source system
	GetRemoteCommand(environment Environment) SyncCommand
	// GetLocalCommand will return the command to be run on the target system
	GetLocalCommand(environment Environment) SyncCommand
	GetTransferResource() SyncerTransferResource
	// PrepareSyncer does any preparations required on a Syncer before it is used
	PrepareSyncer() Syncer
}

type SyncCommand struct {
	command string
}

// SyncerTransferResource describes what it is the is produced by the actions of GetRemoteCommand()
type SyncerTransferResource struct {
	Name        string
	IsDirectory bool
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
	TransferId   string               // a unique id which can be used to identify this entire transaction
}

// SyncherConfigRoot is used to unmarshall yaml config details generally
type SyncherConfigRoot struct {
	Project    string
	LagoonSync LagoonSync `yaml:"lagoon-sync"`
}
