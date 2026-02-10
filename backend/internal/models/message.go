package models

import (
	"time"

	"gorm.io/gorm"
)

type Message struct {
	gorm.Model
	ChatID    uint   `gorm:"index:idx_chat_messages"`
	UserID    uint   `gorm:"index:idx_user_messages"`
	Text      string `gorm:"not null"`
	IsDeleted bool   `gorm:"default:false"`

	ReplyToID   *uint        `gorm:"index:idx_reply_to"`
	EditedAt    *time.Time   `gorm:"default:null"`
	ReplyTo     *Message     `gorm:"foreignKey:ReplyToID"`
	Attachments []Attachment `gorm:"foreignKey:MessageID" json:"attachments,omitempty"`
}
