// internal/handlers/chat.go
package handlers

import (
	"time"

	"messenger/internal/models"
	"messenger/internal/services"

	"github.com/gin-gonic/gin"
)

// ChatHandler handles HTTP API endpoints for chats
// WebSocket functionality moved to websocket.go
type ChatHandler struct {
	chatService *services.ChatService
	userService *services.UserService
}

func NewChatHandler(
	chatService *services.ChatService,
	userService *services.UserService,
) *ChatHandler {
	return &ChatHandler{
		chatService: chatService,
		userService: userService,
	}
}

type messageResponse struct {
	ID          uint                `json:"id"`
	ChatID      uint                `json:"chat_id"`
	UserID      uint                `json:"user_id"`
	Text        string              `json:"text"`
	IV          string              `json:"iv,omitempty"` // AES-GCM nonce; empty = plaintext
	IsDeleted   bool                `json:"is_deleted"`
	CreatedAt   time.Time           `json:"created_at"`
	ReplyToID   *uint               `json:"reply_to_id"`
	EditedAt    *time.Time          `json:"edited_at"`
	Attachments []models.Attachment `json:"attachments"`
}

// chatListItem is now defined in types.go as ChatListItem
// Keeping this as an alias for backward compatibility
type chatListItem = ChatListItem

func toMessageResponses(messages []models.Message) []messageResponse {
	result := make([]messageResponse, 0, len(messages))
	for _, msg := range messages {
		result = append(result, messageResponse{
			ID:          msg.ID,
			ChatID:      msg.ChatID,
			UserID:      msg.UserID,
			Text:        msg.Text,
			IV:          msg.IV,
			ReplyToID:   msg.ReplyToID,
			EditedAt:    msg.EditedAt,
			IsDeleted:   msg.IsDeleted,
			CreatedAt:   msg.CreatedAt,
			Attachments: msg.Attachments,
		})
	}
	return result
}

// formatLastMessage extracts display text and IV from the last message in a chat.
func formatLastMessage(lastMsg *models.Message, err error) (text, iv string) {
	if err != nil || lastMsg == nil {
		return "", ""
	}

	if lastMsg.IsDeleted {
		return "Message deleted", ""
	}

	text = lastMsg.Text
	iv = lastMsg.IV

	// Truncate plaintext previews only (encrypted messages are truncated client-side)
	if iv == "" {
		if runes := []rune(text); len(runes) > MaxChatListPreview {
			text = string(runes[:MaxChatListPreview])
		}
	}

	return text, iv
}

// GetChatData returns chat data with messages in JSON format for dynamic loading
func (h *ChatHandler) GetChatData(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	chatID, err := parseUintParam(c, "id")
	if err != nil {
		sendBadRequest(c, "Invalid chat ID")
		return
	}

	// Check access to chat
	chat, err := h.chatService.FindChatByID(c.Request.Context(), chatID)
	if err != nil || !chat.HasUser(userID) {
		sendForbidden(c, "Access denied")
		return
	}

	// Get recent messages
	messages, err := h.chatService.GetRecentMessages(c.Request.Context(), chatID, MaxRecentMessages)
	if err != nil {
		sendInternalError(c, "Failed to load messages")
		return
	}

	// Get other user info
	otherUser, otherUserID := chat.GetOtherUser(userID)

	sendSuccess(c, gin.H{
		"chatID":        chatID,
		"otherUserID":   otherUserID,
		"otherUsername": otherUser.GetDisplayName(),
		"messages":      toMessageResponses(messages),
	})
}

