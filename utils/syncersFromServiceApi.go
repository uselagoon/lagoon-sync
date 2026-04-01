package utils

import (
	"errors"

	lclient "github.com/uselagoon/machinery/api/lagoon/client"
	"github.com/uselagoon/machinery/api/schema"
	"golang.org/x/net/context"
)

func (r *ApiConn) GetServicesForEnvironment(projectName, environmentName string) ([]schema.EnvironmentService, error) {
	if r.token == "" {
		return nil, errors.New("ApiConn has not been initialized")
	}
	lc := lclient.New(r.graphqlEndpoint, userAgentString, minLagoonApiVersion, &r.token, false)
	// environments := []schema.Environment{}
	environment := schema.Environment{}
	err := lc.EnvironmentByNameAndProjectName(context.TODO(), environmentName, projectName, &environment)
	if err != nil {
		return nil, err
	}

	return environment.Services, nil
}
