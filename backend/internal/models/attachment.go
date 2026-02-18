package models

import (
	"time"

	"gorm.io/gorm"
)

// AttachmentType defines the type of attachment
type AttachmentType string

const (
	// AttachmentTypeImage represents image files (JPEG, PNG, GIF, WebP).
	AttachmentTypeImage AttachmentType = "image"
	// AttachmentTypeVideo represents video files (MP4, WebM).
	AttachmentTypeVideo AttachmentType = "video"
	// AttachmentTypeDocument represents documents and other file types.
	AttachmentTypeDocument AttachmentType = "document"
)

// Attachment represents a file attached to a message
type Attachment struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	MessageID  uint           `gorm:"index:idx_message_attachments;not null" json:"message_id"`
	FileType   AttachmentType `gorm:"type:varchar(20);not null;index:idx_file_type" json:"file_type"`
	StorageKey string         `gorm:"type:varchar(500);not null;unique" json:"storage_key"`
	FileName   string         `gorm:"type:varchar(255);not null" json:"file_name"`
	MimeType   string         `gorm:"type:varchar(100);not null" json:"mime_type"`
	FileSize   int64          `gorm:"not null" json:"file_size"`

	Width    *int     `gorm:"default:null" json:"width,omitempty"`
	Height   *int     `gorm:"default:null" json:"height,omitempty"`
	Duration *int     `gorm:"default:null" json:"duration,omitempty"`
	Message  *Message `gorm:"foreignKey:MessageID" json:"-"`
}
