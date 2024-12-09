package generator

import (
	"github.com/uselagoon/lagoon-sync/synchers"
	"reflect"
	"strings"
	"testing"
)

func TestGenerateMariadbSyncRootFromService(t *testing.T) {
	type args struct {
		definition LagoonServiceDefinition
	}
	tests := []struct {
		name    string
		args    args
		want    synchers.MariadbSyncRoot
		wantErr bool
	}{
		{
			name: "Default case - mariadb service name",
			args: args{definition: LagoonServiceDefinition{
				ServiceName: "mariadb",
				ServiceType: "mariadb",
				Labels:      nil,
			}},
			want: synchers.MariadbSyncRoot{
				Type:        synchers.MariadbSyncPlugin{}.GetPluginId(),
				ServiceName: "mariadb",
				Config: synchers.BaseMariaDbSync{
					DbHostname: "${MARIADB_HOST:-mariadb}",
					DbUsername: "${MARIADB_USERNAME:-drupal}",
					DbPassword: "${MARIADB_PASSWORD:-drupal}",
					DbPort:     "${MARIADB_PORT:-3306}",
					DbDatabase: "${MARIADB_DATABASE:-drupal}",
				},
			},
		},
		{
			name: "Special case - custom service name",
			args: args{definition: LagoonServiceDefinition{
				ServiceName: "db",
				ServiceType: "mariadb",
				Labels:      nil,
			}},
			want: synchers.MariadbSyncRoot{
				Type:        synchers.MariadbSyncPlugin{}.GetPluginId(),
				ServiceName: "db",
				Config: synchers.BaseMariaDbSync{
					DbHostname: "${DB_HOST:-mariadb}",
					DbUsername: "${DB_USERNAME:-drupal}",
					DbPassword: "${DB_PASSWORD:-drupal}",
					DbPort:     "${DB_PORT:-3306}",
					DbDatabase: "${DB_DATABASE:-drupal}",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateMariadbSyncRootFromService(tt.args.definition)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateMariadbSyncRootFromService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GenerateMariadbSyncRootFromService() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateFilesSyncRootFromPersistentService(t *testing.T) {
	type args struct {
		definition LagoonServiceDefinition
	}
	tests := []struct {
		name    string
		args    args
		want    synchers.FilesSyncRoot
		wantErr bool
	}{
		{
			name: "Standard service",
			args: args{definition: LagoonServiceDefinition{
				ServiceName: "cli",
				ServiceType: "cli-persistent",
				Labels: map[string]string{
					"lagoon.persistent": "/app/web/sites/default/files",
				},
			}},
			want: synchers.FilesSyncRoot{
				ServiceName: "cli",
				Type:        synchers.FilesSyncPlugin{}.GetPluginId(),
				Config: synchers.BaseFilesSync{
					SyncPath: "/app/web/sites/default/files",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateFilesSyncRootFromPersistentService(tt.args.definition)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateFilesSyncRootFromPersistentService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GenerateFilesSyncRootFromPersistentService() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildConfigStanzaFromServices(t *testing.T) {
	type args struct {
		services []LagoonServiceDefinition
	}
	tests := []struct {
		name          string
		args          args
		shouldContain []string
		wantErr       bool
	}{
		{
			name: "Single instance",
			shouldContain: []string{
				"db",
				"files1-testmount",
				"files1-nogamount",
			},
			args: args{services: []LagoonServiceDefinition{
				{
					ServiceName: "db",
					ServiceType: "mariadb",
					Labels:      nil,
				},
				{
					ServiceName: "files1",
					ServiceType: "cli-persistent",
					Labels: map[string]string{
						"lagoon.persistent":             "/mainstorage",
						"lagoon.volumes.testmount.path": "/testmount/",
						"lagoon.volumes.nogamount.path": "/testmount2/",
					},
				},
			},
			},
		},
		{
			name: "Postgres instance",
			shouldContain: []string{
				synchers.PostgresSyncPlugin{}.GetPluginId(),
			},
			args: args{services: []LagoonServiceDefinition{
				{
					ServiceName: "postgresdb",
					ServiceType: "postgres",
					Labels:      nil,
				},
			},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildConfigStanzaFromServices(tt.args.services)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildConfigStanzaFromServices() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			for _, v := range tt.shouldContain {
				if !strings.Contains(got, v) {
					t.Errorf("The output string should contain the substring '%v'", v)
				}
			}
		})
	}
}

func TestGenerateFilesSyncRootsFromServiceDefinition(t *testing.T) {
	type args struct {
		definition LagoonServiceDefinition
	}
	tests := []struct {
		name    string
		args    args
		want    []synchers.FilesSyncRoot
		wantErr bool
	}{
		{
			name: "Empty - only standard definition",
			args: args{definition: LagoonServiceDefinition{
				ServiceName: "cli",
				ServiceType: "cli-persistent",
				Labels: map[string]string{
					"lagoon.persistent": "/app/web/sites/default/files",
				},
			}},
			want: []synchers.FilesSyncRoot{},
		},
		{
			name: "Cli with single instance of multiple vol",
			args: args{definition: LagoonServiceDefinition{
				ServiceName: "cli",
				ServiceType: "cli-persistent",
				Labels: map[string]string{
					"lagoon.persistent":        "/app/web/sites/default/files",
					"lagoon.volumes.vol1.path": "/apath/",
				},
			}},
			want: []synchers.FilesSyncRoot{
				{
					Type:        "files",
					ServiceName: "cli-vol1",
					Config: synchers.BaseFilesSync{
						SyncPath: "/apath/",
					},
				},
			},
		},
		{
			name: "Cli with single multiple vols",
			args: args{definition: LagoonServiceDefinition{
				ServiceName: "cli",
				ServiceType: "cli-persistent",
				Labels: map[string]string{
					"lagoon.persistent":        "/app/web/sites/default/files",
					"lagoon.volumes.vol1.path": "/apath/",
					"lagoon.volumes.vol2.path": "/apath2/",
				},
			}},
			want: []synchers.FilesSyncRoot{
				{
					Type:        "files",
					ServiceName: "cli-vol1",
					Config: synchers.BaseFilesSync{
						SyncPath: "/apath/",
					},
				},
				{
					Type:        "files",
					ServiceName: "cli-vol2",
					Config: synchers.BaseFilesSync{
						SyncPath: "/apath2/",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateFilesSyncRootsFromServiceDefinition(tt.args.definition)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateFilesSyncRootsFromServiceDefinition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(tt.want) != len(got) {
				t.Errorf("GenerateFilesSyncRootsFromServiceDefinition() got = %v, want %v", got, tt.want)
			}

			if len(tt.want) != 0 {
				appears := false
				for _, v := range tt.want {
					for _, i := range got {
						if reflect.DeepEqual(v, i) {
							appears = true
						}
					}
				}
				if !appears {
					t.Errorf("GenerateFilesSyncRootsFromServiceDefinition() got = %v, want %v", got, tt.want)
				}
			}
		})
	}
}
