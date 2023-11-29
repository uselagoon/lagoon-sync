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
	err := cmd.Run()
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

func findSSHKeyFiles(directory string) ([]string, error) {
	var keys []string
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Ext(path) == ".pub" {
			privateKeyPath := path[:len(path)-4] // remove ".pub" extension
			keys = append(keys, privateKeyPath)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return keys, nil
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
		authMethods = getAuthmethods(skipAgent, privateKeyfile, sshAuthSock, authMethods)
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
	if validAuthMethod != nil {
		LogDebugInfo("Have a valid auth method", os.Stdout)
		// Connect to the remote server
		config.Auth = []ssh.AuthMethod{
			*validAuthMethod,
		}
		client, err = ssh.Dial("tcp", remoteHost+":"+remotePort, config)
		if err != nil {
			return err, ""
		}
		defer client.Close()
	} else {
		//we need to iterate over the auth methods till we find one that works
		LogDebugInfo("Trying an auth method", os.Stdout)
		for _, am := range authMethods {
			config.Auth = []ssh.AuthMethod{
				am,
			}
			client, err = ssh.Dial("tcp", remoteHost+":"+remotePort, config)
			if err != nil {
				continue
			}
			validAuthMethod = &am // set the valid auth method so that future calls won't need to retry
			break
		}
		if validAuthMethod == nil {
			return errors.New("unable to find valid auth method for ssh"), ""
		}
		defer client.Close()
	}

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
	err = session.Wait()
	if err != nil {
		return err, outputBuffer.String()
	}

	return nil, outputBuffer.String()
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
			files, err := findSSHKeyFiles(userPath)
			if err != nil {
				LogWarning(err.Error(), os.Stdout)
			}
			for _, f := range files {
				am, err := getAuthMethodFromPrivateKey(f)
				if err != nil {
					switch {
					case isPassphraseMissingError(err):
						LogDebugInfo(fmt.Sprintf("Found a passphrase based ssh key: %v", err.Error()), os.Stdout)
					default:
						LogWarning(err.Error(), os.Stdout)
					}
				} else {
					authMethods = append(authMethods, am)
				}
			}
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
