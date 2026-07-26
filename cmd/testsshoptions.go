package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/uselagoon/lagoon-sync/utils"
)

var testsshoptions = &cobra.Command{
	Use:   "testssh",
	Short: "tests ssh portal",
	Long:  "tests ssh portal",
	Run:   testsshoptionsCommandRun,
}

func testsshoptionsCommandRun(cmd *cobra.Command, args []string) {

	// SyncerType can be one of two things
	// 1. a direct reference to a syncer - i.e. mariadb, postgres, files
	// 2. a reference to an alias in the configuration file (.lagoon.yml/.lagoon-sync.yml)
	// 3. a reference to a custom syncer, also defined in the config file.

	// Load configuration
	configRoot, err := loadConfigRoot()
	if err != nil {
		utils.LogFatalError(fmt.Sprintf("Failed to load configuration: %v", err), nil)
	}
	if viper.ConfigFileUsed() == "" {
		utils.LogWarning("No configuration has been given/found for syncer: ", SyncerType)
	}

	// Resolve project name from multiple sources
	ProjectName = resolveProjectName(ProjectName, configRoot)

	// Build source and target environments
	sourceEnvironment, targetEnvironment := buildEnvironments(ProjectName, ServiceName, sourceEnvironmentName, targetEnvironmentName)

	_ = sourceEnvironment
	_ = targetEnvironment

	if ProjectName == "" {
		utils.LogFatalError("No Project name given", nil)
	}

	// fmt.Print(configRoot)
	fmt.Print("aaa\n")
	// Build SSH options from config, env vars, and flags

	

	sshOptions := buildSSHOptions(configRoot, SSHHost, SSHPort, SSHKey, SSHVerbose, SSHSkipAgent, RsyncArguments)
	fmt.Print("bbb\n")
	fmt.Print(sshOptions)

	// Build SSH option wrapper with optional SSH portal integration
	sshOptionWrapper, err := buildSSHOptionWrapper(ProjectName, sshOptions, configRoot, APIEndpoint, useSshPortal)
	if err != nil {
		utils.LogFatalError(fmt.Sprintf("Failed to configure SSH options: %v", err), nil)
	}
	fmt.Print(sshOptionWrapper.GetSSHOptionsForEnvironment("qa"))
}

func init() {
	rootCmd.AddCommand(testsshoptions)
	testsshoptions.PersistentFlags().StringVarP(&ProjectName, "project-name", "p", "", "The Lagoon project name of the remote system")
	testsshoptions.PersistentFlags().StringVarP(&sourceEnvironmentName, "source-environment-name", "e", "", "The Lagoon environment name of the source system")
	testsshoptions.MarkPersistentFlagRequired("source-environment-name")
	testsshoptions.PersistentFlags().StringVarP(&targetEnvironmentName, "target-environment-name", "t", "", "The target environment name (defaults to local)")
	testsshoptions.PersistentFlags().StringVarP(&ServiceName, "service-name", "s", "", "The service name (default is 'cli'")
	testsshoptions.MarkPersistentFlagRequired("remote-environment-name")
	testsshoptions.PersistentFlags().StringVarP(&SSHHost, "ssh-host", "H", "ssh.lagoon.amazeeio.cloud", "Specify your lagoon ssh host, defaults to 'ssh.lagoon.amazeeio.cloud'")
	testsshoptions.PersistentFlags().StringVarP(&SSHPort, "ssh-port", "P", "32222", "Specify your ssh port, defaults to '32222'")
	testsshoptions.PersistentFlags().StringVarP(&SSHKey, "ssh-key", "i", "", "Specify path to a specific SSH key to use for authentication")
	testsshoptions.PersistentFlags().BoolVar(&SSHSkipAgent, "ssh-skip-agent", false, "Do not attempt to use an ssh-agent for key management")
	testsshoptions.PersistentFlags().BoolVar(&SSHVerbose, "verbose", false, "Run ssh commands in verbose (useful for debugging)")
	testsshoptions.PersistentFlags().BoolVarP(&noCliInteraction, "no-interaction", "y", false, "Disallow interaction")
	testsshoptions.PersistentFlags().StringVarP(&RsyncArguments, "rsync-args", "r", "--omit-dir-times --no-perms --no-group --no-owner --chmod=ugo=rwX --recursive --compress", "Pass through arguments to change the behaviour of rsync")
	testsshoptions.PersistentFlags().StringVarP(&APIEndpoint, "api", "A", "https://api.lagoon.amazeeio.cloud/graphql", "Specify your lagoon api endpoint - required for ssh-portal integration")
	testsshoptions.PersistentFlags().BoolVar(&useSshPortal, "use-ssh-portal", false, "This will use the SSH Portal rather than the (soon to be removed) SSH Service on Lagoon core. Will become default in a future release.")

}
