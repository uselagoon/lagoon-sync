package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"

	"github.com/spf13/viper"
)

func LogProcessStep(message string, output interface{}) {
	if output == nil {
		fmt.Printf("\x1b[32;1m%s\x1b[0m\n", message)
	} else if output != nil {
		fmt.Printf("\x1b[32;1m%s\x1b[0m: %s\n", message, output)
	}
}

func LogExecutionStep(message string, output interface{}) {
	if output == nil {
		log.Printf("\x1b[36;1m%s\x1b[0m", message)
	} else if output != nil {
		log.Printf("\x1b[36;1m%s\x1b[0m: \n\x1b[30;1m%s\x1b[0m", message, output)
	}
}

func LogDebugInfo(message string, output interface{}) {
	if debug := viper.Get("show-debug"); debug == true {
		if output == nil {
			log.Printf("\x1b[37;1m(DEBUG)\x1b[0m %s", message)
		} else if output != nil {
			if reflect.TypeOf(output).String() == "string" {
				log.Printf("\x1b[37;1m(DEBUG)\x1b[0m %s: %s", message, output)
			}
			if reflect.TypeOf(output).String() != "string" {
				s, _ := json.MarshalIndent(output, "", "  ")
				log.Printf("\x1b[37;1m(DEBUG)\x1b[0m %s:\n %s", message, string(s))
			}
		}
	}
}

func LogFatalError(message string, output interface{}) {
	if output == nil {
		log.Fatalf("\x1b[31m(ERROR)\x1b[0m - %s", message)
	} else if output != nil {
		log.Fatalf("\x1b[31m(ERROR)\x1b[0m - %s: %s", message, output)
	}
}

func LogWarning(message string, output interface{}) {
	if output == nil {
		log.Printf("\n-----\n\x1b[33;1mWarning:\x1b[0m %s\n-----", message)
	} else if output != nil {
		log.Printf("\n-----\n\x1b[33;1mWarning:\x1b[0m %s: %v\n-----", message, output)
	}
}
