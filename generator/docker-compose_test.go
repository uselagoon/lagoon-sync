package generator

import (
	"testing"
)

func TestProcessServicesFromCompose(t *testing.T) {
	type args struct {
		ComposeFile string
	}
	tests := []struct {
		name    string
		args    args
		want    []LagoonServiceDefinition
		wantErr bool
	}{
		{
			name: "Read service defs",
			args: args{ComposeFile: "./test-assets/drupal-docker-compose.yml"},
			want: []LagoonServiceDefinition{
				{
					ServiceName: "cli",
					ServiceType: "cli-persistent",
				},
				{
					ServiceName: "nginx",
					ServiceType: "nginx-php-persistent",
				},
				{
					ServiceName: "php",
					ServiceType: "nginx-php-persistent",
				},
				{
					ServiceName: "mariadb",
					ServiceType: "mariadb",
				},
			},
			wantErr: false,
		},
		{
			name: "Broken file - should pass because we're using old spec",
			args: args{ComposeFile: "./test-assets/docker-compose-broken.yml"},
			want: []LagoonServiceDefinition{
				{
					ServiceName: "mariadb",
					ServiceType: "mariadb",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project, err := LoadComposeFile(tt.args.ComposeFile)
			if err != nil && tt.wantErr == false {
				t.Errorf("Unexpected error loading file: %v \n", tt.args.ComposeFile)
				return
			}
			if err != nil && tt.wantErr == true {
				return
			}
			services := ProcessServicesFromCompose(project)
			for _, v := range tt.want {
				if !testDockerComposeServiceHasService(services, v) {
					t.Errorf("Could not find service %v in file", v.ServiceName)
				}
			}
		})
	}
}

func testDockerComposeServiceHasService(serviceDefinitions []LagoonServiceDefinition, serviceDef LagoonServiceDefinition) bool {
	// here we match the incoming services to the name
	for _, v := range serviceDefinitions {
		if v.ServiceName == serviceDef.ServiceName && v.ServiceType == serviceDef.ServiceType {
			return true
		}
	}
	return false
}
