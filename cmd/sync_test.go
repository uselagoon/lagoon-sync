package cmd

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/uselagoon/lagoon-sync/synchers"
	"testing"
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
			runSyncProcess: func(sourceEnvironment synchers.Environment, targetEnvironment synchers.Environment, lagoonSyncer synchers.Syncer, syncerType string, dryRun bool, sshOptions synchers.SSHOptions) error {
				if sshOptions.Port != "3222" {
					return errors.New(fmt.Sprintf("Expecting ssh port 3222 - found: %v", sshOptions.Port))
				}

				if sshOptions.Host != "ssh.lagoon.amazeeio.cloud" {
					return errors.New(fmt.Sprintf("Expecting ssh host ssh.lagoon.amazeeio.cloud - found: %v", sshOptions.Port))
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
			syncCommandRun(tt.args.cmd, tt.args.args)
		})
	}
}
