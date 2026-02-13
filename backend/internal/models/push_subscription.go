package models

import "gorm.io/gorm"

// PushSubscription stores a user's Web Push subscription (W3C Push API).
// A user can have multiple subscriptions (one per browser/device).
type PushSubscription struct {
	gorm.Model
	UserID   uint   `gorm:"index;not null"`
	Endpoint string `gorm:"uniqueIndex;type:text;not null"`
	P256dh   string `gorm:"type:varchar(500);not null"`
	Auth     string `gorm:"type:varchar(500);not null"`
}
