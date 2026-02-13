package models

import "time"

// UserKey stores a user's public ECDH key for E2E encryption
type UserKey struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	UserID    uint      `gorm:"uniqueIndex;not null" json:"user_id"`
	PublicKey string    `gorm:"type:text;not null" json:"public_key"` // JWK format (ECDH P-256)
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	User      User      `gorm:"foreignKey:UserID" json:"-"`
}
