package prerequisite

import (
	"os"
)

type EnvVarRsyncPrerequisite struct {
	SyncerType        string
	LagoonVersion     string
	LagoonProject     string
	LagoonEnvironment string
	LagoonRoute       string
	LagoonDomain      string
	Lagoon            string
	MariaDb           DbEnvVars
	MongoDb           DbEnvVars
	PostgresDb        DbEnvVars
}

type DbEnvVars struct {
	DbType   string
	Hostname string
	Username string
	Password string
	Port     string
	Database string
}

func (e *EnvVarRsyncPrerequisite) GetName() string {
	return "env-vars"
}

func (e *EnvVarRsyncPrerequisite) GetValue() bool {
	var lagoonVersion = os.Getenv("LAGOON_VERSION")
	e.LagoonVersion = lagoonVersion

	var lagoonProject = os.Getenv("LAGOON_PROJECT")
	if lagoonProject == "" {
		lagoonProject = os.Getenv("LAGOON_SAFE_PROJECT")
	}
	e.LagoonProject = lagoonProject

	var lagoonEnvironment = os.Getenv("LAGOON_GIT_SAFE_BRANCH")
	e.LagoonEnvironment = lagoonEnvironment

	e.MariaDb = getMariaDbEnvVars()
	e.MongoDb = getMongoDbEnvVars()
	e.PostgresDb = getPostgresDbEnvVars()

	return true
}

func (e *EnvVarRsyncPrerequisite) GatherPrerequisites() ([]GatheredPrerequisite, error) {

	gatheredPrerequisite := []GatheredPrerequisite{
		{
			Name:   "lagoon_version",
			Value:  e.LagoonVersion,
			Status: getStatusFromString(e.LagoonVersion),
		},
		{
			Name:   "lagoon_project",
			Value:  e.LagoonProject,
			Status: getStatusFromString(e.LagoonProject),
		},
		{
			Name:   "lagoon_env",
			Value:  e.LagoonEnvironment,
			Status: getStatusFromString(e.LagoonEnvironment),
		},
	}

	if e.MariaDb.DbType != "" {
		gatheredPrerequisite = append(gatheredPrerequisite,
			GatheredPrerequisite{
				Name:   "mariadb_hostname",
				Value:  e.MariaDb.Hostname,
				Status: getStatusFromString(e.MariaDb.Hostname),
			},
			GatheredPrerequisite{
				Name:   "mariadb_username",
				Value:  e.MariaDb.Username,
				Status: getStatusFromString(e.MariaDb.Username),
			},
			GatheredPrerequisite{
				Name:   "mariadb_password",
				Value:  e.MariaDb.Password,
				Status: getStatusFromString(e.MariaDb.Password),
			},
			GatheredPrerequisite{
				Name:   "mariadb_port",
				Value:  e.MariaDb.Port,
				Status: getStatusFromString(e.MariaDb.Port),
			},
			GatheredPrerequisite{
				Name:   "mariadb_database",
				Value:  e.MariaDb.Database,
				Status: getStatusFromString(e.MariaDb.Database),
			},
		)
	}

	if e.MongoDb.DbType != "" {
		gatheredPrerequisite = append(gatheredPrerequisite,
			GatheredPrerequisite{
				Name:   "mongodb_hostname",
				Value:  e.MongoDb.Hostname,
				Status: getStatusFromString(e.MongoDb.Hostname),
			},
			GatheredPrerequisite{
				Name:   "mongodb_username",
				Value:  e.MongoDb.Username,
				Status: getStatusFromString(e.MongoDb.Username),
			},
			GatheredPrerequisite{
				Name:   "mongodb_password",
				Value:  e.MongoDb.Password,
				Status: getStatusFromString(e.MongoDb.Password),
			},
			GatheredPrerequisite{
				Name:   "mongodb_port",
				Value:  e.MongoDb.Port,
				Status: getStatusFromString(e.MariaDb.Port),
			},
			GatheredPrerequisite{
				Name:   "mongodb_database",
				Value:  e.MongoDb.Database,
				Status: getStatusFromString(e.MariaDb.Database),
			},
		)
	}

	if e.PostgresDb.DbType != "" {
		gatheredPrerequisite = append(gatheredPrerequisite,
			GatheredPrerequisite{
				Name:   "postgresdb_hostname",
				Value:  e.PostgresDb.Hostname,
				Status: getStatusFromString(e.PostgresDb.Hostname),
			},
			GatheredPrerequisite{
				Name:   "postgresdb_username",
				Value:  e.PostgresDb.Username,
				Status: getStatusFromString(e.PostgresDb.Username),
			},
			GatheredPrerequisite{
				Name:   "postgresdb_password",
				Value:  e.PostgresDb.Password,
				Status: getStatusFromString(e.PostgresDb.Password),
			},
			GatheredPrerequisite{
				Name:   "postgresdb_port",
				Value:  e.PostgresDb.Port,
				Status: getStatusFromString(e.PostgresDb.Port),
			},
			GatheredPrerequisite{
				Name:   "postgresdb_database",
				Value:  e.PostgresDb.Database,
				Status: getStatusFromString(e.PostgresDb.Database),
			},
		)
	}

	return gatheredPrerequisite, nil
}

func (e *EnvVarRsyncPrerequisite) Status() int {
	return 0
}

func getMariaDbEnvVars() DbEnvVars {
	var hostname, mariadbHostExists = os.LookupEnv("MARIADB_HOSTNAME")
	if !mariadbHostExists {
		hostname, mariadbHostExists = os.LookupEnv("MARIADB_HOST")
	}
	var username = os.Getenv("MARIADB_USERNAME")
	var password = os.Getenv("MARIADB_PASSWORD")
	var port = os.Getenv("MARIADB_PORT")
	var database = os.Getenv("MARIADB_DATABASE")

	var dbType = ""
	if mariadbHostExists {
		dbType = "mariadb"
	}

	return DbEnvVars{
		DbType:   dbType,
		Hostname: hostname,
		Username: username,
		Password: password,
		Port:     port,
		Database: database,
	}
}

func getMongoDbEnvVars() DbEnvVars {
	var hostname, mongodbHostExists = os.LookupEnv("MONGODB_HOSTNAME")
	var username = os.Getenv("MONGODB_USERNAME")
	var password = os.Getenv("MONGODB_PASSWORD")
	var port = os.Getenv("MONGODB_SERVICE_PORT")
	var database = os.Getenv("MONGODB_DATABASE")
	if database == "" {
		database = "local"
	}

	var dbType = ""
	if mongodbHostExists {
		dbType = "mongodb"
	}

	return DbEnvVars{
		DbType:   dbType,
		Hostname: hostname,
		Username: username,
		Password: password,
		Port:     port,
		Database: database,
	}
}

func getPostgresDbEnvVars() DbEnvVars {
	var hostname, postgresHostExists = os.LookupEnv("POSTGRES_HOST")
	var username = os.Getenv("POSTGRES_USERNAME")
	var password = os.Getenv("POSTGRES_PASSWORD")
	var port = os.Getenv("POSTGRES_PORT")
	if port == "" {
		port = "5432"
	}
	var database = os.Getenv("POSTGRES_DATABASE")

	var dbType = ""
	if postgresHostExists {
		dbType = "postgres"
	}

	return DbEnvVars{
		DbType:   dbType,
		Hostname: hostname,
		Username: username,
		Password: password,
		Port:     port,
		Database: database,
	}
}

func init() {
	RegisterPrerequisiteGatherer("env-vars", &EnvVarRsyncPrerequisite{})
}
