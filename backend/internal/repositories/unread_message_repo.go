package repositories

import (
	"context"

	"messenger/internal/models"

	"gorm.io/gorm"
)

// UnreadMessageRepo handles database operations for unread messages
type UnreadMessageRepo struct {
	db *gorm.DB
}

// NewUnreadMessageRepo creates a new unread message repository
func NewUnreadMessageRepo(db *gorm.DB) *UnreadMessageRepo {
	return &UnreadMessageRepo{db: db}
}

// WithTx creates a new UnreadMessageRepo with the given transaction
func (r *UnreadMessageRepo) WithTx(tx *gorm.DB) *UnreadMessageRepo {
	return &UnreadMessageRepo{db: tx}
}

// Create stores a new unread message record
func (r *UnreadMessageRepo) Create(ctx context.Context, unreadMsg *models.UnreadMessage) error {
	return r.db.WithContext(ctx).Create(unreadMsg).Error
}

// GetByUser retrieves all unread messages for a user
func (r *UnreadMessageRepo) GetByUser(ctx context.Context, userID uint) ([]models.UnreadMessage, error) {
	var unreadMessages []models.UnreadMessage
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Preload("Message").
		Preload("Message.Attachments").
		Find(&unreadMessages).Error
	return unreadMessages, err
}

// DeleteByChat hard-deletes all unread records in a chat for a user (marks as read)
func (r *UnreadMessageRepo) DeleteByChat(ctx context.Context, userID, chatID uint) error {
	return r.db.WithContext(ctx).Unscoped().
		Where("user_id = ? AND chat_id = ?", userID, chatID).
		Delete(&models.UnreadMessage{}).Error
}

// DeleteByUser hard-deletes all unread records for a user (used after pending message delivery)
func (r *UnreadMessageRepo) DeleteByUser(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).Unscoped().
		Where("user_id = ?", userID).
		Delete(&models.UnreadMessage{}).Error
}

// HasUnreadInChat returns true if userID still has at least one unread
// message in chatID. Used by the auto-dismiss path: after a delete /
// clear_chat / etc. we check this per recipient to decide whether to fire
// a SendDismiss push.
func (r *UnreadMessageRepo) HasUnreadInChat(ctx context.Context, userID, chatID uint) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).
		Model(&models.UnreadMessage{}).
		Where("user_id = ? AND chat_id = ?", userID, chatID).
		Limit(1).
		Count(&n).Error
	return n > 0, err
}

// GetUnreadCounts returns count of unread messages per chat for a user
func (r *UnreadMessageRepo) GetUnreadCounts(ctx context.Context, userID uint) (map[uint]int64, error) {
	var results []struct {
		ChatID uint
		Count  int64
	}

	err := r.db.WithContext(ctx).
		Model(&models.UnreadMessage{}).
		Select("chat_id, COUNT(*) as count").
		Where("user_id = ?", userID).
		Group("chat_id").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}

	counts := make(map[uint]int64)
	for _, r := range results {
		counts[r.ChatID] = r.Count
	}
	return counts, nil
}
