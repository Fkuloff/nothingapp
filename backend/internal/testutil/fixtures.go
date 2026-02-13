package testutil

import (
	"testing"

	"messenger/internal/models"

	"gorm.io/gorm"
)

// CreateTestUser creates a user with the given username and returns it.
func CreateTestUser(t *testing.T, db *gorm.DB, username string) *models.User {
	t.Helper()

	user := &models.User{
		Username: username,
		Password: "hashed_password",
		Name:     username,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("failed to create test user %q: %v", username, err)
	}
	return user
}

// CreateTestChat creates a chat between two users and returns it.
func CreateTestChat(t *testing.T, db *gorm.DB, u1, u2 *models.User) *models.Chat {
	t.Helper()

	chat := &models.Chat{
		User1ID: u1.ID,
		User2ID: u2.ID,
	}
	if err := db.Create(chat).Error; err != nil {
		t.Fatalf("failed to create test chat: %v", err)
	}
	return chat
}
