// internal/services/chat_service.go
package services

import (
	"context"
	"errors"
	"fmt"

	"messenger/internal/models"
	"messenger/internal/repositories"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Sentinel errors
var (
	ErrNoMessages = errors.New("no messages found")
)

type ChatService struct {
	logger            *zap.Logger
	chatRepo          *repositories.ChatRepo
	messageRepo       *repositories.MessageRepo
	unreadMessageRepo *repositories.UnreadMessageRepo
}

func NewChatService(
	logger *zap.Logger,
	chatRepo *repositories.ChatRepo,
	messageRepo *repositories.MessageRepo,
	unreadMessageRepo *repositories.UnreadMessageRepo,
) *ChatService {
	return &ChatService{
		logger:            logger,
		chatRepo:          chatRepo,
		messageRepo:       messageRepo,
		unreadMessageRepo: unreadMessageRepo,
	}
}

func (s *ChatService) CreateChat(ctx context.Context, user1ID, user2ID uint) (*models.Chat, error) {
	if user1ID == user2ID {
		return nil, fmt.Errorf("cannot create chat with self")
	}

	// Normalize IDs (smaller always first) to match BeforeCreate hook
	if user1ID > user2ID {
		user1ID, user2ID = user2ID, user1ID
	}

	// Check for existing chat
	existingChat, err := s.chatRepo.FindByUsers(ctx, user1ID, user2ID)
	if err == nil {
		return existingChat, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check for existing chat: %w", err)
	}

	chat := &models.Chat{
		User1ID: user1ID,
		User2ID: user2ID,
	}

	if err := s.chatRepo.Create(ctx, chat); err != nil {
		// If unique constraint violated due to race condition, fetch existing chat
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			raceChat, fetchErr := s.chatRepo.FindByUsers(ctx, user1ID, user2ID)
			if fetchErr == nil {
				return raceChat, nil
			}
			return nil, fmt.Errorf("failed to fetch existing chat after race condition: %w", fetchErr)
		}
		return nil, fmt.Errorf("failed to create chat: %w", err)
	}
	return chat, nil
}

func (s *ChatService) SendMessage(ctx context.Context, chatID, userID uint, text string, replyToID uint) (*models.Message, error) {
	// CRITICAL: Verify user is participant in the chat (use light method for performance)
	chat, err := s.chatRepo.FindByIDLight(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("chat not found")
	}

	if !chat.HasUser(userID) {
		return nil, fmt.Errorf("unauthorized: user is not a participant in this chat")
	}

	var replyPtr *uint
	if replyToID != 0 {
		msg, err := s.messageRepo.FindByID(ctx, replyToID)
		if err != nil || msg.ChatID != chatID {
			return nil, fmt.Errorf("invalid reply message")
		}
		replyPtr = &replyToID
	}

	// Allow empty text if attachments will be added
	message := &models.Message{
		ChatID:    chatID,
		UserID:    userID,
		Text:      text,
		ReplyToID: replyPtr,
	}

	if err := s.messageRepo.Create(ctx, message); err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}
	return message, nil
}

// GetMessageByID returns a message by its ID
func (s *ChatService) GetMessageByID(ctx context.Context, messageID uint) (*models.Message, error) {
	message, err := s.messageRepo.FindByID(ctx, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to find message: %w", err)
	}
	return message, nil
}

