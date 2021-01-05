package prerequisite

import (
	"bytes"
	"log"
	"os/exec"
	"strings"
)

type rsyncPrerequisite struct {
	RsyncPath string
}

func (p *rsyncPrerequisite) initialise() error {
	return nil
}

func (p *rsyncPrerequisite) GetName() string {
	return "rsync_path"
}

func (p *rsyncPrerequisite) GetValue() bool {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.Command("bash", "-c", "which rsync || which /tmp/*rsync*")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		log.Print(err)
	}

	p.RsyncPath = strings.TrimSuffix(stdout.String(), "\n")
	log.Println("Found rsync path: " + p.RsyncPath)

	return true
}

func (p *rsyncPrerequisite) GatherValue() ([]GatheredPrerequisite, error) {
	return []GatheredPrerequisite{
		{
			Name:   p.GetName(),
			Value:  p.RsyncPath,
			Status: p.Status(),
		},
	}, nil
}

func (p *rsyncPrerequisite) Status() int {
	if p.RsyncPath != "" {
		return 1
	}
	return 0
}

func init() {
	RegisterConfigPrerequisite("rsync", &rsyncPrerequisite{})
}
