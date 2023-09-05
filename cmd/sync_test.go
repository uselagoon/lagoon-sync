package cmd

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
				if args.SshOptions.Port != "32222" {
					return errors.New(fmt.Sprintf("Expecting ssh port 32222 - found: %v", args.SshOptions.Port))
				}

				if args.SshOptions.Host != "ssh.lagoon.amazeeio.cloud" {
					return errors.New(fmt.Sprintf("Expecting ssh host ssh.lagoon.amazeeio.cloud - found: %v", args.SshOptions.Host))
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
				if args.SourceEnvironment.ProjectName != "lagoon-sync" {
					return errors.New(fmt.Sprintf("Expecting project name 'lagoon-sync' - found: %v", args.SourceEnvironment.ProjectName))
				}

				if args.SshOptions.Port != "777" {
					return errors.New(fmt.Sprintf("Expecting ssh port 777 - found: %v", args.SshOptions.Port))
				}

				if args.SshOptions.Host != "example.ssh.lagoon.amazeeio.cloud" {
					return errors.New(fmt.Sprintf("Expecting ssh host ssh.lagoon.amazeeio.cloud - found: %v", args.SshOptions.Host))
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
			SkipAPI = true

			viper.SetConfigFile(cfgFile)
			viper.AutomaticEnv()

			syncCommandRun(tt.args.cmd, tt.args.args)
		})
	}
}

func TestSetSSHOptions(t *testing.T) {
	type args struct {
		Project     string
		Source      synchers.Environment
		Target      synchers.Environment
		Type        string
		Config      synchers.SyncherConfigRoot
		EnableDebug bool
	}
	var tests = []struct {
		name string
		args args
		want synchers.SSHOptions
	}{
		{
			name: "Tests config overrides",
			args: args{
				Project: "high-cotton",
				Source:  synchers.Environment{EnvironmentName: "main"},
				Target:  synchers.Environment{EnvironmentName: "dev"},
				Type:    "mariadb",
				Config: synchers.SyncherConfigRoot{
					LagoonSync: map[string]interface{}{
						"ssh": map[string]interface{}{
							"host":       "main.lagoon.example.com",
							"port":       "111",
							"privateKey": "~/.ssh/path/to/key",
							"verbose":    true,
							"rsyncArgs":  "-a",
						},
					},
					Api: "https://api.lagoon.amazeeio.cloud/graphql",
				},
				EnableDebug: false,
			},
			want: synchers.SSHOptions{
				Host:       "main.lagoon.example.com",
				Port:       "111",
				Verbose:    true,
				PrivateKey: "~/.ssh/path/to/key",
				RsyncArgs:  "-a",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Sync{
				Source:      tt.args.Source,
				Target:      tt.args.Target,
				Type:        tt.args.Type,
				EnableDebug: tt.args.EnableDebug,
				Config:      tt.args.Config,
			}

			SkipAPI = true
			noCliInteraction = true

			if got := s.GetSSHOptions(tt.args.Project, tt.args.Source, tt.args.Config); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetSSHOptions() = %v, want %v", got, tt.want)
			}
		})
	}
}
