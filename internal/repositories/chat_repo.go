// internal/repositories/chat_repo.go
package repositories

import (
	"context"
	"messenger/internal/models"

	"gorm.io/gorm"
)

type ChatRepo struct {
	db *gorm.DB
}

func NewChatRepo(db *gorm.DB) *ChatRepo {
	return &ChatRepo{db: db}
}

func (r *ChatRepo) Create(ctx context.Context, chat *models.Chat) error {
	return r.db.WithContext(ctx).Create(chat).Error
}

func (r *ChatRepo) FindByUsers(ctx context.Context, user1ID, user2ID uint) (*models.Chat, error) {
	var chat models.Chat
	err := r.db.WithContext(ctx).Where("(user1_id = ? AND user2_id = ?) OR (user1_id = ? AND user2_id = ?)", user1ID, user2ID, user2ID, user1ID).First(&chat).Error
	if err != nil {
		return nil, err
	}
	return &chat, nil
}

func (r *ChatRepo) GetMessages(ctx context.Context, chatID uint) ([]models.Message, error) {
	var messages []models.Message
	err := r.db.WithContext(ctx).Where("chat_id = ?", chatID).Order("created_at asc").Find(&messages).Error
	return messages, err
}

func (r *ChatRepo) GetUserChats(ctx context.Context, userID uint) ([]models.Chat, error) {
	var chats []models.Chat
	err := r.db.WithContext(ctx).Where("user1_id = ? OR user2_id = ?", userID, userID).
		Preload("User1").Preload("User2").
		Find(&chats).Error
	return chats, err
}

func (r *ChatRepo) FindByID(ctx context.Context, id uint) (*models.Chat, error) {
	var chat models.Chat
	err := r.db.WithContext(ctx).Preload("User1").Preload("User2").First(&chat, id).Error
	if err != nil {
		return nil, err
	}
	return &chat, nil
}

// FindByIDLight finds a chat by ID without preloading users (for access checks)
func (r *ChatRepo) FindByIDLight(ctx context.Context, id uint) (*models.Chat, error) {
	var chat models.Chat
	err := r.db.WithContext(ctx).First(&chat, id).Error
	if err != nil {
		return nil, err
	}
	return &chat, nil
}
