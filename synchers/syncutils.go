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
	SourceEnvironment    Environment
	TargetEnvironment    Environment
	LagoonSyncer         Syncer
	SyncerType           string
	DryRun               bool
	SshOptionWrapper     *SSHOptionWrapper
	SkipSourceCleanup    bool
	SkipTargetCleanup    bool
	SkipTargetImport     bool
	TransferResourceName string
}

type RunSyncProcessFunctionType = func(args RunSyncProcessFunctionTypeArguments) error

func RunSyncProcess(args RunSyncProcessFunctionTypeArguments) error {
	var err error

	if _, err := args.LagoonSyncer.IsInitialized(); err != nil {
		return err
	}

	//TODO: this can come out.
	args.SourceEnvironment, err = RunPrerequisiteCommand(args.SourceEnvironment, args.LagoonSyncer, args.SyncerType, args.DryRun, args.SshOptionWrapper)
	sourceRsyncPath := "rsync" //args.SourceEnvironment.RsyncPath
	args.SourceEnvironment.RsyncPath = "rsync"
	if err != nil {
		_ = PrerequisiteCleanUp(args.SourceEnvironment, sourceRsyncPath, args.DryRun, args.SshOptionWrapper)
		return err
	}

	err = SyncRunSourceCommand(args.SourceEnvironment, args.LagoonSyncer, args.DryRun, args.SshOptionWrapper)
	if err != nil {
		_ = SyncCleanUp(args.SourceEnvironment, args.LagoonSyncer, args.DryRun, args.SshOptionWrapper)
		return err
	}

	args.TargetEnvironment, err = RunPrerequisiteCommand(args.TargetEnvironment, args.LagoonSyncer, args.SyncerType, args.DryRun, args.SshOptionWrapper)
	targetRsyncPath := args.TargetEnvironment.RsyncPath
	if err != nil {
		_ = PrerequisiteCleanUp(args.TargetEnvironment, targetRsyncPath, args.DryRun, args.SshOptionWrapper)
		return err
	}

	err = SyncRunTransfer(args.SourceEnvironment, args.TargetEnvironment, args.LagoonSyncer, args.DryRun, args.SshOptionWrapper)
	if err != nil {
		_ = PrerequisiteCleanUp(args.SourceEnvironment, sourceRsyncPath, args.DryRun, args.SshOptionWrapper)
		_ = SyncCleanUp(args.SourceEnvironment, args.LagoonSyncer, args.DryRun, args.SshOptionWrapper)
		return err
	}

	if !args.SkipTargetImport {
		err = SyncRunTargetCommand(args.TargetEnvironment, args.LagoonSyncer, args.DryRun, args.SshOptionWrapper)
		if err != nil {
			_ = PrerequisiteCleanUp(args.SourceEnvironment, sourceRsyncPath, args.DryRun, args.SshOptionWrapper)
			_ = PrerequisiteCleanUp(args.TargetEnvironment, targetRsyncPath, args.DryRun, args.SshOptionWrapper)
			_ = SyncCleanUp(args.SourceEnvironment, args.LagoonSyncer, args.DryRun, args.SshOptionWrapper)
			_ = SyncCleanUp(args.TargetEnvironment, args.LagoonSyncer, args.DryRun, args.SshOptionWrapper)
			return err
		}
	} else {
		utils.LogProcessStep("Skipping target import step", nil)
	}

	_ = PrerequisiteCleanUp(args.SourceEnvironment, sourceRsyncPath, args.DryRun, args.SshOptionWrapper)
	_ = PrerequisiteCleanUp(args.TargetEnvironment, targetRsyncPath, args.DryRun, args.SshOptionWrapper)
	if !args.SkipSourceCleanup {
		_ = SyncCleanUp(args.SourceEnvironment, args.LagoonSyncer, args.DryRun, args.SshOptionWrapper)
	}
	if !args.SkipTargetCleanup {
		_ = SyncCleanUp(args.TargetEnvironment, args.LagoonSyncer, args.DryRun, args.SshOptionWrapper)
	} else {
		utils.LogProcessStep("File on the target saved as: "+args.LagoonSyncer.GetTransferResource(args.TargetEnvironment).Name, nil)
	}

	return nil
}

func SyncRunSourceCommand(remoteEnvironment Environment, syncer Syncer, dryRun bool, sshOptionWrapper *SSHOptionWrapper) error {

	utils.LogProcessStep("Beginning export on source environment", remoteEnvironment.EnvironmentName)

	sshOptions := sshOptionWrapper.GetSSHOptionsForEnvironment(remoteEnvironment.EnvironmentName)

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
		execString = command

		utils.LogExecutionStep("Running the following for source", execString)

		if !dryRun {

			if remoteEnvironment.EnvironmentName == LOCAL_ENVIRONMENT_NAME {
				err, response, errstring := utils.Shellout(execString)
				if err != nil {
					log.Printf(errstring)
					return err
				}
				if response != "" && debug == false {
					log.Println(response)
				}
			} else {
				err, output := utils.RemoteShellout(execString, remoteEnvironment.GetOpenshiftProjectName(), sshOptions.Host, sshOptions.Port, sshOptions.PrivateKey, sshOptions.SkipAgent)
				utils.LogDebugInfo(output, nil)
				if err != nil {
					utils.LogFatalError("Unable to exec remote command: "+err.Error(), nil)
					return err
				}
			}
		}
	}

	return nil
}

