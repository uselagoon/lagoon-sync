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

type RunSyncProcessFunctionTypeArguments struct {
	SourceEnvironment Environment
	TargetEnvironment Environment
	LagoonSyncer      Syncer
	SyncerType        string
	DryRun            bool
	SkipSourceCleanup bool
	SkipTargetCleanup bool
	SkipTargetImport  bool
}

type RunSyncProcessFunctionType = func(args RunSyncProcessFunctionTypeArguments) error

func RunSyncProcess(args RunSyncProcessFunctionTypeArguments) error {
	var err error

	if _, err := args.LagoonSyncer.IsInitialized(); err != nil {
		return err
	}

	//TODO: this can come out.
	args.SourceEnvironment, err = RunPrerequisiteCommand(args.SourceEnvironment, args.LagoonSyncer, args.SyncerType, args.DryRun)
	sourceRsyncPath := args.SourceEnvironment.RsyncPath
	if err != nil {
		_ = PrerequisiteCleanUp(args.SourceEnvironment, sourceRsyncPath, args.DryRun)
		return err
	}

	err = SyncRunSourceCommand(args.SourceEnvironment, args.LagoonSyncer, args.DryRun)
	if err != nil {
		_ = SyncCleanUp(args.SourceEnvironment, args.LagoonSyncer, args.DryRun)
		return err
	}

	args.TargetEnvironment, err = RunPrerequisiteCommand(args.TargetEnvironment, args.LagoonSyncer, args.SyncerType, args.DryRun)
	targetRsyncPath := args.TargetEnvironment.RsyncPath
	if err != nil {
		_ = PrerequisiteCleanUp(args.TargetEnvironment, targetRsyncPath, args.DryRun)
		return err
	}

	err = SyncRunTransfer(args.SourceEnvironment, args.TargetEnvironment, args.LagoonSyncer, args.DryRun)
	if err != nil {
		_ = PrerequisiteCleanUp(args.SourceEnvironment, sourceRsyncPath, args.DryRun)
		_ = SyncCleanUp(args.SourceEnvironment, args.LagoonSyncer, args.DryRun)
		return err
	}

	if !args.SkipTargetImport {
		err = SyncRunTargetCommand(args.TargetEnvironment, args.LagoonSyncer, args.DryRun)
		if err != nil {
			_ = PrerequisiteCleanUp(args.SourceEnvironment, sourceRsyncPath, args.DryRun)
			_ = PrerequisiteCleanUp(args.TargetEnvironment, targetRsyncPath, args.DryRun)
			_ = SyncCleanUp(args.SourceEnvironment, args.LagoonSyncer, args.DryRun)
			_ = SyncCleanUp(args.TargetEnvironment, args.LagoonSyncer, args.DryRun)
			return err
		}
	} else {
		utils.LogProcessStep("Skipping target import step", nil)
	}

	_ = PrerequisiteCleanUp(args.SourceEnvironment, sourceRsyncPath, args.DryRun)
	_ = PrerequisiteCleanUp(args.TargetEnvironment, targetRsyncPath, args.DryRun)
	if !args.SkipSourceCleanup {
		_ = SyncCleanUp(args.SourceEnvironment, args.LagoonSyncer, args.DryRun)
	}
	if !args.SkipTargetCleanup {
		_ = SyncCleanUp(args.TargetEnvironment, args.LagoonSyncer, args.DryRun)
	} else {
		utils.LogProcessStep("File on the target saved as: "+args.LagoonSyncer.GetTransferResource(args.TargetEnvironment).Name, nil)
	}

	return nil
}

