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
	cmd := exec.Command("sh", "-c", "which lagoon-sync || find . -name lagoon-sync")
	stdout, _ := cmd.Output()

	if string(stdout) == "" {
		fmt.Errorf("lagoon-sync does not exist on environment")
	}

	lagoonPath := strings.TrimSuffix(string(stdout), "\n")
	if lagoonPath != "" {
		return lagoonPath, true
	}
	return "", false
}
