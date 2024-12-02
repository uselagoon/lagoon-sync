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
var syncerMap = []SyncerPlugin{}

type SyncerPlugin interface {
	GetPluginId() string
	GetPluginAliases() []string
	UnmarshallYaml(root SyncherConfigRoot) (Syncer, error)
}

func RegisterSyncer(plugin SyncerPlugin) {
	syncerMap = append(syncerMap, plugin)
}

func GetSyncerForTypeFromConfigRoot(syncerId string, root SyncherConfigRoot) (Syncer, error) {
	for _, v := range syncerMap {
		if v.GetPluginId() == syncerId {
			return v.UnmarshallYaml(root)
		}
	}
	return nil, errors.New(fmt.Sprintf("Syncer of type '%s' not registered", syncerId))
}

// ResolveSyncerIdFromAlias is used in the case where a user has come into the sync process using an
// alias, this resolves it to the proper syncer id
func ResolveSyncerIdFromAlias(syncerId string) (string, error) {
	for _, v := range syncerMap {
		if v.GetPluginId() == syncerId {
			return syncerId, nil
		}
		for _, m := range v.GetPluginAliases() {
			if m == syncerId {
				return v.GetPluginId(), nil
			}
		}
	}
	return "", errors.New(fmt.Sprintf("Syncer of type '%s' not registered", syncerId))
}

func ListSyncers(withAliases bool) []string {
	syncerids := []string{}

	for _, v := range syncerMap {
		syncerids = append(syncerids, v.GetPluginId())
		if withAliases {
			syncerids = append(syncerids, v.GetPluginAliases()...)
		}
	}

	return syncerids
}
