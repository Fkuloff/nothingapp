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
	db                *gorm.DB
	logger            *zap.Logger
	chatRepo          *repositories.ChatRepo
	messageRepo       *repositories.MessageRepo
	unreadMessageRepo *repositories.UnreadMessageRepo
}

func NewChatService(
	db *gorm.DB,
	logger *zap.Logger,
	chatRepo *repositories.ChatRepo,
	messageRepo *repositories.MessageRepo,
	unreadMessageRepo *repositories.UnreadMessageRepo,
) *ChatService {
	return &ChatService{
		db:                db,
		logger:            logger,
		chatRepo:          chatRepo,
		messageRepo:       messageRepo,
		unreadMessageRepo: unreadMessageRepo,
	}
}

func (s *ChatService) CreateChat(ctx context.Context, user1ID, user2ID uint) (*models.Chat, error) {
	if user1ID == user2ID {
		return nil, errors.New("cannot create chat with self")
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
		return nil, fmt.Errorf("check for existing chat: %w", err)
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
			return nil, fmt.Errorf("fetch existing chat after race condition: %w", fetchErr)
		}
		return nil, fmt.Errorf("create chat: %w", err)
	}
	return chat, nil
}

func (s *ChatService) SendMessage(ctx context.Context, chatID, userID uint, text string, replyToID uint) (*models.Message, error) {
	// CRITICAL: Verify user is participant in the chat (use light method for performance)
	chat, err := s.chatRepo.FindByIDLight(ctx, chatID)
	if err != nil {
		return nil, errors.New("chat not found")
	}

	if !chat.HasUser(userID) {
		return nil, errors.New("unauthorized: user is not a participant in this chat")
	}

	var replyPtr *uint
	if replyToID != 0 {
		msg, err := s.messageRepo.FindByID(ctx, replyToID)
		if err != nil || msg.ChatID != chatID {
			return nil, errors.New("invalid reply message")
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
		return nil, fmt.Errorf("create message: %w", err)
	}
	return message, nil
}

// SendMessageAtomic sends a message and creates unread record atomically within a transaction
// This prevents TOCTOU race conditions where a user could come online between the presence check and unread save
func (s *ChatService) SendMessageAtomic(ctx context.Context, chatID, userID, recipientID uint, text, iv string, replyToID uint, isRecipientOffline bool) (*models.Message, error) {
	var message *models.Message

	// Wrap message creation + unread save in a single transaction
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Use transaction-aware repositories
		chatRepo := s.chatRepo.WithTx(tx)
		messageRepo := s.messageRepo.WithTx(tx)
		unreadRepo := s.unreadMessageRepo.WithTx(tx)

		// Verify user access
		if err := s.validateUserAccess(ctx, chatRepo, chatID, userID); err != nil {
			return err
		}

		// Validate reply message if specified
		replyPtr, err := s.validateReplyMessage(ctx, messageRepo, replyToID, chatID)
		if err != nil {
			return err
		}

		// Create the message
		message = &models.Message{
			ChatID:    chatID,
			UserID:    userID,
			Text:      text,
			IV:        iv,
			ReplyToID: replyPtr,
		}

		if err := messageRepo.Create(ctx, message); err != nil {
			return fmt.Errorf("create message: %w", err)
		}

		// Update chat's updated_at to reflect the new message for proper sorting
		if err := tx.Model(&models.Chat{}).Where("id = ?", chatID).Update("updated_at", message.CreatedAt).Error; err != nil {
			return fmt.Errorf("update chat timestamp: %w", err)
		}

		// Create unread record if recipient is offline
		if err := s.createUnreadIfNeeded(ctx, unreadRepo, isRecipientOffline, recipientID, message.ID, chatID); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return message, nil
}

// validateUserAccess checks if user is a participant in the chat
func (s *ChatService) validateUserAccess(ctx context.Context, chatRepo *repositories.ChatRepo, chatID, userID uint) error {
	chat, err := chatRepo.FindByIDLight(ctx, chatID)
	if err != nil {
		return errors.New("chat not found")
	}

	if !chat.HasUser(userID) {
		return errors.New("unauthorized: user is not a participant in this chat")
	}

	return nil
}

// validateReplyMessage validates reply message exists and belongs to the same chat
func (s *ChatService) validateReplyMessage(ctx context.Context, messageRepo *repositories.MessageRepo, replyToID, chatID uint) (*uint, error) {
	if replyToID == 0 {
		return nil, nil //nolint:nilnil // nil pointer and nil error is valid when no reply message specified
	}

	msg, err := messageRepo.FindByID(ctx, replyToID)
	if err != nil || msg.ChatID != chatID {
		return nil, errors.New("invalid reply message")
	}

	return &replyToID, nil
}

// createUnreadIfNeeded creates an unread message record if recipient is offline
func (s *ChatService) createUnreadIfNeeded(ctx context.Context, unreadRepo *repositories.UnreadMessageRepo, isRecipientOffline bool, recipientID, messageID, chatID uint) error {
	if !isRecipientOffline {
		return nil
	}

	unreadMsg := &models.UnreadMessage{
		UserID:    recipientID,
		MessageID: messageID,
		ChatID:    chatID,
	}

	if err := unreadRepo.Create(ctx, unreadMsg); err != nil {
		// If unique constraint violated (duplicate), ignore the error
		// This can happen if the same message is marked unread twice (race condition)
		if !errors.Is(err, gorm.ErrDuplicatedKey) {
			return fmt.Errorf("create unread message: %w", err)
		}
	}

	return nil
}

// GetMessageByID returns a message by its ID
func (s *ChatService) GetMessageByID(ctx context.Context, messageID uint) (*models.Message, error) {
	message, err := s.messageRepo.FindByID(ctx, messageID)
	if err != nil {
		return nil, fmt.Errorf("find message: %w", err)
	}
	return message, nil
}

func (s *ChatService) GetMessages(ctx context.Context, chatID uint) ([]models.Message, error) {
	messages, err := s.messageRepo.GetAllByChatID(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}
	return messages, nil
}

// GetRecentMessages gets the most recent N messages for a chat
func (s *ChatService) GetRecentMessages(ctx context.Context, chatID uint, limit int) ([]models.Message, error) {
	messages, err := s.messageRepo.GetRecentByChatID(ctx, chatID, limit)
	if err != nil {
		return nil, fmt.Errorf("get recent messages: %w", err)
	}
	return messages, nil
}

// GetLastMessageForChat retrieves the most recent message for a chat
func (s *ChatService) GetLastMessageForChat(ctx context.Context, chatID uint) (*models.Message, error) {
	messages, err := s.messageRepo.GetRecentByChatID(ctx, chatID, 1)
	if err != nil {
		return nil, fmt.Errorf("get last message: %w", err)
	}
	if len(messages) == 0 {
		return nil, ErrNoMessages
	}
	return &messages[0], nil
}

// GetUserChats retrieves all chats for a user
// preloadUsers: if true, preloads User1 and User2 (use for display); if false, only loads IDs (use for presence/routing)
func (s *ChatService) GetUserChats(ctx context.Context, userID uint, preloadUsers bool) ([]models.Chat, error) {
	chats, err := s.chatRepo.GetUserChats(ctx, userID, preloadUsers)
	if err != nil {
		return nil, fmt.Errorf("get user chats: %w", err)
	}
	return chats, nil
}

func (s *ChatService) FindChatByID(ctx context.Context, id uint) (*models.Chat, error) {
	chat, err := s.chatRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find chat: %w", err)
	}
	return chat, nil
}

