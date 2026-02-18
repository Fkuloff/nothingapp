package models

import (
	"time"

	"gorm.io/gorm"
)

// MessageType distinguishes user-sent messages from system-generated ones.
type MessageType string

const (
	// MessageTypeUser is a regular message sent by a user.
	MessageTypeUser MessageType = "user"
	// MessageTypeSystem is an auto-generated message (e.g. "user joined the group").
	MessageTypeSystem MessageType = "system"
)

// Message represents a single message in a chat.
type Message struct {
	gorm.Model
	ChatID    uint        `gorm:"index:idx_chat_messages"`
	UserID    uint        `gorm:"index:idx_user_messages"`
	Text      string      `gorm:"not null"`
	IV        string      `gorm:"type:varchar(32)" json:"-"` // AES-GCM nonce (base64); internal use only
	IsDeleted bool        `gorm:"default:false"`
	Type      MessageType `gorm:"type:varchar(20);not null;default:'user'"`

	ReplyToID   *uint        `gorm:"index:idx_reply_to"`
	EditedAt    *time.Time   `gorm:"default:null"`
	ReplyTo     *Message     `gorm:"foreignKey:ReplyToID"`
	Attachments []Attachment `gorm:"foreignKey:MessageID" json:"attachments,omitempty"`
}
