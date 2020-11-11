package synchers

import (
	"fmt"
	"strconv"
	"time"
)

type DrupalconfigSyncRoot struct {
	Config         BaseDrupalconfigSync
	LocalOverrides DrupalconfigSyncLocal `yaml:"local"`
	TransferId     string
}

type DrupalconfigSyncLocal struct {
	Config BaseDrupalconfigSync
}

type BaseDrupalconfigSync struct {
	SyncPath        string
	OutputDirectory string
}

func (root DrupalconfigSyncRoot) PrepareSyncer() Syncer {
	root.TransferId = strconv.FormatInt(time.Now().UnixNano(), 10)
	return root
}

func (root DrupalconfigSyncRoot) GetRemoteCommand() string {
	transferResource := root.GetTransferResource()
	return fmt.Sprintf("drush config-export --destination=%s", transferResource.Name)
}

func (m DrupalconfigSyncRoot) GetLocalCommand() string {
	// l := m.getEffectiveLocalDetails()
	transferResource := m.GetTransferResource()

	return fmt.Sprintf("drush -y config-import --source=%s", transferResource.Name)
}

func (m DrupalconfigSyncRoot) GetTransferResource() SyncerTransferResource {
	return SyncerTransferResource{
		Name:        fmt.Sprintf("%vdrupalconfig-sync-%v", m.GetOutputDirectory(), m.TransferId),
		IsDirectory: true}
}

func (root DrupalconfigSyncRoot) GetOutputDirectory() string {
	m := root.Config
	if len(m.OutputDirectory) == 0 {
		return "/tmp/"
	}
	return m.OutputDirectory
}

func (syncConfig DrupalconfigSyncRoot) getEffectiveLocalDetails() BaseDrupalconfigSync {
	returnDetails := BaseDrupalconfigSync{
		SyncPath:        syncConfig.Config.SyncPath,
		OutputDirectory: syncConfig.Config.OutputDirectory,
	}

	assignLocalOverride := func(target *string, override *string) {
		if len(*override) > 0 {
			*target = *override
		}
	}

	//TODO: can this be replaced with reflection?
	assignLocalOverride(&returnDetails.SyncPath, &syncConfig.LocalOverrides.Config.SyncPath)
	assignLocalOverride(&returnDetails.OutputDirectory, &syncConfig.LocalOverrides.Config.OutputDirectory)
	return returnDetails
}
