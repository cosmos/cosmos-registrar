package gitwrap

import "testing"

func TestGetGlobalGitIdentity(t *testing.T) {
	tests := []struct {
		name      string
		wantName  string
		wantEmail string
	}{
		{"1", "noandrea", "no.andrea@gmail.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotEmail := GetGlobalGitIdentity()
			if gotName != tt.wantName {
				t.Errorf("GetGlobalGitIdentity() gotName = %v, want %v", gotName, tt.wantName)
			}
			if gotEmail != tt.wantEmail {
				t.Errorf("GetGlobalGitIdentity() gotEmail = %v, want %v", gotEmail, tt.wantEmail)
			}
		})
	}
}
