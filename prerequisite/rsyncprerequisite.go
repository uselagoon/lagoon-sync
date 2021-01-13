package prerequisite

import (
	"log"
	"os/exec"
	"strings"
)

type RsyncPrerequisite struct {
	RsyncPath string
}

func (p *RsyncPrerequisite) GetName() string {
	return "rsync_path"
}

func (p *RsyncPrerequisite) GetValue() bool {
	cmd := exec.Command("sh", "-c", "which rsync || which /tmp/*rsync* || true")
	stdoutStderr, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}

	p.RsyncPath = strings.TrimSuffix(string(stdoutStderr), "\n")

	return true
}

func (p *RsyncPrerequisite) GatherPrerequisites() ([]GatheredPrerequisite, error) {
	return []GatheredPrerequisite{
		{
			Name:   p.GetName(),
			Value:  p.RsyncPath,
			Status: p.Status(),
		},
	}, nil
}

func (p *RsyncPrerequisite) Status() int {
	if p.RsyncPath != "" {
		return 1
	}
	return 0
}

// func (p *RsyncPrerequisite) HandlesPrerequisite(name string) bool {
// 	return false
// }

func init() {
	RegisterPrerequisiteGatherer("rsync", &RsyncPrerequisite{})
}
