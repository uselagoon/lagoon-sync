package synchers

import (
	"bytes"
	"errors"
	"fmt"
	"gopkg.in/yaml.v2"
	"os/exec"
	"strconv"
	"time"
)

// UnmarshallLagoonYamlToLagoonSyncStructure will take a bytestream and return a fully parsed lagoon sync config structure
func UnmarshallLagoonYamlToLagoonSyncStructure(data []byte) (SyncherConfigRoot, error) {
	transferId := strconv.FormatInt(time.Now().UnixNano(), 10)
	lagoonConfig := SyncherConfigRoot{
		LagoonSync: LagoonSync{
			TransferId: transferId,
		},
	}
	err := yaml.Unmarshal(data, &lagoonConfig)
	fmt.Print(lagoonConfig)
	if err != nil {
		return SyncherConfigRoot{}, errors.New("Unable to parse lagoon config yaml setup")
	}
	return lagoonConfig, nil
}

func RunSyncProcess(sourceEnvironment RemoteEnvironment, lagoonSyncer Syncer) error {
	var err error
	err = SyncRunRemote(sourceEnvironment, lagoonSyncer)

	if err != nil {
		_ = SyncCleanUp(lagoonSyncer)
		return err
	}
	err = SyncRunTransfer(sourceEnvironment, lagoonSyncer)
	if err != nil {
		_ = SyncCleanUp(lagoonSyncer)
		return err
	}

	err = SyncRunLocal(lagoonSyncer)
	if err != nil {
		_ = SyncCleanUp(lagoonSyncer)
		return err
	}

	return SyncCleanUp(lagoonSyncer)
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

func SyncRunRemote(remoteEnvironment RemoteEnvironment, syncer Syncer) error {
	execString := fmt.Sprintf("ssh -t -o \"UserKnownHostsFile=/dev/null\" -o \"StrictHostKeyChecking=no\" -p 32222 %v@ssh.lagoon.amazeeio.cloud '%v'",
		remoteEnvironment.getOpenshiftProjectName(), syncer.GetRemoteCommand())

	//err, outstring, errstring := Shellout(execString)
	//
	//if err != nil {
	//	fmt.Println(errstring)
	//	return err
	//}
	//fmt.Println(outstring)
	fmt.Println(execString)
	return nil
}

func SyncRunTransfer(remoteEnvironment RemoteEnvironment, syncer Syncer) error {

	remoteResourceName := syncer.GetTransferResource().Name
	if syncer.GetTransferResource().IsDirectory == true {
		remoteResourceName += "/"
	}
	localResourceName := syncer.GetTransferResource().Name

	execString := fmt.Sprintf("rsync -e \"ssh -o LogLevel=ERROR -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -p 32222\" -a %s@ssh.lagoon.amazeeio.cloud:%s %s",
		remoteEnvironment.getOpenshiftProjectName(),
		remoteResourceName,
		localResourceName)

	//err, outstring, errstring := Shellout(execString)
	//
	//if err != nil {
	//	fmt.Println(errstring)
	//	return err
	//}
	//
	//fmt.Println(outstring)
	fmt.Println(execString)
	return nil
}

func SyncRunLocal(syncer Syncer) error {
	execString := syncer.GetLocalCommand()

	//err, outstring, errstring := Shellout(execString)
	//
	//if err != nil {
	//	fmt.Println(errstring)
	//	return err
	//}
	//fmt.Println(outstring)
	fmt.Println(execString)
	return nil
}

func SyncCleanUp(syncer Syncer) error {
	//remove remote resources
	//remove local resources
	fmt.Println("Cleaning up ...")
	return nil
}
