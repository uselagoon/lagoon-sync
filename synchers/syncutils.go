package synchers

import (
	"bytes"
	"errors"
	"fmt"
	"gopkg.in/yaml.v2"
	"log"
	"os/exec"
)

// UnmarshallLagoonYamlToLagoonSyncStructure will take a bytestream and return a fully parsed lagoon sync config structure
func UnmarshallLagoonYamlToLagoonSyncStructure(data []byte) (SyncherConfigRoot, error) {
	lagoonConfig := SyncherConfigRoot{
		LagoonSync: LagoonSync{},
	}
	err := yaml.Unmarshal(data, &lagoonConfig)
	if err != nil {
		return SyncherConfigRoot{}, errors.New("Unable to parse lagoon config yaml setup")
	}
	return lagoonConfig, nil
}

func RunSyncProcess(sourceEnvironment Environment, targetEnvironment Environment, lagoonSyncer Syncer) error {
	var err error
	err = SyncRunSourceCommand(sourceEnvironment, lagoonSyncer)

	if err != nil {
		_ = SyncCleanUp(sourceEnvironment, lagoonSyncer)
		return err
	}
	err = SyncRunTransfer(sourceEnvironment, targetEnvironment, lagoonSyncer)
	if err != nil {
		_ = SyncCleanUp(sourceEnvironment, lagoonSyncer)
		return err
	}

	err = SyncRunTargetCommand(targetEnvironment, lagoonSyncer)
	if err != nil {
		_ = SyncCleanUp(sourceEnvironment, lagoonSyncer)
		_ = SyncCleanUp(targetEnvironment, lagoonSyncer)
		return err
	}

	_ = SyncCleanUp(sourceEnvironment, lagoonSyncer)
	_ = SyncCleanUp(targetEnvironment, lagoonSyncer)

	return nil
}

func SyncRunSourceCommand(remoteEnvironment Environment, syncer Syncer) error {

	log.Println("Beginning export on source environment (%s)", remoteEnvironment.EnvironmentName)

	var execString string


	if remoteEnvironment.EnvironmentName == LOCAL_ENVIRONMENT_NAME {
		execString = syncer.GetRemoteCommand(remoteEnvironment).command
	} else {
		execString = generateRemoteCommand(remoteEnvironment, syncer.GetRemoteCommand(remoteEnvironment).command)
	}

	log.Printf("Running the following for source :- %s", execString)
	err, _, errstring := Shellout(execString)

	if err != nil {
		fmt.Println(errstring)
		return err
	}
	return nil
}

func SyncRunTransfer(sourceEnvironment Environment, targetEnvironment Environment, syncer Syncer) error {

	if sourceEnvironment.EnvironmentName == targetEnvironment.EnvironmentName {
		log.Println("Source and target environments are the same, skipping transfer")
		return nil
	}

	log.Println("Beginning file transfer logic")

	sourceEnvironmentName := syncer.GetTransferResource().Name
	if syncer.GetTransferResource().IsDirectory == true {
		sourceEnvironmentName += "/"
	}
	if sourceEnvironment.EnvironmentName != LOCAL_ENVIRONMENT_NAME {
		sourceEnvironmentName = fmt.Sprintf("%s@ssh.lagoon.amazeeio.cloud:%s", sourceEnvironment.getOpenshiftProjectName(), sourceEnvironmentName)
	}

	targetEnvironmentName := syncer.GetTransferResource().Name
	if targetEnvironment.EnvironmentName != LOCAL_ENVIRONMENT_NAME {
		targetEnvironmentName = fmt.Sprintf("%s@ssh.lagoon.amazeeio.cloud:%s", targetEnvironment.getOpenshiftProjectName(), targetEnvironmentName)
	}

	execString := fmt.Sprintf("rsync -e \"ssh -o LogLevel=ERROR -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -p 32222\" -a %s %s",
		sourceEnvironmentName,
		targetEnvironmentName)

	log.Printf("Running the following for target :- %s", execString)
	err, _, errstring := Shellout(execString)

	if err != nil {
		log.Println(errstring)
		return err
	}

	return nil
}

func SyncRunTargetCommand(targetEnvironment Environment, syncer Syncer) error {

	log.Println("Beginning import on target environment (%s)", targetEnvironment.EnvironmentName)

	var execString string

	if targetEnvironment.EnvironmentName == LOCAL_ENVIRONMENT_NAME {
		execString = syncer.GetLocalCommand(targetEnvironment).command
	} else {
		execString = generateRemoteCommand(targetEnvironment, syncer.GetLocalCommand(targetEnvironment).command)
	}

	log.Printf("Running the following for target :- %s", execString)

	err, _, errstring := Shellout(execString)

	if err != nil {
		fmt.Println(errstring)
		return err
	}

	return nil
}

func SyncCleanUp(environment Environment, syncer Syncer) error {
	transferResourceName := syncer.GetTransferResource().Name

	execString := fmt.Sprintf("rm -r %s", transferResourceName)

	if environment.EnvironmentName != LOCAL_ENVIRONMENT_NAME {
		execString = generateRemoteCommand(environment, execString)
	}

	log.Printf("Beginning resource cleanup on %s", environment.EnvironmentName)
	log.Printf("Running the following: %s", execString)

	err, _, errstring := Shellout(execString)

	if err != nil {
		fmt.Println(errstring)
		return err
	}



	return nil
}

func generateRemoteCommand(remoteEnvironment Environment, command string) string {
	return fmt.Sprintf("ssh -t -o \"UserKnownHostsFile=/dev/null\" -o \"StrictHostKeyChecking=no\" -p 32222 %v@ssh.lagoon.amazeeio.cloud '%v'",
		remoteEnvironment.getOpenshiftProjectName(), command)
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
