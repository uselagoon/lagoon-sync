package utils

import (
	"testing"
)

//func TestLoadComposeFile(t *testing.T) {
//	type args struct {
//		composeFile string
//	}
//	tests := []struct {
//		name    string
//		args    args
//		want    *types.Project
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			got, err := LoadComposeFile(tt.args.composeFile)
//			if (err != nil) != tt.wantErr {
//				t.Errorf("LoadComposeFile() error = %v, wantErr %v", err, tt.wantErr)
//				return
//			}
//			if !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("LoadComposeFile() got = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}

func TestProcessServicesFromCompose(t *testing.T) {
	type args struct {
		ComposeFile string
	}
	tests := []struct {
		name string
		args args
		want []LagoonServiceDefinition
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
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project, _ := LoadComposeFile(tt.args.ComposeFile)
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
