package handlers

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Constants for file upload limits
const (
	MultipartFormSizeAttachment = 512 << 20 // 512 MB
	MultipartFormSizeAvatar     = 10 << 20  // 10 MB
)

// getUserID extracts user ID from Gin context
func getUserID(c *gin.Context) (uint, bool) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	userID, ok := userIDInterface.(uint)
	if !ok {
		return 0, false
	}
	return userID, true
}

// requireUserID extracts user ID and sends 401 if not found
func requireUserID(c *gin.Context) (uint, bool) {
	userID, exists := getUserID(c)
	if !exists {
		sendUnauthorized(c)
		return 0, false
	}
	return userID, true
}

// parseUintParam parses uint from URL parameter
func parseUintParam(c *gin.Context, paramName string) (uint, error) {
	val, err := strconv.ParseUint(c.Param(paramName), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s", paramName)
	}
	return uint(val), nil
}

// Helper functions for common HTTP responses
// These are thin wrappers around response.go functions to maintain backward compatibility

func sendBadRequest(c *gin.Context, message string) {
	badRequest(c, message)
}

func sendUnauthorized(c *gin.Context) {
	unauthorized(c, "Unauthorized")
}

func sendForbidden(c *gin.Context, message string) {
	forbidden(c, message)
}

func sendNotFound(c *gin.Context, message string) {
	notFound(c, message)
}

func sendInternalError(c *gin.Context, message string) {
	internalError(c, message)
}

func sendSuccess(c *gin.Context, data interface{}) {
	jsonSuccess(c, data)
}
