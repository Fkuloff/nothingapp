package models

import (
	"testing"

	"gorm.io/gorm"
)

func TestChat_HasUser(t *testing.T) {
	chat := &Chat{User1ID: 1, User2ID: 2}

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
			chat := &Chat{User1ID: tt.user1ID, User2ID: tt.user2ID}
			if err := chat.BeforeCreate(&gorm.DB{}); err != nil {
				t.Fatalf("BeforeCreate() error = %v", err)
			}
			if chat.User1ID != tt.wantUser1ID {
				t.Errorf("User1ID = %d, want %d", chat.User1ID, tt.wantUser1ID)
			}
			if chat.User2ID != tt.wantUser2ID {
				t.Errorf("User2ID = %d, want %d", chat.User2ID, tt.wantUser2ID)
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
		User1ID: 1,
		User2ID: 2,
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
