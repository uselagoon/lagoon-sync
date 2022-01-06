package assets

import (
	_ "embed"
	"strings"
)

//go:embed .version
var Version []byte

//go:embed rsync
var RsyncLinuxBinBytes []byte

func GetVersion() string {
	return strings.TrimSuffix(string(Version), "\n")
}

func RsyncBin() []byte {
	return RsyncLinuxBinBytes
}