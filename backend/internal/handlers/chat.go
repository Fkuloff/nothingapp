// internal/handlers/chat.go
package handlers

import (
	"context"
	"time"

	"messenger/internal/models"
	"messenger/internal/services"

	"github.com/gin-gonic/gin"
)

// ChatHandler handles HTTP API endpoints for chats
// WebSocket functionality moved to websocket.go
type ChatHandler struct {
	chatService  *services.ChatService
	userService  *services.UserService
	groupService *services.GroupService
	onChatEvent  func(chatID, initiatorUserID uint, action string)
}

// SetOnChatEventCallback registers a callback for chat-level events (clear/delete).
// Called by WebSocketHandler to enable real-time sync between participants.
func (h *ChatHandler) SetOnChatEventCallback(cb func(chatID, initiatorUserID uint, action string)) {
	h.onChatEvent = cb
}

// SetGroupService sets the group service for group chat support in chat handlers.
func (h *ChatHandler) SetGroupService(gs *services.GroupService) {
	h.groupService = gs
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
	Type        string              `json:"type"`
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
		msgType := string(msg.Type)
		if msgType == "" {
			msgType = "user"
		}
		result = append(result, messageResponse{
			ID:          msg.ID,
			ChatID:      msg.ChatID,
			UserID:      msg.UserID,
			Text:        msg.Text,
			Type:        msgType,
			ReplyToID:   msg.ReplyToID,
			EditedAt:    msg.EditedAt,
			IsDeleted:   msg.IsDeleted,
			CreatedAt:   msg.CreatedAt,
			Attachments: msg.Attachments,
		})
	}
	return result
}

// formatLastMessage extracts display text from the last message in a chat.
func formatLastMessage(lastMsg *models.Message, err error) string {
	if err != nil || lastMsg == nil {
		return ""
	}

	if lastMsg.IsDeleted {
		return "Сообщение удалено"
	}

	text := lastMsg.Text
	if runes := []rune(text); len(runes) > MaxChatListPreview {
		text = string(runes[:MaxChatListPreview])
	}

	return text
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

	chat, err := h.chatService.FindChatByID(c.Request.Context(), chatID)
	if err != nil {
		sendForbidden(c, "Access denied")
		return
	}

	// Access check: group or 1-on-1
	if chat.IsGroup {
		if h.groupService == nil {
			sendInternalError(c, "Group service unavailable")
			return
		}
		inGroup, _ := h.groupService.IsUserInGroup(c.Request.Context(), chatID, userID)
		if !inGroup {
			sendForbidden(c, "Access denied")
			return
		}
	} else if !chat.HasUser(userID) {
		sendForbidden(c, "Access denied")
		return
	}

	messages, err := h.chatService.GetRecentMessages(c.Request.Context(), chatID, MaxRecentMessages)
	if err != nil {
		sendInternalError(c, "Failed to load messages")
		return
	}

	if chat.IsGroup {
		groupName := ""
		if chat.GroupName != nil {
			groupName = *chat.GroupName
		}
		memberCount := int64(0)
		if h.groupService != nil {
			members, _ := h.groupService.GetGroupMembers(c.Request.Context(), chatID)
			memberCount = int64(len(members))
		}
		sendSuccess(c, gin.H{
			"chatID":       chatID,
			"is_group":     true,
			"group_name":   groupName,
			"member_count": memberCount,
			"messages":     toMessageResponses(messages),
		})
		return
	}

	otherUser, otherUserID := chat.GetOtherUser(userID)
	sendSuccess(c, gin.H{
		"chatID":        chatID,
		"otherUserID":   otherUserID,
		"otherUsername": otherUser.GetDisplayName(),
		"messages":      toMessageResponses(messages),
	})
}

