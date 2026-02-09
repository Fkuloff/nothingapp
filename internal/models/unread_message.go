package models

import "gorm.io/gorm"

// UnreadMessage tracks which messages a user hasn't read yet
// This allows offline message delivery when user reconnects
type UnreadMessage struct {
	gorm.Model
	Message   Message `gorm:"foreignKey:MessageID"`
	UserID    uint    `gorm:"index:idx_user_unread;not null"`
	MessageID uint    `gorm:"index:idx_user_unread;not null"`
	ChatID    uint    `gorm:"index:idx_chat_unread;not null"`
}
