package prerequisite

import "log"

type GatheredPrerequisite struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Status int    `json:"status"`
}

type ConfigPrerequisite interface {
	initialise() error
	GetName() string
	GetValue() bool
	GatherValue() ([]GatheredPrerequisite, error)
	Status() int
}

var configPrerequisiteList []ConfigPrerequisite

func RegisterConfigPrerequisite(name string, config ConfigPrerequisite) {
	log.Println("Registering: " + name)

	configPrerequisiteList = append(configPrerequisiteList, config)
}

func GetConfigPrerequisite() []ConfigPrerequisite {
	return configPrerequisiteList
}
