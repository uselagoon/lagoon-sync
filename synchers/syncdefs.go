package synchers


//TODO: we may want to have these return slightly more complex types
// if we want to do more interesting stuff with the return details
type Syncer interface {
	GetRemoteCommand() string
	GetLocalCommand() string
	GetTransferResource() SyncerTransferResource
}

type SyncerTransferResource struct {
	Name string
	IsDirectory bool
}

// The following is the root structure for unmarshalling yaml configurations
// Each syncer must register its structure here
type LagoonSync struct {
	Mariadb MariadbSyncRoot
}

// SyncherConfigRoot is used to unmarshall yaml config details generally
type SyncherConfigRoot struct {
	Project string
	LagoonSync LagoonSync `yaml:"lagoon-sync"`
}
