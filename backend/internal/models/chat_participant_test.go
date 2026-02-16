package models

import "testing"

func TestChatParticipant_IsCreator(t *testing.T) {
	tests := []struct {
		name string
		role ParticipantRole
		want bool
	}{
		{"creator role", RoleCreator, true},
		{"admin role", RoleAdmin, false},
		{"member role", RoleMember, false},
		{"empty role", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &ChatParticipant{Role: tt.role}
			if got := p.IsCreator(); got != tt.want {
				t.Errorf("IsCreator() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChatParticipant_IsAdmin(t *testing.T) {
	tests := []struct {
		name string
		role ParticipantRole
		want bool
	}{
		{"creator role returns true", RoleCreator, true},
		{"admin role returns true", RoleAdmin, true},
		{"member role returns false", RoleMember, false},
		{"empty role returns false", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &ChatParticipant{Role: tt.role}
			if got := p.IsAdmin(); got != tt.want {
				t.Errorf("IsAdmin() = %v, want %v", got, tt.want)
			}
		})
	}
}
