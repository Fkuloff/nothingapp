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

// pushHandler handles push notification REST endpoints
type pushHandler struct {
	pushService *services.PushNotificationService
	fcmService  *services.FCMService
	logger      *zap.Logger
}

// newPushHandler creates a new pushHandler
func newPushHandler(pushService *services.PushNotificationService, fcmService *services.FCMService, logger *zap.Logger) *pushHandler {
	return &pushHandler{
		pushService: pushService,
		fcmService:  fcmService,
		logger:      logger,
	}
}

// GetVAPIDKey returns the VAPID public key for frontend subscription
func (h *pushHandler) GetVAPIDKey(c *gin.Context) {
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
func (h *pushHandler) Subscribe(c *gin.Context) {
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
//
//nolint:dupl // Structural twin of UnregisterFCM, but operates on different subscription types.
func (h *pushHandler) Unsubscribe(c *gin.Context) {
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

type fcmRegisterRequest struct {
	Token    string `json:"token" binding:"required"`
	Platform string `json:"platform" binding:"required"`
}

// RegisterFCM stores an FCM device token for the current user.
func (h *pushHandler) RegisterFCM(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	var req fcmRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendBadRequest(c, "Invalid FCM registration data")
		return
	}
	if req.Platform != "android" && req.Platform != "ios" {
		sendBadRequest(c, "Unsupported platform")
		return
	}

	if err := h.fcmService.Register(c.Request.Context(), userID, req.Token, req.Platform); err != nil {
		if strings.Contains(err.Error(), "limit reached") {
			sendBadRequest(c, err.Error())
			return
		}
		h.logger.Error("failed to register FCM token", zap.Error(err), zap.Uint("user_id", userID))
		sendInternalError(c, "Failed to register FCM token")
		return
	}
	sendSuccess(c, gin.H{"message": "FCM token registered"})
}

type fcmUnregisterRequest struct {
	Token string `json:"token" binding:"required"`
}

// UnregisterFCM removes an FCM device token.
//
//nolint:dupl // Structural twin of Unsubscribe.
func (h *pushHandler) UnregisterFCM(c *gin.Context) {
	userID, ok := requireUserID(c)
	if !ok {
		return
	}

	var req fcmUnregisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendBadRequest(c, "Invalid request")
		return
	}

	if err := h.fcmService.Unregister(c.Request.Context(), userID, req.Token); err != nil {
		h.logger.Error("failed to unregister FCM token", zap.Error(err), zap.Uint("user_id", userID))
		sendInternalError(c, "Failed to unregister FCM token")
		return
	}
	sendSuccess(c, gin.H{"message": "FCM token unregistered"})
}

// GetStatus returns whether push notifications are enabled for the current user
func (h *pushHandler) GetStatus(c *gin.Context) {
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
