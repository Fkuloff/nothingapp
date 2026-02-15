// internal/handlers/websocket_actions.go
package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"unicode/utf8"

	"messenger/internal/services"

	"go.uber.org/zap"
)

// handleSendMessage processes a new message
func (h *WebSocketHandler) handleSendMessage(ctx context.Context, userID uint, msgData MessageAction) error {
	if msgData.Text == "" {
		return &wsError{message: "Message cannot be empty"}
	}
	if len(msgData.Text) > MaxMessageSize {
		return &wsError{message: "Message too large (max 10KB)"}
	}

	// Check access to chat and get recipient ID
	chat, err := h.chatService.FindChatByIDLight(ctx, msgData.ChatID)
	if err != nil || !chat.HasUser(userID) {
		return &wsError{message: "Access denied to this chat"}
	}

	// Get other user ID (recipient)
	otherUserID := chat.User1ID
	if chat.User1ID == userID {
		otherUserID = chat.User2ID
	}

	// Check if recipient is offline BEFORE transaction to minimize lock time
	isRecipientOffline := !h.presenceService.IsUserOnline(otherUserID)

	// ATOMIC: Send message and create unread record in single transaction
	// This prevents TOCTOU race where user comes online between presence check and unread save
	message, err := h.chatService.SendMessageAtomic(ctx, msgData.ChatID, userID, otherUserID, msgData.Text, msgData.ReplyToID, isRecipientOffline)
	if err != nil {
		h.logger.Error("error sending message",
			zap.Error(err),
			zap.Uint("user_id", userID),
			zap.Uint("chat_id", msgData.ChatID),
		)
		return &wsError{message: "Failed to send message"}
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

	// Send push notification with actual message text (truncated to 200 runes)
	if h.pushService != nil && h.pushService.IsEnabled() {
		pushText := msgData.Text
		if runes := []rune(pushText); len(runes) > 200 {
			pushText = string(runes[:200]) + "..."
		}
		go h.sendPushNotification(userID, otherUserID, msgData.ChatID, pushText)
	}

	return nil
}

// sendPushNotification sends a push notification to a recipient in a background goroutine.
// Uses context.Background() intentionally: push delivery must not depend on the sender's WebSocket connection.
//
//nolint:contextcheck // intentionally detached from client context
func (h *WebSocketHandler) sendPushNotification(senderID, recipientID, chatID uint, text string) {
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
func (h *WebSocketHandler) handleEditMessage(ctx context.Context, userID uint, msgData MessageAction) error {
	if msgData.MessageID == 0 {
		return &wsError{message: "Message ID required for edit"}
	}
	if msgData.Text == "" {
		return &wsError{message: "New message text cannot be empty"}
	}
	if len(msgData.Text) > MaxMessageSize {
		return &wsError{message: "Message too large (max 10KB)"}
	}

	// Check access to chat
	chat, err := h.chatService.FindChatByIDLight(ctx, msgData.ChatID)
	if err != nil || !chat.HasUser(userID) {
		return &wsError{message: "Access denied to this chat"}
	}

	err = h.chatService.EditMessage(ctx, msgData.MessageID, userID, msgData.Text)
	if err != nil {
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
func (h *WebSocketHandler) handleDeleteMessage(ctx context.Context, userID uint, msgData MessageAction) error {
	if msgData.MessageID == 0 {
		return &wsError{message: "Message ID required for delete"}
	}

	// Check access to chat
	chat, err := h.chatService.FindChatByIDLight(ctx, msgData.ChatID)
	if err != nil || !chat.HasUser(userID) {
		return &wsError{message: "Access denied to this chat"}
	}

	err = h.chatService.DeleteMessage(ctx, msgData.MessageID, userID)
	if err != nil {
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
func (h *WebSocketHandler) handleMarkRead(ctx context.Context, userID uint, msgData MessageAction) error {
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
