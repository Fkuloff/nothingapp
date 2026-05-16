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

// Encryption scheme constants for the messages.scheme column.
//
//   - SchemeServerSide: legacy. Used to mean "server encrypts via the global
//     MESSAGE_ENCRYPTION_KEY". That key has been removed; clients no longer
//     accept this scheme on new sends, and old rows are rendered as a
//     placeholder ("🔒 encrypted message") because nobody can decrypt them.
//   - SchemeClientSide: end-to-end. Text/IV are encrypted by the sender client
//     with chat_key = HKDF(ECDH(my, peer.public_key)). Server stores opaque
//     ciphertext, cannot read plaintext. Only valid scheme for new messages.
//
// The SchemeServerSide constant is kept so we can identify legacy rows during
// migration / cleanup; it is no longer written by the server.
const (
	SchemeServerSide uint8 = 1
	SchemeClientSide uint8 = 2
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
	// Scheme selects the encryption layer responsible for Text/IV. New rows default
	// to SchemeServerSide so the GORM AutoMigrate adding the column doesn't break
	// the live deployment (existing 177-ish rows in prod are server-encrypted).
	Scheme uint8 `gorm:"type:smallint;not null;default:1"`

	ReplyToID   *uint        `gorm:"index:idx_reply_to"`
	EditedAt    *time.Time   `gorm:"default:null"`
	ReplyTo     *Message     `gorm:"foreignKey:ReplyToID"`
	Attachments []Attachment `gorm:"foreignKey:MessageID" json:"attachments,omitempty"`
}
