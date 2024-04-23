package assets

import (
	_ "embed"
	"strings"
)

//go:embed .version
var Version []byte

//go:embed lagoon.yml
var DefaultConfigData []byte

func GetVersion() string {
	return strings.TrimSuffix(string(Version), "\n")
}

func GetDefaultConfig() ([]byte, error) {
	return DefaultConfigData, nil
}
