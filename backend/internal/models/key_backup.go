package models

import "time"

// KeyBackup stores an encrypted backup of a user's private key for multi-device support.
// The private key is encrypted client-side with a password-derived key (PBKDF2 + AES-GCM).
// The server never has access to the plaintext private key.
type KeyBackup struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	UserID       uint      `gorm:"uniqueIndex;not null" json:"user_id"`
	EncryptedKey string    `gorm:"type:text;not null" json:"encrypted_key"` // AES-GCM encrypted private key (base64)
	Salt         string    `gorm:"type:varchar(64);not null" json:"salt"`   // PBKDF2 salt (base64)
	IV           string    `gorm:"type:varchar(32);not null" json:"iv"`     // AES-GCM IV (base64)
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	User         User      `gorm:"foreignKey:UserID" json:"-"`
}
