package synchers

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/uselagoon/lagoon-sync/prerequisite"
	"gopkg.in/yaml.v2"
)

const LOCAL_ENVIRONMENT_NAME = "local"

type Syncer interface {
	// GetPrequisiteCommand will return the command to run on source or target environment to extract information.
	GetPrerequisiteCommand(environmnt Environment, command string) SyncCommand
	// GetRemoteCommand will return the command to be run on the source system
	GetRemoteCommand(environment Environment) SyncCommand
	// GetLocalCommand will return the command to be run on the target system
	GetLocalCommand(environment Environment) SyncCommand
	// GetTransferResource will return the command that executes the transfer
	GetTransferResource(environment Environment) SyncerTransferResource
	// PrepareSyncer does any preparations required on a Syncer before it is used
	PrepareSyncer() (Syncer, error)
	// IsInitialized will tell client code if the syncer is ready to rumble
	IsInitialized() (bool, error)
}

type SyncCommand struct {
	command       string
	substitutions map[string]interface{}
	NoOp          bool // NoOp can be set to true if this command performs no operation (in situations like file transfers)
}

// SyncerTransferResource describes what it is the is produced by the actions of GetRemoteCommand()
type SyncerTransferResource struct {
	Name             string
	IsDirectory      bool
	ExcludeResources []string // ExcludeResources is a string list of any resources that aren't to be included in the transfer
	SkipCleanup      bool
}

type Environment struct {
	ProjectName     string
	EnvironmentName string
	ServiceName     string // This is used to determine which Lagoon service we need to rsync
	RsyncAvailable  bool
	RsyncPath       string
	RsyncLocalPath  string
}

type SSHOptions struct {
	Verbose    bool
	PrivateKey string
	RsyncArgs string
}

func (r Environment) GetOpenshiftProjectName() string {
	return fmt.Sprintf("%s-%s", strings.ToLower(r.ProjectName), strings.ToLower(r.EnvironmentName))
}

// SyncherConfigRoot is used to unmarshall yaml config details generally
type SyncherConfigRoot struct {
	Project       string                 `yaml:"project"`
	LagoonSync    map[string]interface{} `yaml:"lagoon-sync"`
	Prerequisites []prerequisite.GatheredPrerequisite
}

// takes interface, marshals back to []byte, then unmarshals to desired struct
// from https://github.com/go-yaml/yaml/issues/13#issuecomment-428952604
func UnmarshalIntoStruct(pluginIn interface{}, pluginOut interface{}) error {
	b, err := yaml.Marshal(pluginIn)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(b, pluginOut)
}

func GenerateRemoteCommand(remoteEnvironment Environment, command string, sshOptions SSHOptions) string {
	var sshOptionsStr bytes.Buffer
	if sshOptions.Verbose {
		sshOptionsStr.WriteString(" -v")
	}

	if sshOptions.PrivateKey != "" {
		sshOptionsStr.WriteString(fmt.Sprintf(" -i %s", sshOptions.PrivateKey))
	}

	serviceArgument := ""
	if remoteEnvironment.ServiceName != "" {
		serviceArgument = fmt.Sprintf("service=%v", remoteEnvironment.ServiceName)
	}

	return fmt.Sprintf("ssh%s -tt -o \"UserKnownHostsFile=/dev/null\" -o \"StrictHostKeyChecking=no\" -p 32222 %s@ssh.lagoon.amazeeio.cloud %s '%s'",
		sshOptionsStr.String(), remoteEnvironment.GetOpenshiftProjectName(), serviceArgument, command)
}
