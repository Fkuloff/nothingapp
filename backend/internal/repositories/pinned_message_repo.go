package repositories

import (
	"context"

	"messenger/internal/models"

	"gorm.io/gorm"
)

// PinnedMessageRepo handles database operations for pinned messages.
type PinnedMessageRepo struct {
	db *gorm.DB
}

// NewPinnedMessageRepo creates a new pinned message repository.
func NewPinnedMessageRepo(db *gorm.DB) *PinnedMessageRepo {
	return &PinnedMessageRepo{db: db}
}

// WithTx creates a new PinnedMessageRepo using the given transaction.
func (r *PinnedMessageRepo) WithTx(tx *gorm.DB) *PinnedMessageRepo {
	return &PinnedMessageRepo{db: tx}
}

// Create stores a new pinned message record.
func (r *PinnedMessageRepo) Create(ctx context.Context, pin *models.PinnedMessage) error {
	return r.db.WithContext(ctx).Create(pin).Error
}

// Delete hard-deletes a specific pinned message by chat and message ID.
func (r *PinnedMessageRepo) Delete(ctx context.Context, chatID, messageID uint) error {
	return r.db.WithContext(ctx).Unscoped().
		Where("chat_id = ? AND message_id = ?", chatID, messageID).
		Delete(&models.PinnedMessage{}).Error
}

// GetByChatID returns all pinned messages for a chat, most recent first, with preloaded Message and Attachments.
func (r *PinnedMessageRepo) GetByChatID(ctx context.Context, chatID uint) ([]models.PinnedMessage, error) {
	var pins []models.PinnedMessage
	err := r.db.WithContext(ctx).
		Where("chat_id = ?", chatID).
		Order("created_at DESC").
		Preload("Message").
		Preload("Message.Attachments").
		Find(&pins).Error
	return pins, err
}

// CountByChatID returns the number of pinned messages in a chat.
func (r *PinnedMessageRepo) CountByChatID(ctx context.Context, chatID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.PinnedMessage{}).
		Where("chat_id = ?", chatID).
		Count(&count).Error
	return count, err
}

// IsPinned checks whether a specific message is pinned in a chat.
func (r *PinnedMessageRepo) IsPinned(ctx context.Context, chatID, messageID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.PinnedMessage{}).
		Where("chat_id = ? AND message_id = ?", chatID, messageID).
		Count(&count).Error
	return count > 0, err
}

// DeleteByChatID hard-deletes all pinned messages for a chat (cascade cleanup).
func (r *PinnedMessageRepo) DeleteByChatID(ctx context.Context, chatID uint) error {
	return r.db.WithContext(ctx).Unscoped().
		Where("chat_id = ?", chatID).
		Delete(&models.PinnedMessage{}).Error
}
