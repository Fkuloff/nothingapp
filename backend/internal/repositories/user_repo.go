package repositories

import (
	"context"
	"strings"

	"messenger/internal/models"

	"gorm.io/gorm"
)

// UserRepo handles database operations for users.
type UserRepo struct {
	db *gorm.DB
}

// NewUserRepo creates a new UserRepo instance.
func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

// Create stores a new user record.
func (r *UserRepo) Create(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// FindByUsername finds a user by their username.
func (r *UserRepo) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByID finds a user by their ID.
func (r *UserRepo) FindByID(ctx context.Context, id uint) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateAvatar sets or clears the user's avatar URL.
func (r *UserRepo) UpdateAvatar(ctx context.Context, userID uint, avatarURL *string) error {
	return r.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).Update("avatar_url", avatarURL).Error
}

// UpdateName updates a user's display name
func (r *UserRepo) UpdateName(ctx context.Context, userID uint, name string) error {
	return r.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).Update("name", name).Error
}

// UpdatePassword updates a user's hashed password.
func (r *UserRepo) UpdatePassword(ctx context.Context, userID uint, hashedPassword string) error {
	return r.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).Update("password", hashedPassword).Error
}

// UpdateVault overwrites the user's E2E vault material — salt, encrypted account
// key, and the X25519 public key derived from that account key. All three are
// stored as base64 strings; the server never inspects their contents.
//
// Pass empty strings to clear (e.g. when a user opts out of E2E). All three
// must move together: half-state would let other users ECDH against a public_key
// whose private half is no longer accessible.
func (r *UserRepo) UpdateVault(ctx context.Context, userID uint, vaultSalt, encryptedAccountKey, publicKey string) error {
	var saltPtr, keyPtr, pubPtr *string
	if vaultSalt != "" {
		saltPtr = &vaultSalt
	}
	if encryptedAccountKey != "" {
		keyPtr = &encryptedAccountKey
	}
	if publicKey != "" {
		pubPtr = &publicKey
	}
	return r.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).
		Updates(map[string]any{
			"vault_salt":            saltPtr,
			"encrypted_account_key": keyPtr,
			"public_key":            pubPtr,
		}).Error
}

// SearchByUsernameOrName searches users by username or name (case-insensitive, infix match).
// The caller's own user_id is excluded from the results.
//
// The query is normalized:
//   - leading "@" is stripped (users naturally type "@username" copied from the UI)
//   - whitespace splits the query into tokens that must all match (AND)
//
// Each token uses an infix pattern (`%token%`) and is matched against username OR name.
// Multi-token AND means "Иванов Иван" matches a user named "Иван Иванов" — token order
// doesn't matter.
//
// Performance: infix LIKE can't use a B-tree index. For ≤10k users this is fine. Switch
// to pg_trgm + GIN index when the user base grows.
func (r *UserRepo) SearchByUsernameOrName(ctx context.Context, query string, excludeUserID uint) ([]*models.User, error) {
	query = strings.TrimSpace(query)
	query = strings.TrimPrefix(query, "@")
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}

	tokens := strings.Fields(query)
	if len(tokens) == 0 {
		return nil, nil
	}

	db := r.db.WithContext(ctx).Where("id <> ?", excludeUserID)
	for _, token := range tokens {
		pattern := "%" + token + "%"
		db = db.Where("(username ILIKE ? OR name ILIKE ?)", pattern, pattern)
	}

	var users []*models.User
	if err := db.Limit(20).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}
