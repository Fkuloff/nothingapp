package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// healthHandler handles health check requests
type healthHandler struct {
	db *gorm.DB
}

// newHealthHandler creates a new health check handler
func newHealthHandler(db *gorm.DB) *healthHandler {
	return &healthHandler{db: db}
}

// healthResponse represents the health check response
type healthResponse struct {
	Status   string            `json:"status"`
	Time     string            `json:"time"`
	Services map[string]string `json:"services"`
}

// GetHealth returns the health status of the application
func (h *healthHandler) GetHealth(c *gin.Context) {
	response := healthResponse{
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
