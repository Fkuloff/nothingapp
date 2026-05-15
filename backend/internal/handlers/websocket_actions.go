package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"messenger/internal/models"
	"messenger/internal/services"

	"go.uber.org/zap"
)

// pushBodyAttachmentPlaceholder is the fallback body shown in a push notification when the
// sender's message is whitespace-only — this happens when they post a message with only
// file attachments. The composer sends `text: ' '` to satisfy the non-empty-text rule, so
// without the substitution recipients would see an empty notification body.
const pushBodyAttachmentPlaceholder = "📎 Вложение"

// handleSendMessage processes a new message
func (h *webSocketHandler) handleSendMessage(ctx context.Context, userID uint, msgData messageAction) error {
	if msgData.Text == "" {
		return &wsError{message: "Message cannot be empty"}
	}
	if len(msgData.Text) > maxMessageSize {
		return &wsError{message: "Message too large (max 10KB)"}
	}

	// Check access to chat
	chat, err := h.chatService.FindChatByIDLight(ctx, msgData.ChatID)
	if err != nil {
		return &wsError{message: "Access denied to this chat"}
	}

	var message *models.Message

	if chat.IsGroup {
		message, err = h.handleSendGroupMessage(ctx, userID, chat, msgData)
	} else {
		message, err = h.handleSendDirectMessage(ctx, userID, chat, msgData)
	}
	if err != nil {
		return err
	}

	replyToIDVal := uint(0)
	if message.ReplyToID != nil {
		replyToIDVal = *message.ReplyToID
	}

	broadcastData := map[string]any{
		"action":      "new",
		"chat_id":     msgData.ChatID,
		"user_id":     userID,
		"text":        msgData.Text,
		"reply_to_id": replyToIDVal,
		"id":          message.ID,
		"created_at":  message.CreatedAt,
		"is_deleted":  false,
	}
	msgJSON, err := json.Marshal(broadcastData)
	if err != nil {
		h.logger.Error("json marshal error",
			zap.Error(err),
			zap.Uint("message_id", message.ID),
		)
		return &wsError{message: "Server error"}
	}

	// Broadcast to all participants (online users will receive it immediately)
	if err := h.broadcastToChat(ctx, msgData.ChatID, msgJSON); err != nil {
		h.logger.Error("failed to broadcast message",
			zap.Error(err),
			zap.Uint("chat_id", msgData.ChatID),
		)
	}

	return nil
}

// handleSendDirectMessage handles sending a message in a 1-on-1 chat.
func (h *webSocketHandler) handleSendDirectMessage(ctx context.Context, userID uint, chat *models.Chat, msgData messageAction) (*models.Message, error) {
	if !chat.HasUser(userID) {
		return nil, &wsError{message: "Access denied to this chat"}
	}

	otherUserID := chat.GetUser1ID()
	if chat.GetUser1ID() == userID {
		otherUserID = chat.GetUser2ID()
	}

	isRecipientOffline := !h.presenceService.IsUserOnline(otherUserID)

	msg, err := h.chatService.SendMessageAtomic(ctx, msgData.ChatID, userID, otherUserID, msgData.Text, msgData.ReplyToID, isRecipientOffline)
	if err != nil {
		h.logger.Error("error sending message",
			zap.Error(err),
			zap.Uint("user_id", userID),
			zap.Uint("chat_id", msgData.ChatID),
		)
		return nil, &wsError{message: "Failed to send message"}
	}

	// Push notification for direct message
	if h.pushService != nil && h.pushService.IsEnabled() {
		pushText := msgData.Text
		if runes := []rune(pushText); len(runes) > 200 {
			pushText = string(runes[:200]) + "..."
		}
		go h.sendPushNotification(userID, otherUserID, msgData.ChatID, pushText)
	}

	return msg, nil
}

