package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/mitchellh/mapstructure"

	"github.com/manifoldco/promptui"
	"github.com/spf13/viper"
	synchers "github.com/uselagoon/lagoon-sync/synchers"
	"github.com/uselagoon/lagoon-sync/utils"
)

// getServiceName will return the name of the service in which we run the commands themselves. This is typically
// the cli pod in a project
// TODO: this needs to be expanded to be dynamic in the future.
func getServiceName(SyncerType string) string {
	if SyncerType == "mongodb" {
		return SyncerType
	}
	return "cli"
}

func getEnvironmentSshDetails(conn utils.ApiConn, projectName string, defaultSshOptions synchers.SSHOptions) (synchers.SSHOptions, map[string]synchers.SSHOptions, error) {
	environments, err := conn.GetProjectEnvironmentDeployTargets(projectName)
	retMap := map[string]synchers.SSHOptions{}

	if err != nil {
		return synchers.SSHOptions{}, retMap, err
	}

	var defaultOptions synchers.SSHOptions
	defaultSet := false

	for _, environment := range *environments {
		retMap[environment.Name] = synchers.SSHOptions{
			Host:       environment.DeployTarget.SSHHost,
			Port:       environment.DeployTarget.SSHPort,
			Verbose:    defaultSshOptions.Verbose,
			PrivateKey: "",
			SkipAgent:  defaultSshOptions.SkipAgent,
			RsyncArgs:  defaultSshOptions.RsyncArgs,
		}

		if environment.EnvironmentType == "production" {
			defaultOptions = retMap[environment.Name]
			defaultSet = true
		}
	}
	if defaultSet == false {
		return synchers.SSHOptions{}, retMap, errors.New("COULD NOT FIND DEFAULT OPTION SET")
	}
	return defaultOptions, retMap, nil
}

func confirmPrompt(message string) (bool, error) {
	prompt := promptui.Prompt{
		Label:     message,
		IsConfirm: true,
	}
	result, err := prompt.Run()
	if result == "y" {
		return true, err
	}
	return false, err
}

// loadConfigRoot loads and unmarshals the lagoon config file if present
func loadConfigRoot() (synchers.SyncherConfigRoot, error) {
	var configRoot synchers.SyncherConfigRoot

	if viper.ConfigFileUsed() == "" {
		return configRoot, nil
	}

	lagoonConfigBytestream, err := LoadLagoonConfig(viper.ConfigFileUsed())
	if err != nil {
		return configRoot, fmt.Errorf("couldn't load lagoon config file: %w", err)
	}

	loadedConfigRoot, err := synchers.UnmarshallLagoonYamlToLagoonSyncStructure(lagoonConfigBytestream)
	if err != nil {
		return configRoot, fmt.Errorf("issue unmarshalling sync configuration from %v: %w", viper.ConfigFileUsed(), err)
	}

	return loadedConfigRoot, nil
}

// resolveProjectName determines the project name from flags, env vars, or config
// Priority: flagValue -> LAGOON_PROJECT env var -> configRoot.Project
func resolveProjectName(flagValue string, configRoot synchers.SyncherConfigRoot) string {
	if flagValue != "" {
		return flagValue
	}

	project, exists := os.LookupEnv("LAGOON_PROJECT")
	if exists {
		return strings.Replace(project, "_", "-", -1)
	}

	if configRoot.Project != "" {
		return configRoot.Project
	}

	return ""
}

// buildEnvironments creates source and target Environment structs
func buildEnvironments(projectName, serviceName, sourceEnvName, targetEnvName string) (synchers.Environment, synchers.Environment) {
	sourceEnvironment := synchers.Environment{
		ProjectName:     projectName,
		EnvironmentName: sourceEnvName,
		ServiceName:     serviceName,
	}

	// Default target to local if not specified
	if targetEnvName == "" {
		targetEnvName = synchers.LOCAL_ENVIRONMENT_NAME
	}

	targetEnvironment := synchers.Environment{
		ProjectName:     projectName,
		EnvironmentName: targetEnvName,
		ServiceName:     serviceName,
	}

	return sourceEnvironment, targetEnvironment
}

