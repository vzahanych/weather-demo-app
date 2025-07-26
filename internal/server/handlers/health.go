package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
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

func (h *HealthHandler) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status: "alive",
		Uptime: time.Since(h.startTime).String(),
	})
}

func (h *HealthHandler) Readiness(c *gin.Context) {

	c.JSON(http.StatusOK, HealthResponse{
		Status: "ready",
		Uptime: time.Since(h.startTime).String(),
	})
}

func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status:    "ok",
		Uptime:    time.Since(h.startTime).String(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}
