package synchers

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"text/template"

	"github.com/spf13/viper"
	"github.com/uselagoon/lagoon-sync/utils"
	"gopkg.in/yaml.v2"
)

var shellToUse = "sh"
var debug = viper.Get("show-debug")

// UnmarshallLagoonYamlToLagoonSyncStructure will take a bytestream and return a fully parsed lagoon sync config structure
func UnmarshallLagoonYamlToLagoonSyncStructure(data []byte) (SyncherConfigRoot, error) {
	lagoonConfig := SyncherConfigRoot{}
	err := yaml.Unmarshal(data, &lagoonConfig)

	if err != nil && debug == false {
		return SyncherConfigRoot{}, errors.New("Unable to parse lagoon config yaml setup")
	}
	return lagoonConfig, nil
}

type RunSyncProcessFunctionType = func(sourceEnvironment Environment, targetEnvironment Environment, lagoonSyncer Syncer, syncerType string, dryRun bool, sshOptions SSHOptions) error

func RunSyncProcess(sourceEnvironment Environment, targetEnvironment Environment, lagoonSyncer Syncer, syncerType string, dryRun bool, sshOptions SSHOptions) error {
	var err error

	if _, err := lagoonSyncer.IsInitialized(); err != nil {
		return err
	}

	sourceEnvironment, err = RunPrerequisiteCommand(sourceEnvironment, lagoonSyncer, syncerType, dryRun, sshOptions)
	sourceRsyncPath := sourceEnvironment.RsyncPath
	if err != nil {
		_ = PrerequisiteCleanUp(sourceEnvironment, sourceRsyncPath, dryRun, sshOptions)
		return err
	}

	err = SyncRunSourceCommand(sourceEnvironment, lagoonSyncer, dryRun, sshOptions)
	if err != nil {
		_ = SyncCleanUp(sourceEnvironment, lagoonSyncer, dryRun, sshOptions)
		return err
	}

	targetEnvironment, err = RunPrerequisiteCommand(targetEnvironment, lagoonSyncer, syncerType, dryRun, sshOptions)
	targetRsyncPath := targetEnvironment.RsyncPath
	if err != nil {
		_ = PrerequisiteCleanUp(targetEnvironment, targetRsyncPath, dryRun, sshOptions)
		return err
	}

	err = SyncRunTransfer(sourceEnvironment, targetEnvironment, lagoonSyncer, dryRun, sshOptions)
	if err != nil {
		_ = PrerequisiteCleanUp(sourceEnvironment, sourceRsyncPath, dryRun, sshOptions)
		_ = SyncCleanUp(sourceEnvironment, lagoonSyncer, dryRun, sshOptions)
		return err
	}

	err = SyncRunTargetCommand(targetEnvironment, lagoonSyncer, dryRun, sshOptions)
	if err != nil {
		_ = PrerequisiteCleanUp(sourceEnvironment, sourceRsyncPath, dryRun, sshOptions)
		_ = PrerequisiteCleanUp(targetEnvironment, targetRsyncPath, dryRun, sshOptions)
		_ = SyncCleanUp(sourceEnvironment, lagoonSyncer, dryRun, sshOptions)
		_ = SyncCleanUp(targetEnvironment, lagoonSyncer, dryRun, sshOptions)
		return err
	}

	_ = PrerequisiteCleanUp(sourceEnvironment, sourceRsyncPath, dryRun, sshOptions)
	_ = PrerequisiteCleanUp(targetEnvironment, targetRsyncPath, dryRun, sshOptions)
	_ = SyncCleanUp(sourceEnvironment, lagoonSyncer, dryRun, sshOptions)
	_ = SyncCleanUp(targetEnvironment, lagoonSyncer, dryRun, sshOptions)

	return nil
}

