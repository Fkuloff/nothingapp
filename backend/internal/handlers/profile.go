package handlers

import (
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

// GetProfile returns user profile data as JSON
func (h *ProfileHandler) GetProfile(c *gin.Context) {
	currentUserID, ok := requireUserID(c)
	if !ok {
		return
	}

	profileUserIDStr := c.Param("user_id")
	var profileUserID uint
	if profileUserIDStr == "" {
		profileUserID = currentUserID
	} else {
		var err error
		profileUserID, err = parseUintParam(c, "user_id")
		if err != nil {
			sendBadRequest(c, "Invalid user ID")
			return
		}
	}

	user, err := h.userService.GetUserByID(c.Request.Context(), profileUserID)
	if err != nil {
		h.logger.Error("failed to fetch user", zap.Error(err), zap.Uint("user_id", profileUserID))
		sendNotFound(c, "User not found")
		return
	}

	isOwn := profileUserID == currentUserID
	isContact, err := h.contactSvc.IsContact(c.Request.Context(), currentUserID, profileUserID)
	if err != nil {
		h.logger.Warn("failed to check contact status", zap.Error(err), zap.Uint("current_user", currentUserID), zap.Uint("target_user", profileUserID))
		isContact = false
	}

	sendSuccess(c, gin.H{
		"id":         user.ID,
		"username":   user.Username,
		"name":       user.Name,
		"avatar_url": user.AvatarURL,
		"is_own":     isOwn,
		"is_contact": isContact,
	})
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

	var response []UserListItem
	for _, contact := range contacts {
		if contact.ContactUser == nil {
			continue
		}
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
		sendBadRequest(c, err.Error())
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
		sendBadRequest(c, err.Error())
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

	var response []UserListItem
	for _, user := range users {
		response = append(response, UserListItem{
			ID:        user.ID,
			Username:  user.Username,
			Name:      user.GetDisplayName(),
			AvatarURL: user.AvatarURL,
		})
	}

	sendSuccess(c, gin.H{"users": response})
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

	sendSuccess(c, gin.H{
		"id":         user.ID,
		"username":   user.Username,
		"name":       user.Name,
		"avatar_url": user.AvatarURL,
		"is_own":     isOwn,
		"is_contact": isContact,
	})
}
