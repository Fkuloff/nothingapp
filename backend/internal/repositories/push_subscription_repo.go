package repositories

import (
	"context"

	"messenger/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PushSubscriptionRepo handles database operations for push subscriptions
type PushSubscriptionRepo struct {
	db *gorm.DB
}

// NewPushSubscriptionRepo creates a new push subscription repository
func NewPushSubscriptionRepo(db *gorm.DB) *PushSubscriptionRepo {
	return &PushSubscriptionRepo{db: db}
}

// Upsert creates or updates a push subscription (idempotent on endpoint)
func (r *PushSubscriptionRepo) Upsert(ctx context.Context, sub *models.PushSubscription) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "endpoint"}},
			DoUpdates: clause.AssignmentColumns([]string{"user_id", "p256dh", "auth", "updated_at"}),
		}).
		Create(sub).Error
}

// GetByUser retrieves all active subscriptions for a user
func (r *PushSubscriptionRepo) GetByUser(ctx context.Context, userID uint) ([]models.PushSubscription, error) {
	var subs []models.PushSubscription
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Find(&subs).Error
	return subs, err
}

// DeleteByEndpoint removes a subscription by its endpoint for a specific user
func (r *PushSubscriptionRepo) DeleteByEndpoint(ctx context.Context, userID uint, endpoint string) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND endpoint = ?", userID, endpoint).
		Delete(&models.PushSubscription{}).Error
}

// DeleteByEndpointGlobal removes a subscription by endpoint regardless of user.
// Used when push delivery fails with 410 Gone (subscription expired).
func (r *PushSubscriptionRepo) DeleteByEndpointGlobal(ctx context.Context, endpoint string) error {
	return r.db.WithContext(ctx).
		Where("endpoint = ?", endpoint).
		Delete(&models.PushSubscription{}).Error
}

// ExistsByUser checks if a user has at least one push subscription.
func (r *PushSubscriptionRepo) ExistsByUser(ctx context.Context, userID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.PushSubscription{}).
		Where("user_id = ?", userID).
		Limit(1).
		Count(&count).Error
	return count > 0, err
}

// CountByUser returns the number of push subscriptions for a user.
func (r *PushSubscriptionRepo) CountByUser(ctx context.Context, userID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.PushSubscription{}).
		Where("user_id = ?", userID).
		Count(&count).Error
	return count, err
}
