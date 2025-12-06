// internal/repositories/chat_repo.go
package repositories

import (
	"messenger/internal/models"

	"gorm.io/gorm"
)

type ChatRepo struct {
	db *gorm.DB
}

func NewChatRepo(db *gorm.DB) *ChatRepo {
	return &ChatRepo{db: db}
}

func (r *ChatRepo) Create(chat *models.Chat) error {
	return r.db.Create(chat).Error
}

func (r *ChatRepo) FindByUsers(user1ID, user2ID uint) (*models.Chat, error) {
	var chat models.Chat
	err := r.db.Where("(user1_id = ? AND user2_id = ?) OR (user1_id = ? AND user2_id = ?)", user1ID, user2ID, user2ID, user1ID).First(&chat).Error
	if err != nil {
		return nil, err
	}
	return &chat, nil
}

func (r *ChatRepo) GetMessages(chatID uint) ([]models.Message, error) {
	var messages []models.Message
	err := r.db.Where("chat_id = ?", chatID).Order("created_at asc").Find(&messages).Error
	return messages, err
}

func (r *ChatRepo) GetUserChats(userID uint) ([]models.Chat, error) {
	var chats []models.Chat
	err := r.db.Where("user1_id = ? OR user2_id = ?", userID, userID).
		Preload("User1").Preload("User2").
		Find(&chats).Error
	return chats, err
}

func (r *ChatRepo) FindByID(id uint) (*models.Chat, error) {
	var chat models.Chat
	err := r.db.First(&chat, id).Error
	if err != nil {
		return nil, err
	}
	return &chat, nil
}
