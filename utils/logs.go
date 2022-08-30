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
		fmt.Printf("%s\n", message)
	} else if output != nil {
		fmt.Printf("%s: %s\n", message, output)
	}
}

func LogExecutionStep(message string, output interface{}) {
	if output == nil {
		log.Printf("%s\n", message)
	} else if output != nil {
		log.Printf("%s: %s\n", message, output)
	}
}

func LogDebugInfo(message string, output interface{}) {
	if debug := viper.Get("show-debug"); debug == true {
		if output == nil {
			log.Printf("(DEBUG) %s\n", message)
		} else if output != nil {
			if reflect.TypeOf(output).String() == "string" {
				log.Printf("(DEBUG) %s: %s\n", message, output)
			}
			if reflect.TypeOf(output).String() != "string" {
				s, _ := json.MarshalIndent(output, "", "  ")
				log.Printf("(DEBUG) %s:\n %s\n", message, string(s))
			}
		}
	}
}

func LogFatalError(message string, output interface{}) {
	if output == nil {
		log.Fatalf("(ERROR) - %s\n", message)
	} else if output != nil {
		log.Fatalf("(ERROR) - %s: %s\n", message, output)
	}
}

func LogWarning(message string, output interface{}) {
	if output == nil {
		log.Printf("\n-----\nWarning: %s\n-----", message)
	} else if output != nil {
		log.Printf("\n-----\nWarning: %s: %v\n-----", message, output)
	}
}
