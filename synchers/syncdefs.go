package synchers

import (
	"fmt"
	"strings"
)

//TODO: we may want to have these return slightly more complex types
// if we want to do more interesting stuff with the return details
type Syncer interface {
	GetRemoteCommand() string
	GetLocalCommand() string
	GetTransferResource() SyncerTransferResource
	PrepareSyncer() Syncer
}

// SyncerTransferResource describes what it is the is produced by the actions of GetRemoteCommand()
type SyncerTransferResource struct {
	Name        string
	IsDirectory bool
}

type RemoteEnvironment struct {
	ProjectName     string
	EnvironmentName string
}

func (r RemoteEnvironment) getOpenshiftProjectName() string {
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
