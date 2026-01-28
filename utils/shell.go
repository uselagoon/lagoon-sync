package utils

import (
	"bytes"
	"errors"
	"fmt"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"net"
	"os"
	"os/exec"
	"path/filepath"
)

const ShellToUse = "sh"

var validAuthMethod *ssh.AuthMethod

func Shellout(command string) (error, string, string) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command(ShellToUse, "-c", command)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Start()
	if err != nil {
		return err, "", ""
	}
	ShowSpinner()
	defer HideSpinner()
	err = cmd.Wait()
	return err, stdout.String(), stderr.String()
}

func getAuthMethodFromPrivateKey(filename string) (ssh.AuthMethod, error) {
	privateKeyBytes, err := os.ReadFile(filename)

	if err != nil {
		return nil, err
	}

	if len(privateKeyBytes) > 0 {
		// Parse the private key
		signer, err := ssh.ParsePrivateKey(privateKeyBytes)
		if err != nil {
			return nil, err
		}

		// SSH client configuration
		authKeys := ssh.PublicKeys(signer)
		return authKeys, nil

	}
	return nil, errors.New(fmt.Sprint("No data in privateKey: ", filename))
}

func getSSHAuthMethodsFromDirectory(directory string) ([]ssh.AuthMethod, error) {
	var authMethods []ssh.AuthMethod
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Ext(path) != ".pub" {

			// let's test this is a valid ssh key
			am, err := getAuthMethodFromPrivateKey(path)
			if err != nil {
				switch {
				case isPassphraseMissingError(err):
					LogDebugInfo(fmt.Sprintf("Found a passphrase based ssh key at %s: %v", path, err.Error()), os.Stdout)
				default:
					LogDebugInfo(fmt.Sprintf("Skipping %s: %v", path, err.Error()), os.Stdout)
				}
			} else {
				LogDebugInfo(fmt.Sprintf("Found a valid key at %v - will try auth", path), os.Stdout)
				authMethods = append(authMethods, am)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return authMethods, nil
}

func isPassphraseMissingError(err error) bool {
	_, ok := err.(*ssh.PassphraseMissingError)
	return ok
}

func RemoteShellout(command string, remoteUser string, remoteHost string, remotePort string, privateKeyfile string, skipSshAgent bool) (error, string) {

	sshAuthSock, present := os.LookupEnv("SSH_AUTH_SOCK")
	skipAgent := !present || skipSshAgent

	var authMethods []ssh.AuthMethod

	if validAuthMethod == nil { // This makes it so that in subsequent calls, we don't have to recheck all auth methods
		LogDebugInfo("First time running, no cached valid auth methods", os.Stdout)
		authMethods = getAuthmethods(skipAgent, privateKeyfile, sshAuthSock, authMethods)
	} else {
		LogDebugInfo("Found existing auth method", os.Stdout)
		authMethods = []ssh.AuthMethod{
			*validAuthMethod,
		}
	}

	if len(authMethods) == 0 && validAuthMethod == nil {
		return errors.New("No valid authentication methods provided"), ""
	}

	config := &ssh.ClientConfig{
		User:            remoteUser,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	var client *ssh.Client
	var err error

	//we need to iterate over the auth methods till we find one that works
	// for subsequent runs, this will only run once, since only whatever is in
	// validAuthMethod will be attemtped.
	for _, am := range authMethods {
		config.Auth = []ssh.AuthMethod{
			am,
		}
		client, err = ssh.Dial("tcp", remoteHost+":"+remotePort, config)
		if err != nil {
			continue
		}

		if validAuthMethod == nil {
			LogDebugInfo("Dial success - caching auth method for subsequent runs", os.Stdout)
			validAuthMethod = &am // set the valid auth method so that future calls won't need to retry
		}
		break
	}

	if validAuthMethod == nil {
		return errors.New("unable to find valid auth method for ssh"), ""
	}

	defer client.Close()

	// Create a session
	session, err := client.NewSession()
	if err != nil {
		return err, ""
	}
	defer session.Close()

	var outputBuffer bytes.Buffer

	// Set up pipes for stdin, stdout, and stderr
	session.Stdout = &outputBuffer
	session.Stderr = &outputBuffer
	//stdin, err := session.StdinPipe()
	if err != nil {
		return err, ""
	}

	// Start the remote command
	err = session.Start(command)
	if err != nil {
		return err, outputBuffer.String()
	}
	// Wait for the command to complete
	ShowSpinner()
	defer HideSpinner()
	err = session.Wait()

	if err != nil {
		return err, outputBuffer.String()
	}

	return nil, outputBuffer.String()
}

func getAuthmethods(skipAgent bool, privateKeyfile string, sshAuthSock string, authMethods []ssh.AuthMethod) []ssh.AuthMethod {
	// First, try the specified private key file if provided
	keyFileWorked := false
	if privateKeyfile != "" {
		privateKeyFiles := []string{
			privateKeyfile,
		}

		for _, kf := range privateKeyFiles {
			am, err := getAuthMethodFromPrivateKey(kf)
			if err == nil {
				authMethods = append(authMethods, am)
				keyFileWorked = true
			} else {
				LogDebugInfo(fmt.Sprintf("Unable to use specified key file %s: %v", kf, err), os.Stdout)
			}
		}
	}

	// Try SSH agent as fallback if: agent not skipped AND (no key file specified OR key file failed)
	if skipAgent != true && (!keyFileWorked || privateKeyfile == "") {
		// Connect to SSH agent to ask for unencrypted private keys
		if sshAgentConn, err := net.Dial("unix", sshAuthSock); err == nil {
			sshAgent := agent.NewClient(sshAgentConn)
			keys, _ := sshAgent.List()
			if len(keys) > 0 {
				LogDebugInfo("Using SSH agent for authentication", os.Stdout)
				agentAuthmethods := ssh.PublicKeysCallback(sshAgent.Signers)
				authMethods = append(authMethods, agentAuthmethods)
			}
		}
	} else if skipAgent {
		LogDebugInfo("Skipping ssh agent", os.Stdout)
	}

	// If no private key file specified AND agent doesn't have keys, try all keys in ~/.ssh directory
	// Skip directory scan if agent already has keys to avoid warnings for non-key files
	if privateKeyfile == "" && len(authMethods) == 0 {
		userPath, err := os.UserHomeDir()
		if err != nil {
			LogWarning("No ssh key given and no home directory available", nil)
		} else {
			userPath = filepath.Join(userPath, ".ssh")

			if _, err := os.Stat(userPath); err == nil {
				sshAm, err := getSSHAuthMethodsFromDirectory(userPath)
				if err != nil {
					LogWarning(fmt.Sprintf("Error reading SSH keys from %s: %v", userPath, err), nil)
				}
				authMethods = append(authMethods, sshAm...)
			} else {
				LogDebugInfo(fmt.Sprintf("Unable to find .ssh directory at %s", userPath), os.Stdout)
			}
		}
	}

	return authMethods
}
