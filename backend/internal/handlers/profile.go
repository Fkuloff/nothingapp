package handlers

import (
	"errors"
	"strings"

	"messenger/internal/services"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ProfileHandler struct {
	userService *services.UserService
	contactSvc  *services.ContactService
	logger      *zap.Logger
}

func NewProfileHandler(userService *services.UserService, contactSvc *services.ContactService, logger *zap.Logger) *ProfileHandler {
	return &ProfileHandler{
		userService: userService,
		contactSvc:  contactSvc,
		logger:      logger,
	}
}

// GetContacts returns user's contact list as JSON
func (h *ProfileHandler) GetContacts(c *gin.Context) {
	currentUserID, ok := requireUserID(c)
	if !ok {
		return
	}

	contacts, err := h.contactSvc.GetUserContacts(c.Request.Context(), currentUserID)
	if err != nil {
		h.logger.Error("failed to fetch contacts", zap.Error(err), zap.Uint("user_id", currentUserID))
		sendInternalError(c, "Failed to load contacts")
		return
	}

	response := make([]UserListItem, 0, len(contacts))
	for _, contact := range contacts {
		if contact.ContactUser == nil {
			continue
		}
		// Refresh avatar URL for S3 presigned URLs
		h.userService.RefreshUserAvatarURL(contact.ContactUser)
		response = append(response, UserListItem{
			ID:        contact.ContactUser.ID,
			Username:  contact.ContactUser.Username,
			Name:      contact.ContactUser.GetDisplayName(),
			AvatarURL: contact.ContactUser.AvatarURL,
		})
	}

	sendSuccess(c, gin.H{"contacts": response})
}

// AddContactAPI adds user to contacts via JSON API
func (h *ProfileHandler) AddContactAPI(c *gin.Context) {
	currentUserID, ok := requireUserID(c)
	if !ok {
		return
	}

	contactUserID, err := parseUintParam(c, "user_id")
	if err != nil {
		sendBadRequest(c, "Invalid user ID")
		return
	}

	err = h.contactSvc.AddContact(c.Request.Context(), currentUserID, contactUserID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrCannotAddSelf):
			sendBadRequest(c, "Cannot add yourself to contacts")
		case errors.Is(err, services.ErrAlreadyInContacts):
			sendBadRequest(c, "User is already in your contacts")
		default:
			h.logger.Error("failed to add contact", zap.Error(err), zap.Uint("user_id", currentUserID), zap.Uint("contact_user_id", contactUserID))
			sendInternalError(c, "Failed to add contact")
		}
		return
	}

	sendSuccess(c, gin.H{"message": "Contact added"})
}

// RemoveContactAPI removes user from contacts via JSON API
func (h *ProfileHandler) RemoveContactAPI(c *gin.Context) {
	currentUserID, ok := requireUserID(c)
	if !ok {
		return
	}

	contactUserID, err := parseUintParam(c, "user_id")
	if err != nil {
		sendBadRequest(c, "Invalid user ID")
		return
	}

	err = h.contactSvc.RemoveContact(c.Request.Context(), currentUserID, contactUserID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrContactNotFound):
			sendNotFound(c, "Contact not found")
		default:
			h.logger.Error("failed to remove contact", zap.Error(err), zap.Uint("user_id", currentUserID), zap.Uint("contact_user_id", contactUserID))
			sendInternalError(c, "Failed to remove contact")
		}
		return
	}

	sendSuccess(c, gin.H{"message": "Contact removed"})
}

// SearchUsers searches for users by username or name
func (h *ProfileHandler) SearchUsers(c *gin.Context) {
	_, ok := requireUserID(c)
	if !ok {
		return
	}

	query := strings.TrimSpace(c.Query("q"))
	if query == "" || len(query) < 2 {
		sendBadRequest(c, "Search query must be at least 2 characters")
		return
	}

	users, err := h.userService.SearchUsers(c.Request.Context(), query)
	if err != nil {
		h.logger.Error("failed to search users", zap.Error(err), zap.String("query", query))
		sendInternalError(c, "Failed to search users")
		return
	}

	response := make([]UserListItem, 0, len(users))
	for _, user := range users {
		// Refresh avatar URL for S3 presigned URLs
		h.userService.RefreshUserAvatarURL(user)
		response = append(response, UserListItem{
			ID:        user.ID,
			Username:  user.Username,
			Name:      user.GetDisplayName(),
			AvatarURL: user.AvatarURL,
		})
	}

	sendSuccess(c, gin.H{"users": response})
}

// UpdateProfileAPI updates current user's profile
func (h *ProfileHandler) UpdateProfileAPI(c *gin.Context) {
	currentUserID, ok := requireUserID(c)
	if !ok {
		return
	}

	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		sendBadRequest(c, "Name is required")
		return
	}

	if err := h.userService.UpdateProfile(c.Request.Context(), currentUserID, req.Name); err != nil {
		sendBadRequest(c, err.Error())
		return
	}

	sendSuccess(c, gin.H{"message": "Profile updated"})
}

// GetProfileAPI returns profile info for current or specified user
func (h *ProfileHandler) GetProfileAPI(c *gin.Context) {
	currentUserID, ok := requireUserID(c)
	if !ok {
		return
	}

	targetUserID := currentUserID
	if c.Param("user_id") != "" {
		parsedID, err := parseUintParam(c, "user_id")
		if err != nil {
			sendBadRequest(c, "Invalid user ID")
			return
		}
		targetUserID = parsedID
	}

	user, err := h.userService.GetUserByID(c.Request.Context(), targetUserID)
	if err != nil {
		sendNotFound(c, "User not found")
		return
	}

	isOwn := targetUserID == currentUserID
	isContact, err := h.contactSvc.IsContact(c.Request.Context(), currentUserID, targetUserID)
	if err != nil {
		h.logger.Warn("failed to check contact status", zap.Error(err), zap.Uint("current_user", currentUserID), zap.Uint("target_user", targetUserID))
		isContact = false
	}

	// Refresh avatar URL for S3 presigned URLs
	h.userService.RefreshUserAvatarURL(user)

	sendSuccess(c, gin.H{
		"id":         user.ID,
		"username":   user.Username,
		"name":       user.Name,
		"avatar_url": user.AvatarURL,
		"is_own":     isOwn,
		"is_contact": isContact,
	})
}
