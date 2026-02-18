package handlers

import (
	"errors"
	"net/http"

	"messenger/internal/models"
	"messenger/internal/repositories"
	"messenger/internal/services"
	"messenger/internal/storage"

	"github.com/gin-gonic/gin"
)

// AttachmentHandler handles HTTP requests for attachments
type AttachmentHandler struct {
	attachmentService *services.AttachmentService
	chatService       *services.ChatService
	wsHandler         *WebSocketHandler
	participantRepo   *repositories.ChatParticipantRepo
	storage           storage.Storage
}

// NewAttachmentHandler creates a new AttachmentHandler instance
func NewAttachmentHandler(
	attachmentService *services.AttachmentService,
	chatService *services.ChatService,
	wsHandler *WebSocketHandler,
	participantRepo *repositories.ChatParticipantRepo,
	fileStorage storage.Storage,
) *AttachmentHandler {
	return &AttachmentHandler{
		attachmentService: attachmentService,
		chatService:       chatService,
		wsHandler:         wsHandler,
		participantRepo:   participantRepo,
		storage:           fileStorage,
	}
}

// UploadAttachments handles multipart file uploads
func (h *AttachmentHandler) UploadAttachments(c *gin.Context) {
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

	// Verify user has access to chat
	chat, err := h.chatService.FindChatByIDLight(c.Request.Context(), chatID)
	if err != nil {
		sendForbidden(c, "Access denied")
		return
	}

	// Group chats: check membership via participant repo; 1-on-1: use HasUser
	if chat.IsGroup {
		if h.participantRepo == nil {
			sendForbidden(c, "Access denied")
			return
		}
		_, err := h.participantRepo.FindByUserAndChat(c.Request.Context(), chatID, userID)
		if err != nil {
			sendForbidden(c, "Access denied")
			return
		}
	} else if !chat.HasUser(userID) {
		sendForbidden(c, "Access denied")
		return
	}

	// Parse multipart form
	if parseErr := c.Request.ParseMultipartForm(MultipartFormSizeAttachment); parseErr != nil {
		sendBadRequest(c, "Failed to parse form")
		return
	}

	form := c.Request.MultipartForm
	files := form.File["attachments"]

	if len(files) == 0 {
		sendBadRequest(c, "No files uploaded")
		return
	}

	if len(files) > 10 {
		sendBadRequest(c, "Maximum 10 files per message")
		return
	}

	// Upload attachments
	attachments, err := h.attachmentService.UploadAttachments(c.Request.Context(), messageID, userID, files)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrMessageNotFound):
			sendNotFound(c, "Message not found")
		case errors.Is(err, services.ErrNotMessageOwner):
			sendForbidden(c, "Access denied")
		default:
			sendInternalError(c, "Failed to upload attachments")
		}
		return
	}

	// Broadcast attachments_added event to all chat participants via WebSocket
	serialized := serializeAttachmentsWithURL(attachments, h.storage)
	if h.wsHandler != nil {
		h.wsHandler.BroadcastAttachmentsAdded(chatID, messageID, serialized)
	}

	sendSuccess(c, gin.H{
		"success":     true,
		"attachments": serialized,
	})
}

// DownloadAttachment redirects to a presigned S3 URL for direct download (JWT required)
func (h *AttachmentHandler) DownloadAttachment(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	attachmentID, err := parseUintParam(c, "id")
	if err != nil {
		sendBadRequest(c, "Invalid attachment ID")
		return
	}

	attachment, err := h.attachmentService.FindAttachmentWithMessage(c.Request.Context(), attachmentID)
	if err != nil {
		sendNotFound(c, "Attachment not found")
		return
	}

	// Verify user has access to the chat containing this attachment
	if attachment.Message != nil {
		chat, chatErr := h.chatService.FindChatByIDLight(c.Request.Context(), attachment.Message.ChatID)
		if chatErr != nil {
			sendForbidden(c, "Access denied")
			return
		}
		if chat.IsGroup {
			if h.participantRepo == nil {
				sendForbidden(c, "Access denied")
				return
			}
			if _, pErr := h.participantRepo.FindByUserAndChat(c.Request.Context(), chat.ID, userID); pErr != nil {
				sendForbidden(c, "Access denied")
				return
			}
		} else if !chat.HasUser(userID) {
			sendForbidden(c, "Access denied")
			return
		}
	}

	presignedURL := h.storage.GetURL(attachment.StorageKey)
	c.Redirect(http.StatusFound, presignedURL)
}

// DeleteAttachment removes an attachment
func (h *AttachmentHandler) DeleteAttachment(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	attachmentID, err := parseUintParam(c, "id")
	if err != nil {
		sendBadRequest(c, "Invalid attachment ID")
		return
	}

	if err := h.attachmentService.DeleteAttachment(c.Request.Context(), attachmentID, userID); err != nil {
		switch {
		case errors.Is(err, services.ErrAttachmentNotFound):
			sendNotFound(c, "Attachment not found")
		case errors.Is(err, services.ErrNotMessageOwner):
			sendForbidden(c, "Access denied")
		default:
			sendInternalError(c, "Failed to delete attachment")
		}
		return
	}

	sendSuccess(c, gin.H{"success": true})
}

// serializeAttachmentsWithURL converts attachment models to maps with presigned URLs
func serializeAttachmentsWithURL(attachments []*models.Attachment, s storage.Storage) []map[string]any {
	result := make([]map[string]any, 0, len(attachments))
	for _, att := range attachments {
		result = append(result, map[string]any{
			"id":        att.ID,
			"file_type": att.FileType,
			"file_name": att.FileName,
			"file_size": att.FileSize,
			"mime_type": att.MimeType,
			"url":       s.GetURL(att.StorageKey),
		})
	}
	return result
}
