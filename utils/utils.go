package utils

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Reports whether a file exists.
func FileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func FindLagoonSyncOnEnv() (string, bool) {
	whichLagoonSyncCmd := exec.Command("sh", "-c", "which lagoon-sync || false")
	whichStdout, err := whichLagoonSyncCmd.Output()
	if err != nil {
		execPath, err := os.Executable()
		if err != nil {
			fmt.Println(err)
			return "", false
		}
		return execPath, true
	}

	lagoonPath := strings.TrimSuffix(string(whichStdout), "\n")
	if lagoonPath != "" {
		return lagoonPath, true
	}
	return "", false
}