func SyncRunTransfer(sourceEnvironment Environment, targetEnvironment Environment, syncer Syncer, dryRun bool, sshOptionWrapper *SSHOptionWrapper) error {
	utils.LogProcessStep("Beginning file transfer logic", nil)

	// TODO: This is going to be the trickiest of the ssh option calculations.
	// We need to determine ssh endpoints for both environments separately
	sshOptions := sshOptionWrapper.Default

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

	sourceEnvSshOptions := sshOptionWrapper.GetSSHOptionsForEnvironment(targetEnvironment.EnvironmentName)
	rsyncArgs := sshOptions.RsyncArgs

	execString := fmt.Sprintf("%s %s --rsync-path=%s %s -e \"ssh%s -o LogLevel=FATAL -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -p %s -l %s %s service=%s\" %s %s %s",
		targetEnvironment.RsyncPath,
		rsyncArgs,
		sourceEnvironment.RsyncPath,
		verboseFlag,
		sshOptionsStr.String(),
		sourceEnvSshOptions.Port,
		rsyncRemoteSystemUsername,
		sourceEnvSshOptions.Host,
		lagoonRsyncService,
		syncExcludes,
		sourceEnvironmentName,
		targetEnvironmentName)

	utils.LogExecutionStep(fmt.Sprintf("Running the following for target (%s)", targetEnvironment.EnvironmentName), execString)

	if !dryRun {
		if executeRsyncRemotelyOnTarget {
			TargetEnvSshOptions := sshOptionWrapper.GetSSHOptionsForEnvironment(targetEnvironmentName)
			err, output := utils.RemoteShellout(execString, targetEnvironment.GetOpenshiftProjectName(), TargetEnvSshOptions.Host, TargetEnvSshOptions.Port, TargetEnvSshOptions.PrivateKey, TargetEnvSshOptions.SkipAgent)
			utils.LogDebugInfo(output, nil)
			if err != nil {
				utils.LogFatalError("Unable to exec remote command: "+err.Error(), nil)
				return err
			}
		} else {
			if err, _, errstring := utils.Shellout(execString); err != nil {
				utils.LogFatalError(errstring, nil)
				return err
			}
		}
	}

	return nil
}

func SyncRunTargetCommand(targetEnvironment Environment, syncer Syncer, dryRun bool, sshOptionWrapper *SSHOptionWrapper) error {

	utils.LogProcessStep("Beginning import on target environment", targetEnvironment.EnvironmentName)

	sshOptions := sshOptionWrapper.GetSSHOptionsForEnvironment(targetEnvironment.EnvironmentName)

	targetCommands := syncer.GetLocalCommand(targetEnvironment)

	for _, targetCommand := range targetCommands {
		if targetCommand.NoOp {
			log.Printf("Found No Op for environment %s - skipping step", targetEnvironment.EnvironmentName)
			return nil
		}

		var execString string
		tcomm, commandErr := targetCommand.GetCommand()
		if commandErr != nil {
			return commandErr
		}
		execString = tcomm

		utils.LogExecutionStep(fmt.Sprintf("Running the following for target (%s)", targetEnvironment.EnvironmentName), execString)
		if !dryRun {
			if targetEnvironment.EnvironmentName == LOCAL_ENVIRONMENT_NAME {
				err, _, errstring := utils.Shellout(execString)
				if err != nil {
					utils.LogFatalError(errstring, nil)
					return err
				}
			} else {
				err, output := utils.RemoteShellout(execString, targetEnvironment.GetOpenshiftProjectName(), sshOptions.Host, sshOptions.Port, sshOptions.PrivateKey, sshOptions.SkipAgent)
				utils.LogDebugInfo(output, nil)
				if err != nil {
					utils.LogFatalError("Unable to exec remote command: "+err.Error(), nil)
					return err
				}
			}
		}
	}
	return nil
}

func SyncCleanUp(environment Environment, syncer Syncer, dryRun bool, sshOptionWrapper *SSHOptionWrapper) error {
	transferResouce := syncer.GetTransferResource(environment)

	sshOptions := sshOptionWrapper.GetSSHOptionsForEnvironment(environment.EnvironmentName)

	if transferResouce.SkipCleanup == true {
		log.Printf("Skipping cleanup for %v on %v environment", transferResouce.Name, environment.EnvironmentName)
		return nil
	}
	utils.LogProcessStep("Beginning resource cleanup on", environment.EnvironmentName)

	filesToCleanUp := syncer.GetFilesToCleanup(environment)

	for _, fileToCleanup := range filesToCleanUp {
		transferResourceName := fileToCleanup
		execString := fmt.Sprintf("rm -r %s || true", transferResourceName)

		utils.LogExecutionStep("Running the following", execString)
		if !dryRun {
			if environment.EnvironmentName != LOCAL_ENVIRONMENT_NAME {
				err, output := utils.RemoteShellout(execString, environment.GetOpenshiftProjectName(), sshOptions.Host, sshOptions.Port, sshOptions.PrivateKey, sshOptions.SkipAgent)
				utils.LogDebugInfo(output, nil)
				if err != nil {
					utils.LogFatalError("Unable to exec remote command: "+err.Error(), nil)
					return err
				}
			}
			err, _, errstring := utils.Shellout(execString)
			if err != nil {
				utils.LogFatalError(errstring, nil)
				return err
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
