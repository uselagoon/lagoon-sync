package utils

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
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
				case strings.Contains(err.Error(), "no key found"):
					LogDebugInfo(fmt.Sprintf("Not a private key file: %s", path), os.Stdout)
				case isPassphraseMissingError(err):
					LogDebugInfo(fmt.Sprintf("Found a passphrase based ssh key: %v", err.Error()), os.Stdout)
				default:
					LogWarning(err.Error(), os.Stdout)
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

func RemoteShellout(command string, service string, remoteUser string, remoteHost string, remotePort string, privateKeyfile string, skipSshAgent bool) (error, string) {

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

	ShowSpinner()
	defer HideSpinner()

	output, err := session.CombinedOutput(fmt.Sprintf("service=%s %s", service, command))
	if err != nil {
		if exitErr, ok := err.(*ssh.ExitError); ok {
			return fmt.Errorf("remote command failed with exit code %d", exitErr.ExitStatus()), string(output)
		} else {
			return fmt.Errorf("ssh error: %v", err), string(output)
		}
	}

	return nil, string(output)
}

func getAuthmethods(skipAgent bool, privateKeyfile string, sshAuthSock string, authMethods []ssh.AuthMethod) []ssh.AuthMethod {
	if skipAgent != true && privateKeyfile == "" {
		// Connect to SSH agent to ask for unencrypted private keys
		if sshAgentConn, err := net.Dial("unix", sshAuthSock); err == nil {
			sshAgent := agent.NewClient(sshAgentConn)
			keys, _ := sshAgent.List()
			if len(keys) > 0 {
				agentAuthmethods := ssh.PublicKeysCallback(sshAgent.Signers)
				authMethods = append(authMethods, agentAuthmethods)
			}
		}
	} else {
		LogDebugInfo("Skipping ssh agent", os.Stdout)
	}

	if privateKeyfile == "" { //let's try guess it from the OS
		userPath, err := os.UserHomeDir()
		if err != nil {
			LogWarning("No ssh key given and no home directory available", os.Stdout)
		}

		userPath = filepath.Join(userPath, ".ssh")

		if _, err := os.Stat(userPath); err == nil {
			sshAm, err := getSSHAuthMethodsFromDirectory(userPath)
			if err != nil {
				LogWarning(err.Error(), os.Stdout)
			}
			authMethods = append(authMethods, sshAm...)
		} else {
			LogWarning("Unable to find .ssh directory in user home", os.Stdout)
		}
	} else {
		privateKeyFiles := []string{
			privateKeyfile,
		}

		for _, kf := range privateKeyFiles {
			am, err := getAuthMethodFromPrivateKey(kf)
			if err == nil {
				authMethods = append(authMethods, am)
			}
		}
	}
	return authMethods
}
