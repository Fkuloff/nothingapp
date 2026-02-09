package models

import (
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
	Message      *Message `gorm:"foreignKey:MessageID" json:"-"`
	ThumbnailKey *string  `gorm:"type:varchar(500)" json:"thumbnail_key,omitempty"`
	Width        *int     `gorm:"default:null" json:"width,omitempty"`
	Height       *int     `gorm:"default:null" json:"height,omitempty"`
	Duration     *int     `gorm:"default:null" json:"duration,omitempty"`
	gorm.Model
	FileType   AttachmentType `gorm:"type:varchar(20);not null;index:idx_file_type" json:"file_type"`
	StorageKey string         `gorm:"type:varchar(500);not null;unique" json:"storage_key"`
	FileName   string         `gorm:"type:varchar(255);not null" json:"file_name"`
	MimeType   string         `gorm:"type:varchar(100);not null" json:"mime_type"`
	MessageID  uint           `gorm:"index:idx_message_attachments;not null" json:"message_id"`
	FileSize   int64          `gorm:"not null" json:"file_size"`
}

// IsImage returns true if the attachment is an image
func (a *Attachment) IsImage() bool {
	return a.FileType == AttachmentTypeImage
}

// IsVideo returns true if the attachment is a video
func (a *Attachment) IsVideo() bool {
	return a.FileType == AttachmentTypeVideo
}

// RequiresThumbnail returns true if the attachment needs a thumbnail
func (a *Attachment) RequiresThumbnail() bool {
	return a.IsImage() || a.IsVideo()
}
