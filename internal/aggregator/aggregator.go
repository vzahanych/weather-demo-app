package aggregator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/vzahanych/weather-demo-app/internal/config"
	"github.com/vzahanych/weather-demo-app/internal/service"
	"github.com/vzahanych/weather-demo-app/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

type WeatherData struct {
	Services  map[string]interface{} `json:"services"`
	Timestamp string                 `json:"timestamp"`
}

type CacheEntry struct {
	Data      *WeatherData
	Timestamp time.Time
}

type Aggregator struct {
	services map[string]service.WeatherService
	cache    map[string]*CacheEntry
	mutex    sync.RWMutex
	cacheTTL time.Duration
	logger   *zap.Logger
	tele     *telemetry.Telemetry
	metrics  MetricsRecorder
}

// MetricsRecorder interface for recording metrics
type MetricsRecorder interface {
	RecordCacheHit(ctx context.Context, cacheType string)
	RecordCacheMiss(ctx context.Context, cacheType string)
}

func NewAggregator(cfg *config.WeatherConfig, logger *zap.Logger, tele *telemetry.Telemetry) *Aggregator {
	agg := &Aggregator{
		services: make(map[string]service.WeatherService),
		cache:    make(map[string]*CacheEntry),
		cacheTTL: time.Duration(cfg.CacheTTL) * time.Second,
		logger:   logger,
		tele:     tele,
	}

	for name, serviceConfig := range cfg.Services {
		if !serviceConfig.Enabled {
			continue
		}

		svc := agg.createService(name, serviceConfig, logger, tele)
		if svc != nil {
			agg.services[name] = svc
			agg.logger.Info("Registered weather service", zap.String("service", name))
		}
	}

	return agg
}

// SetMetricsRecorder sets the metrics recorder for the aggregator
func (a *Aggregator) SetMetricsRecorder(metrics MetricsRecorder) {
	a.metrics = metrics
}

func (a *Aggregator) createService(name string, cfg config.WeatherServiceConfig, logger *zap.Logger, tele *telemetry.Telemetry) service.WeatherService {
	switch cfg.Type {
	case "open-meteo":
		return service.NewOpenMeteoServiceWithConfig(cfg, logger, tele)
	case "weather-api":
		return service.NewWeatherAPIServiceWithConfig(cfg, logger, tele)
	default:
		logger.Warn("Unknown service type", zap.String("type", cfg.Type), zap.String("service", name))
		return nil
	}
}

func (a *Aggregator) GetWeatherData(ctx context.Context, lat, lon float64) (*WeatherData, error) {
	tracer := a.tele.GetTracer()
	ctx, span := tracer.Start(ctx, "aggregator.GetWeatherData")
	defer span.End()

	// Extract request ID from context for correlated logging
	requestID := ""
	if reqID := ctx.Value("request_id"); reqID != nil {
		if id, ok := reqID.(string); ok {
			requestID = id
		}
	}

	// Create logger with request ID
	reqLogger := a.logger
	if requestID != "" {
		reqLogger = a.logger.With(zap.String("request_id", requestID))
	}

	span.SetAttributes(
		attribute.Float64("lat", lat),
		attribute.Float64("lon", lon),
	)

	cacheKey := fmt.Sprintf("%.6f,%.6f", lat, lon)

	reqLogger.Debug("Weather data requested",
		zap.Float64("lat", lat),
		zap.Float64("lon", lon),
		zap.String("cache_key", cacheKey))

	if cached := a.getFromCache(cacheKey); cached != nil {
		reqLogger.Debug("Cache hit", zap.String("cache_key", cacheKey))
		span.SetAttributes(attribute.Bool("cache_hit", true))

		if a.metrics != nil {
			a.metrics.RecordCacheHit(ctx, "weather_data")
		}

		return cached, nil
	}

	span.SetAttributes(attribute.Bool("cache_hit", false))

	if a.metrics != nil {
		a.metrics.RecordCacheMiss(ctx, "weather_data")
	}

	reqLogger.Info("Cache miss, fetching fresh data",
		zap.String("cache_key", cacheKey),
		zap.Int("enabled_services", len(a.services)))

	data, err := a.fetchWeatherData(ctx, lat, lon)
	if err != nil {
		span.SetAttributes(attribute.Bool("success", false))
		reqLogger.Error("Failed to fetch weather data",
			zap.Error(err),
			zap.String("cache_key", cacheKey))
		return nil, err
	}

	a.setCache(cacheKey, data)
	span.SetAttributes(
		attribute.Bool("success", true),
		attribute.Int("services_count", len(data.Services)),
	)

	reqLogger.Info("Weather data fetched and cached",
		zap.String("cache_key", cacheKey),
		zap.Int("services_count", len(data.Services)))

	return data, nil
}

func (a *Aggregator) fetchWeatherData(ctx context.Context, lat, lon float64) (*WeatherData, error) {
	tracer := a.tele.GetTracer()
	ctx, span := tracer.Start(ctx, "aggregator.fetchWeatherData")
	defer span.End()

	span.SetAttributes(
		attribute.Float64("lat", lat),
		attribute.Float64("lon", lon),
		attribute.Int("services_count", len(a.services)),
	)

	a.mutex.RLock()
	services := make(map[string]service.WeatherService)
	for name, svc := range a.services {
		services[name] = svc
	}
	a.mutex.RUnlock()

	var wg sync.WaitGroup
	results := make(map[string]map[string]interface{})
	resultMutex := sync.Mutex{}

	for name, svc := range services {
		wg.Add(1)
		go func(serviceName string, weatherService interface {
			Get5DayForecast(ctx context.Context, lat, lon float64) (map[string]interface{}, error)
		}) {
			defer wg.Done()

			data, err := weatherService.Get5DayForecast(ctx, lat, lon)
			if err == nil {
				resultMutex.Lock()
				results[serviceName] = data
				resultMutex.Unlock()
			}
		}(name, svc)
	}

	wg.Wait()

	if len(results) == 0 {
		span.SetAttributes(attribute.Bool("success", false))
		return nil, fmt.Errorf("no weather data available")
	}

	servicesData := make(map[string]interface{})
	for name, data := range results {
		servicesData[name] = data
	}

	span.SetAttributes(
		attribute.Bool("success", true),
		attribute.Int("results_count", len(results)),
	)

	return &WeatherData{
		Services:  servicesData,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (a *Aggregator) getFromCache(key string) *WeatherData {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	entry, exists := a.cache[key]
	if !exists {
		return nil
	}

	if time.Since(entry.Timestamp) > a.cacheTTL {
		delete(a.cache, key)
		return nil
	}

	return entry.Data
}

func (a *Aggregator) setCache(key string, data *WeatherData) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.cache[key] = &CacheEntry{
		Data:      data,
		Timestamp: time.Now(),
	}
}

func (a *Aggregator) ClearCache() {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.cache = make(map[string]*CacheEntry)
}

func (a *Aggregator) GetCacheStats() map[string]interface{} {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	enabledServices := make([]string, 0, len(a.services))
	for name := range a.services {
		enabledServices = append(enabledServices, name)
	}

	return map[string]interface{}{
		"cache_size":       len(a.cache),
		"cache_ttl":        a.cacheTTL.String(),
		"enabled_services": enabledServices,
	}
}
