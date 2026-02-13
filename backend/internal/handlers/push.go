package handlers

import (
	"net/url"
	"strings"

	"messenger/internal/services"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// validatePushEndpoint checks that the endpoint is a valid HTTPS URL.
func validatePushEndpoint(endpoint string) bool {
	u, err := url.ParseRequestURI(endpoint)
	if err != nil {
		return false
	}
	return strings.EqualFold(u.Scheme, "https")
}

// PushHandler handles push notification REST endpoints
type PushHandler struct {
	pushService *services.PushNotificationService
	logger      *zap.Logger
}

// NewPushHandler creates a new PushHandler
func NewPushHandler(pushService *services.PushNotificationService, logger *zap.Logger) *PushHandler {
	return &PushHandler{
		pushService: pushService,
		logger:      logger,
	}
}

// GetVAPIDKey returns the VAPID public key for frontend subscription
func (h *PushHandler) GetVAPIDKey(c *gin.Context) {
	if !h.pushService.IsEnabled() {
		sendBadRequest(c, "Push notifications not configured")
		return
	}
	sendSuccess(c, gin.H{
		"vapid_public_key": h.pushService.GetVAPIDPublicKey(),
	})
}

type pushSubscribeRequest struct {
	Endpoint string `json:"endpoint" binding:"required"`
	P256dh   string `json:"p256dh" binding:"required"`
	Auth     string `json:"auth" binding:"required"`
}

// Subscribe stores a new push subscription
func (h *PushHandler) Subscribe(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	var req pushSubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendBadRequest(c, "Invalid subscription data")
		return
	}

	if !validatePushEndpoint(req.Endpoint) {
		sendBadRequest(c, "Invalid push endpoint: must be a valid HTTPS URL")
		return
	}

	if err := h.pushService.Subscribe(c.Request.Context(), userID, req.Endpoint, req.P256dh, req.Auth); err != nil {
		if strings.Contains(err.Error(), "subscription limit reached") {
			sendBadRequest(c, err.Error())
			return
		}
		h.logger.Error("failed to save push subscription",
			zap.Error(err),
			zap.Uint("user_id", userID),
		)
		sendInternalError(c, "Failed to save subscription")
		return
	}

	sendSuccess(c, gin.H{"message": "Subscribed to push notifications"})
}

type pushUnsubscribeRequest struct {
	Endpoint string `json:"endpoint" binding:"required"`
}

// Unsubscribe removes a push subscription
func (h *PushHandler) Unsubscribe(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	var req pushUnsubscribeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendBadRequest(c, "Invalid request")
		return
	}

	if err := h.pushService.Unsubscribe(c.Request.Context(), userID, req.Endpoint); err != nil {
		h.logger.Error("failed to unsubscribe push",
			zap.Error(err),
			zap.Uint("user_id", userID),
		)
		sendInternalError(c, "Failed to unsubscribe")
		return
	}

	sendSuccess(c, gin.H{"message": "Unsubscribed from push notifications"})
}

// GetStatus returns whether push notifications are enabled for the current user
func (h *PushHandler) GetStatus(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	hasSubs, err := h.pushService.HasSubscriptions(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("failed to check push status",
			zap.Error(err),
			zap.Uint("user_id", userID),
		)
		sendInternalError(c, "Failed to check push status")
		return
	}

	sendSuccess(c, gin.H{
		"enabled":          h.pushService.IsEnabled(),
		"has_subscription": hasSubs,
	})
}