func SyncRunSourceCommand(remoteEnvironment Environment, syncer Syncer, dryRun bool, sshOptions SSHOptions) error {

	utils.LogProcessStep("Beginning export on source environment", remoteEnvironment.EnvironmentName)

	if syncer.GetRemoteCommand(remoteEnvironment).NoOp {
		log.Printf("Found No Op for environment %s - skipping step", remoteEnvironment.EnvironmentName)
		return nil
	}

	command, commandErr := syncer.GetRemoteCommand(remoteEnvironment).GetCommand()
	if commandErr != nil {
		return commandErr
	}

	var execString string

	if remoteEnvironment.EnvironmentName == LOCAL_ENVIRONMENT_NAME {
		execString = command
	} else {
		execString = GenerateRemoteCommand(remoteEnvironment, command, sshOptions)
	}

	utils.LogExecutionStep("Running the following for source", execString)

	if !dryRun {
		err, response, errstring := utils.Shellout(execString)
		if err != nil && debug == false {
			fmt.Println(errstring)
			return err
		}
		if response != "" && debug == false {
			fmt.Println(response)
		}
	}
	return nil
}

func SyncRunTransfer(sourceEnvironment Environment, targetEnvironment Environment, syncer Syncer, dryRun bool, sshOptions SSHOptions) error {
	utils.LogProcessStep("Beginning file transfer logic", nil)

	// If we're transferring to the same resource, we can skip this whole process.
	if sourceEnvironment.EnvironmentName == targetEnvironment.EnvironmentName {
		utils.LogDebugInfo("Source and target environments are the same, skipping transfer", nil)
		return nil
	}

	// For now, we assert that _one_ of the environments _has_ to be local
	executeRsyncRemotelyOnTarget := false
	if sourceEnvironment.EnvironmentName != LOCAL_ENVIRONMENT_NAME && targetEnvironment.EnvironmentName != LOCAL_ENVIRONMENT_NAME {
		//TODO: if we have multiple remotes, we need to treat the target environment as local, and run the rysync from there ...
		utils.LogWarning(fmt.Sprintf("Using %s syncer for remote to remote transfer is expirimental at present", viper.Get("syncer-type")), nil)
		utils.LogDebugInfo("Since we're syncing across two remote systems, we're pulling the files to the target", targetEnvironment.EnvironmentName)
		executeRsyncRemotelyOnTarget = true
	}

	if sourceEnvironment.EnvironmentName == LOCAL_ENVIRONMENT_NAME && targetEnvironment.EnvironmentName == LOCAL_ENVIRONMENT_NAME {
		utils.LogFatalError("In order to rsync, at least _one_ of the environments must be remote", nil)
	}

	sourceEnvironmentName := syncer.GetTransferResource(sourceEnvironment).Name
	if syncer.GetTransferResource(sourceEnvironment).IsDirectory == true {
		sourceEnvironmentName += "/"
	}

	// lagoonRsyncService keeps track of precisely where we're going to be rsyncing from.
	lagoonRsyncService := "cli"
	// rsyncRemoteSystemUsername is used by the rsync command to set up the ssh tunnel
	rsyncRemoteSystemUsername := ""

	if sourceEnvironment.EnvironmentName != LOCAL_ENVIRONMENT_NAME {
		//sourceEnvironmentName = fmt.Sprintf("%s@ssh.lagoon.amazeeio.cloud:%s", sourceEnvironment.GetOpenshiftProjectName(), sourceEnvironmentName)
		sourceEnvironmentName = fmt.Sprintf(":%s", sourceEnvironmentName)
		rsyncRemoteSystemUsername = sourceEnvironment.GetOpenshiftProjectName()
		if sourceEnvironment.ServiceName != "" {
			lagoonRsyncService = sourceEnvironment.ServiceName
		}
	}

	targetEnvironmentName := syncer.GetTransferResource(targetEnvironment).Name
	if targetEnvironment.EnvironmentName != LOCAL_ENVIRONMENT_NAME && executeRsyncRemotelyOnTarget == false {
		//targetEnvironmentName = fmt.Sprintf("%s@ssh.lagoon.amazeeio.cloud:%s", targetEnvironment.GetOpenshiftProjectName(), targetEnvironmentName)
		targetEnvironmentName = fmt.Sprintf(":%s", targetEnvironmentName)
		rsyncRemoteSystemUsername = targetEnvironment.GetOpenshiftProjectName()
		if targetEnvironment.ServiceName != "" {
			lagoonRsyncService = targetEnvironment.ServiceName
		}
	}

	syncExcludes := " "
	for _, e := range syncer.GetTransferResource(sourceEnvironment).ExcludeResources {
		syncExcludes += fmt.Sprintf("--exclude=%v ", e)
	}

	var sshOptionsStr bytes.Buffer
	verboseFlag := ""
	if sshOptions.Verbose {
		verboseFlag = "-v"
		sshOptionsStr.WriteString(" -v")
	}

	if sshOptions.PrivateKey != "" {
		sshOptionsStr.WriteString(fmt.Sprintf(" -i %s", sshOptions.PrivateKey))
	}

	rsyncArgs := sshOptions.RsyncArgs
	execString := fmt.Sprintf("%s %s --rsync-path=%s %s -e \"ssh%s -o LogLevel=FATAL -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -p %s -l %s %s service=%s\" %s %s %s",
		targetEnvironment.RsyncPath,
		rsyncArgs,
		sourceEnvironment.RsyncPath,
		verboseFlag,
		sshOptionsStr.String(),
		sshOptions.Port,
		rsyncRemoteSystemUsername,
		sshOptions.Host,
		lagoonRsyncService,
		syncExcludes,
		sourceEnvironmentName,
		targetEnvironmentName)

	if executeRsyncRemotelyOnTarget {
		execString = GenerateRemoteCommand(targetEnvironment, execString, sshOptions)
	}

	utils.LogExecutionStep(fmt.Sprintf("Running the following for target (%s)", targetEnvironment.EnvironmentName), execString)

	if !dryRun {
		if err, _, errstring := utils.Shellout(execString); err != nil {
			utils.LogFatalError(errstring, nil)
		}
	}

	return nil
}

