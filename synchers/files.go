package synchers

import (
	"fmt"
	"strconv"
	"time"
)

type FilesSyncRoot struct {
	Config         BaseFilesSync
	LocalOverrides FilesSyncLocal `yaml:"local"`
	TransferId     string
}

type FilesSyncLocal struct {
	Config BaseFilesSync
}

type BaseFilesSync struct {
	SyncPath string `yaml:"sync-directory"`
	Exclude  []string
}

func (root FilesSyncRoot) PrepareSyncer() (Syncer, error) {
	root.TransferId = strconv.FormatInt(time.Now().UnixNano(), 10)
	return root, nil
}

func (root FilesSyncRoot) GetRemoteCommand(environment Environment) SyncCommand {
	return SyncCommand{
		command: "",
		NoOp:    true,
	}
}

func (m FilesSyncRoot) GetLocalCommand(environment Environment) SyncCommand {
	return SyncCommand{
		command: "",
		NoOp:    true,
	}
}

func (m FilesSyncRoot) GetTransferResource(environment Environment) SyncerTransferResource {
	config := m.Config
	if environment.EnvironmentName == LOCAL_ENVIRONMENT_NAME {
		config = m.getEffectiveLocalDetails()
	}
	return SyncerTransferResource{
		Name:        fmt.Sprintf(config.SyncPath),
		IsDirectory: true,
		SkipCleanup: true,
		ExcludeResources: m.Config.Exclude,
	}
}

func (syncConfig FilesSyncRoot) getEffectiveLocalDetails() BaseFilesSync {
	returnDetails := BaseFilesSync{
		SyncPath: syncConfig.Config.SyncPath,
	}

	assignLocalOverride := func(target *string, override *string) {
		if len(*override) > 0 {
			*target = *override
		}
	}

	//TODO: can this be replaced with reflection?
	assignLocalOverride(&returnDetails.SyncPath, &syncConfig.LocalOverrides.Config.SyncPath)
	return returnDetails
}
