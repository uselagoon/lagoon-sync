package synchers

import "testing"

func TestSyncCommand_GetCommand(t *testing.T) {
	type fields struct {
		command       string
		substitutions map[string]interface{}
		NoOp          bool
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		{
			name: "Tests NoOp error",
			fields: fields{
				command:       "",
				substitutions: nil,
				NoOp:          true,
			},
			wantErr: true,
		},
		{
			name: "No Substitutions",
			fields: fields{
				command:       "No Substitutions",
				NoOp:          false,
				substitutions: nil,
			},
			want:    "No Substitutions",
			wantErr: false,
		},
		{
			name: "Simple single substitution",
			fields: fields{
				command: "{{ .thesubstitute }}",
				substitutions: map[string]interface{}{
					"thesubstitute": "the value",
				},
				NoOp: false,
			},
			want:    "the value",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := SyncCommand{
				command:       tt.fields.command,
				substitutions: tt.fields.substitutions,
				NoOp:          tt.fields.NoOp,
			}
			got, err := c.GetCommand()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetCommand() got = %v, want %v", got, tt.want)
			}
		})
	}
}
