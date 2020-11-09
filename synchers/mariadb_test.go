package synchers

import "testing"

func TestMariadbSync_GetRemoteCommand(t *testing.T) {
	type fields struct {
		DbHostname      string
		DbUsername      string
		DbPassword      string
		DbPort          string
		DbDatabase      string
		OutputDirectory string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name:   "basic DB test",
			fields: fields{DbDatabase: "database", DbHostname: "hostname", DbPort: "port", DbPassword: "password", DbUsername: "username", OutputDirectory: "outputdirectory"},
			want:   "mysqldump -hhostname -uusername -ppassword -Pport database > outputdirectory",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := MariadbSyncRoot{
				Config: BaseMariaDbSync{
					DbHostname:      tt.fields.DbHostname,
					DbUsername:      tt.fields.DbUsername,
					DbPassword:      tt.fields.DbPassword,
					DbPort:          tt.fields.DbPort,
					DbDatabase:      tt.fields.DbDatabase,
					OutputDirectory: tt.fields.OutputDirectory,
				},
			}
			if got := m.GetRemoteCommand(); got != tt.want {
				t.Errorf("GetRemoteCommand() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMariadbSync_GetLocalCommand(t *testing.T) {
	type fields struct {
		DbHostname      string
		DbUsername      string
		DbPassword      string
		DbPort          string
		DbDatabase      string
		OutputDirectory string
		LocalOverrides  BaseMariaDbSync
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name:   "basic DB test",
			fields: fields{DbDatabase: "database", DbHostname: "hostname", DbPort: "port", DbPassword: "password", DbUsername: "username", OutputDirectory: "outputdirectory"},
			want:   "mysql -hhostname -uusername -ppassword -Pport database < outputdirectory",
		},
		{
			name: "Import with Overrides",
			fields: fields{DbDatabase: "database", DbHostname: "hostname", DbPort: "port", DbPassword: "password", DbUsername: "username", OutputDirectory: "outputdirectory", LocalOverrides: BaseMariaDbSync{
				DbDatabase: "localdatabase",
			}},
			want: "mysql -hhostname -uusername -ppassword -Pport localdatabase < outputdirectory",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := MariadbSyncRoot{
				Config: BaseMariaDbSync{
					DbHostname:      tt.fields.DbHostname,
					DbUsername:      tt.fields.DbUsername,
					DbPassword:      tt.fields.DbPassword,
					DbPort:          tt.fields.DbPort,
					DbDatabase:      tt.fields.DbDatabase,
					OutputDirectory: tt.fields.OutputDirectory,
				},
				LocalOverrides: MariadbSyncLocal{
					Config: tt.fields.LocalOverrides,
				},
			}
		if got := m.GetLocalCommand(); got != tt.want {
			t.Errorf("GetLocalCommand() = %v, want %v", got, tt.want)
		}
	})
}
}

//func TestMariadbSync_GetOutputDirectory(t *testing.T) {
//	type fields struct {
//		DbHostname string
//		DbUsername string
//		DbPassword string
//		DbPort     string
//		DbDatabase string
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		want   string
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			m := MariadbSync{
//				DbHostname: tt.fields.DbHostname,
//				DbUsername: tt.fields.DbUsername,
//				DbPassword: tt.fields.DbPassword,
//				DbPort:     tt.fields.DbPort,
//				DbDatabase: tt.fields.DbDatabase,
//			}
//			if got := m.GetOutputDirectory(); got != tt.want {
//				t.Errorf("GetOutputDirectory() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
