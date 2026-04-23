package models

import "gorm.io/gorm"

// FCMToken stores a user's Firebase Cloud Messaging device token.
// A user can have multiple tokens (one per device).
type FCMToken struct {
	gorm.Model
	UserID   uint   `gorm:"index;not null"`
	Token    string `gorm:"uniqueIndex;type:text;not null"`
	Platform string `gorm:"type:varchar(16);not null"` // "android" | "ios"
}
