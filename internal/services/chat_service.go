// internal/services/chat_service.go
package services

import (
	"errors"

	"messenger/internal/models"
	"messenger/internal/repositories"

	"gorm.io/gorm"
)

type ChatService struct {
	chatRepo    *repositories.ChatRepo
	messageRepo *repositories.MessageRepo
}

func NewChatService(chatRepo *repositories.ChatRepo, messageRepo *repositories.MessageRepo) *ChatService {
	return &ChatService{chatRepo: chatRepo, messageRepo: messageRepo}
}

func (s *ChatService) CreateChat(user1ID, user2ID uint) (*models.Chat, error) {
	if user1ID == user2ID {
		return nil, errors.New("cannot create chat with self")
	}

	existingChat, err := s.chatRepo.FindByUsers(user1ID, user2ID)
	if err == nil {
		return existingChat, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err // Propagate other errors
	}

	chat := &models.Chat{
		User1ID: user1ID,
		User2ID: user2ID,
	}

	err = s.chatRepo.Create(chat)
	if err != nil {
		return nil, err
	}
	return chat, nil
}

func (s *ChatService) SendMessage(chatID, userID uint, text string, replyToID uint) (*models.Message, error) {
	var replyPtr *uint
	if replyToID != 0 {
		msg, err := s.messageRepo.FindByID(replyToID)
		if err != nil || msg.ChatID != chatID {
			return nil, errors.New("invalid reply message")
		}
		replyPtr = &replyToID
	} // else replyPtr = nil

	message := &models.Message{
		ChatID:    chatID,
		UserID:    userID,
		Text:      text,
		ReplyToID: replyPtr,
	}

	err := s.messageRepo.Create(message)
	if err != nil {
		return nil, err
	}
	return message, nil
}

func (s *ChatService) GetMessages(chatID uint) ([]models.Message, error) {
	return s.messageRepo.GetAllByChatID(chatID)
}

func (s *ChatService) GetUserChats(userID uint) ([]models.Chat, error) {
	return s.chatRepo.GetUserChats(userID)
}

func (s *ChatService) FindChatByID(id uint) (*models.Chat, error) {
	return s.chatRepo.FindByID(id)
}
