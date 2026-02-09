package handlers

import (
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
