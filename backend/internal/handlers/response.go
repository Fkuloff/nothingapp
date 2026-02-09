package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// jsonError sends a JSON error response with the given status code
func jsonError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}

// jsonSuccess sends a JSON success response with HTTP 200
func jsonSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, data)
}

// Specialized error response functions

func badRequest(c *gin.Context, msg string) {
	jsonError(c, http.StatusBadRequest, msg)
}

func unauthorized(c *gin.Context, msg string) {
	jsonError(c, http.StatusUnauthorized, msg)
}

func forbidden(c *gin.Context, msg string) {
	jsonError(c, http.StatusForbidden, msg)
}

func notFound(c *gin.Context, msg string) {
	jsonError(c, http.StatusNotFound, msg)
}

func internalError(c *gin.Context, msg string) {
	jsonError(c, http.StatusInternalServerError, msg)
}
