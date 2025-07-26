package middlewares

import (
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vzahanych/weather-demo-app/pkg/telemetry"
	"go.uber.org/zap"
)

// HTTPMetrics holds only HTTP request metrics
type HTTPMetrics struct {
	mutex            sync.RWMutex
	requestsTotal    map[string]int64
	requestDurations []float64
	activeRequests   int64
}

type MetricsMiddleware struct {
	logger  *zap.Logger
	tele    *telemetry.Telemetry
	metrics *HTTPMetrics
}

func NewMetricsMiddleware(logger *zap.Logger, tele *telemetry.Telemetry) (*MetricsMiddleware, error) {
	return &MetricsMiddleware{
		logger: logger,
		tele:   tele,
		metrics: &HTTPMetrics{
			requestsTotal:    make(map[string]int64),
			requestDurations: make([]float64, 0),
		},
	}, nil
}

func (m *MetricsMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Increment active requests
		m.metrics.mutex.Lock()
		m.metrics.activeRequests++
		m.metrics.mutex.Unlock()

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start).Seconds()

		// Record metrics
		statusCode := strconv.Itoa(c.Writer.Status())
		route := c.FullPath()
		method := c.Request.Method
		key := method + " " + route + "_" + statusCode

		m.metrics.mutex.Lock()
		m.metrics.requestsTotal[key]++
		m.metrics.requestDurations = append(m.metrics.requestDurations, duration)
		m.metrics.activeRequests--

		// Keep only last 1000 durations to prevent memory leak
		if len(m.metrics.requestDurations) > 1000 {
			m.metrics.requestDurations = m.metrics.requestDurations[len(m.metrics.requestDurations)-1000:]
		}
		m.metrics.mutex.Unlock()

		if m.tele.IsEnabled() {
			m.logger.Debug("HTTP metrics recorded",
				zap.String("method", method),
				zap.String("route", route),
				zap.Int("status", c.Writer.Status()),
				zap.Float64("duration", duration))
		}
	}
}

// GetHTTPMetrics returns the HTTP metrics for the metrics handler to expose
func (m *MetricsMiddleware) GetHTTPMetrics() *HTTPMetrics {
	return m.metrics
}
