// internal/services/chat_service.go
package services

import (
	"context"
	"errors"
	"fmt"

	"messenger/internal/crypto"
	"messenger/internal/models"
	"messenger/internal/repositories"
	"messenger/internal/storage"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var errNoMessages = errors.New("no messages found")

// ChatService handles business logic for chats, messages, and unread tracking.
type ChatService struct {
	db                *gorm.DB
	logger            *zap.Logger
	chatRepo          *repositories.ChatRepo
	participantRepo   *repositories.ChatParticipantRepo
	messageRepo       *repositories.MessageRepo
	unreadMessageRepo *repositories.UnreadMessageRepo
	fileStorage       storage.Storage
	encryptor         *crypto.MessageEncryptor
}

// NewChatService creates a new ChatService.
func NewChatService(
	db *gorm.DB,
	logger *zap.Logger,
	chatRepo *repositories.ChatRepo,
	participantRepo *repositories.ChatParticipantRepo,
	messageRepo *repositories.MessageRepo,
	unreadMessageRepo *repositories.UnreadMessageRepo,
	fileStorage storage.Storage,
	encryptor *crypto.MessageEncryptor,
) *ChatService {
	return &ChatService{
		db:                db,
		logger:            logger,
		chatRepo:          chatRepo,
		participantRepo:   participantRepo,
		messageRepo:       messageRepo,
		unreadMessageRepo: unreadMessageRepo,
		fileStorage:       fileStorage,
		encryptor:         encryptor,
	}
}

// CreateChat creates a 1-on-1 chat between two users, returning the existing chat if one already exists.
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

// SendMessageAtomic sends a message and creates unread record atomically within a transaction
// This prevents TOCTOU race conditions where a user could come online between the presence check and unread save
func (s *ChatService) SendMessageAtomic(ctx context.Context, chatID, userID, recipientID uint, text string, replyToID uint, isRecipientOffline bool) (*models.Message, error) {
	// Encrypt message text before saving
	ciphertext, iv, err := s.encryptor.Encrypt(text)
	if err != nil {
		return nil, fmt.Errorf("encrypt message: %w", err)
	}

	var message *models.Message

	// Wrap message creation + unread save in a single transaction
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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
			Text:      ciphertext,
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

	// Restore plaintext for the caller (e.g. broadcast)
	message.Text = text
	message.IV = ""

	return message, nil
}

// validateUserAccess checks if user is a participant in the chat (1-on-1 or group)
func (s *ChatService) validateUserAccess(ctx context.Context, chatRepo *repositories.ChatRepo, chatID, userID uint) error {
	chat, err := chatRepo.FindByIDLight(ctx, chatID)
	if err != nil {
		return errors.New("chat not found")
	}

	if chat.IsGroup {
		_, err := s.participantRepo.FindByUserAndChat(ctx, chatID, userID)
		if err != nil {
			return errors.New("unauthorized: user is not a participant in this group")
		}
		return nil
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

func (s *ChatService) GetMessages(ctx context.Context, chatID uint) ([]models.Message, error) {
	messages, err := s.messageRepo.GetAllByChatID(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}
	s.decryptMessages(messages)
	return messages, nil
}

// GetRecentMessages gets the most recent N messages for a chat
func (s *ChatService) GetRecentMessages(ctx context.Context, chatID uint, limit int) ([]models.Message, error) {
	messages, err := s.messageRepo.GetRecentByChatID(ctx, chatID, limit)
	if err != nil {
		return nil, fmt.Errorf("get recent messages: %w", err)
	}
	s.decryptMessages(messages)
	return messages, nil
}

// GetLastMessageForChat retrieves the most recent message for a chat
func (s *ChatService) GetLastMessageForChat(ctx context.Context, chatID uint) (*models.Message, error) {
	messages, err := s.messageRepo.GetRecentByChatID(ctx, chatID, 1)
	if err != nil {
		return nil, fmt.Errorf("get last message: %w", err)
	}
	if len(messages) == 0 {
		return nil, errNoMessages
	}
	s.decryptMessages(messages)
	return &messages[0], nil
}

// GetUserChats retrieves all chats for a user (1-on-1 + groups)
// preloadUsers: if true, preloads User1 and User2 (use for display); if false, only loads IDs (use for presence/routing)
func (s *ChatService) GetUserChats(ctx context.Context, userID uint, preloadUsers bool) ([]models.Chat, error) {
	groupChatIDs, err := s.participantRepo.GetUserGroupChatIDs(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user group chats: %w", err)
	}

	chats, err := s.chatRepo.GetUserChatsIncludingGroups(ctx, userID, groupChatIDs, preloadUsers)
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
func (s *ChatService) EditMessage(ctx context.Context, messageID, userID uint, newText string) error {
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

	// Encrypt before update
	ciphertext, newIV, err := s.encryptor.Encrypt(newText)
	if err != nil {
		return fmt.Errorf("encrypt message: %w", err)
	}

	if err := s.messageRepo.UpdateMessage(ctx, messageID, ciphertext, newIV); err != nil {
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

// ClearChat hard-deletes all messages, attachments and unread records in a chat.
// For groups, only admin/creator can clear.
func (s *ChatService) ClearChat(ctx context.Context, chatID, userID uint) error {
	chat, err := s.chatRepo.FindByIDLight(ctx, chatID)
	if err != nil {
		return errors.New("chat not found")
	}

	if err := s.checkChatAccess(ctx, chat, userID, true); err != nil {
		return err
	}

	storageKeys, err := s.purgeMessagesAndAttachments(ctx, chatID)
	if err != nil {
		return err
	}
	s.deleteStorageFiles(storageKeys)
	return nil
}

// DeleteChat hard-deletes a chat and all its messages, attachments, and unread records.
// For groups, use GroupService.DeleteGroup instead — this is for 1-on-1 only.
func (s *ChatService) DeleteChat(ctx context.Context, chatID, userID uint) error {
	chat, err := s.chatRepo.FindByIDLight(ctx, chatID)
	if err != nil {
		return errors.New("chat not found")
	}
	if chat.IsGroup {
		return errors.New("use group delete endpoint for group chats")
	}
	if !chat.HasUser(userID) {
		return errors.New("access denied")
	}

	storageKeys, err := s.purgeChat(ctx, chatID)
	if err != nil {
		return err
	}
	s.deleteStorageFiles(storageKeys)
	return nil
}

// checkChatAccess verifies user has access to the chat. For groups with requireAdmin=true, checks admin role.
func (s *ChatService) checkChatAccess(ctx context.Context, chat *models.Chat, userID uint, requireAdmin bool) error {
	if chat.IsGroup {
		p, err := s.participantRepo.FindByUserAndChat(ctx, chat.ID, userID)
		if err != nil {
			return errors.New("access denied")
		}
		if requireAdmin && !p.IsAdmin() {
			return errors.New("admin or creator role required")
		}
		return nil
	}
	if !chat.HasUser(userID) {
		return errors.New("access denied")
	}
	return nil
}

// SendMessageAtomicGroup sends a message in a group chat, creating unread records for all offline members.
func (s *ChatService) SendMessageAtomicGroup(ctx context.Context, chatID, userID uint, offlineUserIDs []uint, text string, replyToID uint) (*models.Message, error) {
	ciphertext, iv, err := s.encryptor.Encrypt(text)
	if err != nil {
		return nil, fmt.Errorf("encrypt message: %w", err)
	}

	var message *models.Message

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		chatRepo := s.chatRepo.WithTx(tx)
		messageRepo := s.messageRepo.WithTx(tx)
		unreadRepo := s.unreadMessageRepo.WithTx(tx)

		if err := s.validateUserAccess(ctx, chatRepo, chatID, userID); err != nil {
			return err
		}

		replyPtr, err := s.validateReplyMessage(ctx, messageRepo, replyToID, chatID)
		if err != nil {
			return err
		}

		message = &models.Message{
			ChatID:    chatID,
			UserID:    userID,
			Text:      ciphertext,
			IV:        iv,
			ReplyToID: replyPtr,
			Type:      models.MessageTypeUser,
		}

		if err := messageRepo.Create(ctx, message); err != nil {
			return fmt.Errorf("create message: %w", err)
		}

		if err := tx.Model(&models.Chat{}).Where("id = ?", chatID).Update("updated_at", message.CreatedAt).Error; err != nil {
			return fmt.Errorf("update chat timestamp: %w", err)
		}

		for _, recipientID := range offlineUserIDs {
			if err := s.createUnreadIfNeeded(ctx, unreadRepo, true, recipientID, message.ID, chatID); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	message.Text = text
	message.IV = ""
	return message, nil
}

// purgeMessagesAndAttachments hard-deletes messages, attachments and unread records for a chat.
// Returns storage keys of deleted attachments for cleanup.
func (s *ChatService) purgeMessagesAndAttachments(ctx context.Context, chatID uint) ([]string, error) {
	// Collect attachment storage keys before deleting
	var storageKeys []string
	if err := s.db.WithContext(ctx).Model(&models.Attachment{}).
		Where("message_id IN (SELECT id FROM messages WHERE chat_id = ?)", chatID).
		Pluck("storage_key", &storageKeys).Error; err != nil {
		return nil, fmt.Errorf("fetch attachment keys: %w", err)
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Delete pinned messages
		if err := tx.Unscoped().Where("chat_id = ?", chatID).Delete(&models.PinnedMessage{}).Error; err != nil {
			return fmt.Errorf("delete pinned messages: %w", err)
		}
		// Delete unread messages
		if err := tx.Unscoped().Where("chat_id = ?", chatID).Delete(&models.UnreadMessage{}).Error; err != nil {
			return fmt.Errorf("delete unread messages: %w", err)
		}
		// Delete attachments
		if err := tx.Unscoped().Where("message_id IN (SELECT id FROM messages WHERE chat_id = ?)", chatID).Delete(&models.Attachment{}).Error; err != nil {
			return fmt.Errorf("delete attachments: %w", err)
		}
		// Clear reply references to avoid FK constraint issues
		if err := tx.Model(&models.Message{}).Where("chat_id = ? AND reply_to_id IS NOT NULL", chatID).Update("reply_to_id", nil).Error; err != nil {
			return fmt.Errorf("clear reply references: %w", err)
		}
		// Hard-delete messages
		if err := tx.Unscoped().Where("chat_id = ?", chatID).Delete(&models.Message{}).Error; err != nil {
			return fmt.Errorf("delete messages: %w", err)
		}
		return nil
	})

	return storageKeys, err
}

// purgeChat hard-deletes a chat and all its data. Returns storage keys for cleanup.
func (s *ChatService) purgeChat(ctx context.Context, chatID uint) ([]string, error) {
	storageKeys, err := s.purgeMessagesAndAttachments(ctx, chatID)
	if err != nil {
		return nil, err
	}

	// Hard-delete the chat itself (Unscoped bypasses GORM soft-delete)
	if err := s.db.WithContext(ctx).Unscoped().Delete(&models.Chat{}, chatID).Error; err != nil {
		return storageKeys, fmt.Errorf("delete chat: %w", err)
	}

	return storageKeys, nil
}

// deleteStorageFiles removes files from storage (best-effort, errors are logged)
func (s *ChatService) deleteStorageFiles(keys []string) {
	for _, key := range keys {
		if err := s.fileStorage.Delete(key); err != nil {
			s.logger.Warn("failed to delete file from storage", zap.Error(err), zap.String("key", key))
		}
	}
}

// GetUnreadMessagesForUser retrieves all unread messages for a user
func (s *ChatService) GetUnreadMessagesForUser(ctx context.Context, userID uint) ([]models.UnreadMessage, error) {
	unreadMessages, err := s.unreadMessageRepo.GetByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get unread messages: %w", err)
	}
	return unreadMessages, nil
}

// MarkChatAsRead marks all messages in a chat as read for a user
func (s *ChatService) MarkChatAsRead(ctx context.Context, userID, chatID uint) error {
	if err := s.unreadMessageRepo.DeleteByChat(ctx, userID, chatID); err != nil {
		return fmt.Errorf("mark chat as read: %w", err)
	}
	return nil
}

// DeleteUnreadForUser removes all unread records for a user (used after pending message delivery)
func (s *ChatService) DeleteUnreadForUser(ctx context.Context, userID uint) error {
	if err := s.unreadMessageRepo.DeleteByUser(ctx, userID); err != nil {
		return fmt.Errorf("delete unread for user: %w", err)
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

// decryptMessages decrypts all messages in a slice (including ReplyTo).
func (s *ChatService) decryptMessages(messages []models.Message) {
	for i := range messages {
		if messages[i].IV != "" && messages[i].Text != "" && !messages[i].IsDeleted {
			if plaintext, err := s.encryptor.Decrypt(messages[i].Text, messages[i].IV); err == nil {
				messages[i].Text = plaintext
			} else {
				messages[i].Text = "[Ошибка расшифровки]"
			}
			messages[i].IV = ""
		}
		if messages[i].ReplyTo != nil && messages[i].ReplyTo.IV != "" && !messages[i].ReplyTo.IsDeleted {
			if plaintext, err := s.encryptor.Decrypt(messages[i].ReplyTo.Text, messages[i].ReplyTo.IV); err == nil {
				messages[i].ReplyTo.Text = plaintext
			} else {
				messages[i].ReplyTo.Text = "[Ошибка расшифровки]"
			}
			messages[i].ReplyTo.IV = ""
		}
	}
}
