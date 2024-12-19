package synchers

import (
	"reflect"
	"testing"
)

func TestGetSyncerForTypeFromConfigRoot(t *testing.T) {
	type args struct {
		syncerId string
		root     SyncherConfigRoot
	}
	type syncerDef struct {
		Type string
	}
	tests := []struct {
		name           string
		args           args
		wantSyncerType reflect.Type
		wantErr        bool
	}{
		{
			name: "Basic loading of mariadb",
			args: args{
				syncerId: "mariadb",
				root:     SyncherConfigRoot{},
			},
			wantSyncerType: reflect.TypeOf(MariadbSyncPlugin{}),
			wantErr:        false,
		},
		{
			name: "Basic loading of aliased filesystem",
			args: args{
				syncerId: "logs",
				root: SyncherConfigRoot{
					Api:     "",
					Project: "",
					LagoonSync: map[string]interface{}{
						"logs": syncerDef{Type: FilesSyncPlugin{}.GetPluginId()},
					},
					Prerequisites: nil,
				},
			},
			wantSyncerType: reflect.TypeOf(FilesSyncPlugin{}),
			wantErr:        false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetSyncerForTypeFromConfigRoot(tt.args.syncerId, tt.args.root)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSyncerForTypeFromConfigRoot() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if reflect.TypeOf(got) == tt.wantSyncerType {
				t.Errorf("GetSyncerForTypeFromConfigRoot() got = %v, want %v", got, tt.wantSyncerType)
			}
		})
	}
}
