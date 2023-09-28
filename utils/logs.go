package utils

import (
	"encoding/json"
	"os"
	"reflect"

	"github.com/spf13/viper"
	"github.com/withmandala/go-log"
)

var colour bool

func SetColour(c bool) {
	colour = c
}

func LogProcessStep(message string, output interface{}) {
	logger := log.New(os.Stdout)
	if colour {
		logger.WithColor()
	} else {
		logger.WithoutColor()
	}
	if output == nil {
		logger.Info(message)
	} else if output != nil {
		logger.Info(message, output)
	}
}

func LogExecutionStep(message string, output interface{}) {
	logger := log.New(os.Stdout)
	if colour {
		logger.WithColor()
	} else {
		logger.WithoutColor()
	}
	if output == nil {
		logger.Info(message)
	} else if output != nil {
		logger.Info(message, output)
	}
}

func LogDebugInfo(message string, output interface{}) {
	logger := log.New(os.Stdout).WithDebug()
	if colour {
		logger.WithColor()
	} else {
		logger.WithoutColor()
	}
	if debug := viper.Get("show-debug"); debug == true {
		if output == nil {
			logger.Debug(message)
		} else if output != nil {
			if reflect.TypeOf(output).String() == "string" {
				logger.Debug(message, output)
			}
			if reflect.TypeOf(output).String() != "string" {
				s, _ := json.MarshalIndent(output, "", "  ")
				logger.Debug(message, string(s))
			}
		}
	}
}

func LogError(message string, output interface{}) {
	logger := log.New(os.Stdout)
	if colour {
		logger.WithColor()
	} else {
		logger.WithoutColor()
	}
	if output == nil {
		logger.Error(message)
	} else if output != nil {
		logger.Error(message, output)
	}
}

func LogFatalError(message string, output interface{}) {
	logger := log.New(os.Stdout)
	if colour {
		logger.WithColor()
	} else {
		logger.WithoutColor()
	}
	if output == nil {
		logger.Fatal(message)
	} else if output != nil {
		logger.Fatal(message, output)
	}
}

func LogWarning(message string, output interface{}) {
	logger := log.New(os.Stdout)
	if colour {
		logger.WithColor()
	} else {
		logger.WithoutColor()
	}
	if output == nil {
		logger.Warn(message)
	} else if output != nil {
		logger.Warn(message, output)
	}
}
