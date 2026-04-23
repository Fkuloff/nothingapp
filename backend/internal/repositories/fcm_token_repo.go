package repositories

import (
	"context"

	"messenger/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// FCMTokenRepo handles database operations for FCM tokens.
type FCMTokenRepo struct {
	db *gorm.DB
}

// NewFCMTokenRepo creates a new FCM token repository.
func NewFCMTokenRepo(db *gorm.DB) *FCMTokenRepo {
	return &FCMTokenRepo{db: db}
}

// Upsert creates or updates an FCM token (idempotent on token).
func (r *FCMTokenRepo) Upsert(ctx context.Context, t *models.FCMToken) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "token"}},
			DoUpdates: clause.AssignmentColumns([]string{"user_id", "platform", "updated_at"}),
		}).
		Create(t).Error
}

// GetByUser retrieves all active FCM tokens for a user.
func (r *FCMTokenRepo) GetByUser(ctx context.Context, userID uint) ([]models.FCMToken, error) {
	var tokens []models.FCMToken
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Find(&tokens).Error
	return tokens, err
}

// DeleteByToken removes a token for a specific user.
func (r *FCMTokenRepo) DeleteByToken(ctx context.Context, userID uint, token string) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND token = ?", userID, token).
		Delete(&models.FCMToken{}).Error
}

// DeleteByTokenGlobal removes a token regardless of user.
// Used when FCM reports the token as unregistered/invalid.
func (r *FCMTokenRepo) DeleteByTokenGlobal(ctx context.Context, token string) error {
	return r.db.WithContext(ctx).
		Where("token = ?", token).
		Delete(&models.FCMToken{}).Error
}

// CountByUser returns the number of FCM tokens for a user.
func (r *FCMTokenRepo) CountByUser(ctx context.Context, userID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.FCMToken{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	return count, err
}
