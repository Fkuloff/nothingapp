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

// attachmentHandler handles HTTP requests for attachments
type attachmentHandler struct {
	attachmentService *services.AttachmentService
	chatService       *services.ChatService
	wsHandler         *webSocketHandler
	participantRepo   *repositories.ChatParticipantRepo
	storage           storage.Storage
}

// newAttachmentHandler creates a new attachmentHandler instance
func newAttachmentHandler(
	attachmentService *services.AttachmentService,
	chatService *services.ChatService,
	wsHandler *webSocketHandler,
	participantRepo *repositories.ChatParticipantRepo,
	fileStorage storage.Storage,
) *attachmentHandler {
	return &attachmentHandler{
		attachmentService: attachmentService,
		chatService:       chatService,
		wsHandler:         wsHandler,
		participantRepo:   participantRepo,
		storage:           fileStorage,
	}
}

// UploadAttachments handles multipart file uploads
func (h *attachmentHandler) UploadAttachments(c *gin.Context) {
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
	if err := h.verifyChatMembership(c, chatID, userID); err != nil {
		return
	}

	// Parse multipart form
	if parseErr := c.Request.ParseMultipartForm(multipartFormSizeAttachment); parseErr != nil {
		sendBadRequest(c, "Failed to parse form")
		return
	}

	form := c.Request.MultipartForm
	files := form.File["attachments"]

	if len(files) == 0 {
		sendBadRequest(c, "No files uploaded")
		return
	}

	if len(files) > maxAttachmentsPerMessage {
		sendBadRequest(c, "Too many files per message")
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
func (h *attachmentHandler) DownloadAttachment(c *gin.Context) {
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
	if attachment.Message == nil {
		sendInternalError(c, "Attachment has no associated message")
		return
	}

	if err := h.verifyChatMembership(c, attachment.Message.ChatID, userID); err != nil {
		return
	}

	presignedURL := h.storage.GetURL(attachment.StorageKey)
	c.Redirect(http.StatusFound, presignedURL)
}

// DeleteAttachment removes an attachment
func (h *attachmentHandler) DeleteAttachment(c *gin.Context) {
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

// verifyChatMembership checks that the user is a participant of the given chat.
// Responds with an HTTP error and returns a non-nil error on failure.
func (h *attachmentHandler) verifyChatMembership(c *gin.Context, chatID, userID uint) error {
	chat, err := h.chatService.FindChatByIDLight(c.Request.Context(), chatID)
	if err != nil {
		sendForbidden(c, "Access denied")
		return err
	}

	if chat.IsGroup {
		if h.participantRepo == nil {
			sendForbidden(c, "Access denied")
			return errAccessDenied
		}
		if _, pErr := h.participantRepo.FindByUserAndChat(c.Request.Context(), chatID, userID); pErr != nil {
			sendForbidden(c, "Access denied")
			return pErr
		}
	} else if !chat.HasUser(userID) {
		sendForbidden(c, "Access denied")
		return errAccessDenied
	}

	return nil
}

// serializeAttachmentsWithURL converts attachment pointer slice to maps with presigned URLs.
func serializeAttachmentsWithURL(attachments []*models.Attachment, s storage.Storage) []map[string]any {
	result := make([]map[string]any, 0, len(attachments))
	for _, att := range attachments {
		result = append(result, serializeAttachment(att, s))
	}
	return result
}

// serializeAttachmentSlice converts a value-type attachment slice to maps with presigned URLs.
func serializeAttachmentSlice(attachments []models.Attachment, s storage.Storage) []map[string]any {
	result := make([]map[string]any, 0, len(attachments))
	for i := range attachments {
		result = append(result, serializeAttachment(&attachments[i], s))
	}
	return result
}

// serializeAttachment converts a single attachment model to a map with a presigned URL.
func serializeAttachment(att *models.Attachment, s storage.Storage) map[string]any {
	return map[string]any{
		"id":        att.ID,
		"file_type": att.FileType,
		"file_name": att.FileName,
		"file_size": att.FileSize,
		"mime_type": att.MimeType,
		"url":       s.GetURL(att.StorageKey),
	}
}
