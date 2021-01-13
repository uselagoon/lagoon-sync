package prerequisite

import (
	"log"
	"os/exec"
	"strings"
)

type GatheredPrerequisite struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Status int    `json:"status"`
}

type PrerequisiteGatherer interface {
	handlesPrerequisite(name string) bool
	GatherPrerequisites() []GatheredPrerequisite
}

var prereqGathererMap []PrerequisiteGatherer

func RegisterGatherer(name string, gatherer PrerequisiteGatherer) {
	prereqGathererMap = append(prereqGathererMap, gatherer)
}

func GetGatherersMap() []PrerequisiteGatherer {
	return prereqGathererMap
}

func getStatusFromString(prereq string) int {
	if prereq != "" {
		return 1
	}
	return 0
}

//

// type PreRequisiteResponse struct {
// 	Version        string `json:"version"`
// 	LagoonSyncPath string `json:"lagoon-sync-path"`
// 	//EnvPrerequisite   []prerequisite.GatheredPrerequisite `json:"env-config"`
// 	//RysncPrerequisite []prerequisite.GatheredPrerequisite `json:"rsync-config"`
// }

// type ConfigPrerequisite interface {
// 	initialise() error
// 	GetName() string
// 	GetValue() bool
// 	GatherValue() ([]GatheredPrerequisite, error)
// 	Status() int
// }

// var configPrerequisiteList []ConfigPrerequisite

// func RegisterConfigPrerequisite(name string, config ConfigPrerequisite) {
// 	//log.Println("Registering: " + name)

// 	configPrerequisiteList = append(configPrerequisiteList, config)
// }

//

func FindLagoonSyncOnEnv() (string, bool) {
	cmd := exec.Command("sh", "-c", "which ./lagoon-sync || which /tmp/lagoon-sync* || which lagoon-sync || true")
	stdoutStderr, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
		log.Fatal(string(stdoutStderr))
	}

	lagoonPath := strings.TrimSuffix(string(stdoutStderr), "\n")
	if lagoonPath != "" {
		return lagoonPath, true
	}
	return "", false
}
