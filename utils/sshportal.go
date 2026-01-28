package utils

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"

	lclient "github.com/uselagoon/machinery/api/lagoon/client"
	"github.com/uselagoon/machinery/api/schema"
	"github.com/uselagoon/machinery/utils/sshtoken"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
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
	// WORKAROUND: machinery's sshtoken.RetrieveToken has a bug where it returns errors when
	// identity files don't exist instead of falling back to SSH agent keys.
	// See: https://github.com/uselagoon/machinery/blob/main/utils/sshtoken/sshtoken.go
	// Related issue: https://github.com/uselagoon/lagoon-cli/issues/442
	//
	// When using agent-only (no key files), bypass machinery entirely to avoid the bug.

	var token string
	var err error

	// If no key file exists but agent has keys, use our own implementation
	if (sshkeyPath == "" || !fileExists(sshkeyPath)) && agentHasKeys() {
		if sshkeyPath != "" {
			LogDebugInfo(fmt.Sprintf("SSH key file %s does not exist, using SSH agent", sshkeyPath), os.Stdout)
		} else {
			LogDebugInfo("No SSH key file specified, using SSH agent", os.Stdout)
		}
		token, err = retrieveTokenViaAgent(sshHost, sshPort)
		if err != nil {
			return fmt.Errorf("failed to retrieve token via SSH agent: %w", err)
		}
	} else if fileExists(sshkeyPath) {
		// Key file exists, use machinery's implementation
		LogDebugInfo(fmt.Sprintf("Using SSH key file: %s", sshkeyPath), os.Stdout)
		token, err = sshtoken.RetrieveToken(sshkeyPath, sshHost, sshPort, nil, nil, false)
		if err != nil {
			return fmt.Errorf("failed to retrieve token: %w", err)
		}
	} else {
		return fmt.Errorf("no SSH keys available: SSH agent has no keys and no key file specified")
	}

	r.token = token
	r.sshHost = sshHost
	r.sshPort = sshPort
	r.graphqlEndpoint = graphqlEndpoint
	return nil
}

// retrieveTokenViaAgent retrieves a token using SSH agent keys only
// This bypasses machinery's buggy sshtoken.RetrieveToken when using agent-only
func retrieveTokenViaAgent(sshHost, sshPort string) (string, error) {
	sshAuthSock := os.Getenv("SSH_AUTH_SOCK")
	if sshAuthSock == "" {
		return "", fmt.Errorf("SSH_AUTH_SOCK not set")
	}

	// Connect to SSH agent
	agentConn, err := net.Dial("unix", sshAuthSock)
	if err != nil {
		return "", fmt.Errorf("failed to connect to SSH agent: %w", err)
	}
	defer agentConn.Close()

	agentClient := agent.NewClient(agentConn)

	// Create SSH client config using agent
	config := &ssh.ClientConfig{
		User: "lagoon",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeysCallback(agentClient.Signers),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// Connect to SSH host
	sshHostString := fmt.Sprintf("%s:%s", sshHost, sshPort)
	conn, err := ssh.Dial("tcp", sshHostString, config)
	if err != nil {
		return "", fmt.Errorf("failed to connect to %s: %w", sshHostString, err)
	}
	defer conn.Close()

	// Create session
	session, err := conn.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	// Get token
	out, err := session.CombinedOutput("token")
	if err != nil {
		return "", fmt.Errorf("failed to get token: %w", err)
	}

	return strings.TrimSpace(string(out)), nil
}

// agentHasKeys checks if SSH agent is available and has keys
func agentHasKeys() bool {
	sshAuthSock := os.Getenv("SSH_AUTH_SOCK")
	if sshAuthSock == "" {
		return false
	}

	conn, err := net.Dial("unix", sshAuthSock)
	if err != nil {
		return false
	}
	defer conn.Close()

	agentClient := agent.NewClient(conn)
	keys, err := agentClient.List()
	if err != nil {
		return false
	}

	return len(keys) > 0
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
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
