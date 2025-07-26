package handlers

import (
	"context"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AppMetrics holds application-level metrics (cache, services, etc.)
type AppMetrics struct {
	mutex                sync.RWMutex
	cacheHits            int64
	cacheMisses          int64
	weatherServiceCalls  map[string]int64
	weatherServiceErrors map[string]int64
}

type MetricsHandler struct {
	logger     *zap.Logger
	appMetrics *AppMetrics
}

func NewMetricsHandler(logger *zap.Logger) *MetricsHandler {
	return &MetricsHandler{
		logger: logger,
		appMetrics: &AppMetrics{
			weatherServiceCalls:  make(map[string]int64),
			weatherServiceErrors: make(map[string]int64),
		},
	}
}

// RecordCacheHit records a cache hit metric
func (h *MetricsHandler) RecordCacheHit(ctx context.Context, cacheType string) {
	h.appMetrics.mutex.Lock()
	h.appMetrics.cacheHits++
	h.appMetrics.mutex.Unlock()
}

// RecordCacheMiss records a cache miss metric
func (h *MetricsHandler) RecordCacheMiss(ctx context.Context, cacheType string) {
	h.appMetrics.mutex.Lock()
	h.appMetrics.cacheMisses++
	h.appMetrics.mutex.Unlock()
}

// RecordWeatherServiceCall records a weather service API call
func (h *MetricsHandler) RecordWeatherServiceCall(ctx context.Context, service string, success bool) {
	h.appMetrics.mutex.Lock()
	h.appMetrics.weatherServiceCalls[service]++
	if !success {
		h.appMetrics.weatherServiceErrors[service]++
	}
	h.appMetrics.mutex.Unlock()
}

// ServeMetrics returns a Gin handler that exposes metrics in Prometheus format
// This method will be called with HTTP metrics injected via Gin context
func (h *MetricsHandler) ServeMetrics(c *gin.Context) {
	h.appMetrics.mutex.RLock()
	defer h.appMetrics.mutex.RUnlock()

	// Get HTTP metrics from context (will be set by server)
	httpMetrics := h.getHTTPMetricsFromContext(c)

	// Build simple text metrics response
	response := ""

	// HTTP Metrics (if available)
	if httpMetrics != nil {
		httpMetrics.mutex.RLock()

		// Calculate average duration
		var avgDuration float64
		if len(httpMetrics.requestDurations) > 0 {
			sum := 0.0
			for _, d := range httpMetrics.requestDurations {
				sum += d
			}
			avgDuration = sum / float64(len(httpMetrics.requestDurations))
		}

		response += "# HELP http_requests_total Total number of HTTP requests\n"
		response += "# TYPE http_requests_total counter\n"
		for key, count := range httpMetrics.requestsTotal {
			response += "http_requests_total{route_status=\"" + key + "\"} " + strconv.FormatInt(count, 10) + "\n"
		}

		response += "\n# HELP http_request_duration_seconds_avg Average duration of HTTP requests\n"
		response += "# TYPE http_request_duration_seconds_avg gauge\n"
		response += "http_request_duration_seconds_avg " + strconv.FormatFloat(avgDuration, 'f', 6, 64) + "\n"

		response += "\n# HELP http_active_requests Number of active HTTP requests\n"
		response += "# TYPE http_active_requests gauge\n"
		response += "http_active_requests " + strconv.FormatInt(httpMetrics.activeRequests, 10) + "\n"

		httpMetrics.mutex.RUnlock()
	}

	// Application Metrics
	response += "\n# HELP aggregator_cache_hits_total Total cache hits\n"
	response += "# TYPE aggregator_cache_hits_total counter\n"
	response += "aggregator_cache_hits_total " + strconv.FormatInt(h.appMetrics.cacheHits, 10) + "\n"

	response += "\n# HELP aggregator_cache_miss_total Total cache misses\n"
	response += "# TYPE aggregator_cache_miss_total counter\n"
	response += "aggregator_cache_miss_total " + strconv.FormatInt(h.appMetrics.cacheMisses, 10) + "\n"

	response += "\n# HELP weather_service_calls_total Total weather service calls\n"
	response += "# TYPE weather_service_calls_total counter\n"
	for service, count := range h.appMetrics.weatherServiceCalls {
		response += "weather_service_calls_total{service=\"" + service + "\"} " + strconv.FormatInt(count, 10) + "\n"
	}

	response += "\n# HELP weather_service_errors_total Total weather service errors\n"
	response += "# TYPE weather_service_errors_total counter\n"
	for service, count := range h.appMetrics.weatherServiceErrors {
		response += "weather_service_errors_total{service=\"" + service + "\"} " + strconv.FormatInt(count, 10) + "\n"
	}

	c.Header("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	c.String(200, response)
}

// HTTPMetricsProvider interface for getting HTTP metrics from middleware
type HTTPMetricsProvider interface {
	GetHTTPMetrics() interface{}
}

func (h *MetricsHandler) getHTTPMetricsFromContext(c *gin.Context) *HTTPMetrics {
	if value, exists := c.Get("http_metrics"); exists {
		if metrics, ok := value.(*HTTPMetrics); ok {
			return metrics
		}
	}
	return nil
}

// HTTPMetrics struct to match middleware structure
type HTTPMetrics struct {
	mutex            sync.RWMutex
	requestsTotal    map[string]int64
	requestDurations []float64
	activeRequests   int64
}
