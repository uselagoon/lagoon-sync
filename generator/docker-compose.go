package generator

import (
	"github.com/compose-spec/compose-go/cli"
	"github.com/compose-spec/compose-go/loader"
	"github.com/compose-spec/compose-go/types"
	"strings"
)

// docker-compose.go contains all the functionality needed to parse docker compose files for lagoon labels
// and generate sync definitions

type LagoonServiceDefinition struct {
	ServiceName string
	ServiceType string
	image       string
	Labels      map[string]string
}

func LoadComposeFile(composeFile string) (*types.Project, error) {
	// Load the Compose file

	// NOTE: importantly - we're using a very permissive parsing scheme here
	// including the upstream library used in the lagoon build-deploy tool
	// we're only interested in pulling the service definitions here - so it's not so much of a problem.
	projectOptions, err := cli.NewProjectOptions([]string{composeFile},
		cli.WithResolvedPaths(false),
		cli.WithLoadOptions(
			loader.WithSkipValidation,
			loader.WithDiscardEnvFiles,
			func(o *loader.Options) {
				o.IgnoreNonStringKeyErrors = true
				o.IgnoreMissingEnvFileCheck = true
			},
		),
	)
	if err != nil {
		return nil, err
	}

	project, err := cli.ProjectFromOptions(projectOptions)
	if err != nil {
		return nil, err
	}

	return project, nil
}

func ProcessServicesFromCompose(project *types.Project) []LagoonServiceDefinition {
	var serviceDefinitions []LagoonServiceDefinition
	for _, service := range project.Services {
		sd := LagoonServiceDefinition{
			ServiceName: service.Name,
			image:       service.Image,
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
