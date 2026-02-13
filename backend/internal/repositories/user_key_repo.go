package repositories

import (
	"context"

	"messenger/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserKeyRepo struct {
	db *gorm.DB
}

func NewUserKeyRepo(db *gorm.DB) *UserKeyRepo {
	return &UserKeyRepo{db: db}
}

// Upsert creates or updates a user's public key
func (r *UserKeyRepo) Upsert(ctx context.Context, key *models.UserKey) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"public_key", "updated_at"}),
		}).
		Create(key).Error
}

// FindByUserID returns the public key for a user
func (r *UserKeyRepo) FindByUserID(ctx context.Context, userID uint) (*models.UserKey, error) {
	var key models.UserKey
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&key).Error
	return &key, err
}

// FindByUserIDs returns public keys for multiple users
func (r *UserKeyRepo) FindByUserIDs(ctx context.Context, userIDs []uint) ([]models.UserKey, error) {
	var keys []models.UserKey
	err := r.db.WithContext(ctx).Where("user_id IN ?", userIDs).Find(&keys).Error
	return keys, err
}
