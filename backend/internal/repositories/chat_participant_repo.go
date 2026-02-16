package repositories

import (
	"context"

	"messenger/internal/models"

	"gorm.io/gorm"
)

type ChatParticipantRepo struct {
	db *gorm.DB
}

func NewChatParticipantRepo(db *gorm.DB) *ChatParticipantRepo {
	return &ChatParticipantRepo{db: db}
}

func (r *ChatParticipantRepo) WithTx(tx *gorm.DB) *ChatParticipantRepo {
	return &ChatParticipantRepo{db: tx}
}

func (r *ChatParticipantRepo) Create(ctx context.Context, p *models.ChatParticipant) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *ChatParticipantRepo) Delete(ctx context.Context, chatID, userID uint) error {
	return r.db.WithContext(ctx).
		Where("chat_id = ? AND user_id = ?", chatID, userID).
		Delete(&models.ChatParticipant{}).Error
}

func (r *ChatParticipantRepo) GetByChatID(ctx context.Context, chatID uint) ([]models.ChatParticipant, error) {
	var participants []models.ChatParticipant
	err := r.db.WithContext(ctx).
		Where("chat_id = ?", chatID).
		Find(&participants).Error
	return participants, err
}

func (r *ChatParticipantRepo) GetByChatIDWithUsers(ctx context.Context, chatID uint) ([]models.ChatParticipant, error) {
	var participants []models.ChatParticipant
	err := r.db.WithContext(ctx).
		Where("chat_id = ?", chatID).
		Preload("User").
		Find(&participants).Error
	return participants, err
}

func (r *ChatParticipantRepo) GetParticipantUserIDs(ctx context.Context, chatID uint) ([]uint, error) {
	var userIDs []uint
	err := r.db.WithContext(ctx).
		Model(&models.ChatParticipant{}).
		Where("chat_id = ?", chatID).
		Pluck("user_id", &userIDs).Error
	return userIDs, err
}

func (r *ChatParticipantRepo) FindByUserAndChat(ctx context.Context, chatID, userID uint) (*models.ChatParticipant, error) {
	var p models.ChatParticipant
	err := r.db.WithContext(ctx).
		Where("chat_id = ? AND user_id = ?", chatID, userID).
		First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *ChatParticipantRepo) CountByChatID(ctx context.Context, chatID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.ChatParticipant{}).
		Where("chat_id = ?", chatID).
		Count(&count).Error
	return count, err
}

func (r *ChatParticipantRepo) GetUserGroupChatIDs(ctx context.Context, userID uint) ([]uint, error) {
	var chatIDs []uint
	err := r.db.WithContext(ctx).
		Model(&models.ChatParticipant{}).
		Where("user_id = ?", userID).
		Pluck("chat_id", &chatIDs).Error
	return chatIDs, err
}

func (r *ChatParticipantRepo) UpdateRole(ctx context.Context, chatID, userID uint, role models.ParticipantRole) error {
	return r.db.WithContext(ctx).
		Model(&models.ChatParticipant{}).
		Where("chat_id = ? AND user_id = ?", chatID, userID).
		Update("role", role).Error
}
