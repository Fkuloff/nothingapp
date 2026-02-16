package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Constants for file upload limits
const (
	MultipartFormSizeAttachment = 512 << 20 // 512 MB
	MultipartFormSizeAvatar     = 10 << 20  // 10 MB
)

// HTTP response helpers

func sendSuccess(c *gin.Context, data any) {
	c.JSON(http.StatusOK, data)
}

func sendCreated(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, data)
}

func sendBadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, gin.H{"error": message})
}

func sendUnauthorized(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
}

func sendForbidden(c *gin.Context, message string) {
	c.JSON(http.StatusForbidden, gin.H{"error": message})
}

func sendNotFound(c *gin.Context, message string) {
	c.JSON(http.StatusNotFound, gin.H{"error": message})
}

func sendInternalError(c *gin.Context, message string) {
	c.JSON(http.StatusInternalServerError, gin.H{"error": message})
}

// Context helpers

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

func requireUserID(c *gin.Context) (uint, bool) {
	userID, exists := getUserID(c)
	if !exists {
		sendUnauthorized(c)
		return 0, false
	}
	return userID, true
}

func parseUintParam(c *gin.Context, paramName string) (uint, error) {
	val, err := strconv.ParseUint(c.Param(paramName), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s", paramName)
	}
	return uint(val), nil
}

// serveReaderContent streams a reader to the HTTP response with the given content type and cache control.
func serveReaderContent(c *gin.Context, reader io.ReadCloser, contentType, cacheControl string) {
	defer reader.Close()
	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", cacheControl)
	c.Status(http.StatusOK)
	if _, err := io.Copy(c.Writer, reader); err != nil {
		c.Error(err) //nolint:errcheck
	}
}
