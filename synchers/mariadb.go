package synchers

import (
	"fmt"
)

type BaseMariaDbSync struct {
	DbHostname      string `yaml:"hostname"`
	DbUsername      string `yaml:"username"`
	DbPassword      string `yaml:"password"`
	DbPort          string `yaml:"port"`
	DbDatabase      string `yaml:"database"`
	OutputDirectory string
}

type MariadbSyncLocal struct {
	Config BaseMariaDbSync
}

type MariadbSyncRoot struct {
	Config         BaseMariaDbSync
	LocalOverrides MariadbSyncLocal `yaml:"local"`
}

func (root MariadbSyncRoot) GetRemoteCommand() string {
	m := root.Config
	return fmt.Sprintf("mysqldump -h%s -u%s -p%s -P%s %s > %s", m.DbHostname, m.DbUsername, m.DbPassword, m.DbPort, m.DbDatabase, m.OutputDirectory)
}

func (m MariadbSyncRoot) GetLocalCommand() string {
	l := m.getEffectiveLocalDetails()
	return fmt.Sprintf("mysql -h%s -u%s -p%s -P%s %s < %s", l.DbHostname, l.DbUsername, l.DbPassword, l.DbPort, l.DbDatabase, l.OutputDirectory)
}

func (m MariadbSyncRoot) GetTransferResource() SyncerTransferResource {
	return SyncerTransferResource{
		Name:        m.GetOutputDirectory() + "lagoon_sync_mariadb-",
		IsDirectory: false}
}

func (root MariadbSyncRoot) GetOutputDirectory() string {
	m := root.Config
	if(len(m.OutputDirectory) == 0) {
		return "/tmp/"
	}
	return m.OutputDirectory
}

func (syncConfig MariadbSyncRoot) getEffectiveLocalDetails() BaseMariaDbSync {
	returnDetails := BaseMariaDbSync{
		DbHostname:      syncConfig.Config.DbHostname,
		DbUsername:      syncConfig.Config.DbUsername,
		DbPassword:      syncConfig.Config.DbPassword,
		DbPort:          syncConfig.Config.DbPort,
		DbDatabase:      syncConfig.Config.DbDatabase,
		OutputDirectory: syncConfig.Config.OutputDirectory,
	}

	assignLocalOverride := func(target *string, override *string) {
		if len(*override) > 0 {
			*target = *override
		}
	}

	//TODO: can this be replaced with reflection?
	assignLocalOverride(&returnDetails.DbHostname, &syncConfig.LocalOverrides.Config.DbHostname)
	assignLocalOverride(&returnDetails.DbUsername, &syncConfig.LocalOverrides.Config.DbUsername)
	assignLocalOverride(&returnDetails.DbPassword, &syncConfig.LocalOverrides.Config.DbPassword)
	assignLocalOverride(&returnDetails.DbPort, &syncConfig.LocalOverrides.Config.DbPort)
	assignLocalOverride(&returnDetails.DbDatabase, &syncConfig.LocalOverrides.Config.DbDatabase)
	assignLocalOverride(&returnDetails.OutputDirectory, &syncConfig.LocalOverrides.Config.OutputDirectory)
	return returnDetails

}