// resolveSyncer gets the appropriate syncer from the config, falling back to custom syncer
func resolveSyncer(syncerType string, configRoot synchers.SyncherConfigRoot) (synchers.Syncer, error) {
	lagoonSyncer, err := synchers.GetSyncerForTypeFromConfigRoot(syncerType, configRoot)
	if err != nil {
		// Fall back to custom syncer
		lagoonSyncer, err = synchers.GetCustomSync(configRoot, syncerType)
		if err != nil {
			return nil, fmt.Errorf("could not find syncer for type %s: %w", syncerType, err)
		}
	}
	return lagoonSyncer, nil
}

// buildSSHOptions constructs SSH options from config, env vars, and flags
// Priority for host/port: flag (if not default) -> env var -> config -> flag default
func buildSSHOptions(configRoot synchers.SyncherConfigRoot, flagHost, flagPort, flagKey string, flagVerbose, flagSkipAgent bool, rsyncArgs string) synchers.SSHOptions {
	// Decode SSH config from file if present
	sshConfig := synchers.SSHOptions{}
	if configRoot.LagoonSync["ssh"] != nil {
		mapstructure.Decode(configRoot.LagoonSync["ssh"], &sshConfig)
	}

	// Resolve SSH host
	sshHost := flagHost
	if flagHost == "ssh.lagoon.amazeeio.cloud" { // using default, check for overrides
		envSshHost, exists := os.LookupEnv("LAGOON_CONFIG_SSH_HOST")
		if exists {
			sshHost = envSshHost
		} else if sshConfig.Host != "" {
			sshHost = sshConfig.Host
		}
	}

	// Resolve SSH port
	sshPort := flagPort
	if flagPort == "32222" { // using default, check for overrides
		envSshPort, exists := os.LookupEnv("LAGOON_CONFIG_SSH_PORT")
		if exists {
			sshPort = envSshPort
		} else if sshConfig.Port != "" {
			sshPort = sshConfig.Port
		}
	}

	// Resolve SSH key (config takes priority if flag is empty)
	sshKey := flagKey
	if sshConfig.PrivateKey != "" && flagKey == "" {
		sshKey = sshConfig.PrivateKey
	}

	// Resolve verbose flag (config OR flag)
	sshVerbose := flagVerbose
	if sshConfig.Verbose && !flagVerbose {
		sshVerbose = sshConfig.Verbose
	}

	return synchers.SSHOptions{
		Host:       sshHost,
		PrivateKey: sshKey,
		Port:       sshPort,
		Verbose:    sshVerbose,
		RsyncArgs:  rsyncArgs,
		SkipAgent:  flagSkipAgent,
	}
}

// buildSSHOptionWrapper creates and configures an SSH option wrapper, optionally with SSH portal integration
func buildSSHOptionWrapper(projectName string, baseOptions synchers.SSHOptions, configRoot synchers.SyncherConfigRoot, apiEndpoint string, usePortal bool) (*synchers.SSHOptionWrapper, error) {
	sshOptionWrapper := synchers.NewSshOptionWrapper(projectName, baseOptions)

	if !usePortal {
		return sshOptionWrapper, nil
	}

	// Resolve API endpoint
	apiEndPoint := apiEndpoint
	if apiEndpoint == "https://api.lagoon.amazeeio.cloud/graphql" { // using default, check for overrides
		envApiHost, exists := os.LookupEnv("LAGOON_CONFIG_API_HOST")
		if exists {
			apiEndPoint = envApiHost + "/graphql"
		} else if configRoot.Api != "" {
			apiEndPoint = configRoot.Api
		}
	}

	// Initialize API connection and fetch environment SSH details
	apiConn := utils.ApiConn{}
	err := apiConn.Init(apiEndPoint, baseOptions.PrivateKey, baseOptions.Host, baseOptions.Port)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize API connection: %w", err)
	}

	defaultSshOption, sshopts, err := getEnvironmentSshDetails(apiConn, projectName, baseOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to get environment SSH details: %w", err)
	}

	sshOptionWrapper.SetDefaultSshOptions(defaultSshOption)
	for envName, option := range sshopts {
		sshOptionWrapper.AddSsshOptionForEnvironment(envName, option)
	}

	return sshOptionWrapper, nil
}
