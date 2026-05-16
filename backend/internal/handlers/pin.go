package handlers

import (
	"time"

	"messenger/internal/services"
	"messenger/internal/storage"

	"github.com/gin-gonic/gin"
)

// pinHandler handles HTTP requests for pinning/unpinning messages.
type pinHandler struct {
	pinService *services.PinService
	storage    storage.Storage
	onPinEvent func(chatID, messageID uint, action string)
}

// pinnedMessageResponse is the JSON representation of a pinned message.
type pinnedMessageResponse struct {
	ID        uint            `json:"id"`
	ChatID    uint            `json:"chat_id"`
	MessageID uint            `json:"message_id"`
	PinnedBy  uint            `json:"pinned_by"`
	CreatedAt time.Time       `json:"created_at"`
	Message   messageResponse `json:"message"`
}

// newPinHandler creates a new pinHandler.
func newPinHandler(pinService *services.PinService, fileStorage storage.Storage) *pinHandler {
	return &pinHandler{
		pinService: pinService,
		storage:    fileStorage,
	}
}

// setOnPinEventCallback registers a callback for pin/unpin events (WebSocket broadcasting).
func (h *pinHandler) setOnPinEventCallback(cb func(chatID, messageID uint, action string)) {
	h.onPinEvent = cb
}

// PinMessageAPI pins a message in a chat.
// POST /api/chats/:id/messages/:message_id/pin
func (h *pinHandler) PinMessageAPI(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	chatID, err := parseUintParam(c, "id")
	if err != nil {
		sendBadRequest(c, "Invalid chat ID")
		return
	}

	messageID, err := parseUintParam(c, "message_id")
	if err != nil {
		sendBadRequest(c, "Invalid message ID")
		return
	}

	pin, err := h.pinService.PinMessage(c.Request.Context(), chatID, messageID, userID)
	if err != nil {
		handlePinError(c, err)
		return
	}

	if h.onPinEvent != nil {
		h.onPinEvent(chatID, messageID, "message_pinned")
	}

	sendCreated(c, gin.H{
		"id":         pin.ID,
		"chat_id":    pin.ChatID,
		"message_id": pin.MessageID,
		"pinned_by":  pin.PinnedBy,
		"created_at": pin.CreatedAt,
	})
}

// UnpinMessageAPI unpins a message from a chat.
// DELETE /api/chats/:id/messages/:message_id/pin
func (h *pinHandler) UnpinMessageAPI(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	chatID, err := parseUintParam(c, "id")
	if err != nil {
		sendBadRequest(c, "Invalid chat ID")
		return
	}

	messageID, err := parseUintParam(c, "message_id")
	if err != nil {
		sendBadRequest(c, "Invalid message ID")
		return
	}

	if err := h.pinService.UnpinMessage(c.Request.Context(), chatID, messageID, userID); err != nil {
		handlePinError(c, err)
		return
	}

	if h.onPinEvent != nil {
		h.onPinEvent(chatID, messageID, "message_unpinned")
	}

	sendSuccess(c, gin.H{"message": "Message unpinned"})
}

// GetPinnedMessagesAPI returns all pinned messages for a chat.
// GET /api/chats/:id/pins
func (h *pinHandler) GetPinnedMessagesAPI(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	chatID, err := parseUintParam(c, "id")
	if err != nil {
		sendBadRequest(c, "Invalid chat ID")
		return
	}

	pins, err := h.pinService.GetPinnedMessages(c.Request.Context(), chatID, userID)
	if err != nil {
		errMsg := err.Error()
		switch errMsg {
		case errMsgAccessDenied:
			sendForbidden(c, errMsg)
		case errMsgChatNotFound:
			sendNotFound(c, errMsg)
		default:
			sendInternalError(c, "Failed to load pinned messages")
		}
		return
	}

	result := make([]pinnedMessageResponse, 0, len(pins))
	for _, pin := range pins {
		msg := pin.Message
		msgType := string(msg.Type)
		if msgType == "" {
			msgType = "user"
		}

		atts := make([]attachmentResponse, 0, len(msg.Attachments))
		for _, att := range msg.Attachments {
			atts = append(atts, attachmentResponse{
				ID:                att.ID,
				FileSize:          att.FileSize,
				URL:               h.storage.GetURL(att.StorageKey),
				FileIV:            att.FileIV,
				EncryptedMetadata: att.EncryptedMetadata,
				MetadataIV:        att.MetadataIV,
			})
		}

		result = append(result, pinnedMessageResponse{
			ID:        pin.ID,
			ChatID:    pin.ChatID,
			MessageID: pin.MessageID,
			PinnedBy:  pin.PinnedBy,
			CreatedAt: pin.CreatedAt,
			Message: messageResponse{
				ID:          msg.ID,
				ChatID:      msg.ChatID,
				UserID:      msg.UserID,
				Text:        msg.Text,
				Type:        msgType,
				IsDeleted:   msg.IsDeleted,
				CreatedAt:   msg.CreatedAt,
				ReplyToID:   msg.ReplyToID,
				EditedAt:    msg.EditedAt,
				Attachments: atts,
			},
		})
	}

	sendSuccess(c, gin.H{"pinned_messages": result})
}

// handlePinError maps service errors to appropriate HTTP responses for pin/unpin.
func handlePinError(c *gin.Context, err error) {
	errMsg := err.Error()
	switch errMsg {
	case errMsgAccessDenied, "admin or creator role required to pin messages":
		sendForbidden(c, errMsg)
	case errMsgChatNotFound, errMsgMsgNotFound:
		sendNotFound(c, errMsg)
	default:
		sendBadRequest(c, errMsg)
	}
}
