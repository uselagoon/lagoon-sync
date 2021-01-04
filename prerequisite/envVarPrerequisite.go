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
	}, nil
}

func (e *EnvVarSyncPrerequisite) Status() int {
	return 0
}

func init() {
	RegisterConfigPrerequisite("env-vars", &EnvVarSyncPrerequisite{})
}
