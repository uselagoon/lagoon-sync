package prerequisite

import (
	"os"
)

type EnvVaRsyncPrerequisite struct {
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
	Hostname string
	Username string
	Password string
	Port     string
	Database string
}

func (e *EnvVaRsyncPrerequisite) GetName() string {
	return "env-vars"
}

func (e *EnvVaRsyncPrerequisite) GetValue() bool {
	var lagoonVersion = os.Getenv("LAGOON_VERSION")
	if lagoonVersion == "" {
		lagoonVersion = "UNSET"
	}
	e.LagoonVersion = lagoonVersion

	var lagoonProject = os.Getenv("LAGOON_PROJECT")
	if lagoonProject == "" {
		lagoonProject = os.Getenv("LAGOON_SAFE_PROJECT")
	}
	if lagoonProject == "" {
		lagoonProject = "UNSET"
	}
	e.LagoonProject = lagoonProject

	var lagoonEnvironment = os.Getenv("LAGOON_GIT_SAFE_BRANCH")
	if lagoonEnvironment == "" {
		lagoonEnvironment = "UNSET"
	}
	e.LagoonEnvironment = lagoonEnvironment

	e.MariaDb = getMariaDbEnvVars()
	e.MongoDb = getMongoDbEnvVars()
	e.PostgresDb = getPostgresDbEnvVars()

	return true
}

func (e *EnvVaRsyncPrerequisite) GatherPrerequisites() ([]GatheredPrerequisite, error) {
	return []GatheredPrerequisite{
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
		{
			Name:   "mariadb_hostname",
			Value:  e.MariaDb.Hostname,
			Status: getStatusFromString(e.MariaDb.Hostname),
		},
		{
			Name:   "mariadb_username",
			Value:  e.MariaDb.Username,
			Status: getStatusFromString(e.MariaDb.Username),
		},
		{
			Name:   "mariadb_password",
			Value:  e.MariaDb.Password,
			Status: getStatusFromString(e.MariaDb.Password),
		},
		{
			Name:   "mariadb_port",
			Value:  e.MariaDb.Port,
			Status: getStatusFromString(e.MariaDb.Port),
		},
		{
			Name:   "mariadb_database",
			Value:  e.MariaDb.Database,
			Status: getStatusFromString(e.MariaDb.Database),
		},
		{
			Name:   "mongodb_hostname",
			Value:  e.MongoDb.Hostname,
			Status: getStatusFromString(e.MongoDb.Hostname),
		},
		{
			Name:   "mongodb_username",
			Value:  e.MongoDb.Username,
			Status: getStatusFromString(e.MongoDb.Username),
		},
		{
			Name:   "mongodb_password",
			Value:  e.MongoDb.Password,
			Status: getStatusFromString(e.MongoDb.Password),
		},
		{
			Name:   "mongodb_port",
			Value:  e.MongoDb.Port,
			Status: getStatusFromString(e.MariaDb.Port),
		},
		{
			Name:   "mongodb_database",
			Value:  e.MongoDb.Database,
			Status: getStatusFromString(e.MariaDb.Database),
		},
		{
			Name:   "postgresdb_hostname",
			Value:  e.PostgresDb.Hostname,
			Status: getStatusFromString(e.PostgresDb.Hostname),
		},
		{
			Name:   "postgresdb_username",
			Value:  e.PostgresDb.Username,
			Status: getStatusFromString(e.PostgresDb.Username),
		},
		{
			Name:   "postgresdb_password",
			Value:  e.PostgresDb.Password,
			Status: getStatusFromString(e.PostgresDb.Password),
		},
		{
			Name:   "postgresdb_port",
			Value:  e.PostgresDb.Port,
			Status: getStatusFromString(e.PostgresDb.Port),
		},
		{
			Name:   "postgresdb_database",
			Value:  e.PostgresDb.Database,
			Status: getStatusFromString(e.PostgresDb.Database),
		},
	}, nil
}

func (e *EnvVaRsyncPrerequisite) Status() int {
	return 0
}

func getMariaDbEnvVars() DbEnvVars {
	var hostname = os.Getenv("MARIADB_HOSTNAME")
	if hostname == "" {
		hostname = "UNSET"
	}

	var username = os.Getenv("MARIADB_USERNAME")
	if username == "" {
		username = "UNSET"
	}

	var password = os.Getenv("MARIADB_PASSWORD")
	if password == "" {
		password = "UNSET"
	}

	var port = os.Getenv("MARIADB_PORT")
	if port == "" {
		port = "UNSET"
	}

	var database = os.Getenv("MARIADB_DATABASE")
	if database == "" {
		database = "UNSET"
	}

	return DbEnvVars{
		Hostname: hostname,
		Username: username,
		Password: password,
		Port:     port,
		Database: database,
	}
}

func getMongoDbEnvVars() DbEnvVars {
	var hostname = os.Getenv("HOSTNAME")
	if hostname == "" {
		hostname = "UNSET"
	}

	var username = os.Getenv("MONGODB_USERNAME")
	if username == "" {
		username = "UNSET"
	}

	var password = os.Getenv("MONGODB_PASSWORD")
	if password == "" {
		password = "UNSET"
	}

	var port = os.Getenv("MONGODB_SERVICE_PORT")
	if port == "" {
		port = "UNSET"
	}

	var database = os.Getenv("MONGODB_DATABASE")
	if database == "" {
		database = "local"
	}

	return DbEnvVars{
		Hostname: hostname,
		Username: username,
		Password: password,
		Port:     port,
		Database: database,
	}
}

func getPostgresDbEnvVars() DbEnvVars {
	var hostname = os.Getenv("POSTGRES_HOST")
	if hostname == "" {
		hostname = "UNSET"
	}

	var username = os.Getenv("POSTGRES_USERNAME")
	if username == "" {
		username = "UNSET"
	}

	var password = os.Getenv("POSTGRES_PASSWORD")
	if password == "" {
		password = "UNSET"
	}

	var port = os.Getenv("POSTGRES_PORT")
	if port == "" {
		port = "5432"
	}

	var database = os.Getenv("POSTGRES_DATABASE")
	if database == "" {
		database = "UNSET"
	}

	return DbEnvVars{
		Hostname: hostname,
		Username: username,
		Password: password,
		Port:     port,
		Database: database,
	}
}

// func (e *EnvVaRsyncPrerequisite) HandlesPrerequisite(name string) bool {
// 	return false
// }

func init() {
	RegisterPrerequisiteGatherer("env-vars", &EnvVaRsyncPrerequisite{})
}
