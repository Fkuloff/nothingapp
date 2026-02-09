package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	db *gorm.DB
}

// NewHealthHandler creates a new health check handler
func NewHealthHandler(db *gorm.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status   string            `json:"status"`
	Time     string            `json:"time"`
	Services map[string]string `json:"services"`
}

// GetHealth returns the health status of the application
func (h *HealthHandler) GetHealth(c *gin.Context) {
	response := HealthResponse{
		Status:   "healthy",
		Time:     time.Now().UTC().Format(time.RFC3339),
		Services: make(map[string]string),
	}

	// Check database connection
	sqlDB, err := h.db.DB()
	if err != nil {
		response.Status = "unhealthy"
		response.Services["database"] = "error: " + err.Error()
		c.JSON(http.StatusServiceUnavailable, response)
		return
	}

	if err := sqlDB.Ping(); err != nil {
		response.Status = "unhealthy"
		response.Services["database"] = "error: " + err.Error()
		c.JSON(http.StatusServiceUnavailable, response)
		return
	}

	response.Services["database"] = "healthy"
	c.JSON(http.StatusOK, response)
}
