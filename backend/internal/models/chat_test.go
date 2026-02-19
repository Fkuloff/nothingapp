package models

import (
	"testing"

	"gorm.io/gorm"
)

func uintPtr(v uint) *uint { return &v }

func TestChat_HasUser(t *testing.T) {
	chat := &Chat{User1ID: uintPtr(1), User2ID: uintPtr(2)}

	tests := []struct {
		name   string
		userID uint
		want   bool
	}{
		{"user1 is participant", 1, true},
		{"user2 is participant", 2, true},
		{"stranger is not participant", 3, false},
		{"zero ID is not participant", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := chat.HasUser(tt.userID); got != tt.want {
				t.Errorf("HasUser(%d) = %v, want %v", tt.userID, got, tt.want)
			}
		})
	}
}

func TestChat_BeforeCreate_NormalizesUserIDs(t *testing.T) {
	tests := []struct {
		name        string
		user1ID     uint
		user2ID     uint
		wantUser1ID uint
		wantUser2ID uint
	}{
		{
			name:        "already normalized (smaller first)",
			user1ID:     1,
			user2ID:     5,
			wantUser1ID: 1,
			wantUser2ID: 5,
		},
		{
			name:        "swaps when larger ID first",
			user1ID:     10,
			user2ID:     3,
			wantUser1ID: 3,
			wantUser2ID: 10,
		},
		{
			name:        "equal IDs remain unchanged",
			user1ID:     7,
			user2ID:     7,
			wantUser1ID: 7,
			wantUser2ID: 7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chat := &Chat{User1ID: uintPtr(tt.user1ID), User2ID: uintPtr(tt.user2ID)}
			if err := chat.BeforeCreate(&gorm.DB{}); err != nil {
				t.Fatalf("BeforeCreate() error = %v", err)
			}
			if chat.GetUser1ID() != tt.wantUser1ID {
				t.Errorf("User1ID = %d, want %d", chat.GetUser1ID(), tt.wantUser1ID)
			}
			if chat.GetUser2ID() != tt.wantUser2ID {
				t.Errorf("User2ID = %d, want %d", chat.GetUser2ID(), tt.wantUser2ID)
			}
		})
	}
}

func TestChat_HasUser_GroupChat(t *testing.T) {
	chat := &Chat{IsGroup: true}

	tests := []struct {
		name   string
		userID uint
		want   bool
	}{
		{"any user returns false for group chat", 1, false},
		{"zero ID returns false for group chat", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := chat.HasUser(tt.userID); got != tt.want {
				t.Errorf("HasUser(%d) on group chat = %v, want %v", tt.userID, got, tt.want)
			}
		})
	}
}

func TestChat_BeforeCreate_SkipsNormalizationForGroup(t *testing.T) {
	chat := &Chat{IsGroup: true, User1ID: uintPtr(10), User2ID: uintPtr(3)}
	if err := chat.BeforeCreate(&gorm.DB{}); err != nil {
		t.Fatalf("BeforeCreate() error = %v", err)
	}
	// Group chats should NOT normalize user IDs
	if chat.GetUser1ID() != 10 {
		t.Errorf("User1ID = %d, want 10 (unchanged)", chat.GetUser1ID())
	}
	if chat.GetUser2ID() != 3 {
		t.Errorf("User2ID = %d, want 3 (unchanged)", chat.GetUser2ID())
	}
}

func TestChat_GetGroupName(t *testing.T) {
	teamName := "Team Alpha"
	empty := ""

	tests := []struct {
		name      string
		groupName *string
		want      string
	}{
		{"returns name when set", &teamName, "Team Alpha"},
		{"returns empty when nil", nil, ""},
		{"returns empty when empty string pointer", &empty, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chat := &Chat{IsGroup: true, GroupName: tt.groupName}
			if got := chat.GetGroupName(); got != tt.want {
				t.Errorf("GetGroupName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestChat_GetOtherUser(t *testing.T) {
	user1 := User{Username: "alice"}
	user1.ID = 1
	user2 := User{Username: "bob"}
	user2.ID = 2

	chat := &Chat{
		User1ID: uintPtr(1),
		User2ID: uintPtr(2),
		User1:   user1,
		User2:   user2,
	}

	t.Run("current user is User1, returns User2", func(t *testing.T) {
		other, otherID := chat.GetOtherUser(1)
		if otherID != 2 {
			t.Errorf("otherID = %d, want 2", otherID)
		}
		if other.Username != "bob" {
			t.Errorf("other.Username = %q, want %q", other.Username, "bob")
		}
	})

	t.Run("current user is User2, returns User1", func(t *testing.T) {
		other, otherID := chat.GetOtherUser(2)
		if otherID != 1 {
			t.Errorf("otherID = %d, want 1", otherID)
		}
		if other.Username != "alice" {
			t.Errorf("other.Username = %q, want %q", other.Username, "alice")
		}
	})
}

func TestChat_GetUserID_NilSafety(t *testing.T) {
	chat := &Chat{IsGroup: true}

	if got := chat.GetUser1ID(); got != 0 {
		t.Errorf("GetUser1ID() on nil = %d, want 0", got)
	}
	if got := chat.GetUser2ID(); got != 0 {
		t.Errorf("GetUser2ID() on nil = %d, want 0", got)
	}
}
