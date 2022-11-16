package cmd

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/uselagoon/lagoon-sync/synchers"
	"log"
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
				if args.SourceEnvironment.SSH.Port != "32222" {
					return errors.New(fmt.Sprintf("Expecting ssh port 32222 - found: %v", args.SourceEnvironment.SSH.Port))
				}

				if args.SourceEnvironment.SSH.Host != "ssh.lagoon.amazeeio.cloud" {
					return errors.New(fmt.Sprintf("Expecting ssh host ssh.lagoon.amazeeio.cloud - found: %v", args.SourceEnvironment.SSH.Host))
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
				if args.SourceEnvironment.SSH.Port != "777" {
					return errors.New(fmt.Sprintf("Expecting ssh port 777 - found: %v", args.SourceEnvironment.SSH.Port))
				}

				if args.SourceEnvironment.SSH.Host != "example.ssh.lagoon.amazeeio.cloud" {
					return errors.New(fmt.Sprintf("Expecting ssh host ssh.lagoon.amazeeio.cloud - found: %v", args.SourceEnvironment.SSH.Host))
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
			name: "Tests default SSHOptions",
			args: args{
				Project: "high-cotton",
				Source:  synchers.Environment{EnvironmentName: "main"},
				Target:  synchers.Environment{EnvironmentName: "dev"},
				Type:    "mariadb",
				Config: synchers.SyncherConfigRoot{
					LagoonSync: map[string]interface{}{
						"ssh": map[string]interface{}{},
					},
					LagoonAPI: synchers.LagoonAPI{
						Endpoint: "https://api.lagoon.amazeeio.cloud/graphql",
						SSHKey:   "~/$HOME/.ssh/id_rsa",
						SSHHost:  "ssh.lagoon.amazeeio.cloud",
						SSHPort:  "32222",
					},
				},
				EnableDebug: false,
			},
			want: synchers.SSHOptions{
				Host:      "ssh.lagoon.amazeeio.cloud",
				Port:      "32222",
				RsyncArgs: "--omit-dir-times --no-perms --no-group --no-owner --chmod=ugo=rwX --recursive --compress",
			},
		},
		{
			name: "Tests overrides",
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
					LagoonAPI: synchers.LagoonAPI{
						Endpoint: "https://api.lagoon.amazeeio.cloud/graphql",
						SSHKey:   "~/$HOME/.ssh/id_rsa",
						SSHHost:  "ssh.lagoon.amazeeio.cloud",
						SSHPort:  "32222",
					},
				},
				EnableDebug: false,
			},
			want: synchers.SSHOptions{
				"main.lagoon.example.com",
				"111",
				true,
				"~/.ssh/path/to/key",
				"-a",
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
			}

			log.Println(tt.args)

			if got := s.GetSSHOptions(tt.args.Project, tt.args.Source.EnvironmentName, tt.args.Config); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetSSHOptions() = %v, want %v", got, tt.want)
			}
		})
	}
}
