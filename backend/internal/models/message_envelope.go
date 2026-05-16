package models

// MessageEnvelope holds the per-recipient ciphertext for a group chat message
// that uses client-side encryption (scheme=2). Variant A "Pairwise" E2E means
// the sender encrypts the plaintext once per recipient — derived from
// ECDH(sender.private, recipient.public) — and one envelope per recipient
// gets stored here.
//
// For 1-on-1 scheme=2 messages there is exactly one ciphertext, stored on the
// Message row itself (Text + IV); no envelopes are written. For group scheme=2
// messages Message.Text and Message.IV are empty and the real ciphertexts live
// here, addressed by RecipientID.
//
// The sender also writes an envelope for *themselves* (ECDH self-with-self is
// well-defined for X25519 and only the sender can recompute it) so they can
// re-read their own outgoing messages from another device.
type MessageEnvelope struct {
	// Composite primary key — one envelope per (message, recipient). Index by
	// (recipient_id, message_id) supports the "give me my envelopes for these
	// messages" lookup that fires on chat open + WS reconnect replay.
	MessageID   uint `gorm:"primaryKey;autoIncrement:false;index:idx_envelope_recipient_message,priority:2"`
	RecipientID uint `gorm:"primaryKey;autoIncrement:false;index:idx_envelope_recipient_message,priority:1"`

	// Ciphertext is the base64-encoded AES-GCM output for this recipient.
	Ciphertext string `gorm:"not null"`
	// IV is the base64-encoded 12-byte AES-GCM nonce for this envelope.
	IV string `gorm:"type:varchar(32);not null"`
}

// TableName fixes the plural since GORM's default would be "message_envelopes"
// which matches our intent — keeping it explicit so a future rename of the
// struct doesn't silently break migrations.
func (MessageEnvelope) TableName() string {
	return "message_envelopes"
}
