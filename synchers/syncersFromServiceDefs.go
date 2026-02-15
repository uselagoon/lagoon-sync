package synchers

import (
	"fmt"
	"strings"

	"github.com/uselagoon/lagoon-sync/utils"
)

// NewBaseMariaDbSyncFromService returns a MariadbSyncRoot populated from a service definition.
// Returns an error for unsupported service types.
func NewBaseMariaDbSyncRootFromService(service utils.Service) (Syncer, error) {
	allowed := map[string]bool{
		"mariadb-single": true,
		"mariadb-dbaas":  true,
		"mariadb":        true,
	}

	serviceType := service.Type
	if serviceType == "" {
		serviceType = service.Name
	}

	if !allowed[serviceType] {
		return &MariadbSyncRoot{}, fmt.Errorf("unsupported mariadb service type: %s", serviceType)
	}

	name := strings.ToUpper(service.Name)
	retSyncRoot := &MariadbSyncRoot{
		ServiceName: service.Name,
		Type:        MariadbSyncPlugin{}.GetPluginId(),
		Config: BaseMariaDbSync{
			DbHostname: fmt.Sprintf("${%v_HOST}", name),
			DbUsername: fmt.Sprintf("${%v_USERNAME}", name),
			DbPassword: fmt.Sprintf("${%v_PASSWORD}", name),
			DbPort:     fmt.Sprintf("${%v_PORT}", name),
			DbDatabase: fmt.Sprintf("${%v_DATABASE}", name),
		},
	}

	retSyncRoot.PrepareSyncer()

	return retSyncRoot, nil
}

// NewBasePostgresSyncFromService returns a PostgresSyncRoot populated from a service definition.
// Returns an error for unsupported service types.
func NewBasePostgresSyncRootFromService(service utils.Service) (Syncer, error) {
	allowed := map[string]bool{
		"postgres-single": true,
		"postgres-dbaas":  true,
		"postgres":        true,
	}

	serviceType := service.Type
	if serviceType == "" {
		serviceType = service.Name
	}

	if !allowed[serviceType] {
		return &PostgresSyncRoot{}, fmt.Errorf("unsupported postgres service type: %s", serviceType)
	}

	name := strings.ToUpper(service.Name)
	retSyncRoot := &PostgresSyncRoot{
		ServiceName: service.Name,
		Type:        PostgresSyncPlugin{}.GetPluginId(),
		Config: BasePostgresSync{
			DbHostname: fmt.Sprintf("${%v_HOST}", name),
			DbUsername: fmt.Sprintf("{${%v_USERNAME}", name),
			DbPassword: fmt.Sprintf("{${%v_PASSWORD}", name),
			DbPort:     fmt.Sprintf("{${%v_PORT}", name),
			DbDatabase: fmt.Sprintf("{${%v_DATABASE}", name),
		},
	}
	retSyncRoot.PrepareSyncer()

	return retSyncRoot, nil
}

// NewBaseFilesSyncFromService returns a FilesSyncRoot populated from a service definition.
// Returns an error if the service has no volumes.
func NewBaseFilesSyncRootFromService(service utils.Service, volumePath string) (Syncer, error) {
	if len(service.Volumes) == 0 {
		return &FilesSyncRoot{}, fmt.Errorf("service %s has no volumes to sync", service.Name)
	}

	if volumePath == "" {
		return &FilesSyncRoot{}, fmt.Errorf("volume path is required for file sync")
	}

	retSyncRoot := &FilesSyncRoot{
		ServiceName: service.Name,
		Type:        FilesSyncPlugin{}.GetPluginId(),
		Config: BaseFilesSync{
			SyncPath: volumePath,
		},
	}
	retSyncRoot.PrepareSyncer()

	return retSyncRoot, nil
}
