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

	MessageID uint `gorm:"index:idx_message_attachments;not null" json:"message_id"`
	// FileType, FileName, MimeType: kept on the row for legacy (pre-encrypted-
	// metadata) attachments. New scheme=2 uploads write empty strings here and
	// stash the real values in EncryptedMetadata instead, so the server never
	// sees the user's filename or claimed mime. Receiving clients prefer
	// EncryptedMetadata when present and fall back to these for legacy rows.
	FileType   AttachmentType `gorm:"type:varchar(20);not null;default:'';index:idx_file_type" json:"file_type"`
	StorageKey string         `gorm:"type:varchar(500);not null;unique" json:"storage_key"`
	FileName   string         `gorm:"type:varchar(255);not null;default:''" json:"file_name"`
	MimeType   string         `gorm:"type:varchar(100);not null;default:''" json:"mime_type"`
	FileSize   int64          `gorm:"not null" json:"file_size"`
	// FileIV is the base64 AES-GCM nonce used to encrypt the file body
	// client-side. Same nonce for every recipient — they all decrypt the
	// same ciphertext with the same per-file file_key (which itself is
	// wrapped per-recipient in attachment_envelopes). Empty for legacy
	// (pre-E2E) attachments which shouldn't exist after the cleanup.
	FileIV string `gorm:"type:varchar(32)" json:"file_iv,omitempty"`
	// EncryptedMetadata + MetadataIV: AES-GCM ciphertext of a small JSON blob
	// {"file_name": "...", "mime_type": "..."} encrypted under the same
	// per-file file_key that encrypts the body. Server never sees the
	// plaintext — including filename and claimed mime, which previously
	// leaked to the operator via the FileName / MimeType columns above.
	// Empty for legacy attachments uploaded before this encryption layer
	// shipped; clients fall back to FileName / MimeType then.
	EncryptedMetadata string `gorm:"type:text;default:''" json:"encrypted_metadata,omitempty"`
	MetadataIV        string `gorm:"type:varchar(32);default:''" json:"metadata_iv,omitempty"`

	Width    *int     `gorm:"default:null" json:"width,omitempty"`
	Height   *int     `gorm:"default:null" json:"height,omitempty"`
	Duration *int     `gorm:"default:null" json:"duration,omitempty"`
	Message  *Message `gorm:"foreignKey:MessageID" json:"-"`
}
