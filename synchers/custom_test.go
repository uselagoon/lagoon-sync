package synchers

import (
	"reflect"
	"testing"
)

func TestCustomSyncPlugin_UnmarshallYaml(t *testing.T) {
	type fields struct {
		isConfigEmpty bool
	}
	type args struct {
		root SyncherConfigRoot
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    Syncer
		wantErr bool
	}{
		{
			name:   "simple unmarshalling",
			fields: fields{isConfigEmpty: false},
			args: args{
				root: SyncherConfigRoot{
					Project: "",
					LagoonSync: map[string]interface{}{
						"custom": CustomSyncRoot{
							TransferResource: "testing",
							Source:           BaseCustomSyncCommands{Commands: []string{"first"}},
							Target:           BaseCustomSyncCommands{Commands: []string{"second"}},
						},
					},
					Prerequisites: nil,
				},
			},
			want: CustomSyncRoot{
				TransferResource: "testing",
				Source:           BaseCustomSyncCommands{Commands: []string{"first"}},
				Target:           BaseCustomSyncCommands{Commands: []string{"second"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := CustomSyncPlugin{
				isConfigEmpty: tt.fields.isConfigEmpty,
			}
			got, err := m.UnmarshallYaml(tt.args.root, "custom")
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshallYaml() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UnmarshallYaml() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetCustomSync(t *testing.T) {
	type args struct {
		configRoot SyncherConfigRoot
		syncerName string
	}
	tests := []struct {
		name    string
		args    args
		want    Syncer
		wantErr bool
	}{
		{
			name: "simple unmarshalling with custom root name",
			//fields: fields{isConfigEmpty: false},
			args: args{
				syncerName: "customroot",
				configRoot: SyncherConfigRoot{
					Project: "",
					LagoonSync: map[string]interface{}{
						"customroot": CustomSyncRoot{
							TransferResource: "testing",
							Source:           BaseCustomSyncCommands{Commands: []string{"first"}},
							Target:           BaseCustomSyncCommands{Commands: []string{"second"}},
						},
					},
					Prerequisites: nil,
				},
			},
			want: CustomSyncRoot{
				TransferResource: "testing",
				Source:           BaseCustomSyncCommands{Commands: []string{"first"}},
				Target:           BaseCustomSyncCommands{Commands: []string{"second"}},
			},
		},
		{
			name: "simple unmarshalling with multiple commands",
			//fields: fields{isConfigEmpty: false},
			args: args{
				syncerName: "customroot",
				configRoot: SyncherConfigRoot{
					Project: "",
					LagoonSync: map[string]interface{}{
						"customroot": CustomSyncRoot{
							TransferResource: "testing",
							Source:           BaseCustomSyncCommands{Commands: []string{"first of one", "second of one"}},
							Target:           BaseCustomSyncCommands{Commands: []string{"first of two", "second of two"}},
						},
					},
					Prerequisites: nil,
				},
			},
			want: CustomSyncRoot{
				TransferResource: "testing",
				Source:           BaseCustomSyncCommands{Commands: []string{"first of one", "second of one"}},
				Target:           BaseCustomSyncCommands{Commands: []string{"first of two", "second of two"}},
			},
		},
		{
			name: "Fails because of empty transfer resource",
			//fields: fields{isConfigEmpty: false},
			wantErr: true,
			args: args{
				syncerName: "customroot",
				configRoot: SyncherConfigRoot{
					Project: "",
					LagoonSync: map[string]interface{}{
						"customroot": CustomSyncRoot{
							TransferResource: "",
							Source:           BaseCustomSyncCommands{Commands: []string{"first of one", "second of one"}},
							Target:           BaseCustomSyncCommands{Commands: []string{"first of two", "second of two"}},
						},
					},
					Prerequisites: nil,
				},
			},
			want: CustomSyncRoot{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			got, errs := GetCustomSync(tt.args.configRoot, tt.args.syncerName)
			if (errs != nil) != tt.wantErr {
				t.Errorf("GetCustomSync() error = %v, wantErr %v", errs, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetCustomSync() got = %v, want %v", got, tt.want)
			}
		})
	}
}
