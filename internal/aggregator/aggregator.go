package aggregator

import (
	"fmt"
	"sync"
	"time"

	"github.com/vzahanych/weather-demo-app/internal/config"
	"github.com/vzahanych/weather-demo-app/internal/service"
)

var (
	integratedServices = map[string]bool{
		"open-meteo":  true,
		"weather-api": true,
	}
)

// AggregatedWeatherData represents flexible weather data from multiple services
type AggregatedWeatherData struct {
	Services  map[string]interface{} `json:"services,omitempty"`
	Timestamp string                 `json:"timestamp"`
}

type CacheEntry struct {
	Data      *AggregatedWeatherData
	Timestamp time.Time
	TTL       time.Duration
}

type Aggregator struct {
	services map[string]service.WeatherService
	cache    map[string]*CacheEntry
	mutex    sync.RWMutex
	cacheTTL time.Duration
}

func NewAggregator(cfg *config.WeatherConfig) *Aggregator {
	agg := &Aggregator{
		services: make(map[string]service.WeatherService),
		cache:    make(map[string]*CacheEntry),
		cacheTTL: time.Duration(cfg.CacheTTL) * time.Second,
	}

	agg.registerServicesFromConfig(cfg)
	return agg
}

func (a *Aggregator) registerServicesFromConfig(cfg *config.WeatherConfig) {
	enabledServices := cfg.GetEnabledServices()

	for name, serviceConfig := range enabledServices {
		if !integratedServices[name] {
			continue
		}

		service := a.createServiceFromConfig(name, serviceConfig)
		if service != nil {
			a.RegisterService(service)
		}
	}
}

func (a *Aggregator) createServiceFromConfig(name string, cfg config.WeatherServiceConfig) service.WeatherService {
	switch name {
	case "open-meteo":
		return service.NewOpenMeteoServiceWithConfig(cfg)
	case "weather-api":
		return service.NewWeatherAPIServiceWithConfig(cfg)
	default:
		return nil
	}
}

func (a *Aggregator) RegisterService(service service.WeatherService) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.services[service.Name()] = service
}

func (a *Aggregator) GetWeatherData(lat, lon float64) (*AggregatedWeatherData, error) {
	cacheKey := fmt.Sprintf("%.6f,%.6f", lat, lon)

	if cached := a.getFromCache(cacheKey); cached != nil {
		return cached, nil
	}

	data, err := a.fetchWeatherData(lat, lon)
	if err != nil {
		return nil, err
	}

	a.setCache(cacheKey, data)
	return data, nil
}

func (a *Aggregator) fetchWeatherData(lat, lon float64) (*AggregatedWeatherData, error) {
	a.mutex.RLock()
	services := make(map[string]service.WeatherService, len(a.services))
	for name, service := range a.services {
		services[name] = service
	}
	a.mutex.RUnlock()

	var wg sync.WaitGroup
	results := make(map[string]map[string]interface{})
	errors := make(map[string]error)
	resultMutex := sync.Mutex{}

	for name, srv := range services {
		wg.Add(1)
		go func(serviceName string, weatherService service.WeatherService) {
			defer wg.Done()

			data, err := weatherService.Get5DayForecast(lat, lon)
			resultMutex.Lock()
			defer resultMutex.Unlock()

			if err != nil {
				errors[serviceName] = err
			} else {
				results[serviceName] = data
			}
		}(name, srv)
	}

	// Wait for all services to complete either successfully or with an error
	wg.Wait()

	aggregated := &AggregatedWeatherData{
		Services:  make(map[string]interface{}),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	for name, data := range results {
		aggregated.Services[name] = data
	}

	if len(errors) > 0 {
		return aggregated, fmt.Errorf("some services failed: %v", errors)
	}

	return aggregated, nil
}

func (a *Aggregator) getFromCache(key string) *AggregatedWeatherData {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	entry, exists := a.cache[key]
	if !exists {
		return nil
	}

	if time.Since(entry.Timestamp) > entry.TTL {
		delete(a.cache, key)
		return nil
	}

	return entry.Data
}

func (a *Aggregator) setCache(key string, data *AggregatedWeatherData) {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.cache[key] = &CacheEntry{
		Data:      data,
		Timestamp: time.Now(),
		TTL:       a.cacheTTL,
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

	return map[string]interface{}{
		"cache_size": len(a.cache),
		"cache_ttl":  a.cacheTTL.String(),
	}
}

// GetServiceData returns weather data for a specific service
func (data *AggregatedWeatherData) GetServiceData(serviceName string) (map[string]interface{}, bool) {
	if data.Services == nil {
		return nil, false
	}
	serviceData, exists := data.Services[serviceName]
	if !exists {
		return nil, false
	}

	if result, ok := serviceData.(map[string]interface{}); ok {
		return result, true
	}
	return nil, false
}

// GetAvailableServices returns a list of available service names
func (data *AggregatedWeatherData) GetAvailableServices() []string {
	if data.Services == nil {
		return []string{}
	}

	services := make([]string, 0, len(data.Services))
	for serviceName := range data.Services {
		services = append(services, serviceName)
	}
	return services
}

// HasService checks if data from a specific service is available
func (data *AggregatedWeatherData) HasService(serviceName string) bool {
	if data.Services == nil {
		return false
	}
	_, exists := data.Services[serviceName]
	return exists
}
