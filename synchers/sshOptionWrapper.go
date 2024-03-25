package synchers

// sshOptionWrapper.go contains the logic for the new system for passing ssh portal data

type SSHOptionWrapper struct {
	ProjectName string                // this is primarily used to ensure someone doesn't do something silly - it's an assertion
	Options     map[string]SSHOptions // a map off all named ssh options - environment => ssh config
	Default     SSHOptions            // this will be returned if no explicit match is found in `Options`
}

func NewSshOptionWrapper(projectName string, defaultSshOptions SSHOptions) *SSHOptionWrapper {
	return &SSHOptionWrapper{
		ProjectName: projectName,
		Options:     map[string]SSHOptions{},
		Default:     defaultSshOptions,
	}
}

func (receiver *SSHOptionWrapper) getSSHOptionsForEnvironment(environmentName string) SSHOptions {
	sshOptionsMapValue, ok := receiver.Options[environmentName]
	if ok {
		return sshOptionsMapValue
	}
	return receiver.Default
}

func (receiver *SSHOptionWrapper) addSsshOptionForEnvironment(environmentName string, sshOptions SSHOptions) {
	receiver.Options[environmentName] = sshOptions
}
