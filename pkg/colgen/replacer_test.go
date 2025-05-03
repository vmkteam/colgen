package colgen

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseReplaceRule(t *testing.T) {
	type args struct {
		rule string
	}
	tests := []struct {
		name    string
		args    args
		want    ReplaceRule
		wantErr bool
	}{
		{
			name: "//colgen@NewCall(db)",
			args: args{
				rule: "//colgen@NewCall(db)",
			},
			want: ReplaceRule{
				Find:   "//colgen@NewCall(db)",
				Cmd:    "New",
				Entity: "Call",
				Arg:    "db.Call",
			},
			wantErr: false,
		},
		{
			name: "//colgen@newUserSummary(dating.User,full,json)",
			args: args{
				rule: "//colgen@newUserSummary(dating.User,full,json)",
			},
			want: ReplaceRule{
				Find:     "//colgen@newUserSummary(dating.User,full,json)",
				Cmd:      "new",
				Entity:   "UserSummary",
				Arg:      "dating.User",
				IsFull:   true,
				WithJSON: true,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseReplaceRule(tt.args.rule)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseReplaceRule() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseReplaceRule() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReplacer_Generate(t *testing.T) {
	tests := []struct {
		skip    bool
		name    string
		arg     string
		want    string
		wantErr bool
	}{
		{
			name: "",
			arg:  "//colgen@NewCall(db)",
			want: `
type Call struct { 
    db.Call
}

func NewCall(in *db.Call) *Call {
	if in == nil {
		return nil
	}

	return &Call{ 
        Call: *in,
	}
}
`,
			wantErr: false,
		},
		{
			name: "",
			arg:  "//colgen@NewUser(db)",
			want: `
type User struct { 
    db.User
}

func NewUser(in *db.User) *User {
	if in == nil {
		return nil
	}

	return &User{ 
        User: *in,
	}
}
`,
			wantErr: false,
		},
		{
			skip: true,
			name: "",
			arg:  "//colgen@newUserSummary(db.User,full,json)",
			want: `
type UserSummary struct { 
    ID int |json:"userSummaryId"|
    CreatedAt time.Time |json:"createdAt"|
    Login string |json:"login"|
    Password string |json:"password"|
    AuthKey string |json:"authKey"|
    LastActivityAt *time.Time |json:"lastActivityAt"|
    StatusID int |json:"statusId"|
}

func newUserSummary(in *db.User) *UserSummary {
	if in == nil {
		return nil
	}

	return &UserSummary{
        ID: in.ID,
        CreatedAt: in.CreatedAt,
        Login: in.Login,
        Password: in.Password,
        AuthKey: in.AuthKey,
        LastActivityAt: in.LastActivityAt,
        StatusID: in.StatusID,
	}
}
`,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		// load packages
		rl := NewReplacer()
		err := rl.UsePackageDir(filepath.Dir("."))
		if err != nil {
			t.Errorf("UsePackageDir() error = %v, wantErr %v", err, tt.wantErr)
			return
		}

		t.Run(tt.name, func(t *testing.T) {
			if tt.skip {
				t.Log("skip")
				return
			}
			// generate
			got, err := rl.Generate([]string{tt.arg})
			if (err != nil) != tt.wantErr {
				t.Errorf("Generate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			tt.want = strings.ReplaceAll(tt.want, "|", "`")
			if !reflect.DeepEqual(got[0].Replace, tt.want) {
				t.Errorf("Generate() got = %v, want %v", got[0].Replace, tt.want)
				os.Stdout.WriteString(got[0].Replace)
			}
		})
	}
}
