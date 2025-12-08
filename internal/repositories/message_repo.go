// internal/repositories/message_repo.go
package repositories

import (
	"messenger/internal/models"

	"gorm.io/gorm"
)

type MessageRepo struct {
	db *gorm.DB
}

func NewMessageRepo(db *gorm.DB) *MessageRepo {
	return &MessageRepo{db: db}
}

func (r *MessageRepo) Create(message *models.Message) error {
	return r.db.Create(message).Error
}

func (r *MessageRepo) GetAllByChatID(chatID uint) ([]models.Message, error) {
	var messages []models.Message
	err := r.db.Where("chat_id = ?", chatID).
		Order("created_at asc").
		Preload("ReplyTo").
		Find(&messages).Error
	return messages, err
}

func (r *MessageRepo) FindByID(id uint) (*models.Message, error) {
	var msg models.Message
	err := r.db.First(&msg, id).Error
	return &msg, err
}
