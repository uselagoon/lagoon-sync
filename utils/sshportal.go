package utils

import (
	"errors"
	lclient "github.com/uselagoon/machinery/api/lagoon/client"
	"github.com/uselagoon/machinery/api/schema"
	"github.com/uselagoon/machinery/utils/sshtoken"
	"golang.org/x/net/context"
)

// sshportal.go contains the functionality we need for connecting to the ssh portal and grab a list of deploy targets and environments

const userAgentString = "lagoon-sync"

type ApiConn struct {
	graphqlEndpoint string
	token           string
	sshHost         string
	sshPort         string
}

func (r *ApiConn) Init(graphqlEndpoint, sshkeyPath, sshHost, sshPort string) error {
	token, err := sshtoken.RetrieveToken(sshkeyPath, sshHost, sshPort, nil, nil, false)
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
	lc := lclient.New(r.graphqlEndpoint, userAgentString, "", &r.token, false)
	environments := []schema.Environment{}
	err := lc.EnvironmentsByProjectName(context.TODO(), projectName, &environments)
	if err != nil {
		return nil, errors.New("ApiConn has not been initialized")
	}

	return &environments, nil
}
