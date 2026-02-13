// internal/repositories/message_repo.go
package repositories

import (
	"context"
	"time"

	"messenger/internal/models"

	"gorm.io/gorm"
)

type MessageRepo struct {
	db *gorm.DB
}

func NewMessageRepo(db *gorm.DB) *MessageRepo {
	return &MessageRepo{db: db}
}

// WithTx creates a new MessageRepo with the given transaction
func (r *MessageRepo) WithTx(tx *gorm.DB) *MessageRepo {
	return &MessageRepo{db: tx}
}

func (r *MessageRepo) Create(ctx context.Context, message *models.Message) error {
	return r.db.WithContext(ctx).Create(message).Error
}

func (r *MessageRepo) GetAllByChatID(ctx context.Context, chatID uint) ([]models.Message, error) {
	var messages []models.Message
	err := r.db.WithContext(ctx).Where("chat_id = ?", chatID).
		Order("created_at asc").
		Preload("ReplyTo").
		Preload("ReplyTo.Attachments"). // Nested preload to avoid N+1 for reply attachments
		Preload("Attachments").
		Find(&messages).Error
	return messages, err
}

// GetRecentByChatID gets the most recent N messages for a chat
func (r *MessageRepo) GetRecentByChatID(ctx context.Context, chatID uint, limit int) ([]models.Message, error) {
	var messages []models.Message
	err := r.db.WithContext(ctx).Where("chat_id = ?", chatID).
		Order("created_at desc").
		Limit(limit).
		Preload("ReplyTo").
		Preload("ReplyTo.Attachments"). // Nested preload to avoid N+1 for reply attachments
		Preload("Attachments").
		Find(&messages).Error

	// Reverse to get chronological order (oldest first)
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, err
}

func (r *MessageRepo) FindByID(ctx context.Context, id uint) (*models.Message, error) {
	var msg models.Message
	err := r.db.WithContext(ctx).Preload("Attachments").First(&msg, id).Error
	return &msg, err
}

// UpdateMessage updates message text (and IV for E2E encryption) and sets EditedAt timestamp
func (r *MessageRepo) UpdateMessage(ctx context.Context, id uint, newText, iv string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.Message{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"text":      newText,
			"iv":        iv,
			"edited_at": now,
		}).Error
}

// SoftDeleteMessage marks message as deleted without removing it from DB
func (r *MessageRepo) SoftDeleteMessage(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&models.Message{}).
		Where("id = ?", id).
		Update("is_deleted", true).Error
}
