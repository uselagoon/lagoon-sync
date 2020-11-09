package synchers

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v2"
)

// UnmarshallLagoonYamlToLagoonSyncStructure will take a bytestream and return a fully parsed lagoon sync config structure
func UnmarshallLagoonYamlToLagoonSyncStructure(data []byte) (SyncherConfigRoot, error) {
	lagoonConfig := SyncherConfigRoot{}
	err := yaml.Unmarshal(data, &lagoonConfig)
	if(err != nil) {
		return SyncherConfigRoot{}, errors.New("Unable to parse lagoon config yaml setup")
	}
	return lagoonConfig, nil
}


func SyncRunRemote(syncer Syncer) error {
	fmt.Print("I'm going to be running the following: ")
	println(syncer.GetRemoteCommand())
	return nil
}

func SyncRunTransfer(syncer Syncer) error {
	fmt.Print("I'm going to be rsyncing the following resource: ")
	return nil
}

func SyncRunLocal(syncer Syncer) error {
	fmt.Print("I'm going to be running the following: ")
	fmt.Println(syncer.GetLocalCommand())
	return nil
}