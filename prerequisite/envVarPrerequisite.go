package prerequisite

import (
	"os"
)

type EnvVarSyncPrerequisite struct {
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

func (e *EnvVarSyncPrerequisite) initialise() error {
	return nil
}

func (e *EnvVarSyncPrerequisite) GetName() string {
	return "env-vars"
}

func (e *EnvVarSyncPrerequisite) GetValue() bool {
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

func (e *EnvVarSyncPrerequisite) GatherValue() ([]GatheredPrerequisite, error) {
	return []GatheredPrerequisite{
		{
			Name:   "lagoon_version",
			Value:  e.LagoonVersion,
			Status: 1,
		},
		{
			Name:   "lagoon_project",
			Value:  e.LagoonProject,
			Status: 1,
		},
		{
			Name:   "lagoon_env",
			Value:  e.LagoonEnvironment,
			Status: 1,
		},
		{
			Name:   "mariadb_hostname",
			Value:  e.MariaDb.Hostname,
			Status: 1,
		},
		{
			Name:   "mariadb_username",
			Value:  e.MariaDb.Username,
			Status: 1,
		},
		{
			Name:   "mariadb_password",
			Value:  e.MariaDb.Password,
			Status: 1,
		},
		{
			Name:   "mariadb_port",
			Value:  e.MariaDb.Port,
			Status: 1,
		},
		{
			Name:   "mongodb_hostname",
			Value:  e.MongoDb.Hostname,
			Status: 1,
		},
		{
			Name:   "mongodb_username",
			Value:  e.MongoDb.Username,
			Status: 1,
		},
		{
			Name:   "mongodb_password",
			Value:  e.MongoDb.Password,
			Status: 1,
		},
		{
			Name:   "mongodb_port",
			Value:  e.MongoDb.Port,
			Status: 1,
		},
		{
			Name:   "mongodb_database",
			Value:  e.MongoDb.Database,
			Status: 1,
		},
		{
			Name:   "postgresdb_database",
			Value:  e.PostgresDb.Database,
			Status: 1,
		},
		{
			Name:   "postgresdb_hostname",
			Value:  e.PostgresDb.Hostname,
			Status: 1,
		},
		{
			Name:   "postgresdb_username",
			Value:  e.PostgresDb.Username,
			Status: 1,
		},
		{
			Name:   "postgresdb_password",
			Value:  e.PostgresDb.Password,
			Status: 1,
		},
		{
			Name:   "postgresdb_port",
			Value:  e.PostgresDb.Port,
			Status: 1,
		},
		{
			Name:   "postgresdb_database",
			Value:  e.PostgresDb.Database,
			Status: 1,
		},
	}, nil
}

func (e *EnvVarSyncPrerequisite) Status() int {
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

func init() {
	RegisterConfigPrerequisite("env-vars", &EnvVarSyncPrerequisite{})
}