// ListChatsAPI returns list of chats for external UI (1-on-1 + groups)
func (h *ChatHandler) ListChatsAPI(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	chats, err := h.chatService.GetUserChats(c.Request.Context(), userID, true)
	if err != nil {
		sendInternalError(c, "Failed to load chats")
		return
	}

	unreadCounts, err := h.chatService.GetUnreadCounts(c.Request.Context(), userID)
	if err != nil {
		unreadCounts = make(map[uint]int64)
	}

	items := make([]chatListItem, 0, len(chats))
	for _, chat := range chats {
		lastMsg, lErr := h.chatService.GetLastMessageForChat(c.Request.Context(), chat.ID)
		lastMessageText := formatLastMessage(lastMsg, lErr)

		if chat.IsGroup {
			groupName := ""
			if chat.GroupName != nil {
				groupName = *chat.GroupName
			}
			var avatarURL *string
			if chat.AvatarURL != nil && *chat.AvatarURL != "" {
				url := h.userService.GetAvatarURL(chat.AvatarURL)
				avatarURL = url
			}
			memberCount := 0
			if h.groupService != nil {
				cnt, _ := h.groupService.GetParticipantUserIDs(c.Request.Context(), chat.ID)
				memberCount = len(cnt)
			}
			items = append(items, chatListItem{
				ID:          chat.ID,
				IsGroup:     true,
				GroupName:   groupName,
				MemberCount: memberCount,
				AvatarURL:   avatarURL,
				LastMessage: lastMessageText,
				UnreadCount: int(unreadCounts[chat.ID]),
				UpdatedAt:   chat.UpdatedAt,
			})
		} else {
			otherUser, otherUserID := chat.GetOtherUser(userID)
			h.userService.RefreshUserAvatarURL(otherUser)

			items = append(items, chatListItem{
				ID:            chat.ID,
				IsGroup:       false,
				OtherUserID:   otherUserID,
				OtherUserName: otherUser.GetDisplayName(),
				AvatarURL:     otherUser.AvatarURL,
				LastMessage:   lastMessageText,
				UnreadCount:   int(unreadCounts[chat.ID]),
				UpdatedAt:     chat.UpdatedAt,
			})
		}
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

// chatAction executes a chat operation (clear/delete) with shared validation logic.
// Broadcasts the event to the other participant BEFORE the destructive DB action,
// because DeleteChat hard-deletes the chat record (FindChatByIDLight would fail after).
func (h *ChatHandler) chatAction(c *gin.Context, action func(ctx context.Context, chatID, userID uint) error, failMsg, successMsg, eventAction string) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	chatID, err := parseUintParam(c, "id")
	if err != nil {
		sendBadRequest(c, "Invalid chat ID")
		return
	}

	// Pre-validate access
	chat, err := h.chatService.FindChatByIDLight(c.Request.Context(), chatID)
	if err != nil {
		sendForbidden(c, "Access denied")
		return
	}

	if chat.IsGroup {
		if h.groupService == nil {
			sendInternalError(c, "Group service unavailable")
			return
		}
		inGroup, _ := h.groupService.IsUserInGroup(c.Request.Context(), chatID, userID)
		if !inGroup {
			sendForbidden(c, "Access denied")
			return
		}
	} else if !chat.HasUser(userID) {
		sendForbidden(c, "Access denied")
		return
	}

	// Broadcast BEFORE destructive action (chat record must exist for participant lookup)
	if h.onChatEvent != nil {
		h.onChatEvent(chatID, userID, eventAction)
	}

	if err := action(c.Request.Context(), chatID, userID); err != nil {
		errMsg := err.Error()
		if errMsg == "access denied" || errMsg == "admin or creator role required" {
			sendForbidden(c, errMsg)
		} else {
			sendInternalError(c, failMsg)
		}
		return
	}

	sendSuccess(c, gin.H{"message": successMsg})
}

// ClearChatAPI clears all messages in a chat
func (h *ChatHandler) ClearChatAPI(c *gin.Context) {
	h.chatAction(c, h.chatService.ClearChat, "Failed to clear chat", "Chat cleared", "chat_cleared")
}

// DeleteChatAPI deletes a chat
func (h *ChatHandler) DeleteChatAPI(c *gin.Context) {
	h.chatAction(c, h.chatService.DeleteChat, "Failed to delete chat", "Chat deleted", "chat_deleted")
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

	chat, err := h.chatService.FindChatByIDLight(c.Request.Context(), chatID)
	if err != nil {
		sendForbidden(c, "Access denied")
		return
	}

	// Access check
	if chat.IsGroup {
		if h.groupService == nil {
			sendInternalError(c, "Group service unavailable")
			return
		}
		inGroup, _ := h.groupService.IsUserInGroup(c.Request.Context(), chatID, userID)
		if !inGroup {
			sendForbidden(c, "Access denied")
			return
		}
	} else if !chat.HasUser(userID) {
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
