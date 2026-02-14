package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"messenger/internal/services"

	"github.com/gin-gonic/gin"
)

// AttachmentHandler handles HTTP requests for attachments
type AttachmentHandler struct {
	attachmentService *services.AttachmentService
	chatService       *services.ChatService
}

// NewAttachmentHandler creates a new AttachmentHandler instance
func NewAttachmentHandler(
	attachmentService *services.AttachmentService,
	chatService *services.ChatService,
) *AttachmentHandler {
	return &AttachmentHandler{
		attachmentService: attachmentService,
		chatService:       chatService,
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

	// Verify user has access to chat (use light method for performance)
	chat, err := h.chatService.FindChatByIDLight(c.Request.Context(), chatID)
	if err != nil || !chat.HasUser(userID) {
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

	sendSuccess(c, gin.H{
		"success":     true,
		"attachments": attachments,
	})
}

// DownloadAttachment streams a file (PUBLIC endpoint - no JWT required)
// Access control: Attachments are publicly accessible via S3 presigned URLs
// This endpoint simply proxies the request to the storage backend
func (h *AttachmentHandler) DownloadAttachment(c *gin.Context) {
	attachmentID, err := parseUintParam(c, "id")
	if err != nil {
		sendBadRequest(c, "Invalid attachment ID")
		return
	}

	attachment, reader, err := h.attachmentService.GetAttachment(c.Request.Context(), attachmentID)
	if err != nil {
		sendNotFound(c, "Attachment not found")
		return
	}
	defer reader.Close()

	// Set headers for download
	c.Header("Content-Type", attachment.MimeType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", attachment.FileName))
	c.Header("Content-Length", strconv.FormatInt(attachment.FileSize, 10))

	// Stream file
	c.DataFromReader(http.StatusOK, attachment.FileSize, attachment.MimeType, reader, nil)
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
