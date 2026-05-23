// internal/services/chat_service.go
package services

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"messenger/internal/models"
	"messenger/internal/repositories"
	"messenger/internal/storage"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	errNoMessages               = errors.New("no messages found")
	errMissingIVForClientScheme = errors.New("iv is required for client-encrypted scheme")
	errServerSchemeRemoved      = errors.New("server-side encryption (scheme=1) is no longer supported; please update your client")
)

// MessageEnvelopeInput is a single per-recipient ciphertext inside a group
// scheme=2 send. The sender (client) encrypts the plaintext once per
// recipient (including themselves) and ships the resulting envelopes here.
type MessageEnvelopeInput struct {
	RecipientID uint
	Ciphertext  string
	IV          string
}

// SendMessageInput captures the per-message payload for SendMessageAtomic /
// SendMessageAtomicGroup. After the scheme=1 removal there are exactly two
// valid shapes (the service rejects anything else):
//
//   - 1-on-1 scheme=2: Scheme == models.SchemeClientSide, Text + IV both
//     non-empty, no Envelopes. Server stores both as opaque blobs.
//   - Group scheme=2 pairwise: Scheme == models.SchemeClientSide, Envelopes
//     non-empty (one per current participant including the sender). Text + IV
//     on the persisted message row are empty by convention; per-recipient
//     ciphertexts live in the message_envelopes table.
type SendMessageInput struct {
	Text      string
	IV        string
	Scheme    uint8
	ReplyToID uint
	Envelopes []MessageEnvelopeInput
}

// ChatService handles business logic for chats, messages, and unread tracking.
type ChatService struct {
	db                *gorm.DB
	logger            *zap.Logger
	chatRepo          *repositories.ChatRepo
	participantRepo   *repositories.ChatParticipantRepo
	messageRepo       *repositories.MessageRepo
	envelopeRepo      *repositories.MessageEnvelopeRepo
	unreadMessageRepo *repositories.UnreadMessageRepo
	fileStorage       storage.Storage
}

// NewChatService creates a new ChatService.
func NewChatService(
	db *gorm.DB,
	logger *zap.Logger,
	chatRepo *repositories.ChatRepo,
	participantRepo *repositories.ChatParticipantRepo,
	messageRepo *repositories.MessageRepo,
	envelopeRepo *repositories.MessageEnvelopeRepo,
	unreadMessageRepo *repositories.UnreadMessageRepo,
	fileStorage storage.Storage,
) *ChatService {
	return &ChatService{
		db:                db,
		logger:            logger,
		chatRepo:          chatRepo,
		participantRepo:   participantRepo,
		messageRepo:       messageRepo,
		envelopeRepo:      envelopeRepo,
		unreadMessageRepo: unreadMessageRepo,
		fileStorage:       fileStorage,
	}
}

// prepareMessageBody normalizes a SendMessageInput into the (ciphertext, iv,
// scheme) triple that gets persisted. After the scheme=1 removal there are
// exactly two valid shapes:
//
//   - 1-on-1 scheme=2: Text + IV both non-empty, no Envelopes.
//   - Group scheme=2 pairwise: Envelopes non-empty, Text + IV ignored on the
//     row (per-recipient ciphertexts live in message_envelopes).
//
// Anything else (server-side scheme=1, missing IV, envelopes without scheme,
// etc.) is rejected — the server no longer encrypts on behalf of clients.
func (*ChatService) prepareMessageBody(in SendMessageInput) (ciphertext, iv string, scheme uint8, err error) {
	if in.Scheme != models.SchemeClientSide {
		return "", "", 0, errServerSchemeRemoved
	}
	if len(in.Envelopes) > 0 {
		// Group pairwise E2E: messages row carries no ciphertext, just the
		// scheme marker. Each recipient's ciphertext is in message_envelopes.
		return "", "", models.SchemeClientSide, nil
	}
	if in.IV == "" {
		return "", "", 0, errMissingIVForClientScheme
	}
	return in.Text, in.IV, models.SchemeClientSide, nil
}

