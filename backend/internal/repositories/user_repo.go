package repositories

import (
	"context"

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

// SearchByUsernameOrName searches users by username or name (case-insensitive, partial match).
// The caller's own user_id is excluded from the results.
func (r *UserRepo) SearchByUsernameOrName(ctx context.Context, query string, excludeUserID uint) ([]*models.User, error) {
	var users []*models.User
	searchPattern := "%" + query + "%"
	err := r.db.WithContext(ctx).
		Where("(username ILIKE ? OR name ILIKE ?) AND id <> ?", searchPattern, searchPattern, excludeUserID).
		Limit(20).
		Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}