// handleSendGroupMessage handles sending a message in a group chat.
func (h *webSocketHandler) handleSendGroupMessage(ctx context.Context, userID uint, chat *models.Chat, msgData messageAction) (*models.Message, error) {
	if h.participantRepo == nil {
		return nil, &wsError{message: "Group chat not supported"}
	}

	// Verify membership
	_, err := h.participantRepo.FindByUserAndChat(ctx, chat.ID, userID)
	if err != nil {
		return nil, &wsError{message: "Access denied to this chat"}
	}

	// Get all member IDs and determine who is offline
	memberIDs, err := h.participantRepo.GetParticipantUserIDs(ctx, chat.ID)
	if err != nil {
		return nil, &wsError{message: "Failed to get group members"}
	}

	var offlineUserIDs []uint
	for _, memberID := range memberIDs {
		if memberID != userID && !h.presenceService.IsUserOnline(memberID) {
			offlineUserIDs = append(offlineUserIDs, memberID)
		}
	}

	msg, err := h.chatService.SendMessageAtomicGroup(ctx, msgData.ChatID, userID, offlineUserIDs, msgData.Text, msgData.ReplyToID)
	if err != nil {
		h.logger.Error("error sending group message",
			zap.Error(err),
			zap.Uint("user_id", userID),
			zap.Uint("chat_id", msgData.ChatID),
		)
		return nil, &wsError{message: "Failed to send message"}
	}

	// Push notification for each offline group member
	if h.pushService != nil && h.pushService.IsEnabled() {
		pushText := msgData.Text
		if runes := []rune(pushText); len(runes) > 200 {
			pushText = string(runes[:200]) + "..."
		}
		for _, memberID := range offlineUserIDs {
			go h.sendPushNotification(userID, memberID, msgData.ChatID, pushText)
		}
	}

	return msg, nil
}

// sendPushNotification sends a push notification to a recipient in a background goroutine.
// Uses context.Background() intentionally: push delivery must not depend on the sender's WebSocket connection.
//
//nolint:contextcheck // intentionally detached from client context
func (h *webSocketHandler) sendPushNotification(senderID, recipientID, chatID uint, text string) {
	defer func() {
		if r := recover(); r != nil {
			h.logger.Error("panic in push notification goroutine",
				zap.Any("panic", r),
				zap.Uint("recipient_id", recipientID),
			)
		}
	}()

	h.logger.Debug("attempting push notification",
		zap.Uint("sender_id", senderID),
		zap.Uint("recipient_id", recipientID),
		zap.Uint("chat_id", chatID),
	)

	senderName := "New message"
	if sender, err := h.userService.GetUserByID(context.Background(), senderID); err == nil {
		senderName = sender.GetDisplayName()
	}

	body := text
	if strings.TrimSpace(body) == "" {
		body = pushBodyAttachmentPlaceholder
	}
	if utf8.RuneCountInString(body) > 200 {
		runes := []rune(body)
		body = string(runes[:200]) + "..."
	}

	payload := services.PushPayload{
		Title:  senderName,
		Body:   body,
		ChatID: chatID,
		UserID: senderID,
		Tag:    fmt.Sprintf("chat-%d", chatID),
	}

	if err := h.pushService.SendNotification(context.Background(), recipientID, payload); err != nil {
		h.logger.Error("failed to send push notification",
			zap.Error(err),
			zap.Uint("recipient_id", recipientID),
		)
	}
}

