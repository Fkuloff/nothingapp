package handlers

import (
	"crypto/subtle"
	"errors"
	"net/http"

	"messenger/internal/services"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// appReleaseHandler exposes the public latest-release lookup + the
// admin-only release-registration endpoint.
type appReleaseHandler struct {
	service     *services.AppReleaseService
	log         *zap.Logger
	adminAPIKey string
}

func newAppReleaseHandler(svc *services.AppReleaseService, adminAPIKey string, log *zap.Logger) *appReleaseHandler {
	return &appReleaseHandler{service: svc, adminAPIKey: adminAPIKey, log: log}
}

// GetLatestRelease is the public endpoint clients poll on startup.
// 200 + JSON   → release exists; client compares VersionCode locally
// 204 No Content → no releases registered for this platform yet
//
// Cache-Control is `no-store` because the in-app updater wants the freshest
// row possible, and the response is so small (~300 bytes) that caching adds
// no value.
func (h *appReleaseHandler) GetLatestRelease(c *gin.Context) {
	platform := c.DefaultQuery("platform", services.PlatformAndroid)
	rel, err := h.service.GetLatest(c.Request.Context(), platform)
	if err != nil {
		h.log.Error("get latest release", zap.Error(err), zap.String("platform", platform))
		sendInternalError(c, "internal error")
		return
	}
	c.Header("Cache-Control", "no-store")
	if rel == nil {
		c.Status(http.StatusNoContent)
		return
	}
	sendSuccess(c, rel)
}

// CreateRelease is the admin-only registration POST. Auth via X-Admin-Key
// header compared with constant-time equality so it doesn't leak the
// length of the configured secret through timing.
//
// CI workflow (when set up):
//   1. CI builds + signs APK, computes its SHA-256
//   2. CI uploads APK to a public bucket
//   3. CI POSTs the metadata here with X-Admin-Key=$ADMIN_API_KEY
//   4. Backend persists the row + fires the WS broadcast hook
func (h *appReleaseHandler) CreateRelease(c *gin.Context) {
	if h.adminAPIKey == "" {
		// Refuse rather than 401 — clearer signal to admins that the env
		// var simply isn't set, not that their key is wrong.
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "admin endpoints disabled (ADMIN_API_KEY not set)"})
		return
	}
	provided := c.GetHeader("X-Admin-Key")
	if subtle.ConstantTimeCompare([]byte(provided), []byte(h.adminAPIKey)) != 1 {
		sendUnauthorized(c)
		return
	}

	var req services.CreateReleaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendBadRequest(c, "invalid JSON body")
		return
	}

	rel, err := h.service.CreateRelease(c.Request.Context(), req)
	if err != nil {
		if errors.Is(err, services.ErrReleaseConflict) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		// Admin endpoint — trusted caller. Surface validation + persistence
		// errors verbatim so the operator can fix env / payload directly
		// instead of digging through server logs. Still log for audit.
		h.log.Warn("create release rejected", zap.Error(err))
		sendBadRequest(c, err.Error())
		return
	}

	sendCreated(c, rel)
}
