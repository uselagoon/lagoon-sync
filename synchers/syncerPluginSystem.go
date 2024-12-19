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
	UnmarshallYaml(root SyncherConfigRoot, targetService string) (Syncer, error)
}

func RegisterSyncer(plugin SyncerPlugin) {
	syncerMap[plugin.GetPluginId()] = plugin
}

func GetSyncerForTypeFromConfigRoot(syncerId string, root SyncherConfigRoot) (Syncer, error) {

	// we may want to first check if there's an explicit type attached to this syncerId
	SyncerConfig, exists := root.LagoonSync[syncerId]
	if exists {
		configTypeStruct := struct {
			Type string `yaml:"type" json:"type"`
		}{Type: ""}
		_ = UnmarshalIntoStruct(SyncerConfig, &configTypeStruct)

		// We've found an alias in the config that implements a "type"
		if configTypeStruct.Type != "" {
			return syncerMap[configTypeStruct.Type].UnmarshallYaml(root, syncerId)
		}
	}

	if syncerMap[syncerId] == nil {
		return nil, errors.New(fmt.Sprintf("Syncer of type '%s' not registered", syncerId))
	}

	return syncerMap[syncerId].UnmarshallYaml(root, syncerId)

}
