package synchers

type Syncer interface {
	GetRemoteCommand() string
	GetLocalCommand() string
	GetTransferResourceName() string
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
