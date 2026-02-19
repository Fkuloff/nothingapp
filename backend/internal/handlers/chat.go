package handlers

import (
	"context"
	"time"

	"messenger/internal/models"
	"messenger/internal/services"
	"messenger/internal/storage"

	"github.com/gin-gonic/gin"
)

// chatHandler handles HTTP API endpoints for chats.
type chatHandler struct {
	chatService  *services.ChatService
	userService  *services.UserService
	groupService *services.GroupService
	storage      storage.Storage
	onChatEvent  func(chatID, initiatorUserID uint, action string)
}

// SetOnChatEventCallback registers a callback for chat-level events (clear/delete).
// Called by webSocketHandler to enable real-time sync between participants.
func (h *chatHandler) SetOnChatEventCallback(cb func(chatID, initiatorUserID uint, action string)) {
	h.onChatEvent = cb
}

// SetGroupService sets the group service for group chat support in chat handlers.
func (h *chatHandler) SetGroupService(gs *services.GroupService) {
	h.groupService = gs
}

// newChatHandler creates a new chatHandler.
func newChatHandler(
	chatService *services.ChatService,
	userService *services.UserService,
	fileStorage storage.Storage,
) *chatHandler {
	return &chatHandler{
		chatService: chatService,
		userService: userService,
		storage:     fileStorage,
	}
}

type attachmentResponse struct {
	ID       uint                  `json:"id"`
	FileType models.AttachmentType `json:"file_type"`
	FileName string                `json:"file_name"`
	FileSize int64                 `json:"file_size"`
	MimeType string                `json:"mime_type"`
	URL      string                `json:"url"`
}

type messageResponse struct {
	ID          uint                 `json:"id"`
	ChatID      uint                 `json:"chat_id"`
	UserID      uint                 `json:"user_id"`
	Text        string               `json:"text"`
	Type        string               `json:"type"`
	IsDeleted   bool                 `json:"is_deleted"`
	CreatedAt   time.Time            `json:"created_at"`
	ReplyToID   *uint                `json:"reply_to_id"`
	EditedAt    *time.Time           `json:"edited_at"`
	Attachments []attachmentResponse `json:"attachments"`
}

func (h *chatHandler) toMessageResponses(messages []models.Message) []messageResponse {
	result := make([]messageResponse, 0, len(messages))
	for _, msg := range messages {
		msgType := string(msg.Type)
		if msgType == "" {
			msgType = "user"
		}

		atts := make([]attachmentResponse, 0, len(msg.Attachments))
		for _, att := range msg.Attachments {
			atts = append(atts, attachmentResponse{
				ID:       att.ID,
				FileType: att.FileType,
				FileName: att.FileName,
				FileSize: att.FileSize,
				MimeType: att.MimeType,
				URL:      h.storage.GetURL(att.StorageKey),
			})
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
			Attachments: atts,
		})
	}
	return result
}

// getGroupMemberCount returns the number of members in a group (0 if unavailable).
func (h *chatHandler) getGroupMemberCount(ctx context.Context, chatID uint) int {
	if h.groupService == nil {
		return 0
	}
	members, err := h.groupService.GetGroupMembers(ctx, chatID)
	if err != nil {
		return 0
	}
	return len(members)
}

// getGroupAvatarURL returns a refreshed avatar URL for the group (nil if unset).
func (h *chatHandler) getGroupAvatarURL(chat *models.Chat) *string {
	if chat.AvatarURL != nil && *chat.AvatarURL != "" {
		return h.userService.GetAvatarURL(chat.AvatarURL)
	}
	return nil
}

// getParticipantCount returns the participant count for a group (0 if unavailable).
func (h *chatHandler) getParticipantCount(ctx context.Context, chatID uint) int {
	if h.groupService == nil {
		return 0
	}
	ids, err := h.groupService.GetParticipantUserIDs(ctx, chatID)
	if err != nil {
		return 0
	}
	return len(ids)
}

// buildGroupListItem creates a chatListItem for a group chat.
func (h *chatHandler) buildGroupListItem(ctx context.Context, chat *models.Chat, lastMessage string, unreadCounts map[uint]int64) chatListItem {
	return chatListItem{
		ID:          chat.ID,
		IsGroup:     true,
		GroupName:   chat.GetGroupName(),
		MemberCount: h.getParticipantCount(ctx, chat.ID),
		AvatarURL:   h.getGroupAvatarURL(chat),
		LastMessage: lastMessage,
		UnreadCount: int(unreadCounts[chat.ID]),
		UpdatedAt:   chat.UpdatedAt,
	}
}

