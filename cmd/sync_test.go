package cmd

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/uselagoon/lagoon-sync/synchers"
	"reflect"
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
			runSyncProcess: func(args synchers.RunSyncProcessFunctionTypeArguments) error {
				if args.SSHOptions.Port != "32222" {
					return errors.New(fmt.Sprintf("Expecting ssh port 32222 - found: %v", args.SSHOptions.Port))
				}

				if args.SSHOptions.Host != "ssh.lagoon.amazeeio.cloud" {
					return errors.New(fmt.Sprintf("Expecting ssh host ssh.lagoon.amazeeio.cloud - found: %v", args.SSHOptions.Host))
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
				if args.SSHOptions.Port != "777" {
					return errors.New(fmt.Sprintf("Expecting ssh port 777 - found: %v", args.SSHOptions.Port))
				}

				if args.SSHOptions.Host != "example.ssh.lagoon.amazeeio.cloud" {
					return errors.New(fmt.Sprintf("Expecting ssh host ssh.lagoon.amazeeio.cloud - found: %v", args.SSHOptions.Host))
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

func TestSetSSHOptions(t *testing.T) {
	type args struct {
		configRoot synchers.SyncherConfigRoot
		args       []string
	}
	var tests = []struct {
		name       string
		configRoot synchers.SyncherConfigRoot
		args       args
		want       synchers.SSHOptions
	}{
		{
			name: "Tests default SSHOptions",
			args: args{
				configRoot: synchers.SyncherConfigRoot{
					LagoonSync: map[string]interface{}{
						"ssh": map[string]interface{}{},
					},
				},
			},
			want: synchers.SSHOptions{
				"ssh.lagoon.amazeeio.cloud",
				"32222",
				false,
				"",
				"--omit-dir-times --no-perms --no-group --no-owner --chmod=ugo=rwX --recursive --compress",
			},
		},
		{
			name: "Tests overrides",
			args: args{
				configRoot: synchers.SyncherConfigRoot{
					LagoonSync: map[string]interface{}{
						"ssh": map[string]interface{}{
							"host":       "override.lagoon.example.com",
							"port":       "111",
							"privateKey": "~/.ssh/path/to/key",
							"verbose":    true,
							"rsyncArgs":  "-a",
						},
					},
				},
			},
			want: synchers.SSHOptions{
				"override.lagoon.example.com",
				"111",
				true,
				"~/.ssh/path/to/key",
				"-a",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SetSSHOptions(tt.args.configRoot); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SetSSHOptions() = %v, want %v", got, tt.want)
			}
		})
	}
}
