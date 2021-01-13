package prerequisite

import (
	"os"
)

type EnvironmentVariablePrerequisiteGatherer struct {
	GatheredPrerequisites []GatheredPrerequisite
}

func (p *EnvironmentVariablePrerequisiteGatherer) handlesPrerequisite(name string) bool {
	if name == "LAGOON_ENVIRONMENT" {
		return true
	}
	return false
}

func (p *EnvironmentVariablePrerequisiteGatherer) GatherPrerequisites() []GatheredPrerequisite {

	var lagoonEnvironment = os.Getenv("LAGOON_GIT_SAFE_BRANCH")

	return []GatheredPrerequisite{
		{Name: "hostname", Value: "mariadb", Status: 1},
		{
			Name:   "LAGOON_ENVIRONMENT",
			Value:  lagoonEnvironment,
			Status: getStatusFromString(lagoonEnvironment),
		},
	}
}

func (p *EnvironmentVariablePrerequisiteGatherer) GetGatherName() string {
	return "env-vars"
}

func init() {
	RegisterGatherer("Env var gatherer", &EnvironmentVariablePrerequisiteGatherer{})
}