// CreateChat creates a 1-on-1 chat between two users, returning the existing chat if one already exists.
// Self-chats (user1==user2) are rejected here — use EnsureFavoritesChat instead, which is the
// only sanctioned path for creating the special "Saved Messages" chat.
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
		User1ID: &user1ID,
		User2ID: &user2ID,
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

// EnsureFavoritesChat returns the user's "Saved Messages" chat, creating it if it doesn't
// exist yet. The favorites chat is a 1-on-1 chat where user1_id == user2_id == userID;
// the unique index on (user1_id, user2_id) guarantees idempotency — concurrent calls
// land on the same row. Called on registration and on startup backfill.
func (s *ChatService) EnsureFavoritesChat(ctx context.Context, userID uint) (*models.Chat, error) {
	existing, err := s.chatRepo.FindByUsers(ctx, userID, userID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("check for favorites chat: %w", err)
	}

	chat := &models.Chat{
		User1ID: &userID,
		User2ID: &userID,
	}
	if err := s.chatRepo.Create(ctx, chat); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			raceChat, fetchErr := s.chatRepo.FindByUsers(ctx, userID, userID)
			if fetchErr == nil {
				return raceChat, nil
			}
			return nil, fmt.Errorf("fetch favorites chat after race: %w", fetchErr)
		}
		return nil, fmt.Errorf("create favorites chat: %w", err)
	}
	return chat, nil
}

// EnsureFavoritesChatsForAllUsers backfills the favorites chat for every existing user.
// Run once at startup — idempotent thanks to the unique constraint. Errors per-user
// are logged but don't abort the batch.
func (s *ChatService) EnsureFavoritesChatsForAllUsers(ctx context.Context, userIDs []uint) {
	for _, uid := range userIDs {
		if _, err := s.EnsureFavoritesChat(ctx, uid); err != nil {
			s.logger.Warn("backfill favorites chat failed", zap.Uint("user_id", uid), zap.Error(err))
		}
	}
}

// IsFavoritesChat reports whether a chat is a user's self-chat (Saved Messages).
// For 1-on-1 chats only; group chats always return false (their user1/user2 are NULL).
func IsFavoritesChat(chat *models.Chat) bool {
	if chat.IsGroup {
		return false
	}
	return chat.User1ID != nil && chat.User2ID != nil && *chat.User1ID == *chat.User2ID
}

