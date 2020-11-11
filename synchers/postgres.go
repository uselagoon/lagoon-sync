package synchers

import (
	"fmt"
	"strconv"
	"time"
)

type PostgresSyncRoot struct {
	Config         BasePostgresSync  `yaml:"config"`
	LocalOverrides PostgresSyncLocal `yaml:"local"`
	TransferId     string
}

type PostgresSyncLocal struct {
	Config BasePostgresSync
}

type BasePostgresSync struct {
	DbHostname       string   `yaml:"hostname"`
	DbUsername       string   `yaml:"username"`
	DbPassword       string   `yaml:"password"`
	DbPort           string   `yaml:"port"`
	DbDatabase       string   `yaml:"database"`
	ExcludeTable     []string `yaml:"exclude-table"`
	ExcludeTableData []string `yaml:"exclude-table-data"`
	OutputDirectory  string
}

func (root PostgresSyncRoot) PrepareSyncer() Syncer {
	root.TransferId = strconv.FormatInt(time.Now().UnixNano(), 10)
	return root
}

func (root PostgresSyncRoot) GetRemoteCommand() string {
	m := root.Config
	transferResource := root.GetTransferResource()

	var tablesToExclude string
	for _, s := range m.ExcludeTable {
		tablesToExclude += fmt.Sprintf("--exclude-table=%s.%s ", m.DbDatabase, s)
	}

	var tablesWhoseDataToExclude string
	for _, s := range m.ExcludeTableData {
		tablesWhoseDataToExclude += fmt.Sprintf("--exclude-table-data=%s.%s ", m.DbDatabase, s)
	}

	return fmt.Sprintf("PGPASSWORD=\"%s\" pg_dump --no-owner -h%s -U%s -p%s %s %s %s > %s", m.DbPassword, m.DbHostname, m.DbUsername, m.DbPort, tablesToExclude, tablesWhoseDataToExclude, m.DbDatabase, transferResource.Name)
}

func (m PostgresSyncRoot) GetLocalCommand() string {
	l := m.getEffectiveLocalDetails()
	transferResource := m.GetTransferResource()
	return fmt.Sprintf("pg_restore --no-privileges --no-owner -U%s -d%s --clean < %s", l.DbUsername, l.DbDatabase, transferResource.Name)
}

func (m PostgresSyncRoot) GetTransferResource() SyncerTransferResource {
	return SyncerTransferResource{
		Name:        fmt.Sprintf("%vlagoon_sync_postgres_%v.sql", m.GetOutputDirectory(), m.TransferId),
		IsDirectory: false}
}

func (root PostgresSyncRoot) GetOutputDirectory() string {
	m := root.Config
	if len(m.OutputDirectory) == 0 {
		return "/tmp/"
	}
	return m.OutputDirectory
}

func (syncConfig PostgresSyncRoot) getEffectiveLocalDetails() BasePostgresSync {
	returnDetails := BasePostgresSync{
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
