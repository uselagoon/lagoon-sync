package generator

import (
	"github.com/compose-spec/compose-go/v2/cli"
	"github.com/compose-spec/compose-go/v2/types"
	"golang.org/x/net/context"
	"strings"
)

// syncdefGenerator.go contains all the functionality needed to parse docker compose files for lagoon labels
// and generate sync definitions

type LagoonServiceDefinition struct {
	ServiceName string
	ServiceType string
	Labels      map[string]string
}

func LoadComposeFile(composeFile string) (*types.Project, error) {
	// Load the Compose file
	projectOptions, err := cli.NewProjectOptions([]string{composeFile}, cli.WithDefaultConfigPath)
	if err != nil {
		return nil, err
	}

	project, err := cli.ProjectFromOptions(context.TODO(), projectOptions)
	if err != nil {
		return nil, err
	}

	return project, nil
}

func ProcessServicesFromCompose(project *types.Project) []LagoonServiceDefinition {
	serviceDefinitions := []LagoonServiceDefinition{}
	for _, service := range project.Services {
		sd := LagoonServiceDefinition{
			ServiceName: service.Name,
			Labels:      map[string]string{},
		}
		// we only process this if this _has_ a lagoon.type, and that is not "none"
		lt, exists := service.Labels["lagoon.type"]
		lt = strings.ToLower(lt)
		if exists && lt != "none" {
			sd.ServiceType = lt
			for k, v := range service.Labels {
				sd.Labels[k] = v
			}
			serviceDefinitions = append(serviceDefinitions, sd)
		}
	}
	return serviceDefinitions
}
