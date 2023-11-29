package utils

import (
	"reflect"
	"testing"
)

func Test_findSSHKeyFiles(t *testing.T) {
	type args struct {
		directory string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "Run on test directory",
			args: args{directory: "../test-resources/shell-tests/test_findSSHKeyFiles"},
			want: []string{
				"../test-resources/shell-tests/test_findSSHKeyFiles/key1",
				"../test-resources/shell-tests/test_findSSHKeyFiles/key2",
				"../test-resources/shell-tests/test_findSSHKeyFiles/key3",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := findSSHKeyFiles(tt.args.directory)
			if (err != nil) != tt.wantErr {
				t.Errorf("findSSHKeyFiles() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("findSSHKeyFiles() got = %v, want %v", got, tt.want)
			}
		})
	}
}