// FindChatByIDLight finds a chat without preloading users (for access checks only)
func (s *ChatService) FindChatByIDLight(ctx context.Context, id uint) (*models.Chat, error) {
	chat, err := s.chatRepo.FindByIDLight(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find chat: %w", err)
	}
	return chat, nil
}

// EditMessage allows a user to edit their own message
func (s *ChatService) EditMessage(ctx context.Context, messageID, userID uint, newText, iv string) error {
	// Find the message
	message, err := s.messageRepo.FindByID(ctx, messageID)
	if err != nil {
		return errors.New("message not found")
	}

	// Verify ownership
	if message.UserID != userID {
		return errors.New("unauthorized: you can only edit your own messages")
	}

	// Check if already deleted
	if message.IsDeleted {
		return errors.New("cannot edit deleted message")
	}

	// Validate new text
	if newText == "" || len(newText) > 10000 {
		return errors.New("invalid message length")
	}

	if err := s.messageRepo.UpdateMessage(ctx, messageID, newText, iv); err != nil {
		return fmt.Errorf("update message: %w", err)
	}

	return nil
}

// DeleteMessage allows a user to delete their own message
func (s *ChatService) DeleteMessage(ctx context.Context, messageID, userID uint) error {
	// Find the message
	message, err := s.messageRepo.FindByID(ctx, messageID)
	if err != nil {
		return errors.New("message not found")
	}

	// Verify ownership
	if message.UserID != userID {
		return errors.New("unauthorized: you can only delete your own messages")
	}

	// Check if already deleted
	if message.IsDeleted {
		return errors.New("message already deleted")
	}

	if err := s.messageRepo.SoftDeleteMessage(ctx, messageID); err != nil {
		return fmt.Errorf("delete message: %w", err)
	}

	return nil
}

// GetUnreadMessagesForUser retrieves all unread messages for a user
func (s *ChatService) GetUnreadMessagesForUser(ctx context.Context, userID uint) ([]models.UnreadMessage, error) {
	unreadMessages, err := s.unreadMessageRepo.GetByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get unread messages: %w", err)
	}
	return unreadMessages, nil
}

// CreateUnreadMessage creates a new unread message record
func (s *ChatService) CreateUnreadMessage(ctx context.Context, unreadMsg *models.UnreadMessage) error {
	if err := s.unreadMessageRepo.Create(ctx, unreadMsg); err != nil {
		return fmt.Errorf("create unread message: %w", err)
	}
	return nil
}

// MarkChatAsRead marks all messages in a chat as read for a user
func (s *ChatService) MarkChatAsRead(ctx context.Context, userID, chatID uint) error {
	if err := s.unreadMessageRepo.DeleteByChat(ctx, userID, chatID); err != nil {
		return fmt.Errorf("mark chat as read: %w", err)
	}
	return nil
}

// GetUnreadCounts returns count of unread messages per chat for a user
func (s *ChatService) GetUnreadCounts(ctx context.Context, userID uint) (map[uint]int64, error) {
	counts, err := s.unreadMessageRepo.GetUnreadCounts(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get unread counts: %w", err)
	}
	return counts, nil
}
