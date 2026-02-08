package synchers

import (
	"fmt"
	"strings"

	"github.com/uselagoon/lagoon-sync/utils"
)

// NewBaseMariaDbSyncFromService returns a BaseMariaDbSync populated from a service definition.
// Returns an error for unsupported service types.
func NewBaseMariaDbSyncFromService(service utils.Service) (BaseMariaDbSync, error) {
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
		return BaseMariaDbSync{}, fmt.Errorf("unsupported mariadb service type: %s", serviceType)
	}

	name := strings.ToUpper(service.Name)
	return BaseMariaDbSync{
		DbHostname: fmt.Sprintf("%v_HOST", name),
		DbUsername: fmt.Sprintf("%v_USERNAME", name),
		DbPassword: fmt.Sprintf("%v_PASSWORD", name),
		DbPort:     fmt.Sprintf("%v_PORT", name),
		DbDatabase: fmt.Sprintf("%v_DATABASE", name),
	}, nil
}