func SyncRunTargetCommand(targetEnvironment Environment, syncer Syncer, dryRun bool, sshOptions SSHOptions) error {

	utils.LogProcessStep("Beginning import on target environment", targetEnvironment.EnvironmentName)

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
		execString = GenerateRemoteCommand(targetEnvironment, command, sshOptions)
	}

	utils.LogExecutionStep(fmt.Sprintf("Running the following for target (%s)", targetEnvironment.EnvironmentName), execString)
	if !dryRun {
		err, _, errstring := utils.Shellout(execString)
		if err != nil {
			utils.LogFatalError(errstring, nil)
		}
	}

	return nil
}

func SyncCleanUp(environment Environment, syncer Syncer, dryRun bool, sshOptions SSHOptions) error {
	transferResouce := syncer.GetTransferResource(environment)

	if transferResouce.SkipCleanup == true {
		log.Printf("Skipping cleanup for %v on %v environment", transferResouce.Name, environment.EnvironmentName)
		return nil
	}

	transferResourceName := transferResouce.Name
	execString := fmt.Sprintf("rm -r %s || true", transferResourceName)

	if environment.EnvironmentName != LOCAL_ENVIRONMENT_NAME {
		execString = GenerateRemoteCommand(environment, execString, sshOptions)
	}

	utils.LogProcessStep("Beginning resource cleanup on", environment.EnvironmentName)
	utils.LogExecutionStep("Running the following", execString)

	if !dryRun {
		err, _, errstring := utils.Shellout(execString)
		if err != nil {
			utils.LogFatalError(errstring, nil)
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
		command:       commandString,
		substitutions: substitutions,
		NoOp:          false,
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

func getEnv(key string, defaultVal string) string {
	if _, exists := os.LookupEnv(key); exists {
		return key
	}
	return defaultVal
}
