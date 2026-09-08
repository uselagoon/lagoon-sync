package utils

import (
	"errors"
	"fmt"

	lclient "github.com/uselagoon/machinery/api/lagoon/client"
	"github.com/uselagoon/machinery/api/schema"
	"github.com/uselagoon/machinery/utils/sshtoken"
	"golang.org/x/net/context"
)

// sshportal.go contains the functionality we need for connecting to the ssh portal and grab a list of deploy targets and environments

const userAgentString = "lagoon-sync"
const minLagoonApiVersion = "2.18.0"

type ApiConn struct {
	graphqlEndpoint string
	token           string
	sshHost         string
	sshPort         string
}

// Init initialises the API connection, retrieving an SSH token and storing connection details.
// tokenHost and tokenPort, if non-empty, override sshHost and sshPort solely for token retrieval
// (e.g. when an SSH portal is in use and the token endpoint differs from the tunnel endpoint).
func (r *ApiConn) Init(graphqlEndpoint, sshkeyPath, sshHost, sshPort, tokenHost, tokenPort string) error {
	// Determine which host/port to use for token retrieval.
	resolvedTokenHost := sshHost
	if tokenHost != "" {
		resolvedTokenHost = tokenHost
	}
	resolvedTokenPort := sshPort
	if tokenPort != "" {
		resolvedTokenPort = tokenPort
	}

	token, err := sshtoken.RetrieveToken(sshkeyPath, resolvedTokenHost, resolvedTokenPort, nil, nil, false)
	if err != nil {
		return err
	}

	// TODO: we could, perhaps, add some assertions here regarding the format of these incoming values
	r.token = token
	r.sshHost = sshHost
	r.sshPort = sshPort
	r.graphqlEndpoint = graphqlEndpoint
	return nil
}

func (r *ApiConn) GetProjectEnvironmentDeployTargets(projectName string) (*[]schema.Environment, error) {
	if r.token == "" {
		return nil, errors.New("ApiConn has not been initialized")
	}
	lc := lclient.New(r.graphqlEndpoint, userAgentString, minLagoonApiVersion, &r.token, false)
	environments := []schema.Environment{}
	err := lc.EnvironmentsByProjectName(context.TODO(), projectName, &environments)
	if err != nil {
		return nil, fmt.Errorf("could not get deploy targets: %w", err)
	}

	return &environments, nil
}
