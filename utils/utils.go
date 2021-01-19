package utils

import (
	"log"
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
	cmd := exec.Command("sh", "-c", "lagoon_sync=$(which lagoon-sync || false) && $lagoon_sync")
	stdoutStderr, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
		log.Fatal(string(stdoutStderr))
	}

	lagoonPath := strings.TrimSuffix(string(stdoutStderr), "\n")
	if lagoonPath != "" {
		return lagoonPath, true
	}
	return "", false
}
