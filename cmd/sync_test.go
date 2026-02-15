package cmd

import (
	"errors"
	"fmt"
	"testing"

	"github.com/spf13/cobra"
	"github.com/uselagoon/lagoon-sync/synchers"
)

func Test_syncCommandRun(t *testing.T) {
	type args struct {
		cmd  *cobra.Command
		args []string
	}
	tests := []struct {
		name           string
		lagoonYmlFile  string
		args           args
		runSyncProcess synchers.RunSyncProcessFunctionType //This will be the thing that drives the actual test
		wantsError     bool
	}{
		{
			name:          "Tests defaults",
			lagoonYmlFile: "../test-resources/sync-test/tests-defaults/.lagoon.yml",
			args: args{
				cmd: nil,
				args: []string{
					"mariadb",
				},
			},
			runSyncProcess: func(args synchers.RunSyncProcessFunctionTypeArguments) error {
				sshOptions := args.SshOptionWrapper.Default
				if sshOptions.Port != "32222" {
					return errors.New(fmt.Sprintf("Expecting ssh port 32222 - found: %v", sshOptions.Port))
				}

				if sshOptions.Host != "ssh.lagoon.amazeeio.cloud" {
					return errors.New(fmt.Sprintf("Expecting ssh host ssh.lagoon.amazeeio.cloud - found: %v", sshOptions.Host))
				}

				return nil
			},
			wantsError: false,
		},
		{
			name:          "Tests Lagoon yaml",
			lagoonYmlFile: "../test-resources/sync-test/tests-lagoon-yml/.lagoon.yml",
			args: args{
				cmd: nil,
				args: []string{
					"mariadb",
				},
			},
			runSyncProcess: func(args synchers.RunSyncProcessFunctionTypeArguments) error {
				sshOptions := args.SshOptionWrapper.Default
				if sshOptions.Port != "777" {
					return errors.New(fmt.Sprintf("Expecting ssh port 777 - found: %v", sshOptions.Port))
				}

				if sshOptions.Host != "example.ssh.lagoon.amazeeio.cloud" {
					return errors.New(fmt.Sprintf("Expecting ssh host ssh.lagoon.amazeeio.cloud - found: %v", sshOptions.Host))
				}

				return nil
			},
			wantsError: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runSyncProcess = tt.runSyncProcess
			cfgFile = tt.lagoonYmlFile
			noCliInteraction = true
			// Initialize viper with the config file
			processConfig(cfgFile)
			syncCommandRun(tt.args.cmd, tt.args.args)
		})
	}
}
