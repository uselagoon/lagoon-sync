package prerequisite

import "reflect"

type GatheredPrerequisite struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Status int    `json:"status"`
}

type PreRequisiteResponse struct {
	Version           string                 `json:"version"`
	LagoonSyncPath    string                 `json:"lagoon-sync-path"`
	EnvPrerequisite   []GatheredPrerequisite `json:"env-config"`
	RysncPrerequisite []GatheredPrerequisite `json:"rsync-config"`
}

type PrerequisiteGatherer interface {
	GetName() string
	//GetValue() bool
	HandlesPrerequisite(string) bool
	GatherPrerequisites() ([]GatheredPrerequisite, error)
	//Status() int
}

var PrerequisiteGathererList []PrerequisiteGatherer

func RegisterPrerequisiteGatherer(name string, config PrerequisiteGatherer) {
	PrerequisiteGathererList = append(PrerequisiteGathererList, config)
}

func getPrerequisiteGatherers() []PrerequisiteGatherer {
	return PrerequisiteGathererList
}

func (p *PreRequisiteResponse) IsPrerequisiteResponseEmpty() bool {
	return reflect.DeepEqual(&PreRequisiteResponse{}, p)
}

func getStatusFromString(prereq string) int {
	if prereq != "" {
		return 1
	}
	return 0
}
