package prerequisite

// import (
// 	"log"
// 	"os/exec"
// 	"strings"
// )

// type rsyncPrerequisite struct {
// 	RsyncPath string
// }

// func (p *rsyncPrerequisite) initialise() error {
// 	return nil
// }

// func (p *rsyncPrerequisite) GetName() string {
// 	return "rsync_path"
// }

// func (p *rsyncPrerequisite) GetValue() bool {
// 	cmd := exec.Command("sh", "-c", "which rsync || which /tmp/*rsync* || true")
// 	stdoutStderr, err := cmd.Output()
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	p.RsyncPath = strings.TrimSuffix(string(stdoutStderr), "\n")
// 	//log.Println("Found rsync path: " + p.RsyncPath)

// 	return true
// }

// func (p *rsyncPrerequisite) GatherValue() ([]GatheredPrerequisite, error) {
// 	return []GatheredPrerequisite{
// 		{
// 			Name:   p.GetName(),
// 			Value:  p.RsyncPath,
// 			Status: p.Status(),
// 		},
// 	}, nil
// }

// func (p *rsyncPrerequisite) Status() int {
// 	if p.RsyncPath != "" {
// 		return 1
// 	}
// 	return 0
// }

// func init() {
// 	RegisterConfigPrerequisite("rsync", &rsyncPrerequisite{})
// }
