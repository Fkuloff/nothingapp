package repositories

import (
	"context"

	"messenger/internal/models"

	"gorm.io/gorm"
)

type UserRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *UserRepo) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepo) FindByID(ctx context.Context, id uint) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepo) UpdateAvatar(ctx context.Context, userID uint, avatarURL *string) error {
	return r.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).Update("avatar_url", avatarURL).Error
}

// SearchByUsernameOrName searches users by username or name (case-insensitive, partial match)
func (r *UserRepo) SearchByUsernameOrName(ctx context.Context, query string) ([]*models.User, error) {
	var users []*models.User
	searchPattern := "%" + query + "%"
	err := r.db.WithContext(ctx).Where("username ILIKE ? OR name ILIKE ?", searchPattern, searchPattern).
		Limit(20).
		Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}
