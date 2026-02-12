package utils

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

// This file contains the utilities to read the service definitions in a docker-compose.yml file and output a list of services and files to sync.
// The assumption is that if they exist on one service, they'll exist on another.
// Assuming this works, then the next step will be to pull these same data from the services api

// DockerCompose represents the root structure of a docker-compose.yml file
type DockerCompose struct {
	Services map[string]*Service `yaml:"services"`
}

// Service represents a single service definition in docker-compose.yml
type Service struct {
	Labels  map[string]string `yaml:"labels"`
	Name    string
	Type    string
	Volumes map[string]string
}

// UnmarshalYAML implements custom unmarshaling for Service to handle malformed YAML
func (s *Service) UnmarshalYAML(unmarshal func(interface{}) error) error {
	// Use a raw map to unmarshal the service definition
	type rawService struct {
		Labels interface{} `yaml:"labels"`
	}

	var raw rawService
	err := unmarshal(&raw)
	if err != nil {
		return err
	}

	// Initialize fields
	s.Labels = make(map[string]string)
	s.Volumes = make(map[string]string)

	// Handle labels - could be a map or other types
	if raw.Labels != nil {
		switch labels := raw.Labels.(type) {
		case map[interface{}]interface{}:
			// Convert map[interface{}]interface{} to map[string]string, only including string values
			for k, v := range labels {
				if keyStr, ok := k.(string); ok {
					if valStr, ok := v.(string); ok {
						s.Labels[keyStr] = valStr
					}
				}
			}
		case map[string]interface{}:
			// Convert map[string]interface{} to map[string]string, only including string values
			for k, v := range labels {
				if valStr, ok := v.(string); ok {
					s.Labels[k] = valStr
				}
			}
		case map[string]string:
			// Already the right type
			s.Labels = labels
		}
	}

	return nil
}

// Build represents the build configuration for a service
type Build struct {
	Context    string `yaml:"context"`
	Dockerfile string `yaml:"dockerfile"`
}

func LoadDockerCompose(path string) (map[string]Service, error) {
	// Check if file exists
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", path)
		}
		return nil, fmt.Errorf("error checking file: %w", err)
	}

	// Read file contents
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	// Unmarshal YAML into DockerCompose struct
	var compose DockerCompose
	err = yaml.Unmarshal(data, &compose)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling YAML: %w", err)
	}

	// now we iterate through the services and see if we have any lagoon services
	for serviceName, serviceDef := range compose.Services {
		compose.Services[serviceName].Name = serviceName
		compose.Services[serviceName].Volumes = map[string]string{} // let's assign the volumes here so we can fill it.
		for k, v := range serviceDef.Labels {
			switch {
			case k == "lagoon.type":
				compose.Services[serviceName].Type = v
			case k == "lagoon.persistent":
				compose.Services[serviceName].Volumes[v] = v
			case strings.HasPrefix(k, "lagoon.volumes.") && strings.HasSuffix(k, ".path"):
				parts := strings.Split(k, ".")
				if len(parts) >= 4 {
					volName := parts[2]
					compose.Services[serviceName].Volumes[volName] = v
				}
			}
		}
	}

	// Convert pointers to values for the return map
	result := make(map[string]Service)
	for name, service := range compose.Services {
		result[name] = *service
	}

	return result, nil
}
