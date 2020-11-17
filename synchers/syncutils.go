package synchers

import (
	"bytes"
	"errors"
	"fmt"
	"gopkg.in/yaml.v2"
	"log"
	"os/exec"
	"text/template"
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

func RunSyncProcess(sourceEnvironment Environment, targetEnvironment Environment, lagoonSyncer Syncer, dryRun bool) error {
	var err error
	err = SyncRunSourceCommand(sourceEnvironment, lagoonSyncer, dryRun)

	if err != nil {
		_ = SyncCleanUp(sourceEnvironment, lagoonSyncer, dryRun)
		return err
	}
	err = SyncRunTransfer(sourceEnvironment, targetEnvironment, lagoonSyncer, dryRun)
	if err != nil {
		_ = SyncCleanUp(sourceEnvironment, lagoonSyncer, dryRun)
		return err
	}

	err = SyncRunTargetCommand(targetEnvironment, lagoonSyncer, dryRun)
	if err != nil {
		_ = SyncCleanUp(sourceEnvironment, lagoonSyncer, dryRun)
		_ = SyncCleanUp(targetEnvironment, lagoonSyncer, dryRun)
		return err
	}

	_ = SyncCleanUp(sourceEnvironment, lagoonSyncer, dryRun)
	_ = SyncCleanUp(targetEnvironment, lagoonSyncer, dryRun)

	return nil
}

func SyncRunSourceCommand(remoteEnvironment Environment, syncer Syncer, dryRun bool) error {

	log.Printf("Beginning export on source environment (%s)", remoteEnvironment.EnvironmentName)

	if syncer.GetRemoteCommand(remoteEnvironment).NoOp {
		log.Printf("Found No Op for environment %s - skipping step", remoteEnvironment.EnvironmentName)
		return nil
	}

	var execString string

	command, commandErr := syncer.GetRemoteCommand(remoteEnvironment).GetCommand()
	if commandErr != nil {
		return commandErr
	}

	if remoteEnvironment.EnvironmentName == LOCAL_ENVIRONMENT_NAME {
		execString = command
	} else {
		execString = generateRemoteCommand(remoteEnvironment, command)
	}

	log.Printf("Running the following for source :- %s", execString)

	if !dryRun {
		err, _, errstring := Shellout(execString)
		if err != nil {
			fmt.Println(errstring)
			return err
		}
	}
	return nil
}

func SyncRunTransfer(sourceEnvironment Environment, targetEnvironment Environment, syncer Syncer, dryRun bool) error {

	if sourceEnvironment.EnvironmentName == targetEnvironment.EnvironmentName {
		log.Println("Source and target environments are the same, skipping transfer")
		return nil
	}

	log.Println("Beginning file transfer logic")

	sourceEnvironmentName := syncer.GetTransferResource(sourceEnvironment).Name
	if syncer.GetTransferResource(sourceEnvironment).IsDirectory == true {
		sourceEnvironmentName += "/"
	}
	if sourceEnvironment.EnvironmentName != LOCAL_ENVIRONMENT_NAME {
		sourceEnvironmentName = fmt.Sprintf("%s@ssh.lagoon.amazeeio.cloud:%s", sourceEnvironment.getOpenshiftProjectName(), sourceEnvironmentName)
	}

	targetEnvironmentName := syncer.GetTransferResource(targetEnvironment).Name
	if targetEnvironment.EnvironmentName != LOCAL_ENVIRONMENT_NAME {
		targetEnvironmentName = fmt.Sprintf("%s@ssh.lagoon.amazeeio.cloud:%s", targetEnvironment.getOpenshiftProjectName(), targetEnvironmentName)
	}

	syncExcludes := " "
	for _, e := range syncer.GetTransferResource(sourceEnvironment).ExcludeResources {
		syncExcludes += fmt.Sprintf("--exclude=%v ", e)
	}

	execString := fmt.Sprintf("rsync -e \"ssh -o LogLevel=ERROR -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -p 32222\" %v -a %s %s",
		syncExcludes,
		sourceEnvironmentName,
		targetEnvironmentName)

	log.Printf("Running the following for target :- %s", execString)

	if !dryRun {
		err, _, errstring := Shellout(execString)

		if err != nil {
			log.Println(errstring)
			return err
		}
	}

	return nil
}

func SyncRunTargetCommand(targetEnvironment Environment, syncer Syncer, dryRun bool) error {

	log.Printf("Beginning import on target environment (%s)", targetEnvironment.EnvironmentName)

	if syncer.GetRemoteCommand(targetEnvironment).NoOp {
		log.Printf("Found No Op for environment %s - skipping step", targetEnvironment.EnvironmentName)
		return nil
	}

	var execString string
	command, commandErr := syncer.GetLocalCommand(targetEnvironment).GetCommand()
	if commandErr != nil {
		return commandErr
	}

	if targetEnvironment.EnvironmentName == LOCAL_ENVIRONMENT_NAME {
		execString = command
	} else {
		execString = generateRemoteCommand(targetEnvironment, command)
	}

	log.Printf("Running the following for target :- %s", execString)
	if !dryRun {
		err, _, errstring := Shellout(execString)

		if err != nil {
			fmt.Println(errstring)
			return err
		}
	}

	return nil
}

func SyncCleanUp(environment Environment, syncer Syncer, dryRun bool) error {
	transferResouce := syncer.GetTransferResource(environment)

	if transferResouce.SkipCleanup == true {
		log.Printf("Skipping cleanup for %v on %v environment", transferResouce.Name, environment.EnvironmentName)
		return nil
	}

	transferResourceName := transferResouce.Name


	execString := fmt.Sprintf("rm -r %s", transferResourceName)

	if environment.EnvironmentName != LOCAL_ENVIRONMENT_NAME {
		execString = generateRemoteCommand(environment, execString)
	}

	log.Printf("Beginning resource cleanup on %s", environment.EnvironmentName)
	log.Printf("Running the following: %s", execString)

	if !dryRun {
		err, _, errstring := Shellout(execString)

		if err != nil {
			fmt.Println(errstring)
			return err
		}
	}



	return nil
}


func generateNoOpSyncCommand() SyncCommand {
	return SyncCommand{
		NoOp: true,
	}
}

func generateSyncCommand(commandString string, substitutions map[string]interface{}) SyncCommand {
	return SyncCommand{
		command: commandString,
		substitutions: substitutions,
		NoOp: false,
	}
}

func (c SyncCommand) GetCommand() (string, error) {
	if c.NoOp == true {
		return "", errors.New("The command is marked as NoOp(eration) and does not generate a string")
	}
	templ, err := template.New("Command Template").Parse(c.command)
	if err != nil {
		return "", err
	}
	var output bytes.Buffer
	if err := templ.Execute(&output, c.substitutions); err != nil {
		return "", err
	}
	return output.String(), nil
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
