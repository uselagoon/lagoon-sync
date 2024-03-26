package synchers

import (
	"reflect"
	"testing"
)

var testOptions = map[string]SSHOptions{
	"env1": {
		Host: "env1s.host.com", // Note, we're only really setting the host to differentiate during the test
	},
	"env2": {
		Host: "env2s.host.com",
	},
}

func TestSSHOptionWrapper_getSSHOptionsForEnvironment(t *testing.T) {
	type fields struct {
		ProjectName string
		Options     map[string]SSHOptions
		Default     SSHOptions
	}
	type args struct {
		environmentName string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   SSHOptions
	}{
		{
			name: "Falls back to default",
			fields: fields{
				ProjectName: "test",
				Options:     testOptions,
				Default: SSHOptions{
					Host: "defaulthost",
				},
			},
			want: SSHOptions{
				Host: "defaulthost",
			},
			args: args{environmentName: "shoulddefault"},
		},
		{
			name: "Gets named environment ssh details",
			fields: fields{
				ProjectName: "test",
				Options:     testOptions,
				Default: SSHOptions{
					Host: "defaulthost",
				},
			},
			want: SSHOptions{
				Host: "env1s.host.com",
			},
			args: args{environmentName: "env1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receiver := &SSHOptionWrapper{
				ProjectName: tt.fields.ProjectName,
				Options:     tt.fields.Options,
				Default:     tt.fields.Default,
			}
			if got := receiver.getSSHOptionsForEnvironment(tt.args.environmentName); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getSSHOptionsForEnvironment() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSSHOptionWrapper_addSsshOptionForEnvironment(t *testing.T) {
	type fields struct {
		ProjectName string
		Options     map[string]SSHOptions
		Default     SSHOptions
	}
	type args struct {
		environmentName       string
		environmentSSHOptions SSHOptions
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   SSHOptions
	}{
		{
			name: "Adds a new item to the list",
			fields: fields{
				ProjectName: "test",
				Options:     testOptions,
				Default: SSHOptions{
					Host: "defaulthost",
				},
			},
			want: SSHOptions{
				Host: "newItem.ssh.com",
			},
			args: args{
				environmentSSHOptions: SSHOptions{
					Host: "newItem.ssh.com",
				},
				environmentName: "newItem",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receiver := &SSHOptionWrapper{
				ProjectName: tt.fields.ProjectName,
				Options:     tt.fields.Options,
				Default:     tt.fields.Default,
			}
			receiver.addSsshOptionForEnvironment(tt.args.environmentName, tt.args.environmentSSHOptions)
			if got := receiver.getSSHOptionsForEnvironment(tt.args.environmentName); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getSSHOptionsForEnvironment() = %v, want %v", got, tt.want)
			}
		})
	}
}
