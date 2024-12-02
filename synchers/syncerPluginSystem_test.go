package synchers

import "testing"

func TestResolveSyncerIdFromAlias(t *testing.T) {
	type args struct {
		syncerId string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "No syncer registered",
			args: args{
				syncerId: "doesntexist",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "No alias but registered",
			args: args{
				syncerId: "mongodb",
			},
			want:    "mongodb",
			wantErr: false,
		},
		{
			name: "Mysql mariadb alias",
			args: args{
				syncerId: "mysql",
			},
			want:    "mariadb",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveSyncerIdFromAlias(tt.args.syncerId)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResolveSyncerIdFromAlias() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ResolveSyncerIdFromAlias() got = %v, want %v", got, tt.want)
			}
		})
	}
}
