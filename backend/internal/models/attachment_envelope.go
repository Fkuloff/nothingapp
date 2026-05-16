package models

// AttachmentEnvelope holds the per-recipient wrapped file_key for a scheme=2
// attachment. The actual file blob is encrypted client-side with a random
// AES-GCM file_key (stored nowhere on the server); that file_key is then
// wrapped once per recipient via AES-GCM(file_key, chat_key) where chat_key is
// the standard 1-on-1 or group ECDH-derived key.
//
// Server stores opaque blobs (the encrypted file in MinIO) and these opaque
// envelopes (the wrapped file_keys), and can read neither without a
// participant's account_key.
type AttachmentEnvelope struct {
	// Composite primary key — one envelope per (attachment, recipient).
	// Secondary index on (recipient_id, attachment_id) supports the lookup
	// "give me my envelopes for these attachments" when rendering a message
	// with N files for the current user.
	AttachmentID uint `gorm:"primaryKey;autoIncrement:false;index:idx_att_envelope_recipient,priority:2"`
	RecipientID  uint `gorm:"primaryKey;autoIncrement:false;index:idx_att_envelope_recipient,priority:1"`

	// EncryptedFileKey is base64 AES-GCM(file_key, chat_key) — the file_key
	// itself is a random 32-byte AES key chosen client-side per attachment.
	EncryptedFileKey string `gorm:"not null"`
	// IV is the base64-encoded 12-byte nonce used to AES-GCM-wrap the file_key.
	// Separate from the file's own IV (the file body has its own AES-GCM nonce).
	IV string `gorm:"type:varchar(32);not null"`
}

// TableName pins the plural so a future struct rename doesn't silently move
// the table.
func (AttachmentEnvelope) TableName() string {
	return "attachment_envelopes"
}
