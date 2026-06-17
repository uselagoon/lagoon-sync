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
	// GetPrerequisiteCommand will return the command to run on source or target environment to extract information.
	GetPrerequisiteCommand(environment Environment, command string) SyncCommand
	// GetRemoteCommand will return the command to be run on the source system
	GetRemoteCommand(environment Environment) []SyncCommand
	// GetLocalCommand will return the command to be run on the target system
	GetLocalCommand(environment Environment) []SyncCommand
	// GetTransferResource will return the command that executes the transfer
	GetTransferResource(environment Environment) SyncerTransferResource
	// SetTransferResource allows for the overriding of the resource transfer name that's typically generated
	SetTransferResource(transferResourceName string) error
	// GetFilesToCleanup will return a list of files to be deleted after completing the transfer
	GetFilesToCleanup(environment Environment) []string
	// PrepareSyncer does any preparations required on a Syncer before it is used
	PrepareSyncer() (Syncer, error)
	// IsInitialized will tell client code if the syncer is ready to rumble
	IsInitialized() (bool, error)
}

type SyncCommand struct {
	command       string
	substitutions map[string]string
	NoOp          bool // NoOp can be set to true if this command performs no operation (in situations like file transfers)
}

// SyncerTransferResource describes what it is the is produced by the actions of GetRemoteCommand()
type SyncerTransferResource struct {
	Name             string   `yaml:"name,omitempty" json:"name,omitempty"`
	IsDirectory      bool     `yaml:"isDirectory,omitempty" json:"isDirectory,omitempty"`
	ExcludeResources []string `yaml:"excludeResources,omitempty" json:"excludeResources,omitempty"`
	SkipCleanup      bool     `yaml:"skipCleanup,omitempty" json:"skipCleanup,omitempty"`
}

type Environment struct {
	ProjectName     string `yaml:"projectName"`
	EnvironmentName string `yaml:"environmentName"`
	ServiceName     string `yaml:"serviceName"` // This is used to determine which Lagoon service we need to rsync
	RsyncAvailable  bool   `yaml:"rsyncAvailable"`
	RsyncPath       string `yaml:"rsyncPath"`
	RsyncLocalPath  string `yaml:"rsyncLocalPath"`
}

// SyncherConfigRoot is used to unmarshall yaml config details generally
type SyncherConfigRoot struct {
	Api           string                              `yaml:"api,omitempty" json:"api,omitempty"`
	Project       string                              `yaml:"project" json:"project,omitempty"`
	LagoonSync    map[string]interface{}              `yaml:"lagoon-sync" json:"lagoonSync,omitempty"`
	Prerequisites []prerequisite.GatheredPrerequisite `yaml:"prerequisites" json:"prerequisites,omitempty"`
}

type SSHConfig struct {
	SSH SSHOptions `yaml:"ssh,omitempty" json:"SSH"`
}

type SSHOptions struct {
	Host       string `yaml:"host,omitempty" json:"host,omitempty"`
	Port       string `yaml:"port,omitempty" json:"port,omitempty"`
	Verbose    bool   `yaml:"verbose,omitempty" json:"verbose,omitempty"`
	PrivateKey string `yaml:"privateKey,omitempty" json:"privateKey,omitempty"`
	SkipAgent  bool
	RsyncArgs  string `yaml:"rsyncArgs,omitempty" json:"rsyncArgs,omitempty"`
}

func (r Environment) GetOpenshiftProjectName() string {
	return fmt.Sprintf("%s-%s", strings.ToLower(r.ProjectName), strings.ToLower(r.EnvironmentName))
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

	return fmt.Sprintf("ssh%s -tt -o LogLevel=FATAL -o \"UserKnownHostsFile=/dev/null\" -o \"StrictHostKeyChecking=no\" -p %s %s@%s %s '%s'",
		sshOptionsStr.String(), sshOptions.Port, remoteEnvironment.GetOpenshiftProjectName(), sshOptions.Host, serviceArgument, command)
}
