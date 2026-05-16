package models

import (
	"time"

	"gorm.io/gorm"
)

// Attachment represents a file attached to a message.
//
// All user-facing metadata (filename + mime type) is encrypted client-side
// under the per-file file_key and stored in EncryptedMetadata + MetadataIV.
// The body itself is opaque ciphertext in MinIO; only the byte size leaks
// (inherent to network transfer). Legacy plaintext columns `file_name`,
// `mime_type`, `file_type` and the `AttachmentType` enum were removed —
// rows from before the encrypted-metadata migration are unreadable and
// must be cleaned up via the cleanup runbook.
type Attachment struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`

	MessageID  uint   `gorm:"index:idx_message_attachments;not null" json:"message_id"`
	StorageKey string `gorm:"type:varchar(500);not null;unique" json:"storage_key"`
	FileSize   int64  `gorm:"not null" json:"file_size"`
	// FileIV is the base64 AES-GCM nonce used to encrypt the file body
	// client-side. Same nonce for every recipient — they all decrypt the
	// same ciphertext with the same per-file file_key (which itself is
	// wrapped per-recipient in attachment_envelopes).
	FileIV string `gorm:"type:varchar(32);not null" json:"file_iv"`
	// EncryptedMetadata + MetadataIV: AES-GCM ciphertext of a small JSON blob
	// {"fileName": "...", "mimeType": "..."} encrypted under the same
	// per-file file_key that encrypts the body. Server never sees the
	// plaintext. Required for every new upload — receiver derives the render
	// bucket (image / video / document) from the decrypted mime.
	EncryptedMetadata string `gorm:"type:text;not null" json:"encrypted_metadata"`
	MetadataIV        string `gorm:"type:varchar(32);not null" json:"metadata_iv"`

	Width    *int     `gorm:"default:null" json:"width,omitempty"`
	Height   *int     `gorm:"default:null" json:"height,omitempty"`
	Duration *int     `gorm:"default:null" json:"duration,omitempty"`
	Message  *Message `gorm:"foreignKey:MessageID" json:"-"`
}
