package handlers

import (
	"messenger/internal/models"
	"messenger/internal/services"

	"github.com/gin-gonic/gin"
)

// groupEvent represents a group-level event for WebSocket broadcasting.
type groupEvent struct {
	Action  string            `json:"action"`
	ChatID  uint              `json:"chat_id"`
	ActorID uint              `json:"actor_id,omitempty"`
	UserID  uint              `json:"user_id,omitempty"`
	Members []groupMemberItem `json:"members,omitempty"`
	Name    string            `json:"name,omitempty"`
	Avatar  string            `json:"avatar_url,omitempty"`
	NewRole string            `json:"new_role,omitempty"`
}

// groupHandler handles HTTP requests for group chat operations.
type groupHandler struct {
	groupService    *services.GroupService
	presenceService *services.PresenceService
	userService     *services.UserService
	onGroupEvent    func(event groupEvent)
}

// newGroupHandler creates a new groupHandler.
func newGroupHandler(
	groupService *services.GroupService,
	presenceService *services.PresenceService,
	userService *services.UserService,
) *groupHandler {
	return &groupHandler{
		groupService:    groupService,
		presenceService: presenceService,
		userService:     userService,
	}
}

// SetOnGroupEventCallback registers a callback for group events (member changes, updates).
func (h *groupHandler) setOnGroupEventCallback(cb func(event groupEvent)) {
	h.onGroupEvent = cb
}

// CreateGroupAPI creates a new group chat.
// POST /api/groups { "name": "...", "member_ids": [2,3,5] }
func (h *groupHandler) CreateGroupAPI(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	var req struct {
		Name      string `json:"name"`
		MemberIDs []uint `json:"member_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		sendBadRequest(c, "Invalid input")
		return
	}

	chat, err := h.groupService.CreateGroup(c.Request.Context(), userID, req.Name, req.MemberIDs)
	if err != nil {
		sendBadRequest(c, err.Error())
		return
	}

	// Build member list for response
	members := h.buildMemberList(c, chat.ID)

	sendCreated(c, gin.H{
		"id":         chat.ID,
		"name":       req.Name,
		"is_group":   true,
		"members":    members,
		"created_at": chat.CreatedAt,
	})
}

// GetGroupInfoAPI returns group info with members.
// GET /api/groups/:id
func (h *groupHandler) GetGroupInfoAPI(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	chatID, err := parseUintParam(c, "id")
	if err != nil {
		sendBadRequest(c, "Invalid group ID")
		return
	}

	inGroup, gErr := h.groupService.IsUserInGroup(c.Request.Context(), chatID, userID)
	if gErr != nil {
		sendInternalError(c, "Failed to check group membership")
		return
	}
	if !inGroup {
		sendForbidden(c, "Access denied")
		return
	}

	chat, err := h.groupService.GetGroupInfo(c.Request.Context(), chatID)
	if err != nil {
		sendInternalError(c, "Failed to load group info")
		return
	}

	members := h.buildMemberList(c, chatID)

	groupName := ""
	if chat.GroupName != nil {
		groupName = *chat.GroupName
	}

	var avatarURL *string
	if chat.AvatarURL != nil && *chat.AvatarURL != "" {
		avatarURL = h.userService.GetAvatarURL(chat.AvatarURL)
	}

	var creatorID uint
	if chat.CreatorID != nil {
		creatorID = *chat.CreatorID
	}

	sendSuccess(c, gin.H{
		"id":         chat.ID,
		"name":       groupName,
		"avatar_url": avatarURL,
		"creator_id": creatorID,
		"members":    members,
		"created_at": chat.CreatedAt,
	})
}

// UpdateGroupInfoAPI updates group name.
// PUT /api/groups/:id { "name": "New Name" }
func (h *groupHandler) UpdateGroupInfoAPI(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	chatID, err := parseUintParam(c, "id")
	if err != nil {
		sendBadRequest(c, "Invalid group ID")
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		sendBadRequest(c, "Invalid input")
		return
	}

	if err := h.groupService.UpdateGroupInfo(c.Request.Context(), chatID, userID, req.Name); err != nil {
		sendBadRequest(c, err.Error())
		return
	}

	if h.onGroupEvent != nil {
		h.onGroupEvent(groupEvent{
			Action:  "group_updated",
			ChatID:  chatID,
			ActorID: userID,
			Name:    req.Name,
		})
	}

	sendSuccess(c, gin.H{"message": "Group updated"})
}

// AddMembersAPI adds members to a group.
// POST /api/groups/:id/members { "user_ids": [4,6] }
func (h *groupHandler) AddMembersAPI(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	chatID, err := parseUintParam(c, "id")
	if err != nil {
		sendBadRequest(c, "Invalid group ID")
		return
	}

	var req struct {
		UserIDs []uint `json:"user_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		sendBadRequest(c, "Invalid input")
		return
	}

	addedIDs, err := h.groupService.AddMembers(c.Request.Context(), chatID, userID, req.UserIDs)
	if err != nil {
		sendBadRequest(c, err.Error())
		return
	}

	if h.onGroupEvent != nil && len(addedIDs) > 0 {
		// Build member items for added users
		var addedMembers []groupMemberItem
		for _, id := range addedIDs {
			user, err := h.userService.GetUserByID(c.Request.Context(), id)
			if err != nil {
				continue
			}
			h.userService.RefreshUserAvatarURL(user)
			addedMembers = append(addedMembers, groupMemberItem{
				UserID:    user.ID,
				Username:  user.Username,
				Name:      user.GetDisplayName(),
				AvatarURL: user.AvatarURL,
				Role:      string(models.RoleMember),
				IsOnline:  h.presenceService.IsUserOnline(user.ID),
			})
		}
		h.onGroupEvent(groupEvent{
			Action:  "member_added",
			ChatID:  chatID,
			ActorID: userID,
			Members: addedMembers,
		})
	}

	sendSuccess(c, gin.H{"message": "Members added"})
}

