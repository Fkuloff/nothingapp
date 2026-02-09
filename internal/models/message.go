package models

import (
	"time"

	"gorm.io/gorm"
)

type Message struct {
	ReplyToID *uint      `gorm:"index:idx_reply_to"`
	ReplyTo   *Message   `gorm:"foreignKey:ReplyToID"`
	EditedAt  *time.Time `gorm:"default:null"`
	gorm.Model
	Text        string       `gorm:"not null"`
	Attachments []Attachment `gorm:"foreignKey:MessageID" json:"attachments,omitempty"`
	ChatID      uint         `gorm:"index:idx_chat_messages"`
	UserID      uint         `gorm:"index:idx_user_messages"`
	IsDeleted   bool         `gorm:"default:false"`
}
