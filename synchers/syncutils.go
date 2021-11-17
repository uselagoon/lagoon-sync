package synchers

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"text/template"

	"github.com/amazeeio/lagoon-sync/utils"
	"github.com/spf13/viper"
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

func RunSyncProcess(sourceEnvironment Environment, targetEnvironment Environment, lagoonSyncer Syncer, syncerType string, commandOptions SyncCommandOptions, dryRun bool, verboseSSH bool) error {
	var err error

	if _, err := lagoonSyncer.IsInitialized(); err != nil {
		return err
	}

	sourceEnvironment, err = RunPrerequisiteCommand(sourceEnvironment, lagoonSyncer, syncerType, dryRun, verboseSSH)
	sourceRsyncPath := sourceEnvironment.RsyncPath
	if err != nil {
		_ = PrerequisiteCleanUp(sourceEnvironment, sourceRsyncPath, dryRun, verboseSSH)
		return err
	}

	// Preflight source checks to determine command input
	// e.g we need to determine tables that are available in source env so that we can compare against wildcard args
	lagoonSyncer, err = RunPreflightCommand(sourceEnvironment, lagoonSyncer, commandOptions, syncerType, dryRun, verboseSSH)
	if err != nil {
		return err
	}

	err = SyncRunSourceCommand(sourceEnvironment, lagoonSyncer, commandOptions, dryRun, verboseSSH)
	if err != nil {
		_ = SyncCleanUp(sourceEnvironment, lagoonSyncer, dryRun, verboseSSH)
		return err
	}

	targetEnvironment, err = RunPrerequisiteCommand(targetEnvironment, lagoonSyncer, syncerType, dryRun, verboseSSH)
	targetRsyncPath := targetEnvironment.RsyncPath
	if err != nil {
		_ = PrerequisiteCleanUp(targetEnvironment, targetRsyncPath, dryRun, verboseSSH)
		return err
	}

	err = SyncRunTransfer(sourceEnvironment, targetEnvironment, lagoonSyncer, dryRun, verboseSSH)
	if err != nil {
		_ = PrerequisiteCleanUp(sourceEnvironment, sourceRsyncPath, dryRun, verboseSSH)
		_ = SyncCleanUp(sourceEnvironment, lagoonSyncer, dryRun, verboseSSH)
		return err
	}

	err = SyncRunTargetCommand(targetEnvironment, lagoonSyncer, commandOptions, dryRun, verboseSSH)
	if err != nil {
		_ = PrerequisiteCleanUp(sourceEnvironment, sourceRsyncPath, dryRun, verboseSSH)
		_ = PrerequisiteCleanUp(targetEnvironment, targetRsyncPath, dryRun, verboseSSH)
		_ = SyncCleanUp(sourceEnvironment, lagoonSyncer, dryRun, verboseSSH)
		_ = SyncCleanUp(targetEnvironment, lagoonSyncer, dryRun, verboseSSH)
		return err
	}

	_ = PrerequisiteCleanUp(sourceEnvironment, sourceRsyncPath, dryRun, verboseSSH)
	_ = PrerequisiteCleanUp(targetEnvironment, targetRsyncPath, dryRun, verboseSSH)
	_ = SyncCleanUp(sourceEnvironment, lagoonSyncer, dryRun, verboseSSH)
	_ = SyncCleanUp(targetEnvironment, lagoonSyncer, dryRun, verboseSSH)

	return nil
}

