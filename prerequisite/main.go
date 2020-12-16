package prerequisite

import (
	"log"
)

type ConfigPrerequisite interface {
	initialise() error
	getName() string
	getValue() string
	status() int
}

var configPrerequisiteList []ConfigPrerequisite

func RegisterConfigPrerequisite(name string, config ConfigPrerequisite) {
	log.Print("registering: " + name)
	configPrerequisiteList = append(configPrerequisiteList, config)
}

func GetConfigPrerequisite() []ConfigPrerequisite {
	return configPrerequisiteList
}
