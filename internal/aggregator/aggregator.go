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

var (
	workers = 5 // just a default value, can be overridden by config
)

// There is an affort to implement throttling
// That should be tested and probably refactored
type HandlerTimeoutError struct {
	Message string
	Timeout time.Duration
}

func (e *HandlerTimeoutError) Error() string {
	return e.Message
}

func (e *HandlerTimeoutError) IsTimeout() bool {
	return true
}

type QueueFullError struct {
	Message string
}

func (e *QueueFullError) Error() string {
	return e.Message
}

func (e *QueueFullError) IsOverloaded() bool {
	return true
}

type WeatherData struct {
	Services  map[string]interface{} `json:"services"`
	Timestamp string                 `json:"timestamp"`
}

type CacheEntry struct {
	Data      *WeatherData
	Timestamp time.Time
}

// Task represents a weather data fetch task 
// (I think this is the right place for it, but not 100% sure, 
// TODO: revisit if we need more fields here)
type Task struct {
	ID        string
	Lat       float64
	Lon       float64
	Context   context.Context
	ResultCh  chan TaskResult
	CreatedAt time.Time
}


type TaskResult struct {
	Data  *WeatherData
	Error error
}

// Aggregator is supposed to be a singleton,
// It should have a logic for autorecovering from panics
// TODO: revisit to make it more stable
type Aggregator struct {
	services   map[string]service.WeatherService
	cache      map[string]*CacheEntry
	cacheMutex sync.RWMutex
	cacheTTL   time.Duration
	logger     *zap.Logger
	tele       *telemetry.Telemetry
	metrics    MetricsRecorder

	// Worker pool components
	taskQueue    chan *Task
	workers      int
	workerWg     sync.WaitGroup
	shutdownCh   chan struct{}
	isRunning    bool
	runningMutex sync.RWMutex

	// Request deduplication
	pendingTasks map[string][]*Task
	pendingMutex sync.Mutex
}

// Metrics is broken right now
// It is not visible in Grafana
// TODO: fix it
type MetricsRecorder interface {
	RecordCacheHit(ctx context.Context, cacheType string)
	RecordCacheMiss(ctx context.Context, cacheType string)
	RecordHandlerTimeout(ctx context.Context, reason string)
	RecordQueueFull(ctx context.Context)
}

func NewAggregator(cfg *config.WeatherConfig, logger *zap.Logger, tele *telemetry.Telemetry) *Aggregator {
	
	if cfg.Workers > 0 {
		workers = cfg.Workers
	}

	agg := &Aggregator{
		services:     make(map[string]service.WeatherService),
		cache:        make(map[string]*CacheEntry),
		cacheTTL:     time.Duration(cfg.CacheTTL) * time.Second,
		logger:       logger,
		tele:         tele,
		taskQueue:    make(chan *Task, 100),
		workers:      workers,
		shutdownCh:   make(chan struct{}),
		pendingTasks: make(map[string][]*Task),
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

// Start begins the just worker pool
// TODO: add logic for autorecovering from panics
func (a *Aggregator) Start(ctx context.Context) error {
	a.runningMutex.Lock()
	defer a.runningMutex.Unlock()

	if a.isRunning {
		return fmt.Errorf("aggregator is already running")
	}

	a.logger.Info("Starting aggregator worker pool", zap.Int("workers", a.workers))

	for i := 0; i < a.workers; i++ {
		a.workerWg.Add(1)
		worker := NewAggregatorWorker(a, i)
		go worker.Start(ctx)
	}

	a.isRunning = true
	return nil
}

func (a *Aggregator) Stop(ctx context.Context) error {
	a.runningMutex.Lock()
	defer a.runningMutex.Unlock()

	if !a.isRunning {
		return nil
	}

	a.logger.Info("Stopping aggregator worker pool")

	close(a.shutdownCh)
	close(a.taskQueue)

	done := make(chan struct{})
	go func() {
		a.workerWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		a.logger.Info("All workers stopped gracefully")
	case <-ctx.Done():
		a.logger.Warn("Worker shutdown timeout")
	}

	a.isRunning = false
	return nil
}

// That is part of the request deduplication logic
// The main idea is to avoid server overload
// TODO: need to be carefully tested  under load
func (a *Aggregator) notifyPendingTasks(taskKey string, result TaskResult) {
	a.pendingMutex.Lock() 
	defer a.pendingMutex.Unlock()

	tasks, exists := a.pendingTasks[taskKey]
	if !exists {
		return
	}

	for _, task := range tasks {
		select {
		case task.ResultCh <- result:
		default:
			// Can't write to the channel, skip this task
		}
	}

	delete(a.pendingTasks, taskKey)
}

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
		logger.Warn("Given service isnot integrated", zap.String("type", cfg.Type), zap.String("service", name))
		return nil
	}
}

