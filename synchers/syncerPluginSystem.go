package synchers

import (
	"errors"
	"fmt"
)

/**
* This file contains the global logic for syncer plugin registration
*
* The idea is, roughly, that syncers will register themselves here and that will make them dynamically available
* to unmarshall Yaml config data and be instantiated by client code generally
 */

// syncerMap maps plugin identifiers (eg. "mariadb", "files", etc.) with code that will unmarshall yaml
// and instantiate the syncers themselves.
var syncerMap = map[string]SyncerPlugin{}

type SyncerPlugin interface {
	GetPluginId() string
	UnmarshallYaml(root SyncherConfigRoot) (Syncer, error)
}

func RegisterSyncer(plugin SyncerPlugin) {
	syncerMap[plugin.GetPluginId()] = plugin
}

func GetSyncerForTypeFromConfigRoot(syncerId string, root SyncherConfigRoot) (Syncer, error) {
	if syncerMap[syncerId] == nil {
		return nil, errors.New(fmt.Sprintf("Syncer of type '%s' not registered", syncerId))
	}
	return syncerMap[syncerId].UnmarshallYaml(root)
}