// RemoveMemberAPI removes a member from the group.
// DELETE /api/groups/:id/members/:user_id
func (h *groupHandler) RemoveMemberAPI(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	chatID, err := parseUintParam(c, "id")
	if err != nil {
		sendBadRequest(c, "Invalid group ID")
		return
	}

	targetUserID, err := parseUintParam(c, "user_id")
	if err != nil {
		sendBadRequest(c, "Invalid user ID")
		return
	}

	// Broadcast BEFORE removal (need participant list with the user still in it)
	if h.onGroupEvent != nil {
		h.onGroupEvent(groupEvent{
			Action:  "member_removed",
			ChatID:  chatID,
			ActorID: userID,
			UserID:  targetUserID,
		})
	}

	if err := h.groupService.RemoveMember(c.Request.Context(), chatID, userID, targetUserID); err != nil {
		sendBadRequest(c, err.Error())
		return
	}

	sendSuccess(c, gin.H{"message": "Member removed"})
}

// LeaveGroupAPI allows a member to leave the group.
// POST /api/groups/:id/leave
func (h *groupHandler) LeaveGroupAPI(c *gin.Context) {
	h.groupAction(c, groupEvent{Action: "member_left"}, func(ctx *gin.Context, chatID, userID uint) error {
		return h.groupService.LeaveGroup(ctx.Request.Context(), chatID, userID)
	}, "Left group")
}

// ChangeRoleAPI changes a member's role.
// PUT /api/groups/:id/members/:user_id/role { "role": "admin" }
func (h *groupHandler) ChangeRoleAPI(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	chatID, err := parseUintParam(c, "id")
	if err != nil {
		sendBadRequest(c, "Invalid group ID")
		return
	}

	targetUserID, err := parseUintParam(c, "user_id")
	if err != nil {
		sendBadRequest(c, "Invalid user ID")
		return
	}

	var req struct {
		Role string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		sendBadRequest(c, "Invalid input")
		return
	}

	if err := h.groupService.ChangeRole(c.Request.Context(), chatID, userID, targetUserID, models.ParticipantRole(req.Role)); err != nil {
		sendBadRequest(c, err.Error())
		return
	}

	if h.onGroupEvent != nil {
		h.onGroupEvent(groupEvent{
			Action:  "role_changed",
			ChatID:  chatID,
			ActorID: userID,
			UserID:  targetUserID,
			NewRole: req.Role,
		})
	}

	sendSuccess(c, gin.H{"message": "Role changed"})
}

