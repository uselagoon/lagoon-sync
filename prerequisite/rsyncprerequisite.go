package prerequisite

import "fmt"

type SyncPrerequisite struct {
	RsyncPath string
}

func (c *SyncPrerequisite) initialise() error {
	return nil
}

func (c *SyncPrerequisite) getName() string {
	return "rsync"
}

func (c *SyncPrerequisite) getValue() string {
	return "/usr/bin/rsync"
}

func (c *SyncPrerequisite) status() int {
	return 0
}

func init() {
	fmt.Print("Sdfsfsd")
	RegisterConfigPrerequisite("rsync-prereq", &SyncPrerequisite{})
}
