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
				paths:                 []string{"../test-resources/config-tests/initial-test"},
				DefaultConfigFileName: "intial-test-lagoon.yml",
			},
			want:    "intial-test-lagoon.yml",
			wantErr: false,
		},
		{
			name: "Initial test - Empty path",
			args: args{
				paths:                 []string{},
				DefaultConfigFileName: "../test-resources/config-tests/initial-test/intial-test-lagoon.yml",
			},
			want:    "../test-resources/config-tests/initial-test/intial-test-lagoon.yml",
			wantErr: false,
		},
		{
			name: "Initial test - Multiple paths",
			args: args{
				paths:                 []string{"../test-resources/sync-test/tests-defaults", "../test-resources/sync-test/tests-lagoon-yml", "../test-resources/config-tests/initial-test"},
				DefaultConfigFileName: ".lagoon.yml",
			},
			want:    ".lagoon.yml",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := processConfigEnv(tt.args.paths, tt.args.DefaultConfigFileName)
			if (err != nil) != tt.wantErr {
				t.Errorf("processConfigEnv() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("processConfigEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}