func (s *ChatService) GetMessages(ctx context.Context, chatID uint) ([]models.Message, error) {
	messages, err := s.messageRepo.GetAllByChatID(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	return messages, nil
}

// GetRecentMessages gets the most recent N messages for a chat
func (s *ChatService) GetRecentMessages(ctx context.Context, chatID uint, limit int) ([]models.Message, error) {
	messages, err := s.messageRepo.GetRecentByChatID(ctx, chatID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent messages: %w", err)
	}
	return messages, nil
}

// GetLastMessageForChat retrieves the most recent message for a chat
func (s *ChatService) GetLastMessageForChat(ctx context.Context, chatID uint) (*models.Message, error) {
	messages, err := s.messageRepo.GetRecentByChatID(ctx, chatID, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to get last message: %w", err)
	}
	if len(messages) == 0 {
		return nil, ErrNoMessages
	}
	return &messages[0], nil
}

func (s *ChatService) GetUserChats(ctx context.Context, userID uint) ([]models.Chat, error) {
	chats, err := s.chatRepo.GetUserChats(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user chats: %w", err)
	}
	return chats, nil
}

func (s *ChatService) FindChatByID(ctx context.Context, id uint) (*models.Chat, error) {
	chat, err := s.chatRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find chat: %w", err)
	}
	return chat, nil
}

// FindChatByIDLight finds a chat without preloading users (for access checks only)
func (s *ChatService) FindChatByIDLight(ctx context.Context, id uint) (*models.Chat, error) {
	chat, err := s.chatRepo.FindByIDLight(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to find chat: %w", err)
	}
	return chat, nil
}

// EditMessage allows a user to edit their own message
func (s *ChatService) EditMessage(ctx context.Context, messageID, userID uint, newText string) error {
	// Find the message
	message, err := s.messageRepo.FindByID(ctx, messageID)
	if err != nil {
		return fmt.Errorf("message not found")
	}

	// Verify ownership
	if message.UserID != userID {
		return fmt.Errorf("unauthorized: you can only edit your own messages")
	}

	// Check if already deleted
	if message.IsDeleted {
		return fmt.Errorf("cannot edit deleted message")
	}

	// Validate new text
	if newText == "" || len(newText) > 10000 {
		return fmt.Errorf("invalid message length")
	}

	if err := s.messageRepo.UpdateMessage(ctx, messageID, newText); err != nil {
		return fmt.Errorf("failed to update message: %w", err)
	}

	return nil
}

// DeleteMessage allows a user to delete their own message
func (s *ChatService) DeleteMessage(ctx context.Context, messageID, userID uint) error {
	// Find the message
	message, err := s.messageRepo.FindByID(ctx, messageID)
	if err != nil {
		return fmt.Errorf("message not found")
	}

	// Verify ownership
	if message.UserID != userID {
		return fmt.Errorf("unauthorized: you can only delete your own messages")
	}

	// Check if already deleted
	if message.IsDeleted {
		return fmt.Errorf("message already deleted")
	}

	if err := s.messageRepo.SoftDeleteMessage(ctx, messageID); err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}

	return nil
}

// GetUnreadMessagesForUser retrieves all unread messages for a user
func (s *ChatService) GetUnreadMessagesForUser(ctx context.Context, userID uint) ([]models.UnreadMessage, error) {
	unreadMessages, err := s.unreadMessageRepo.GetByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get unread messages: %w", err)
	}
	return unreadMessages, nil
}

// CreateUnreadMessage creates a new unread message record
func (s *ChatService) CreateUnreadMessage(ctx context.Context, unreadMsg *models.UnreadMessage) error {
	if err := s.unreadMessageRepo.Create(ctx, unreadMsg); err != nil {
		return fmt.Errorf("failed to create unread message: %w", err)
	}
	return nil
}

// MarkChatAsRead marks all messages in a chat as read for a user
func (s *ChatService) MarkChatAsRead(ctx context.Context, userID, chatID uint) error {
	if err := s.unreadMessageRepo.DeleteByChat(ctx, userID, chatID); err != nil {
		return fmt.Errorf("failed to mark chat as read: %w", err)
	}
	return nil
}

// GetUnreadCounts returns count of unread messages per chat for a user
func (s *ChatService) GetUnreadCounts(ctx context.Context, userID uint) (map[uint]int64, error) {
	counts, err := s.unreadMessageRepo.GetUnreadCounts(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get unread counts: %w", err)
	}
	return counts, nil
}
