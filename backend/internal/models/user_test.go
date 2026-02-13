package models

import "testing"

func TestUser_GetDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		user     User
		wantName string
	}{
		{
			name:     "returns Name when set",
			user:     User{Username: "johndoe", Name: "John Doe"},
			wantName: "John Doe",
		},
		{
			name:     "returns Username when Name is empty",
			user:     User{Username: "johndoe", Name: ""},
			wantName: "johndoe",
		},
		{
			name:     "returns Name even if same as Username",
			user:     User{Username: "johndoe", Name: "johndoe"},
			wantName: "johndoe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.user.GetDisplayName(); got != tt.wantName {
				t.Errorf("GetDisplayName() = %q, want %q", got, tt.wantName)
			}
		})
	}
}
