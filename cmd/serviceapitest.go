package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/uselagoon/lagoon-sync/utils"
)

var testPackage string
var testRun string
var testVerbose bool

var serviceApiTestCmd = &cobra.Command{
	Use:   "serviceapitest",
	Short: "runs whatever I want it to run",
	Long:  `runs whatever I want it to run`,
	Run:   serviceApiTestRun,
}

func serviceApiTestRun(cmd *cobra.Command, args []string) {
	// let's quickly spin up the api and get this party started - looking at environments

	preRunSetSSHDetailsFromEnvars(cmd, args)

	apiConn := utils.ApiConn{}
	fmt.Printf("%v, %v, %v, %v\n", testAPIEndpoint, testSSHKey, testSSHHost, testSSHPort)
	err := apiConn.Init(testAPIEndpoint, testSSHKey, testSSHHost, testSSHPort)
	if err != nil {
		utils.LogFatalError(fmt.Sprintf("failed to initialize API connection: %w", err), nil)
	}

	// let's try get the longsuffering test1copy's service list
	services, err := apiConn.GetServicesForEnvironment("test6-drupal-example-simple", "test1copy")

	if err != nil {
		utils.LogFatalError(fmt.Sprintf("failed to get services: %w", err), nil)
	}

	for _, service := range services {
		// Pretty-print the service for debugging/visibility
		if b, err := json.MarshalIndent(service, "", "  "); err == nil {
			fmt.Printf("Service:\n%s\n", string(b))
		} else {
			fmt.Printf("Service (error marshaling): %+v\n", service)
		}
	}
}

var testAPIEndpoint, testSSHHost, testSSHPort, testSSHKey string

func init() {
	rootCmd.AddCommand(serviceApiTestCmd)
	serviceApiTestCmd.PersistentFlags().StringVarP(&testAPIEndpoint, "api", "A", "https://api.main.lagoon-core.test6.amazee.io/graphql", "Specify your lagoon api endpoint")
	serviceApiTestCmd.PersistentFlags().StringVarP(&testSSHHost, "ssh-host", "H", "ssh.main.lagoon-core.test6.amazee.io", "Specify your lagoon ssh host, defaults to 'ssh.lagoon.amazeeio.cloud'")
	serviceApiTestCmd.PersistentFlags().StringVarP(&testSSHPort, "ssh-port", "P", "22", "Specify your ssh port, defaults to '22'")
	serviceApiTestCmd.PersistentFlags().StringVarP(&testSSHKey, "ssh-key", "i", "", "Specify path to a specific SSH key to use for authentication")
}
