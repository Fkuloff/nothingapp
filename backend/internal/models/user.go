package models

import "gorm.io/gorm"

// User represents a registered user account.
//
// VaultSalt + EncryptedAccountKey + PublicKey hold the client-side E2E key material:
//   - VaultSalt is the random PBKDF2 salt used to derive a vault_key from the user's
//     password. Base64-encoded, populated by the client on first E2E onboarding.
//   - EncryptedAccountKey is the user's per-account symmetric key (used to seed the
//     X25519 keypair) encrypted with the derived vault_key. Stored as opaque base64;
//     the server never sees the cleartext account key.
//   - PublicKey is the user's X25519 public key (32 bytes, base64). Derived
//     deterministically from account_key on the client and uploaded so that other
//     users can ECDH against it. Published openly — it's the "public" half of the
//     keypair and is what makes scheme=2 inter-user messaging possible.
//
// All three fields are nullable so existing users without E2E setup continue to work:
// they use SchemeServerSide (scheme=1) messages until/unless they opt into E2E by
// uploading vault material via PUT /api/auth/vault.
type User struct {
	gorm.Model
	Username            string  `gorm:"unique;not null"`
	Password            string  `gorm:"not null"`
	Name                string  `gorm:"not null"`
	AvatarURL           *string `gorm:"type:varchar(500)"`
	VaultSalt           *string `gorm:"type:varchar(64)"`
	EncryptedAccountKey *string `gorm:"type:text"`
	PublicKey           *string `gorm:"type:varchar(64)"`
}

// GetDisplayName returns the display name for the user (Name if set, otherwise Username)
func (u *User) GetDisplayName() string {
	if u.Name != "" {
		return u.Name
	}
	return u.Username
}
