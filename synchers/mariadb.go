package synchers

import "fmt"

type BaseMariaDbSync struct {
	DbHostname      string
	DbUsername      string
	DbPassword      string
	DbPort          string
	DbDatabase      string
	OutputDirectory string
}

type MariadbSync struct {
	BaseMariaDbSync
	LocalOverrides  BaseMariaDbSync
}

func (m MariadbSync) GetRemoteCommand() string {
	return fmt.Sprintf("mysqldump -h%s -u%s -p%s -P%s %s > %s", m.DbHostname, m.DbUsername, m.DbPassword, m.DbPort, m.DbDatabase, m.OutputDirectory)
}

func (m MariadbSync) GetLocalCommand() string {
	l := m.getEffectiveLocalDetails()
	return fmt.Sprintf("mysql -h%s -u%s -p%s -P%s %s < %s", l.DbHostname, l.DbUsername, l.DbPassword, l.DbPort, l.DbDatabase, l.OutputDirectory)
}

func (m MariadbSync) GetTransferResourceName() string {
	return m.GetOutputDirectory() + "lagoon_sync_mariadb-"
}

func (m MariadbSync) GetOutputDirectory() string {
	return "/tmp/"
}

func (syncConfig MariadbSync) getEffectiveLocalDetails() BaseMariaDbSync {
	returnDetails := BaseMariaDbSync{
		DbHostname:      syncConfig.DbHostname,
		DbUsername:      syncConfig.DbUsername,
		DbPassword:      syncConfig.DbPassword,
		DbPort:          syncConfig.DbPort,
		DbDatabase:      syncConfig.DbDatabase,
		OutputDirectory: syncConfig.OutputDirectory,
	}

	assignLocalOverride := func(target *string, override *string) {
		fmt.Println(*override)
		if len(*override) > 0 {
			*target = *override
		}
	}

	//TODO: can this be replaced with reflection?
	assignLocalOverride(&returnDetails.DbHostname, &syncConfig.LocalOverrides.DbHostname)
	assignLocalOverride(&returnDetails.DbUsername, &syncConfig.LocalOverrides.DbUsername)
	assignLocalOverride(&returnDetails.DbPassword, &syncConfig.LocalOverrides.DbPassword)
	assignLocalOverride(&returnDetails.DbPort, &syncConfig.LocalOverrides.DbPort)
	assignLocalOverride(&returnDetails.DbDatabase, &syncConfig.LocalOverrides.DbDatabase)
	assignLocalOverride(&returnDetails.OutputDirectory, &syncConfig.LocalOverrides.OutputDirectory)
	return returnDetails

}