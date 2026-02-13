package testutil

import (
	"os"
	"testing"

	"messenger/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// SetupTestDB connects to PostgreSQL using DB_URL env, runs AutoMigrate,
// and registers a cleanup function that truncates all tables after the test.
func SetupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		t.Skip("DB_URL not set, skipping integration test")
	}

	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	// Run migrations
	err = db.AutoMigrate(
		&models.User{},
		&models.Chat{},
		&models.Message{},
		&models.Attachment{},
		&models.Contact{},
		&models.UnreadMessage{},
		&models.UserKey{},
		&models.KeyBackup{},
		&models.PushSubscription{},
	)
	if err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	t.Cleanup(func() {
		// Truncate all tables in reverse dependency order
		tables := []string{
			"attachments",
			"unread_messages",
			"messages",
			"contacts",
			"push_subscriptions",
			"key_backups",
			"user_keys",
			"chats",
			"users",
		}
		for _, table := range tables {
			db.Exec("TRUNCATE TABLE " + table + " CASCADE")
		}
	})

	return db
}