func (a *Aggregator) GetWeatherData(ctx context.Context, lat, lon float64) (*WeatherData, error) {
	tracer := a.tele.GetTracer()
	ctx, span := tracer.Start(ctx, "aggregator.GetWeatherData")
	defer span.End()

	// No time to make that smatter yet	
	requestID := ""
	if reqID := ctx.Value("request_id"); reqID != nil {
		if id, ok := reqID.(string); ok {
			requestID = id
		}
	}

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

	// Our cache is vary simple just for  demo purposes
	// Maybe we might cache single days to make less requests to the services
	// TODO: revisit that
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

	a.runningMutex.RLock()
	isRunning := a.isRunning
	a.runningMutex.RUnlock()

	if !isRunning {
		return nil, fmt.Errorf("aggregator worker pool is not running")
	}

	cfg := config.GetConfig()
	handlerTimeout := time.Duration(cfg.Weather.HandlerTimeout) * time.Second

	timeoutCtx, timeoutCancel := context.WithTimeout(ctx, handlerTimeout)
	defer timeoutCancel()

	a.pendingMutex.Lock()

	// Check if there's already a pending task for these coordinates
	if existingTasks, exists := a.pendingTasks[cacheKey]; exists {
		// Add this request to the pending list
		task := &Task{
			ID:        fmt.Sprintf("%s-%d", cacheKey, len(existingTasks)),
			Lat:       lat,
			Lon:       lon,
			Context:   timeoutCtx,
			ResultCh:  make(chan TaskResult, 1),
			CreatedAt: time.Now(),
		}

		a.pendingTasks[cacheKey] = append(existingTasks, task)
		a.pendingMutex.Unlock()

		reqLogger.Debug("Joining existing task",
			zap.String("cache_key", cacheKey),
			zap.Duration("timeout", handlerTimeout))

		// Wait for the result with timeout
		select {
		case result := <-task.ResultCh:
			return result.Data, result.Error
		case <-timeoutCtx.Done():
			reqLogger.Warn("Handler timeout while waiting for existing task",
				zap.String("cache_key", cacheKey),
				zap.Duration("timeout", handlerTimeout))
			if a.metrics != nil {
				a.metrics.RecordHandlerTimeout(ctx, "existing_task")
			}
			return nil, &HandlerTimeoutError{
				Message: "request timeout: server is overloaded",
				Timeout: handlerTimeout,
			}
		}
	}

	task := &Task{
		ID:        cacheKey,
		Lat:       lat,
		Lon:       lon,
		Context:   timeoutCtx,
		ResultCh:  make(chan TaskResult, 1),
		CreatedAt: time.Now(),
	}

	a.pendingTasks[cacheKey] = []*Task{task}
	a.pendingMutex.Unlock()

	reqLogger.Debug("Cache miss, creating new task",
		zap.String("cache_key", cacheKey),
		zap.Int("enabled_services", len(a.services)),
		zap.Duration("timeout", handlerTimeout))

	select {
	case a.taskQueue <- task:
	case <-timeoutCtx.Done():
		a.pendingMutex.Lock()
		delete(a.pendingTasks, cacheKey)
		a.pendingMutex.Unlock()

		reqLogger.Warn("Handler timeout while submitting task",
			zap.String("cache_key", cacheKey),
			zap.Duration("timeout", handlerTimeout))
		if a.metrics != nil {
			a.metrics.RecordHandlerTimeout(ctx, "queue_submit")
		}
		return nil, &HandlerTimeoutError{
			Message: "request timeout: server is overloaded",
			Timeout: handlerTimeout,
		}
	default:
		// Queue is full, lets trigger trottling
		a.pendingMutex.Lock()
		delete(a.pendingTasks, cacheKey)
		a.pendingMutex.Unlock()

		reqLogger.Error("Task queue is full", zap.String("cache_key", cacheKey))
		if a.metrics != nil {
			a.metrics.RecordQueueFull(ctx)
		}
		return nil, &QueueFullError{
			Message: "server is overloaded, please try again later",
		}
	}


	select {
	case result := <-task.ResultCh:
		if result.Error != nil {
			reqLogger.Error("Task failed", zap.Error(result.Error))
			return nil, result.Error
		}

		reqLogger.Info("Weather data fetched and cached",
			zap.String("cache_key", cacheKey),
			zap.Int("services_count", len(result.Data.Services)))

		return result.Data, nil

	case <-timeoutCtx.Done():
		reqLogger.Warn("Handler timeout while waiting for task result",
			zap.String("cache_key", cacheKey),
			zap.Duration("timeout", handlerTimeout))
		if a.metrics != nil {
			a.metrics.RecordHandlerTimeout(ctx, "task_execution")
		}
		return nil, &HandlerTimeoutError{
			Message: "request timeout: server is overloaded",
			Timeout: handlerTimeout,
		}
	}
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

	a.cacheMutex.RLock()
	services := make(map[string]service.WeatherService)
	for name, svc := range a.services {
		services[name] = svc
	}
	a.cacheMutex.RUnlock()

	var wg sync.WaitGroup
	results := make(map[string]map[string]interface{})
	resultMutex := sync.Mutex{}

	// TODO: add logic for retrying failed services
	// maybe with attemtps and resonable exponential backoff
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
	a.cacheMutex.RLock()
	defer a.cacheMutex.RUnlock()

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
	a.cacheMutex.Lock()
	defer a.cacheMutex.Unlock()

	a.cache[key] = &CacheEntry{
		Data:      data,
		Timestamp: time.Now(),
	}
}

func (a *Aggregator) ClearCache() {
	a.cacheMutex.Lock()
	defer a.cacheMutex.Unlock()
	a.cache = make(map[string]*CacheEntry)
}

func (a *Aggregator) GetCacheStats() map[string]interface{} {
	a.cacheMutex.RLock()
	defer a.cacheMutex.RUnlock()

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
