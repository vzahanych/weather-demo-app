package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vzahanych/weather-demo-app/internal/server/utils"
	"go.uber.org/zap"
)

type HealthHandler struct {
	logger    *zap.Logger
	startTime time.Time
}

func NewHealthHandler(logger *zap.Logger) *HealthHandler {
	return &HealthHandler{
		logger:    logger,
		startTime: time.Now(),
	}
}

// validateAndRespond validates health response and sends it
func (h *HealthHandler) validateAndRespond(c *gin.Context, response HealthResponse) {
	if validationErrors := utils.ValidateStruct(response); validationErrors != nil {
		h.logger.Error("Health response validation failed",
			zap.Any("validation_errors", validationErrors))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error: invalid health response",
			Code:    "HEALTH_VALIDATION_ERROR",
			Details: "Health response failed validation",
		})
		return
	}
	c.JSON(http.StatusOK, response)
}

func (h *HealthHandler) Liveness(c *gin.Context) {
	response := HealthResponse{
		Status: "ok",
		Uptime: time.Since(h.startTime).String(),
	}
	h.validateAndRespond(c, response)
}

func (h *HealthHandler) Readiness(c *gin.Context) {
	response := HealthResponse{
		Status: "ok",
		Uptime: time.Since(h.startTime).String(),
	}
	h.validateAndRespond(c, response)
}

func (h *HealthHandler) Health(c *gin.Context) {
	response := HealthResponse{
		Status:    "ok",
		Uptime:    time.Since(h.startTime).String(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	h.validateAndRespond(c, response)
}
