package handlers

import (
	"fmt"
	"io"
	"messenger/internal/services"
	"net/http"
	"strconv"

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

	chatID, err := parseUintParam(c, "chatId")
	if err != nil {
		sendBadRequest(c, "Invalid chat ID")
		return
	}

	messageID, err := parseUintParam(c, "messageId")
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
		sendInternalError(c, err.Error())
		return
	}

	sendSuccess(c, gin.H{
		"success":     true,
		"attachments": attachments,
	})
}

// DownloadAttachment streams a file
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

	attachment, reader, err := h.attachmentService.GetAttachment(c.Request.Context(), attachmentID)
	if err != nil {
		sendNotFound(c, "Attachment not found")
		return
	}
	defer reader.Close()

	// Verify user has access (through message -> chat)
	message, err := h.chatService.GetMessageByID(c.Request.Context(), attachment.MessageID)
	if err != nil {
		sendNotFound(c, "Message not found")
		return
	}

	chat, err := h.chatService.FindChatByIDLight(c.Request.Context(), message.ChatID)
	if err != nil || !chat.HasUser(userID) {
		sendForbidden(c, "Access denied")
		return
	}

	// Set headers for download
	c.Header("Content-Type", attachment.MimeType)
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, attachment.FileName))
	c.Header("Content-Length", strconv.FormatInt(attachment.FileSize, 10))

	// Stream file
	c.DataFromReader(http.StatusOK, attachment.FileSize, attachment.MimeType, reader, nil)
}

// GetThumbnail serves thumbnail
func (h *AttachmentHandler) GetThumbnail(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	attachmentID, err := parseUintParam(c, "id")
	if err != nil {
		sendBadRequest(c, "Invalid attachment ID")
		return
	}

	attachment, reader, err := h.attachmentService.GetThumbnail(c.Request.Context(), attachmentID)
	if err != nil {
		sendNotFound(c, err.Error())
		return
	}
	defer reader.Close()

	// Verify user has access (through message -> chat)
	message, err := h.chatService.GetMessageByID(c.Request.Context(), attachment.MessageID)
	if err != nil {
		sendNotFound(c, "Message not found")
		return
	}

	chat, err := h.chatService.FindChatByIDLight(c.Request.Context(), message.ChatID)
	if err != nil || !chat.HasUser(userID) {
		sendForbidden(c, "Access denied")
		return
	}

	// Set headers for inline display
	c.Header("Content-Type", "image/jpeg")
	c.Status(http.StatusOK)

	// Stream thumbnail without setting Content-Length (auto-detected)
	if _, copyErr := io.Copy(c.Writer, reader); copyErr != nil {
		// Log error but don't send response as headers already sent
		// c.Error returns *Error which we can safely ignore here
		c.Error(copyErr) //nolint:errcheck // error already being logged via Gin's error handling
	}
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
		sendInternalError(c, err.Error())
		return
	}

	sendSuccess(c, gin.H{"success": true})
}