// checkGroupAccess verifies the user is a member of the group chat.
// Returns nil on success, or responds with an appropriate error and returns a non-nil error.
func (h *chatHandler) checkGroupAccess(c *gin.Context, chatID, userID uint) error {
	if h.groupService == nil {
		sendInternalError(c, "Group service unavailable")
		return errAccessDenied
	}
	inGroup, err := h.groupService.IsUserInGroup(c.Request.Context(), chatID, userID)
	if err != nil {
		sendInternalError(c, "Failed to check group membership")
		return err
	}
	if !inGroup {
		sendForbidden(c, "Access denied")
		return errAccessDenied
	}
	return nil
}

// checkChatAccess verifies the user has access to the chat (group or 1-on-1).
func (h *chatHandler) checkChatAccess(c *gin.Context, chat *models.Chat, userID uint) error {
	if chat.IsGroup {
		return h.checkGroupAccess(c, chat.ID, userID)
	}
	if !chat.HasUser(userID) {
		sendForbidden(c, "Access denied")
		return errAccessDenied
	}
	return nil
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
	if runes := []rune(text); len(runes) > maxChatListPreview {
		text = string(runes[:maxChatListPreview])
	}

	return text
}

// GetChatData returns chat data with messages in JSON format for dynamic loading
func (h *chatHandler) GetChatData(c *gin.Context) {
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
	if err := h.checkChatAccess(c, chat, userID); err != nil {
		return
	}

	messages, err := h.chatService.GetRecentMessages(c.Request.Context(), chatID, maxRecentMessages)
	if err != nil {
		sendInternalError(c, "Failed to load messages")
		return
	}

	if chat.IsGroup {
		sendSuccess(c, gin.H{
			"chatID":       chatID,
			"is_group":     true,
			"group_name":   chat.GetGroupName(),
			"member_count": h.getGroupMemberCount(c.Request.Context(), chatID),
			"messages":     h.toMessageResponses(messages),
		})
		return
	}

	otherUser, otherUserID := chat.GetOtherUser(userID)
	sendSuccess(c, gin.H{
		"chatID":        chatID,
		"otherUserID":   otherUserID,
		"otherUsername": otherUser.GetDisplayName(),
		"messages":      h.toMessageResponses(messages),
	})
}

// ListChatsAPI returns list of chats for external UI (1-on-1 + groups)
func (h *chatHandler) ListChatsAPI(c *gin.Context) {
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
			items = append(items, h.buildGroupListItem(c.Request.Context(), &chat, lastMessageText, unreadCounts))
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
func (h *chatHandler) CreateChatAPI(c *gin.Context) {
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
		"user1_id":   chat.GetUser1ID(),
		"user2_id":   chat.GetUser2ID(),
		"created_at": chat.CreatedAt,
	})
}

// chatAction executes a chat operation (clear/delete) with shared validation logic.
// Broadcasts the event to the other participant BEFORE the destructive DB action,
// because DeleteChat hard-deletes the chat record (FindChatByIDLight would fail after).
func (h *chatHandler) chatAction(c *gin.Context, action func(ctx context.Context, chatID, userID uint) error, failMsg, successMsg, eventAction string) {
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

	if err := h.checkChatAccess(c, chat, userID); err != nil {
		return
	}

	// Broadcast BEFORE destructive action (chat record must exist for participant lookup)
	if h.onChatEvent != nil {
		h.onChatEvent(chatID, userID, eventAction)
	}

	if err := action(c.Request.Context(), chatID, userID); err != nil {
		errMsg := err.Error()
		if errMsg == errMsgAccessDenied || errMsg == "admin or creator role required" {
			sendForbidden(c, errMsg)
		} else {
			sendInternalError(c, failMsg)
		}
		return
	}

	sendSuccess(c, gin.H{"message": successMsg})
}

// ClearChatAPI clears all messages in a chat
func (h *chatHandler) ClearChatAPI(c *gin.Context) {
	h.chatAction(c, h.chatService.ClearChat, "Failed to clear chat", "Chat cleared", "chat_cleared")
}

// DeleteChatAPI deletes a chat
func (h *chatHandler) DeleteChatAPI(c *gin.Context) {
	h.chatAction(c, h.chatService.DeleteChat, "Failed to delete chat", "Chat deleted", "chat_deleted")
}

// GetChatMessagesAPI returns chat messages for external UI
func (h *chatHandler) GetChatMessagesAPI(c *gin.Context) {
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
	if err := h.checkChatAccess(c, chat, userID); err != nil {
		return
	}

	messages, err := h.chatService.GetMessages(c.Request.Context(), chatID)
	if err != nil {
		sendInternalError(c, "Failed to load messages")
		return
	}

	sendSuccess(c, gin.H{
		"messages": h.toMessageResponses(messages),
	})
}
