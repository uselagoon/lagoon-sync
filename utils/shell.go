package utils

import (
	"bytes"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"net"
	"os"
	"os/exec"
)

const ShellToUse = "sh"

func Shellout(command string) (error, string, string) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command(ShellToUse, "-c", command)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return err, stdout.String(), stderr.String()
}

func RemoteShellout(command string, remoteUser string, remoteHost string, remotePort string, privateKeyfile string) (error, string, string) {
	// Read the private key file

	skipAgent := false

	var authMethods []ssh.AuthMethod

	if skipAgent != true {
		// Connect to SSH agent to ask for unencrypted private keys
		if sshAgentConn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
			sshAgent := agent.NewClient(sshAgentConn)
			keys, _ := sshAgent.List()
			if len(keys) > 0 {
				agentAuthmethods := ssh.PublicKeysCallback(sshAgent.Signers)
				authMethods = append(authMethods, agentAuthmethods)
			}
		}
	}

	privateKeyBytes, err := os.ReadFile(privateKeyfile)

	// if there are authMethods already, let's keep going
	if err != nil && len(authMethods) == 0 {
		return err, "", ""
	}

	if len(privateKeyBytes) > 0 {
		// Parse the private key
		signer, err := ssh.ParsePrivateKey(privateKeyBytes)
		if err != nil {
			return err, "", ""
		}

		// SSH client configuration
		authKeys := ssh.PublicKeys(signer)
		authMethods = append(authMethods, authKeys)
	}

	config := &ssh.ClientConfig{
		User:            remoteUser,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// Connect to the remote server
	client, err := ssh.Dial("tcp", remoteHost+":"+remotePort, config)
	if err != nil {
		return err, "", ""
	}

	defer client.Close()

	// Create a session
	session, err := client.NewSession()
	if err != nil {
		return err, "", ""
	}
	defer session.Close()

	// Set up pipes for stdin, stdout, and stderr
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	stdin, err := session.StdinPipe()
	if err != nil {
		return err, "", ""
	}

	// Start the remote command
	err = session.Start(command)
	if err != nil {
		return err, "", ""
	}
	// Wait for the command to complete
	err = session.Wait()
	if err != nil {
		return err, "", ""
	}

	// Close the stdin pipe
	stdin.Close()

	return nil, "", ""
}
