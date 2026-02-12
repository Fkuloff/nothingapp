package handlers

import (
	"io"
	"net/http"

	"messenger/internal/services"

	"github.com/gin-gonic/gin"
)

// UserHandler handles HTTP requests for user operations
type UserHandler struct {
	userService *services.UserService
}

// NewUserHandler creates a new UserHandler instance
func NewUserHandler(userService *services.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// UploadAvatar handles avatar upload
func (h *UserHandler) UploadAvatar(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	// Parse multipart form
	if err := c.Request.ParseMultipartForm(MultipartFormSizeAvatar); err != nil {
		sendBadRequest(c, "Failed to parse form")
		return
	}

	// Get file from form
	file, err := c.FormFile("avatar")
	if err != nil {
		sendBadRequest(c, "No avatar file provided")
		return
	}

	// Upload avatar
	avatarURL, err := h.userService.UploadAvatar(c.Request.Context(), userID, file)
	if err != nil {
		sendBadRequest(c, err.Error())
		return
	}

	sendSuccess(c, gin.H{
		"success":    true,
		"avatar_url": avatarURL,
	})
}

// DeleteAvatar handles avatar deletion
func (h *UserHandler) DeleteAvatar(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	if err := h.userService.DeleteAvatar(c.Request.Context(), userID); err != nil {
		sendBadRequest(c, err.Error())
		return
	}

	sendSuccess(c, gin.H{"success": true})
}

// GetAvatar serves avatar image by user ID (PUBLIC endpoint)
func (h *UserHandler) GetAvatar(c *gin.Context) {
	userID, err := parseUintParam(c, "user_id")
	if err != nil {
		sendBadRequest(c, "Invalid user ID")
		return
	}

	reader, contentType, err := h.userService.GetAvatarReader(c.Request.Context(), userID)
	if err != nil {
		sendNotFound(c, "Avatar not found")
		return
	}
	defer reader.Close()

	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "public, max-age=3600")
	c.Status(http.StatusOK)

	if _, copyErr := io.Copy(c.Writer, reader); copyErr != nil {
		c.Error(copyErr)
	}
}