func SyncRunSourceCommand(remoteEnvironment Environment, syncer Syncer, dryRun bool) error {
	utils.LogProcessStep("Beginning export on source environment", remoteEnvironment.EnvironmentName)

	remoteCommands := syncer.GetRemoteCommand(remoteEnvironment)
	for _, remoteCommand := range remoteCommands {
		if remoteCommand.NoOp {
			log.Printf("Found No Op for environment %s - skipping step", remoteEnvironment.EnvironmentName)
			return nil
		}

		command, commandErr := remoteCommand.GetCommand()
		if commandErr != nil {
			return commandErr
		}

		var execString string

		if remoteEnvironment.EnvironmentName == LOCAL_ENVIRONMENT_NAME {
			execString = command
		} else {
			execString = GenerateRemoteCommand(remoteEnvironment, command)
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
	}

	return nil
}

func SyncRunTransfer(sourceEnvironment Environment, targetEnvironment Environment, syncer Syncer, dryRun bool) error {
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
	if sourceEnvironment.SSH.Verbose || targetEnvironment.SSH.Verbose {
		verboseFlag = "-v"
		sshOptionsStr.WriteString(" -v")
	}

	if sourceEnvironment.SSH.PrivateKey != "" {
		sshOptionsStr.WriteString(fmt.Sprintf(" -i %s", sourceEnvironment.SSH.PrivateKey))
	}

	rsyncArgs := sourceEnvironment.SSH.RsyncArgs
	execString := fmt.Sprintf("%s %s --rsync-path=%s %s -e \"ssh%s -o LogLevel=FATAL -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -p %s -l %s %s service=%s\" %s %s %s",
		targetEnvironment.RsyncPath,
		rsyncArgs,
		sourceEnvironment.RsyncPath,
		verboseFlag,
		sshOptionsStr.String(),
		sourceEnvironment.SSH.Port,
		rsyncRemoteSystemUsername,
		sourceEnvironment.SSH.Host,
		lagoonRsyncService,
		syncExcludes,
		sourceEnvironmentName,
		targetEnvironmentName)

	if executeRsyncRemotelyOnTarget {
		execString = GenerateRemoteCommand(targetEnvironment, execString)
	}

	utils.LogExecutionStep(fmt.Sprintf("Running the following for target (%s)", targetEnvironment.EnvironmentName), execString)

	if !dryRun {
		if err, _, errstring := utils.Shellout(execString); err != nil {
			utils.LogFatalError(errstring, nil)
		}
	}

	return nil
}

func SyncRunTargetCommand(targetEnvironment Environment, syncer Syncer, dryRun bool) error {
	utils.LogProcessStep("Beginning import on target environment", targetEnvironment.EnvironmentName)

	targetCommands := syncer.GetLocalCommand(targetEnvironment)

	for _, targetCommand := range targetCommands {
		if targetCommand.NoOp {
			log.Printf("Found No Op for environment %s - skipping step", targetEnvironment.EnvironmentName)
			return nil
		}

		var execString string
		targetCommands, commandErr := targetCommand.GetCommand()
		if commandErr != nil {
			return commandErr
		}

		if targetEnvironment.EnvironmentName == LOCAL_ENVIRONMENT_NAME {
			execString = targetCommands
		} else {
			execString = GenerateRemoteCommand(targetEnvironment, targetCommands)
		}

		utils.LogExecutionStep(fmt.Sprintf("Running the following for target (%s)", targetEnvironment.EnvironmentName), execString)
		if !dryRun {
			err, _, errstring := utils.Shellout(execString)
			if err != nil {
				utils.LogFatalError(errstring, nil)
			}
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
	utils.LogProcessStep("Beginning resource cleanup on", environment.EnvironmentName)

	filesToCleanUp := syncer.GetFilesToCleanup(environment)

	for _, fileToCleanup := range filesToCleanUp {
		transferResourceName := fileToCleanup
		execString := fmt.Sprintf("rm -r %s || true", transferResourceName)

		if environment.EnvironmentName != LOCAL_ENVIRONMENT_NAME {
			execString = GenerateRemoteCommand(environment, execString)
		}

		utils.LogExecutionStep("Running the following", execString)

		if !dryRun {
			err, _, errstring := utils.Shellout(execString)
			if err != nil {
				utils.LogFatalError(errstring, nil)
			}
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
