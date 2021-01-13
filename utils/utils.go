package utils

import (
	"log"
	"os/exec"
	"strings"
)

func FindLagoonSyncOnEnv() (string, bool) {
	cmd := exec.Command("sh", "-c", "which ./lagoon-sync || which /tmp/lagoon-sync* || which lagoon-sync || true")
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
