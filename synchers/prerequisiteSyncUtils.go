package synchers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/uselagoon/lagoon-sync/assets"

	"github.com/uselagoon/lagoon-sync/prerequisite"
	"github.com/uselagoon/lagoon-sync/utils"
)

func RunPrerequisiteCommand(environment Environment, syncer Syncer, syncerType string, dryRun bool, sshOptions SSHOptions) (Environment, error) {
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

	lagoonSyncVersion := "unknown"
	if data.Version != "" {
		lagoonSyncVersion = data.Version
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
		// Add rsync to env
		rsyncPath, err := createRsync(environment, syncer, lagoonSyncVersion, sshOptions)
		if err != nil {
			return environment, err
		}

		utils.LogDebugInfo("Rsync path", rsyncPath)
		environment.RsyncPath = rsyncPath
		return environment, nil
	}

	return environment, nil
}

func PrerequisiteCleanUp(environment Environment, rsyncPath string, dryRun bool, sshOptions SSHOptions) error {
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

var RsyncAssetPath = "/tmp/rsync"

// will add bundled rsync onto environment and return the new rsync path as string
func createRsync(environment Environment, syncer Syncer, lagoonSyncVersion string, sshOptions SSHOptions) (string, error) {
	utils.LogDebugInfo("%v environment doesn't have rsync", environment.EnvironmentName)
	utils.LogDebugInfo("Downloading rsync asset on", environment.EnvironmentName)

	environmentName := syncer.GetTransferResource(environment).Name
	if syncer.GetTransferResource(environment).IsDirectory == true {
		environmentName += "/"
	}

	// Create rsync asset
	RsyncAssetPath, err := createRsyncAssetFromBytes(lagoonSyncVersion)
	if err != nil {
		log.Println(err)
	}

	rsyncDestinationPath := fmt.Sprintf("%vlagoon_sync_rsync_%v", "/tmp/", strings.ReplaceAll(lagoonSyncVersion, ".", "_"))

	environment.RsyncLocalPath = rsyncDestinationPath
	environment.RsyncPath = RsyncAssetPath

	// If local we bail out here
	if environment.EnvironmentName == LOCAL_ENVIRONMENT_NAME {
		return rsyncDestinationPath, nil
	}

	var execString string
	command := fmt.Sprintf("'cat > %v && chmod +x %v' < %s",
		rsyncDestinationPath,
		rsyncDestinationPath,
		rsyncDestinationPath,
	)

	if environment.EnvironmentName == LOCAL_ENVIRONMENT_NAME {
		execString = command
	} else {
		serviceArgument := ""
		if environment.ServiceName != "" {
			serviceArgument = fmt.Sprintf("service=%v", environment.ServiceName)

		}

		execString = fmt.Sprintf("ssh -t -o \"UserKnownHostsFile=/dev/null\" -o \"StrictHostKeyChecking=no\" -p %s %s@%s %s %s",
			sshOptions.Port, environment.GetOpenshiftProjectName(), sshOptions.Host, serviceArgument, command)
	}

	utils.LogExecutionStep(fmt.Sprintf("Running the following for %s", environment.EnvironmentName), execString)

	if err, _, errstring := utils.Shellout(execString); err != nil {
		log.Println(errstring)
		return "", err
	}

	// Remove local versioned rsync (post ssh transfer) - otherwise rsync will be avialable on target at /tmp/
	log.Printf("Removing rsync binary locally stored: %s", rsyncDestinationPath)
	if err := os.Remove(rsyncDestinationPath); err != nil {
		log.Println(err.Error())
	}

	return rsyncDestinationPath, nil
}

func createRsyncAssetFromBytes(lagoonSyncVersion string) (string, error) {
	tempRsyncPath := "/tmp/rsync"
	err := ioutil.WriteFile(tempRsyncPath, assets.RsyncBin(), 0774)
	if err != nil {
		utils.LogFatalError("Unable to write to file", err)
	}

	if lagoonSyncVersion == "" {
		utils.LogDebugInfo("No lagoon-sync version set", nil)
	}

	// Rename rsync binary with latest lagoon version appended.
	versionedRsyncPath := fmt.Sprintf("%vlagoon_sync_rsync_%v", "/tmp/", strings.ReplaceAll(lagoonSyncVersion, ".", "_"))
	cpRsyncCmd := fmt.Sprintf("cp %s %s",
		tempRsyncPath,
		versionedRsyncPath,
	)

	if err, _, errstring := utils.Shellout(cpRsyncCmd); err != nil {
		log.Println(errstring)
		return "", err
	}

	log.Printf("Removing temp rsync binary: %s", tempRsyncPath)
	if err := os.Remove(tempRsyncPath); err != nil {
		log.Println(err.Error())
	}

	return versionedRsyncPath, nil
}
