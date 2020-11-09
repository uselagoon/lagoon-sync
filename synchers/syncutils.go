package synchers

import (
	"bytes"
	"errors"
	"fmt"
	"gopkg.in/yaml.v2"
	"os/exec"
)

// UnmarshallLagoonYamlToLagoonSyncStructure will take a bytestream and return a fully parsed lagoon sync config structure
func UnmarshallLagoonYamlToLagoonSyncStructure(data []byte) (SyncherConfigRoot, error) {
	lagoonConfig := SyncherConfigRoot{}
	err := yaml.Unmarshal(data, &lagoonConfig)
	if(err != nil) {
		return SyncherConfigRoot{}, errors.New("Unable to parse lagoon config yaml setup")
	}
	return lagoonConfig, nil
}

const ShellToUse = "bash"

func Shellout(command string) (error, string, string) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command(ShellToUse, "-c", command)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return err, stdout.String(), stderr.String()
}


func SyncRunRemote(syncer Syncer) error {
	println(syncer.GetRemoteCommand())

	execString := fmt.Sprintf("ssh -t -o \"UserKnownHostsFile=/dev/null\" -o \"StrictHostKeyChecking=no\" -p 32222 %v@ssh.lagoon.amazeeio.cloud '%v'",
	"amazeelabsv4-com-dev", "ls")

	err, outstring, errstring := Shellout(execString)

	if err != nil {
		fmt.Println(errstring)
		return err
	}
	fmt.Println(outstring)
	return nil
}

func SyncRunTransfer(syncer Syncer) error {
	fmt.Print("I'm going to be rsyncing the following resource: ")
	return nil
}

func SyncRunLocal(syncer Syncer) error {
	fmt.Print("I'm going to be running the following: ")
	fmt.Println(syncer.GetLocalCommand())
	return nil
}