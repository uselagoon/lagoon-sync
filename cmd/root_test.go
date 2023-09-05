package cmd

import "testing"

func Test_processConfigEnv(t *testing.T) {
	type args struct {
		paths                 []string
		DefaultConfigFileName string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "Initial test",
			args: args{
				DefaultConfigFileName: "../test-resources/config-tests/initial-test/intial-test-lagoon.yml",
			},
			want:    "intial-test-lagoon.yml",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := processConfig(tt.args.DefaultConfigFileName)
			if (err != nil) != tt.wantErr {
				t.Errorf("processConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
