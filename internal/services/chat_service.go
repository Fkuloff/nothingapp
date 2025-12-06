// internal/services/chat_service.go
package services

import (
	"errors"

	"messenger/internal/models"
	"messenger/internal/repositories"
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

	existingChat, _ := s.chatRepo.FindByUsers(user1ID, user2ID)
	if existingChat != nil {
		return existingChat, nil
	}

	chat := &models.Chat{
		User1ID: user1ID,
		User2ID: user2ID,
	}

	err := s.chatRepo.Create(chat)
	return chat, err
}

func (s *ChatService) SendMessage(chatID, userID uint, text string) error {
	message := &models.Message{
		ChatID: chatID,
		UserID: userID,
		Text:   text,
	}

	return s.messageRepo.Create(message)
}

func (s *ChatService) GetMessages(chatID uint) ([]models.Message, error) {
	return s.chatRepo.GetMessages(chatID)
}

func (s *ChatService) GetUserChats(userID uint) ([]models.Chat, error) {
	return s.chatRepo.GetUserChats(userID)
}

func (s *ChatService) FindChatByID(id uint) (*models.Chat, error) {
	return s.chatRepo.FindByID(id)
}
