package repositories

import (
	"context"

	"messenger/internal/models"

	"gorm.io/gorm"
)

// ChatParticipantRepo handles database operations for group chat participants.
type ChatParticipantRepo struct {
	db *gorm.DB
}

// NewChatParticipantRepo creates a new ChatParticipantRepo instance.
func NewChatParticipantRepo(db *gorm.DB) *ChatParticipantRepo {
	return &ChatParticipantRepo{db: db}
}

// WithTx creates a new ChatParticipantRepo using the given transaction.
func (r *ChatParticipantRepo) WithTx(tx *gorm.DB) *ChatParticipantRepo {
	return &ChatParticipantRepo{db: tx}
}

// Create stores a new chat participant record.
func (r *ChatParticipantRepo) Create(ctx context.Context, p *models.ChatParticipant) error {
	return r.db.WithContext(ctx).Create(p).Error
}

// Delete removes a participant from a chat.
func (r *ChatParticipantRepo) Delete(ctx context.Context, chatID, userID uint) error {
	return r.db.WithContext(ctx).
		Where("chat_id = ? AND user_id = ?", chatID, userID).
		Delete(&models.ChatParticipant{}).Error
}

// GetByChatID returns all participants for a chat.
func (r *ChatParticipantRepo) GetByChatID(ctx context.Context, chatID uint) ([]models.ChatParticipant, error) {
	var participants []models.ChatParticipant
	err := r.db.WithContext(ctx).
		Where("chat_id = ?", chatID).
		Find(&participants).Error
	return participants, err
}

// GetByChatIDWithUsers returns all participants for a chat with user details preloaded.
func (r *ChatParticipantRepo) GetByChatIDWithUsers(ctx context.Context, chatID uint) ([]models.ChatParticipant, error) {
	var participants []models.ChatParticipant
	err := r.db.WithContext(ctx).
		Where("chat_id = ?", chatID).
		Preload("User").
		Find(&participants).Error
	return participants, err
}

// GetParticipantUserIDs returns only the user IDs of all participants in a chat.
func (r *ChatParticipantRepo) GetParticipantUserIDs(ctx context.Context, chatID uint) ([]uint, error) {
	var userIDs []uint
	err := r.db.WithContext(ctx).
		Model(&models.ChatParticipant{}).
		Where("chat_id = ?", chatID).
		Pluck("user_id", &userIDs).Error
	return userIDs, err
}

// FindByUserAndChat finds a participant record for a specific user in a chat.
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

// CountByChatID returns the number of participants in a chat.
func (r *ChatParticipantRepo) CountByChatID(ctx context.Context, chatID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.ChatParticipant{}).
		Where("chat_id = ?", chatID).
		Count(&count).Error
	return count, err
}

// GetUserGroupChatIDs returns all chat IDs where the user is a participant.
func (r *ChatParticipantRepo) GetUserGroupChatIDs(ctx context.Context, userID uint) ([]uint, error) {
	var chatIDs []uint
	err := r.db.WithContext(ctx).
		Model(&models.ChatParticipant{}).
		Where("user_id = ?", userID).
		Pluck("chat_id", &chatIDs).Error
	return chatIDs, err
}

// UpdateRole changes a participant's role in a chat.
func (r *ChatParticipantRepo) UpdateRole(ctx context.Context, chatID, userID uint, role models.ParticipantRole) error {
	return r.db.WithContext(ctx).
		Model(&models.ChatParticipant{}).
		Where("chat_id = ? AND user_id = ?", chatID, userID).
		Update("role", role).Error
}
