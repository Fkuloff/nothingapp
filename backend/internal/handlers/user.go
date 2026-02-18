package handlers

import (
	"messenger/internal/services"

	"github.com/gin-gonic/gin"
)

// userHandler handles HTTP requests for user operations
type userHandler struct {
	userService *services.UserService
}

// newUserHandler creates a new userHandler instance
func newUserHandler(userService *services.UserService) *userHandler {
	return &userHandler{
		userService: userService,
	}
}

// UploadAvatar handles avatar upload
func (h *userHandler) UploadAvatar(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	// Parse multipart form
	if err := c.Request.ParseMultipartForm(multipartFormSizeAvatar); err != nil {
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
func (h *userHandler) DeleteAvatar(c *gin.Context) {
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
func (h *userHandler) GetAvatar(c *gin.Context) {
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
	serveReaderContent(c, reader, contentType, "no-cache")
}