// handleEditMessage processes a message edit
func (h *webSocketHandler) handleEditMessage(ctx context.Context, userID uint, msgData messageAction) error {
	if msgData.MessageID == 0 {
		return &wsError{message: "Message ID required for edit"}
	}
	if msgData.Text == "" {
		return &wsError{message: "New message text cannot be empty"}
	}
	if len(msgData.Text) > maxMessageSize {
		return &wsError{message: "Message too large (max 10KB)"}
	}

	// Check access to chat
	if err := h.checkWSChatAccess(ctx, msgData.ChatID, userID); err != nil {
		return err
	}

	if err := h.chatService.EditMessage(ctx, msgData.MessageID, userID, msgData.Text); err != nil {
		h.logger.Error("error editing message",
			zap.Error(err),
			zap.Uint("message_id", msgData.MessageID),
			zap.Uint("user_id", userID),
		)
		return &wsError{message: "Failed to edit message"}
	}

	broadcastData := map[string]any{
		"action":  "edit",
		"chat_id": msgData.ChatID,
		"id":      msgData.MessageID,
		"text":    msgData.Text,
		"user_id": userID,
	}
	msgJSON, err := json.Marshal(broadcastData)
	if err != nil {
		h.logger.Error("json marshal error",
			zap.Error(err),
			zap.Uint("message_id", msgData.MessageID),
		)
		return &wsError{message: "Server error"}
	}

	if err := h.broadcastToChat(ctx, msgData.ChatID, msgJSON); err != nil {
		h.logger.Error("failed to broadcast edit",
			zap.Error(err),
			zap.Uint("chat_id", msgData.ChatID),
		)
	}
	return nil
}

// handleDeleteMessage processes a message deletion
func (h *webSocketHandler) handleDeleteMessage(ctx context.Context, userID uint, msgData messageAction) error {
	if msgData.MessageID == 0 {
		return &wsError{message: "Message ID required for delete"}
	}

	// Check access to chat
	if err := h.checkWSChatAccess(ctx, msgData.ChatID, userID); err != nil {
		return err
	}

	if err := h.chatService.DeleteMessage(ctx, msgData.MessageID, userID); err != nil {
		h.logger.Error("error deleting message",
			zap.Error(err),
			zap.Uint("message_id", msgData.MessageID),
			zap.Uint("user_id", userID),
		)
		return &wsError{message: "Failed to delete message"}
	}

	broadcastData := map[string]any{
		"action":     "delete",
		"chat_id":    msgData.ChatID,
		"id":         msgData.MessageID,
		"user_id":    userID,
		"is_deleted": true,
	}
	msgJSON, err := json.Marshal(broadcastData)
	if err != nil {
		h.logger.Error("json marshal error",
			zap.Error(err),
			zap.Uint("message_id", msgData.MessageID),
		)
		return &wsError{message: "Server error"}
	}

	if err := h.broadcastToChat(ctx, msgData.ChatID, msgJSON); err != nil {
		h.logger.Error("failed to broadcast delete",
			zap.Error(err),
			zap.Uint("chat_id", msgData.ChatID),
		)
	}
	return nil
}

// handleMarkRead marks messages as read
func (h *webSocketHandler) handleMarkRead(ctx context.Context, userID uint, msgData messageAction) error {
	if msgData.ChatID == 0 {
		return &wsError{message: "chat_id is required"}
	}

	// Delete all unread messages for this user in this chat
	if err := h.chatService.MarkChatAsRead(ctx, userID, msgData.ChatID); err != nil {
		h.logger.Error("failed to mark messages as read",
			zap.Error(err),
			zap.Uint("user_id", userID),
			zap.Uint("chat_id", msgData.ChatID),
		)
		return &wsError{message: "Failed to mark messages as read"}
	}

	return nil
}

// checkWSChatAccess verifies the user has access to the chat (1-on-1 or group).
func (h *webSocketHandler) checkWSChatAccess(ctx context.Context, chatID, userID uint) error {
	chat, err := h.chatService.FindChatByIDLight(ctx, chatID)
	if err != nil {
		return &wsError{message: "Access denied to this chat"}
	}

	if chat.IsGroup {
		if h.participantRepo == nil {
			return &wsError{message: "Group chat not supported"}
		}
		if _, err := h.participantRepo.FindByUserAndChat(ctx, chatID, userID); err != nil {
			return &wsError{message: "Access denied to this chat"}
		}
		return nil
	}

	if !chat.HasUser(userID) {
		return &wsError{message: "Access denied to this chat"}
	}
	return nil
}
