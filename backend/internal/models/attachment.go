package models

import (
	"time"

	"gorm.io/gorm"
)

// AttachmentType defines the type of attachment
type AttachmentType string

const (
	AttachmentTypeImage    AttachmentType = "image"
	AttachmentTypeVideo    AttachmentType = "video"
	AttachmentTypeDocument AttachmentType = "document"
	AttachmentTypeAudio    AttachmentType = "audio"
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

	Width        *int     `gorm:"default:null" json:"width,omitempty"`
	Height       *int     `gorm:"default:null" json:"height,omitempty"`
	Duration     *int     `gorm:"default:null" json:"duration,omitempty"`
	IV           string   `gorm:"type:varchar(32)" json:"iv,omitempty"`             // AES-GCM nonce (base64); empty = unencrypted
	OriginalType string   `gorm:"type:varchar(100)" json:"original_type,omitempty"` // Original MIME type before encryption
	OriginalName string   `gorm:"type:varchar(255)" json:"original_name,omitempty"` // Original filename before encryption
	Message      *Message `gorm:"foreignKey:MessageID" json:"-"`
}

// IsImage returns true if the attachment is an image
func (a *Attachment) IsImage() bool {
	return a.FileType == AttachmentTypeImage
}

// IsVideo returns true if the attachment is a video
func (a *Attachment) IsVideo() bool {
	return a.FileType == AttachmentTypeVideo
}