// Takes config from .yml and parsed command-line arguments to determine what needs to be done on the source environment.
func RunPreflightCommand(sourceEnvironment Environment, syncher Syncer, commandOptions SyncCommandOptions, syncerType string, dryRun bool, verboseSSH bool) (Syncer, error) {
	if syncerType == "files" || syncerType == "drupalconfig" {
		return syncher, nil
	}

	if verboseSSH {
		utils.LogProcessStep("Running preflight checks on", sourceEnvironment.EnvironmentName)
	}

	var execString string
	command, commandErr := syncher.GetPreflightCommand(sourceEnvironment, verboseSSH).GetCommand()
	if commandErr != nil {
		return syncher, commandErr
	}

	if sourceEnvironment.EnvironmentName == LOCAL_ENVIRONMENT_NAME {
		execString = command
	} else {
		execString = GenerateRemoteCommand(sourceEnvironment, command, verboseSSH)
	}

	utils.LogExecutionStep("Running the following preflight command", execString)

	err, preflightResponse, errstring := utils.Shellout(execString)
	if err != nil {
		fmt.Println(errstring)
	}

	// Apply preflight response changes to input arguments
	lagoonSyncher, err := syncher.ApplyPreflightResponseChecks(preflightResponse, commandOptions)
	fmt.Println("lagoonSyncher: ", lagoonSyncher)

	return lagoonSyncher, nil
}

func SyncRunSourceCommand(remoteEnvironment Environment, syncer Syncer, commandOptions SyncCommandOptions, dryRun bool, verboseSSH bool) error {
	utils.LogProcessStep("Beginning export on source environment", remoteEnvironment.EnvironmentName)

	if syncer.GetRemoteCommand(remoteEnvironment, commandOptions).NoOp {
		log.Printf("Found No Op for environment %s - skipping step", remoteEnvironment.EnvironmentName)
		return nil
	}

	command, commandErr := syncer.GetRemoteCommand(remoteEnvironment, commandOptions).GetCommand()
	if commandErr != nil {
		return commandErr
	}

	var execString string

	if remoteEnvironment.EnvironmentName == LOCAL_ENVIRONMENT_NAME {
		execString = command
	} else {
		execString = GenerateRemoteCommand(remoteEnvironment, command, verboseSSH)
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

func SyncRunTransfer(sourceEnvironment Environment, targetEnvironment Environment, syncer Syncer, dryRun bool, verboseSSH bool) error {
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

	verboseSSHArgument := ""
	if verboseSSH {
		verboseSSHArgument = "-v"
	}

	execString := fmt.Sprintf("%v --omit-dir-times --rsync-path=%v %v -e \"ssh %v -o LogLevel=FATAL -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -p 32222 -l %v ssh.lagoon.amazeeio.cloud service=%v\" %v -a %s %s",
		targetEnvironment.RsyncPath,
		sourceEnvironment.RsyncPath,
		verboseSSHArgument,
		verboseSSHArgument,
		rsyncRemoteSystemUsername,
		lagoonRsyncService,
		syncExcludes,
		sourceEnvironmentName,
		targetEnvironmentName)

	if executeRsyncRemotelyOnTarget {
		execString = GenerateRemoteCommand(targetEnvironment, execString, verboseSSH)
	}

	utils.LogExecutionStep(fmt.Sprintf("Running the following for target (%s)", targetEnvironment.EnvironmentName), execString)

	if !dryRun {
		if err, _, errstring := utils.Shellout(execString); err != nil {
			utils.LogFatalError(errstring, nil)
		}
	}

	return nil
}

func SyncRunTargetCommand(targetEnvironment Environment, syncer Syncer, commandOptions SyncCommandOptions, dryRun bool, verboseSSH bool) error {

	utils.LogProcessStep("Beginning import on target environment", targetEnvironment.EnvironmentName)

	if syncer.GetRemoteCommand(targetEnvironment, commandOptions).NoOp {
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
		execString = GenerateRemoteCommand(targetEnvironment, command, verboseSSH)
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

func SyncCleanUp(environment Environment, syncer Syncer, dryRun bool, verboseSSH bool) error {
	transferResouce := syncer.GetTransferResource(environment)

	if transferResouce.SkipCleanup == true {
		log.Printf("Skipping cleanup for %v on %v environment", transferResouce.Name, environment.EnvironmentName)
		return nil
	}

	transferResourceName := transferResouce.Name
	execString := fmt.Sprintf("rm -r %s", transferResourceName)

	if environment.EnvironmentName != LOCAL_ENVIRONMENT_NAME {
		execString = GenerateRemoteCommand(environment, execString, verboseSSH)
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