// ListChatsAPI returns list of chats for external UI
func (h *ChatHandler) ListChatsAPI(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	// Preload users for display (names, avatars)
	chats, err := h.chatService.GetUserChats(c.Request.Context(), userID, true)
	if err != nil {
		sendInternalError(c, "Failed to load chats")
		return
	}

	// Get unread counts for all chats
	unreadCounts, err := h.chatService.GetUnreadCounts(c.Request.Context(), userID)
	if err != nil {
		// Log error but continue - unread counts are not critical
		unreadCounts = make(map[uint]int64)
	}

	items := make([]chatListItem, 0, len(chats))
	for _, chat := range chats {
		otherUser, otherUserID := chat.GetOtherUser(userID)

		// Refresh avatar URL for S3 presigned URLs
		h.userService.RefreshUserAvatarURL(otherUser)

		lastMsg, err := h.chatService.GetLastMessageForChat(c.Request.Context(), chat.ID)
		lastMessageText, lastMessageIV := formatLastMessage(lastMsg, err)

		items = append(items, chatListItem{
			ID:            chat.ID,
			OtherUserID:   otherUserID,
			OtherUserName: otherUser.GetDisplayName(),
			AvatarURL:     otherUser.AvatarURL,
			LastMessage:   lastMessageText,
			LastMessageIV: lastMessageIV,
			UnreadCount:   int(unreadCounts[chat.ID]),
			UpdatedAt:     chat.UpdatedAt,
		})
	}

	sendSuccess(c, gin.H{"chats": items})
}

// CreateChatAPI creates a new chat via JSON request
func (h *ChatHandler) CreateChatAPI(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	var req struct {
		OtherUsername string `json:"other_username"`
		OtherUserID   uint   `json:"other_user_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		sendBadRequest(c, "Invalid input")
		return
	}

	var targetUserID uint
	switch {
	case req.OtherUserID != 0:
		targetUserID = req.OtherUserID
	case req.OtherUsername != "":
		otherUser, err := h.userService.FindByUsername(c.Request.Context(), req.OtherUsername)
		if err != nil {
			sendBadRequest(c, "User not found")
			return
		}
		targetUserID = otherUser.ID
	default:
		sendBadRequest(c, "other_user_id or other_username is required")
		return
	}

	if targetUserID == userID {
		sendBadRequest(c, "Cannot create chat with yourself")
		return
	}

	if req.OtherUserID != 0 {
		if _, err := h.userService.GetUserByID(c.Request.Context(), targetUserID); err != nil {
			sendBadRequest(c, "User not found")
			return
		}
	}

	chat, err := h.chatService.CreateChat(c.Request.Context(), userID, targetUserID)
	if err != nil {
		sendBadRequest(c, err.Error())
		return
	}

	sendCreated(c, gin.H{
		"id":         chat.ID,
		"user1_id":   chat.User1ID,
		"user2_id":   chat.User2ID,
		"created_at": chat.CreatedAt,
	})
}

// ClearChatAPI clears all messages in a chat
func (h *ChatHandler) ClearChatAPI(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	chatID, err := parseUintParam(c, "id")
	if err != nil {
		sendBadRequest(c, "Invalid chat ID")
		return
	}

	if err := h.chatService.ClearChat(c.Request.Context(), chatID, userID); err != nil {
		if err.Error() == "access denied" {
			sendForbidden(c, "Access denied")
		} else {
			sendInternalError(c, "Failed to clear chat")
		}
		return
	}

	sendSuccess(c, gin.H{"message": "Chat cleared"})
}

// DeleteChatAPI deletes a chat
func (h *ChatHandler) DeleteChatAPI(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	chatID, err := parseUintParam(c, "id")
	if err != nil {
		sendBadRequest(c, "Invalid chat ID")
		return
	}

	if err := h.chatService.DeleteChat(c.Request.Context(), chatID, userID); err != nil {
		if err.Error() == "access denied" {
			sendForbidden(c, "Access denied")
		} else {
			sendInternalError(c, "Failed to delete chat")
		}
		return
	}

	sendSuccess(c, gin.H{"message": "Chat deleted"})
}

// GetChatMessagesAPI returns chat messages for external UI
func (h *ChatHandler) GetChatMessagesAPI(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	chatID, err := parseUintParam(c, "id")
	if err != nil {
		sendBadRequest(c, "Invalid chat ID")
		return
	}

	chat, err := h.chatService.FindChatByID(c.Request.Context(), chatID)
	if err != nil || !chat.HasUser(userID) {
		sendForbidden(c, "Access denied")
		return
	}

	messages, err := h.chatService.GetMessages(c.Request.Context(), chatID)
	if err != nil {
		sendInternalError(c, "Failed to load messages")
		return
	}

	sendSuccess(c, gin.H{
		"messages": toMessageResponses(messages),
	})
}
