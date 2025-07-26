package aggregator

import (
	"testing"
	"time"

	"github.com/vzahanych/weather-demo-app/internal/config"
	"github.com/vzahanych/weather-demo-app/pkg/telemetry"
	"go.uber.org/zap"
)

func TestNewAggregator(t *testing.T) {
	cfg := &config.WeatherConfig{
		CacheTTL: 300,
		Services: map[string]config.WeatherServiceConfig{
			"open-meteo": {
				Type:    "open-meteo",
				Enabled: true,
				BaseURL: "https://api.open-meteo.com/v1",
			},
		},
	}

	logger, _ := zap.NewDevelopment()
	tele := &telemetry.Telemetry{}

	agg := NewAggregator(cfg, logger, tele)
	if agg == nil {
		t.Fatal("NewAggregator returned nil")
	}

	if agg.cacheTTL != 5*time.Minute {
		t.Errorf("Expected cache TTL to be 5 minutes, got %v", agg.cacheTTL)
	}

	if len(agg.services) != 1 {
		t.Errorf("Expected 1 service to be registered, got %d", len(agg.services))
	}
}

func TestAggregatorCache(t *testing.T) {
	cfg := &config.WeatherConfig{
		CacheTTL: 1,
		Services: map[string]config.WeatherServiceConfig{},
	}

	logger, _ := zap.NewDevelopment()
	tele := &telemetry.Telemetry{}

	agg := NewAggregator(cfg, logger, tele)

	cacheKey := "52.520000,13.410000"

	if agg.getFromCache(cacheKey) != nil {
		t.Error("Cache should be empty initially")
	}

	testData := &WeatherData{
		Services:  make(map[string]interface{}),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	agg.setCache(cacheKey, testData)

	cached := agg.getFromCache(cacheKey)
	if cached == nil {
		t.Error("Expected to find data in cache")
	}

	if cached.Timestamp != testData.Timestamp {
		t.Errorf("Expected timestamp %s, got %s", testData.Timestamp, cached.Timestamp)
	}
}

func TestAggregatorCacheExpiration(t *testing.T) {
	cfg := &config.WeatherConfig{
		CacheTTL: 0,
		Services: map[string]config.WeatherServiceConfig{},
	}

	logger, _ := zap.NewDevelopment()
	tele := &telemetry.Telemetry{}

	agg := NewAggregator(cfg, logger, tele)

	cacheKey := "52.520000,13.410000"

	testData := &WeatherData{
		Services:  make(map[string]interface{}),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	agg.setCache(cacheKey, testData)

	time.Sleep(10 * time.Millisecond)

	cached := agg.getFromCache(cacheKey)
	if cached != nil {
		t.Error("Cache entry should have expired")
	}
}

func TestAggregatorGetCacheStats(t *testing.T) {
	cfg := &config.WeatherConfig{
		CacheTTL: 300,
		Services: map[string]config.WeatherServiceConfig{},
	}

	logger, _ := zap.NewDevelopment()
	tele := &telemetry.Telemetry{}

	agg := NewAggregator(cfg, logger, tele)

	stats := agg.GetCacheStats()
	if stats["cache_size"] != 0 {
		t.Errorf("Expected cache size 0, got %v", stats["cache_size"])
	}

	if stats["cache_ttl"] != "5m0s" {
		t.Errorf("Expected cache TTL 5m0s, got %v", stats["cache_ttl"])
	}
}

func TestWeatherDataBasic(t *testing.T) {
	// Test with empty data
	emptyData := &WeatherData{
		Services:  make(map[string]interface{}),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	if len(emptyData.Services) != 0 {
		t.Error("Empty data should not have any services")
	}

	// Test with populated data
	populatedData := &WeatherData{
		Services: map[string]interface{}{
			"open-meteo": map[string]interface{}{
				"temperature": 20.5,
				"humidity":    65,
			},
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	if len(populatedData.Services) != 1 {
		t.Errorf("Expected 1 service, got %d", len(populatedData.Services))
	}

	if _, exists := populatedData.Services["open-meteo"]; !exists {
		t.Error("Should have open-meteo service")
	}
}
