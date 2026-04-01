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

	apiConn := utils.ApiConn{}
	err := apiConn.Init("https://api.main.lagoon-core.test6.amazee.io/graphql", "/home/bomoko/.ssh/id_rsa", "ssh.main.lagoon-core.test6.amazee.io", "22")
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

func init() {
	rootCmd.AddCommand(serviceApiTestCmd)

}
