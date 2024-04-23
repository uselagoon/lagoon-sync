package synchers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/uselagoon/lagoon-sync/prerequisite"
	"github.com/uselagoon/lagoon-sync/utils"
)

func RunPrerequisiteCommand(environment Environment, syncer Syncer, syncerType string, dryRun bool, sshOptionWrapper *SSHOptionWrapper) (Environment, error) {

	sshOptions := sshOptionWrapper.GetSSHOptionsForEnvironment(environment.EnvironmentName)

	// We don't run prerequisite checks on these syncers for now.
	if syncerType == "files" || syncerType == "drupalconfig" {
		environment.RsyncPath = "rsync"
		return environment, nil
	}

	utils.LogProcessStep("Running prerequisite checks on", environment.EnvironmentName)

	var execString string
	var configRespSuccessful bool

	execString, commandErr := syncer.GetPrerequisiteCommand(environment, "config").GetCommand()
	if commandErr != nil {
		return environment, commandErr
	}

	utils.LogExecutionStep("Running the following prerequisite command", execString)

	var output string

	if environment.EnvironmentName == LOCAL_ENVIRONMENT_NAME {
		err, response, errstring := utils.Shellout(execString)
		if err != nil {
			log.Printf(errstring)
			return environment, err
		}
		if response != "" && debug == false {
			log.Println(response)
		}
	} else {
		err, output := utils.RemoteShellout(execString, environment.GetOpenshiftProjectName(), sshOptions.Host, sshOptions.Port, sshOptions.PrivateKey, sshOptions.SkipAgent)
		utils.LogDebugInfo(output, nil)
		if err != nil {
			utils.LogFatalError("Unable to exec remote command: "+err.Error(), nil)
			return environment, err
		}
	}

	data := &prerequisite.PreRequisiteResponse{}
	json.Unmarshal([]byte(output), &data)

	if !data.IsPrerequisiteResponseEmpty() {
		utils.LogDebugInfo("'lagoon-sync config' response", output)
		configRespSuccessful = true
	} else {
		utils.LogWarning("'lagoon-sync' is not available on", environment.EnvironmentName)
		configRespSuccessful = false
	}

	// Check if environment has rsync
	if data.RysncPrerequisite != nil {
		for _, c := range data.RysncPrerequisite {
			if c.Value != "" {
				environment.RsyncAvailable = true
				environment.RsyncPath = c.Value
			}
		}
	}

	// Check if prerequisite checks were successful.
	if !configRespSuccessful {
		utils.LogDebugInfo("Unable to determine rsync config, will attempt to use 'rsync' path instead", environment.EnvironmentName)
		environment.RsyncPath = "rsync"
		return environment, nil
	}
	if environment.RsyncAvailable {
		utils.LogDebugInfo("Rsync found", environment.RsyncPath)
	}

	if !dryRun && !environment.RsyncAvailable {
		return environment, errors.New("Unable to find rsync - unable to continue")
	}

	return environment, nil
}

func PrerequisiteCleanUp(environment Environment, rsyncPath string, dryRun bool, sshOptionWrapper *SSHOptionWrapper) error {

	sshOptions := sshOptionWrapper.GetSSHOptionsForEnvironment(environment.EnvironmentName)

	if rsyncPath == "" || rsyncPath == "rsync" || !strings.Contains(rsyncPath, "/tmp/") {
		return nil
	}

	utils.LogProcessStep("Beginning prerequisite resource cleanup on", environment.EnvironmentName)

	execString := fmt.Sprintf("rm -r %s", rsyncPath)

	if environment.EnvironmentName != LOCAL_ENVIRONMENT_NAME {
		execString = GenerateRemoteCommand(environment, execString, sshOptions)
	}

	utils.LogExecutionStep("Running the following", execString)

	if !dryRun {
		err, _, errstring := utils.Shellout(execString)

		if err != nil {
			fmt.Println(errstring)
			return err
		}
	}

	return nil
}