// DeleteGroupAPI deletes a group. Creator only.
// DELETE /api/groups/:id
func (h *groupHandler) DeleteGroupAPI(c *gin.Context) {
	h.groupAction(c, groupEvent{Action: "group_deleted"}, func(ctx *gin.Context, chatID, userID uint) error {
		return h.groupService.DeleteGroup(ctx.Request.Context(), chatID, userID)
	}, "Group deleted")
}

// groupAction extracts common logic for group actions that broadcast BEFORE executing.
func (h *groupHandler) groupAction(c *gin.Context, event groupEvent, action func(*gin.Context, uint, uint) error, successMsg string) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	chatID, err := parseUintParam(c, "id")
	if err != nil {
		sendBadRequest(c, "Invalid group ID")
		return
	}

	// Broadcast BEFORE action
	event.ChatID = chatID
	event.UserID = userID
	if event.Action == "group_deleted" {
		event.ActorID = userID
	}
	if h.onGroupEvent != nil {
		h.onGroupEvent(event)
	}

	if err := action(c, chatID, userID); err != nil {
		sendBadRequest(c, err.Error())
		return
	}

	sendSuccess(c, gin.H{"message": successMsg})
}

// UploadGroupAvatarAPI uploads a group avatar.
// POST /api/groups/:id/avatar
func (h *groupHandler) UploadGroupAvatarAPI(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	chatID, err := parseUintParam(c, "id")
	if err != nil {
		sendBadRequest(c, "Invalid group ID")
		return
	}

	if err := c.Request.ParseMultipartForm(multipartFormSizeAvatar); err != nil {
		sendBadRequest(c, "File too large")
		return
	}

	fileHeader, err := c.FormFile("avatar")
	if err != nil {
		sendBadRequest(c, "No file provided")
		return
	}

	avatarURL, err := h.groupService.UploadGroupAvatar(c.Request.Context(), chatID, userID, fileHeader)
	if err != nil {
		sendBadRequest(c, err.Error())
		return
	}

	if h.onGroupEvent != nil {
		h.onGroupEvent(groupEvent{
			Action:  "group_updated",
			ChatID:  chatID,
			ActorID: userID,
			Avatar:  avatarURL,
		})
	}

	sendSuccess(c, gin.H{
		"success":    true,
		"avatar_url": avatarURL,
	})
}

// DeleteGroupAvatarAPI deletes a group avatar.
// DELETE /api/groups/:id/avatar
func (h *groupHandler) DeleteGroupAvatarAPI(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	chatID, err := parseUintParam(c, "id")
	if err != nil {
		sendBadRequest(c, "Invalid group ID")
		return
	}

	if err := h.groupService.DeleteGroupAvatar(c.Request.Context(), chatID, userID); err != nil {
		sendBadRequest(c, err.Error())
		return
	}

	if h.onGroupEvent != nil {
		h.onGroupEvent(groupEvent{
			Action:  "group_updated",
			ChatID:  chatID,
			ActorID: userID,
		})
	}

	sendSuccess(c, gin.H{"message": "Avatar deleted"})
}

// GetGroupAvatar serves the group avatar image.
// GET /api/group-avatars/:chat_id (public)
func (h *groupHandler) GetGroupAvatar(c *gin.Context) {
	chatID, err := parseUintParam(c, "chat_id")
	if err != nil {
		sendBadRequest(c, "Invalid group ID")
		return
	}

	reader, contentType, err := h.groupService.GetGroupAvatarReader(c.Request.Context(), chatID)
	if err != nil {
		sendNotFound(c, "Avatar not found")
		return
	}
	serveReaderContent(c, reader, contentType, "public, max-age=3600")
}

// buildMemberList builds the member list for group API responses.
func (h *groupHandler) buildMemberList(c *gin.Context, chatID uint) []groupMemberItem {
	participants, err := h.groupService.GetGroupMembers(c.Request.Context(), chatID)
	if err != nil {
		return nil
	}

	members := make([]groupMemberItem, 0, len(participants))
	for _, p := range participants {
		h.userService.RefreshUserAvatarURL(&p.User)
		members = append(members, groupMemberItem{
			UserID:    p.UserID,
			Username:  p.User.Username,
			Name:      p.User.GetDisplayName(),
			AvatarURL: p.User.AvatarURL,
			Role:      string(p.Role),
			IsOnline:  h.presenceService.IsUserOnline(p.UserID),
		})
	}
	return members
}
