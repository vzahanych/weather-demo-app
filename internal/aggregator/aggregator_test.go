package aggregator

import (
	"testing"
	"time"

	"github.com/vzahanych/weather-demo-app/internal/config"
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

	agg := NewAggregator(cfg)
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

	agg := NewAggregator(cfg)

	cacheKey := "52.520000,13.410000"

	if agg.getFromCache(cacheKey) != nil {
		t.Error("Cache should be empty initially")
	}

	testData := &AggregatedWeatherData{
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

	agg := NewAggregator(cfg)

	cacheKey := "52.520000,13.410000"

	testData := &AggregatedWeatherData{
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

	agg := NewAggregator(cfg)

	stats := agg.GetCacheStats()
	if stats["cache_size"] != 0 {
		t.Errorf("Expected cache size 0, got %v", stats["cache_size"])
	}

	if stats["cache_ttl"] != "5m0s" {
		t.Errorf("Expected cache TTL 5m0s, got %v", stats["cache_ttl"])
	}
}

func TestAggregatedWeatherDataHelpers(t *testing.T) {
	// Test with empty data
	emptyData := &AggregatedWeatherData{
		Services:  make(map[string]interface{}),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	if emptyData.HasService("open-meteo") {
		t.Error("Empty data should not have any services")
	}

	services := emptyData.GetAvailableServices()
	if len(services) != 0 {
		t.Errorf("Expected 0 services, got %d", len(services))
	}

	_, exists := emptyData.GetServiceData("open-meteo")
	if exists {
		t.Error("Should not find service data for non-existent service")
	}

	// Test with populated data
	populatedData := &AggregatedWeatherData{
		Services: map[string]interface{}{
			"open-meteo": map[string]interface{}{
				"temperature": 20.5,
				"humidity":    65,
			},
			"weather-api": map[string]interface{}{
				"temp": 22.0,
				"hum":  60,
			},
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	if !populatedData.HasService("open-meteo") {
		t.Error("Should have open-meteo service")
	}

	if !populatedData.HasService("weather-api") {
		t.Error("Should have weather-api service")
	}

	if populatedData.HasService("non-existent") {
		t.Error("Should not have non-existent service")
	}

	services = populatedData.GetAvailableServices()
	if len(services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(services))
	}

	// Check that both services are in the list
	hasOpenMeteo := false
	hasWeatherAPI := false
	for _, service := range services {
		if service == "open-meteo" {
			hasOpenMeteo = true
		}
		if service == "weather-api" {
			hasWeatherAPI = true
		}
	}

	if !hasOpenMeteo {
		t.Error("open-meteo service should be in available services")
	}

	if !hasWeatherAPI {
		t.Error("weather-api service should be in available services")
	}

	// Test GetServiceData
	openMeteoData, exists := populatedData.GetServiceData("open-meteo")
	if !exists {
		t.Error("Should find open-meteo service data")
	}

	if openMeteoData["temperature"] != 20.5 {
		t.Errorf("Expected temperature 20.5, got %v", openMeteoData["temperature"])
	}

	if openMeteoData["humidity"] != 65 {
		t.Errorf("Expected humidity 65, got %v", openMeteoData["humidity"])
	}

	weatherAPIData, exists := populatedData.GetServiceData("weather-api")
	if !exists {
		t.Error("Should find weather-api service data")
	}

	if weatherAPIData["temp"] != 22.0 {
		t.Errorf("Expected temp 22.0, got %v", weatherAPIData["temp"])
	}

	if weatherAPIData["hum"] != 60 {
		t.Errorf("Expected hum 60, got %v", weatherAPIData["hum"])
	}
}