// SendMessageAtomic sends a message and creates unread record atomically within a transaction
// This prevents TOCTOU race conditions where a user could come online between the presence check and unread save.
//
// For scheme=SchemeClientSide messages the server stores the supplied ciphertext + IV
// verbatim and the returned Message keeps both populated so callers can broadcast them
// to peer clients. For scheme=SchemeServerSide (or 0) the legacy behavior applies:
// server encrypts, then the returned Message has Text restored to plaintext + IV cleared.
func (s *ChatService) SendMessageAtomic(ctx context.Context, chatID, userID, recipientID uint, in SendMessageInput, isRecipientOffline bool) (*models.Message, error) {
	ciphertext, storedIV, effectiveScheme, err := s.prepareMessageBody(in)
	if err != nil {
		return nil, err
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
		replyPtr, err := s.validateReplyMessage(ctx, messageRepo, in.ReplyToID, chatID)
		if err != nil {
			return err
		}

		// Create the message
		message = &models.Message{
			ChatID:    chatID,
			UserID:    userID,
			Text:      ciphertext,
			IV:        storedIV,
			Scheme:    effectiveScheme,
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

	// Hand back the same content the caller would broadcast to other clients:
	//   - scheme=1: plaintext + empty IV (matches legacy callers / clients).
	//   - scheme=2: original ciphertext + client-supplied IV intact.
	if effectiveScheme == models.SchemeClientSide {
		message.Text = in.Text
		message.IV = in.IV
	} else {
		message.Text = in.Text
		message.IV = ""
	}

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

// GetMessages returns the full message history of a chat addressed to userID.
// For group scheme=2 messages the per-user envelope is resolved into Text/IV
// so the caller sees a uniform per-message ciphertext shape regardless of
// whether the chat is 1-on-1 or group.
func (s *ChatService) GetMessages(ctx context.Context, chatID, userID uint) ([]models.Message, error) {
	messages, err := s.messageRepo.GetAllByChatID(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}
	if err := s.resolveEnvelopesForUser(ctx, messages, userID); err != nil {
		return nil, fmt.Errorf("resolve envelopes: %w", err)
	}
	return messages, nil
}

// GetRecentMessages gets the most recent N messages for a chat addressed to userID.
func (s *ChatService) GetRecentMessages(ctx context.Context, chatID, userID uint, limit int) ([]models.Message, error) {
	messages, err := s.messageRepo.GetRecentByChatID(ctx, chatID, limit)
	if err != nil {
		return nil, fmt.Errorf("get recent messages: %w", err)
	}
	if err := s.resolveEnvelopesForUser(ctx, messages, userID); err != nil {
		return nil, fmt.Errorf("resolve envelopes: %w", err)
	}
	return messages, nil
}

// GetLastMessageForChat retrieves the most recent non-deleted message for a chat.
// Deleted messages are skipped entirely so the chat-list preview never shows a "deleted" placeholder.
// For group scheme=2 messages userID's envelope is resolved so the chat-list preview
// path sees the same ciphertext shape as a 1-on-1 scheme=2 message — i.e. the
// "🔒 placeholder" path in formatLastMessage applies uniformly.
func (s *ChatService) GetLastMessageForChat(ctx context.Context, chatID, userID uint) (*models.Message, error) {
	msg, err := s.messageRepo.GetLastNonDeletedByChatID(ctx, chatID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errNoMessages
		}
		return nil, fmt.Errorf("get last message: %w", err)
	}
	messages := []models.Message{*msg}
	if err := s.resolveEnvelopesForUser(ctx, messages, userID); err != nil {
		return nil, fmt.Errorf("resolve envelopes: %w", err)
	}
	return &messages[0], nil
}

// FillEnvelopeForUser is the single-message helper for callers outside the
// chat-list / chat-window fetch path (notably WebSocket unread-replay) that
// already have one models.Message in hand and need to substitute its Text/IV
// with the recipient's per-user ciphertext.
//
// Returns true when an envelope was applied. Returns false (without error) for
// non-group / non-scheme=2 messages, or when this user has no envelope for the
// message (they joined the group after the message was sent). Callers should
// treat false on a group scheme=2 message as "skip — this user isn't a recipient."
func (s *ChatService) FillEnvelopeForUser(ctx context.Context, m *models.Message, userID uint) (bool, error) {
	if m == nil ||
		m.Scheme != models.SchemeClientSide ||
		m.Text != "" || m.IV != "" {
		return false, nil
	}
	envelopes, err := s.envelopeRepo.FindForRecipient(ctx, userID, []uint{m.ID})
	if err != nil {
		return false, err
	}
	env, ok := envelopes[m.ID]
	if !ok {
		return false, nil
	}
	m.Text = env.Ciphertext
	m.IV = env.IV
	return true, nil
}

// resolveEnvelopesForUser fills in Text/IV from message_envelopes for every
// group scheme=2 message in the slice, addressed to userID. Messages without
// an envelope for this user (because they joined after the message was sent,
// for example) keep their empty Text/IV and the client renders a "🔒" placeholder.
// Embedded ReplyTo previews go through the same lookup so reply quotes don't
// silently show empty bubbles.
func (s *ChatService) resolveEnvelopesForUser(ctx context.Context, messages []models.Message, userID uint) error {
	if len(messages) == 0 {
		return nil
	}
	targetIDs := collectEnvelopeTargets(messages)
	if len(targetIDs) == 0 {
		return nil
	}
	envelopes, err := s.envelopeRepo.FindForRecipient(ctx, userID, targetIDs)
	if err != nil {
		return err
	}
	applyEnvelopes(messages, envelopes)
	return nil
}

// needsEnvelopeResolution returns true for scheme=2 messages whose ciphertext
// hasn't been resolved yet (group envelope rows: Text/IV both empty).
func needsEnvelopeResolution(m *models.Message) bool {
	return m != nil &&
		m.Scheme == models.SchemeClientSide &&
		m.Text == "" && m.IV == ""
}

// collectEnvelopeTargets walks the message slice and returns the IDs of every
// message (plus embedded ReplyTo preview) that still needs envelope resolution.
func collectEnvelopeTargets(messages []models.Message) []uint {
	var ids []uint
	for i := range messages {
		m := &messages[i]
		if needsEnvelopeResolution(m) {
			ids = append(ids, m.ID)
		}
		if needsEnvelopeResolution(m.ReplyTo) {
			ids = append(ids, m.ReplyTo.ID)
		}
	}
	return ids
}

// applyEnvelopes copies ciphertext+iv from the envelope map back into the
// matching message rows (and any embedded ReplyTo previews).
func applyEnvelopes(messages []models.Message, envelopes map[uint]models.MessageEnvelope) {
	fill := func(m *models.Message) {
		if env, ok := envelopes[m.ID]; ok {
			m.Text = env.Ciphertext
			m.IV = env.IV
		}
	}
	for i := range messages {
		m := &messages[i]
		fill(m)
		if m.ReplyTo != nil {
			fill(m.ReplyTo)
		}
	}
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

// EditMessage allows a user to edit their own message. The newText / newIV /
// scheme trio follows the same convention as SendMessageInput — for 1-on-1
// scheme=2 edits pass new ciphertext + iv; for group scheme=2 edits pass the
// new envelope set in in.Envelopes and the server replaces the old envelopes
// wholesale.
//
// EditMessage refuses to switch a message between schemes — once a message
// is stored under one scheme, edits must keep that scheme (mixing would
// leave half the readers unable to decrypt).
func (s *ChatService) EditMessage(ctx context.Context, messageID, userID uint, in SendMessageInput) error {
	message, err := s.messageRepo.FindByID(ctx, messageID)
	if err != nil {
		return errors.New("message not found")
	}
	if err := validateEditPrechecks(message, userID, in); err != nil {
		return err
	}

	// Group scheme=2 edit: validate against current participants, replace the
	// whole envelope set, keep messages.text/iv empty.
	if len(in.Envelopes) > 0 {
		if err := s.validateGroupEnvelopes(ctx, message.ChatID, userID, in); err != nil {
			return err
		}
		return s.replaceEnvelopes(ctx, messageID, in.Envelopes)
	}

	ciphertext, newIV, _, err := s.prepareMessageBody(in)
	if err != nil {
		return err
	}
	if err := s.messageRepo.UpdateMessage(ctx, messageID, ciphertext, newIV); err != nil {
		return fmt.Errorf("update message: %w", err)
	}
	return nil
}

// validateEditPrechecks runs all the ownership / state / scheme-match checks
// that apply regardless of whether the edit is 1-on-1 or group envelopes.
// Extracted so EditMessage stays under the gocognit threshold.
func validateEditPrechecks(message *models.Message, userID uint, in SendMessageInput) error {
	if message.UserID != userID {
		return errors.New("unauthorized: you can only edit your own messages")
	}
	if message.IsDeleted {
		return errors.New("cannot edit deleted message")
	}
	// Envelope-based group edits leave in.Text empty by convention; only run
	// the length check on the other branches.
	if len(in.Envelopes) == 0 {
		if in.Text == "" || len(in.Text) > 10000 {
			return errors.New("invalid message length")
		}
	}
	// Disallow scheme switches mid-life. scheme==0 is treated as scheme=1 on
	// both sides for legacy rows that pre-date the column.
	storedScheme := message.Scheme
	if storedScheme == 0 {
		storedScheme = models.SchemeServerSide
	}
	requestedScheme := in.Scheme
	if requestedScheme == 0 {
		requestedScheme = models.SchemeServerSide
	}
	if requestedScheme != storedScheme {
		return errors.New("cannot change encryption scheme on edit")
	}
	return nil
}

// replaceEnvelopes wipes the existing envelope set for messageID and writes a
// fresh one inside a single transaction. The message row's edited_at gets
// bumped via UpdateMessage so callers don't need a second round-trip.
func (s *ChatService) replaceEnvelopes(ctx context.Context, messageID uint, in []MessageEnvelopeInput) error {
	envelopes := make([]models.MessageEnvelope, len(in))
	for i, e := range in {
		envelopes[i] = models.MessageEnvelope{
			MessageID:   messageID,
			RecipientID: e.RecipientID,
			Ciphertext:  e.Ciphertext,
			IV:          e.IV,
		}
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		envelopeRepo := s.envelopeRepo.WithTx(tx)
		messageRepo := s.messageRepo.WithTx(tx)
		if err := envelopeRepo.DeleteByMessageID(ctx, messageID); err != nil {
			return fmt.Errorf("delete old envelopes: %w", err)
		}
		if err := envelopeRepo.CreateBatch(ctx, envelopes); err != nil {
			return fmt.Errorf("create new envelopes: %w", err)
		}
		if err := messageRepo.UpdateMessage(ctx, messageID, "", ""); err != nil {
			return fmt.Errorf("touch message: %w", err)
		}
		return nil
	})
}

// DeleteMessage marks a message as deleted, wipes its content from the DB, and purges any attachments
// (both the rows and the files in object storage). The row itself is kept so that replies pointing
// at this message still render a "deleted" placeholder in their quote block.
func (s *ChatService) DeleteMessage(ctx context.Context, messageID, userID uint) error {
	message, err := s.messageRepo.FindByID(ctx, messageID)
	if err != nil {
		return errors.New("message not found")
	}

	if message.UserID != userID {
		return errors.New("unauthorized: you can only delete your own messages")
	}

	if message.IsDeleted {
		return errors.New("message already deleted")
	}

	// Collect attachment storage keys before we blow away the rows.
	var storageKeys []string
	if err := s.db.WithContext(ctx).Model(&models.Attachment{}).
		Where("message_id = ?", messageID).
		Pluck("storage_key", &storageKeys).Error; err != nil {
		return fmt.Errorf("fetch attachment keys: %w", err)
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Wipe attachment envelopes first (subquery on attachments) before
		// the attachments themselves disappear and the subquery returns empty.
		if err := tx.Where("attachment_id IN (SELECT id FROM attachments WHERE message_id = ?)", messageID).Delete(&models.AttachmentEnvelope{}).Error; err != nil {
			return fmt.Errorf("delete attachment envelopes: %w", err)
		}
		// Wipe attachment rows for this message
		if err := tx.Unscoped().Where("message_id = ?", messageID).Delete(&models.Attachment{}).Error; err != nil {
			return fmt.Errorf("delete attachments: %w", err)
		}
		// Drop unread entries so offline recipients don't see a ghost of this message
		if err := tx.Unscoped().Where("message_id = ?", messageID).Delete(&models.UnreadMessage{}).Error; err != nil {
			return fmt.Errorf("delete unread entries: %w", err)
		}
		// Drop pin entries — a deleted message can't remain pinned
		if err := tx.Unscoped().Where("message_id = ?", messageID).Delete(&models.PinnedMessage{}).Error; err != nil {
			return fmt.Errorf("delete pin entries: %w", err)
		}
		// Drop per-recipient envelopes (no-op for non-group / scheme=1 messages).
		if err := tx.Where("message_id = ?", messageID).Delete(&models.MessageEnvelope{}).Error; err != nil {
			return fmt.Errorf("delete envelopes: %w", err)
		}
		// Mark the message deleted and clear its content. The row stays so reply chains keep resolving.
		if err := tx.Model(&models.Message{}).
			Where("id = ?", messageID).
			Updates(map[string]interface{}{
				"is_deleted": true,
				"text":       "",
				"iv":         "",
			}).Error; err != nil {
			return fmt.Errorf("mark message deleted: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Best-effort storage cleanup (doesn't block the delete op)
	s.deleteStorageFiles(storageKeys)

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
	if IsFavoritesChat(chat) {
		return errors.New("favorites chat cannot be deleted, only cleared")
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

// SendMessageAtomicGroup sends a message in a group chat, creating unread
// records for all offline members. For group pairwise E2E (scheme=2 with
// in.Envelopes set) each envelope is a per-recipient ciphertext stored in
// message_envelopes; Message.Text/IV are empty on the row.
func (s *ChatService) SendMessageAtomicGroup(ctx context.Context, chatID, userID uint, offlineUserIDs []uint, in SendMessageInput) (*models.Message, error) {
	if err := s.validateGroupEnvelopes(ctx, chatID, userID, in); err != nil {
		return nil, err
	}
	ciphertext, storedIV, effectiveScheme, err := s.prepareMessageBody(in)
	if err != nil {
		return nil, err
	}

	var message *models.Message
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.persistGroupMessage(ctx, tx, persistGroupArgs{
			chatID:          chatID,
			userID:          userID,
			offlineUserIDs:  offlineUserIDs,
			in:              in,
			ciphertext:      ciphertext,
			storedIV:        storedIV,
			effectiveScheme: effectiveScheme,
			out:             &message,
		})
	})
	if err != nil {
		return nil, err
	}

	// Hand back what callers can broadcast to clients without re-fetching:
	//   - scheme=1: plaintext + empty IV.
	//   - scheme=2 1-on-1: caller-supplied ciphertext + iv (no envelopes).
	//   - scheme=2 group: empty Text/IV on the row; envelopes are the source of truth.
	switch {
	case effectiveScheme == models.SchemeClientSide && len(in.Envelopes) > 0:
		message.Text = ""
		message.IV = ""
	case effectiveScheme == models.SchemeClientSide:
		message.Text = in.Text
		message.IV = in.IV
	default:
		message.Text = in.Text
		message.IV = ""
	}
	return message, nil
}

// persistGroupArgs bundles the inputs to persistGroupMessage so the helper's
// signature stays short (otherwise we'd be over the linter's parameter cap
// on top of the cognitive-complexity cap we're trying to dodge).
type persistGroupArgs struct {
	in              SendMessageInput
	out             **models.Message
	ciphertext      string
	storedIV        string
	offlineUserIDs  []uint
	chatID          uint
	userID          uint
	effectiveScheme uint8
}

// persistGroupMessage runs the inner transactional body of
// SendMessageAtomicGroup: access check, reply validation, message row
// insertion, envelope batch insertion, chat-timestamp bump, unread fan-out.
// Extracted from SendMessageAtomicGroup so its caller's gocognit score stays
// under the linter threshold.
func (s *ChatService) persistGroupMessage(ctx context.Context, tx *gorm.DB, a persistGroupArgs) error {
	chatRepo := s.chatRepo.WithTx(tx)
	messageRepo := s.messageRepo.WithTx(tx)
	envelopeRepo := s.envelopeRepo.WithTx(tx)
	unreadRepo := s.unreadMessageRepo.WithTx(tx)

	if err := s.validateUserAccess(ctx, chatRepo, a.chatID, a.userID); err != nil {
		return err
	}
	replyPtr, err := s.validateReplyMessage(ctx, messageRepo, a.in.ReplyToID, a.chatID)
	if err != nil {
		return err
	}
	message := &models.Message{
		ChatID:    a.chatID,
		UserID:    a.userID,
		Text:      a.ciphertext,
		IV:        a.storedIV,
		Scheme:    a.effectiveScheme,
		ReplyToID: replyPtr,
		Type:      models.MessageTypeUser,
	}
	if err := messageRepo.Create(ctx, message); err != nil {
		return fmt.Errorf("create message: %w", err)
	}
	*a.out = message

	if len(a.in.Envelopes) > 0 {
		envelopes := make([]models.MessageEnvelope, len(a.in.Envelopes))
		for i, e := range a.in.Envelopes {
			envelopes[i] = models.MessageEnvelope{
				MessageID:   message.ID,
				RecipientID: e.RecipientID,
				Ciphertext:  e.Ciphertext,
				IV:          e.IV,
			}
		}
		if err := envelopeRepo.CreateBatch(ctx, envelopes); err != nil {
			return fmt.Errorf("create envelopes: %w", err)
		}
	}

	if err := tx.Model(&models.Chat{}).Where("id = ?", a.chatID).Update("updated_at", message.CreatedAt).Error; err != nil {
		return fmt.Errorf("update chat timestamp: %w", err)
	}
	for _, recipientID := range a.offlineUserIDs {
		if err := s.createUnreadIfNeeded(ctx, unreadRepo, true, recipientID, message.ID, a.chatID); err != nil {
			return err
		}
	}
	return nil
}

// validateGroupEnvelopes enforces the pairwise E2E shape for a group send:
// every current participant must be addressed exactly once (no duplicates,
// no missing recipients, no strangers). The lazy-vault modal in the client
// guarantees every active user has a public_key, so this strict invariant
// is honest in practice. Performed before opening the transaction so we
// can fail fast with a clean error.
func (s *ChatService) validateGroupEnvelopes(ctx context.Context, chatID, senderID uint, in SendMessageInput) error {
	if in.Scheme != models.SchemeClientSide || len(in.Envelopes) == 0 {
		return nil
	}
	participantIDs, err := s.participantRepo.GetParticipantUserIDs(ctx, chatID)
	if err != nil {
		return fmt.Errorf("fetch participants: %w", err)
	}
	want := make(map[uint]struct{}, len(participantIDs))
	for _, pid := range participantIDs {
		want[pid] = struct{}{}
	}
	if _, ok := want[senderID]; !ok {
		// Caught later by validateUserAccess too, but earlier here keeps the
		// error message tight.
		return errors.New("sender is not a participant in this group")
	}
	seen := make(map[uint]struct{}, len(in.Envelopes))
	for _, e := range in.Envelopes {
		if e.Ciphertext == "" || e.IV == "" {
			return errors.New("envelope ciphertext and iv must be non-empty")
		}
		if _, dup := seen[e.RecipientID]; dup {
			return fmt.Errorf("duplicate envelope for recipient %d", e.RecipientID)
		}
		if _, ok := want[e.RecipientID]; !ok {
			return fmt.Errorf("envelope for non-participant %d", e.RecipientID)
		}
		seen[e.RecipientID] = struct{}{}
	}
	if len(seen) != len(want) {
		return fmt.Errorf("envelopes cover %d of %d participants", len(seen), len(want))
	}
	return nil
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
		// Delete attachment envelopes (subquery on attachments) BEFORE the
		// attachments themselves are gone — same reasoning as DeleteMessage.
		if err := tx.Where("attachment_id IN (SELECT id FROM attachments WHERE message_id IN (SELECT id FROM messages WHERE chat_id = ?))", chatID).Delete(&models.AttachmentEnvelope{}).Error; err != nil {
			return fmt.Errorf("delete attachment envelopes: %w", err)
		}
		// Delete attachments
		if err := tx.Unscoped().Where("message_id IN (SELECT id FROM messages WHERE chat_id = ?)", chatID).Delete(&models.Attachment{}).Error; err != nil {
			return fmt.Errorf("delete attachments: %w", err)
		}
		// Delete per-recipient envelopes for all messages in this chat
		if err := tx.Where("message_id IN (SELECT id FROM messages WHERE chat_id = ?)", chatID).Delete(&models.MessageEnvelope{}).Error; err != nil {
			return fmt.Errorf("delete envelopes: %w", err)
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

// deleteStorageFiles removes files from storage in parallel (best-effort, errors are logged).
// A bounded pool keeps concurrency sane — deleting 10 attachments sequentially on a slow
// link easily stalls the caller for seconds; parallel deletes complete in ~1 round-trip.
func (s *ChatService) deleteStorageFiles(keys []string) {
	if len(keys) == 0 {
		return
	}

	const maxParallel = 8
	sem := make(chan struct{}, maxParallel)
	var wg sync.WaitGroup

	for _, key := range keys {
		wg.Add(1)
		sem <- struct{}{}
		go func(k string) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := s.fileStorage.Delete(k); err != nil {
				s.logger.Warn("failed to delete file from storage", zap.Error(err), zap.String("key", k))
			}
		}(key)
	}
	wg.Wait()
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
