package synchers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strings"

	"github.com/amazeeio/lagoon-sync/assets"
	"github.com/amazeeio/lagoon-sync/prerequisite"
	"github.com/amazeeio/lagoon-sync/utils"
	"github.com/spf13/viper"
)

func RunPrerequisiteCommand(environment Environment, syncer Syncer, syncerType string, dryRun bool, verboseSSH bool) (Environment, error) {
	// We don't run prerequisite checks on these syncers for now.
	if syncerType == "files" || syncerType == "drupalconfig" {
		environment.RsyncPath = "rsync"
		return environment, nil
	}

	log.Printf("Running prerequisite checks on %s environment", environment.EnvironmentName)

	var execString string
	var configRespSuccessful bool

	command, commandErr := syncer.GetPrerequisiteCommand(environment, "config --no-debug").GetCommand()
	if commandErr != nil {
		return environment, commandErr
	}

	if environment.EnvironmentName == LOCAL_ENVIRONMENT_NAME {
		execString = command
	} else {
		execString = GenerateRemoteCommand(environment, command, verboseSSH)
	}

	log.Printf("Running the following prerequisite command:- %s", execString)

	err, configResponseJson, errstring := utils.Shellout(execString)
	if err != nil {
		fmt.Println(errstring)
	}

	data := &prerequisite.PreRequisiteResponse{}
	json.Unmarshal([]byte(configResponseJson), &data)

	if !data.IsPrerequisiteResponseEmpty() {
		if debug := viper.Get("no-debug"); debug == false {
			log.Printf("Config response: %v", configResponseJson)
		}
		configRespSuccessful = true
	} else {
		log.Printf("%v\n-----\nWarning: lagoon-sync is not available on %s\n-----", configResponseJson, environment.EnvironmentName)
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
		log.Printf("Unable to determine rsync config on %v, will attempt to use 'rsync' path instead", environment.EnvironmentName)
		environment.RsyncPath = "rsync"
		return environment, nil
	}
	if environment.RsyncAvailable {
		log.Printf("Rsync found: %v", environment.RsyncPath)
	}

	if !dryRun && !environment.RsyncAvailable {
		// Add rsync to env
		rsyncPath, err := createRsync(environment, syncer, lagoonSyncVersion)
		if err != nil {
			fmt.Println(errstring)
			return environment, err
		}

		log.Printf("Rsync path: %s", rsyncPath)
		environment.RsyncPath = rsyncPath
		return environment, nil
	}

	return environment, nil
}

func PrerequisiteCleanUp(environment Environment, rsyncPath string, dryRun bool, verboseSSH bool) error {
	log.Printf("Beginning prerequisite resource cleanup on %s", environment.EnvironmentName)
	if rsyncPath == "" || rsyncPath == "rsync" || !strings.Contains(rsyncPath, "/tmp/") {
		return nil
	}
	execString := fmt.Sprintf("rm -r %s", rsyncPath)

	if environment.EnvironmentName != LOCAL_ENVIRONMENT_NAME {
		execString = GenerateRemoteCommand(environment, execString, verboseSSH)
	}

	log.Printf("Running the following: %s", execString)

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
func createRsync(environment Environment, syncer Syncer, lagoonSyncVersion string) (string, error) {
	log.Printf("%v environment doesn't have rsync", environment.EnvironmentName)
	log.Printf("Downloading rsync asset on %v", environment.EnvironmentName)

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

		execString = fmt.Sprintf("ssh -t -o \"UserKnownHostsFile=/dev/null\" -o \"StrictHostKeyChecking=no\" -p 32222 %v@ssh.lagoon.amazeeio.cloud %v %v",
			environment.GetOpenshiftProjectName(), serviceArgument, command)
	}

	log.Printf("Running the following for:- %s", execString)

	if err, _, errstring := utils.Shellout(execString); err != nil {
		log.Println(errstring)
		return "", err
	}

	// Remove local versioned rsync (post ssh transfer) - otherwise rsync will be avialable on target at /tmp/
	removeLocalRsyncCopyExecString := fmt.Sprintf("rm -rf %v", rsyncDestinationPath)
	log.Printf("Removing rsync binary locally stored: %v", removeLocalRsyncCopyExecString)
	if err, _, errstring := utils.Shellout(removeLocalRsyncCopyExecString); err != nil {
		log.Println(errstring)
		return "", err
	}

	return rsyncDestinationPath, nil
}

func createRsyncAssetFromBytes(lagoonSyncVersion string) (string, error) {
	tempRsyncPath := "/tmp/rsync"
	err := ioutil.WriteFile(tempRsyncPath, assets.GetRSYNC(), 0774)
	if err != nil {
		log.Fatal(err)
	}

	if lagoonSyncVersion == "" {
		log.Print("No lagoon-sync version set")
	}

	// Test running local rsync binary (won't run on darwin OS)
	// localRsyncCommand := exec.Command(tempRsyncPath", "--version")
	// stdout, err := localRsyncCommand.StdoutPipe()
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// if err := localRsyncCommand.Start(); err != nil {
	// 	log.Fatal(err)
	// }

	// r := bufio.NewReader(stdout)
	// b, err := r.ReadBytes('\n')
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Println("Local rsync ran:", string(b))

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

	removeTempLocalRsyncCopyExecString := fmt.Sprintf("rm -rf %v", tempRsyncPath)
	log.Printf("Removing temp rsync binary: %v", removeTempLocalRsyncCopyExecString)
	if err, _, errstring := utils.Shellout(removeTempLocalRsyncCopyExecString); err != nil {
		log.Println(errstring)
		return "", err
	}

	return versionedRsyncPath, nil
}
