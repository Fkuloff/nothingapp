package models

import "gorm.io/gorm"

// PinnedMessage represents a message pinned in a chat.
// Each chat can have multiple pinned messages (up to MaxPinnedPerChat).
type PinnedMessage struct {
	gorm.Model
	ChatID    uint    `gorm:"uniqueIndex:idx_pinned_chat_message;index:idx_pinned_chat;not null"`
	MessageID uint    `gorm:"uniqueIndex:idx_pinned_chat_message;not null"`
	PinnedBy  uint    `gorm:"not null"`
	Message   Message `gorm:"foreignKey:MessageID"`
}
