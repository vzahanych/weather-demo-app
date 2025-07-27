package aggregator

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/vzahanych/weather-demo-app/internal/config"
	"github.com/vzahanych/weather-demo-app/pkg/telemetry"
	"go.uber.org/zap/zaptest"
)

func createTestAggregator(t *testing.T) *Aggregator {
	logger := zaptest.NewLogger(t)

	tele := &telemetry.Telemetry{} 

	cfg := &config.WeatherConfig{
		CacheTTL: 300,
		Workers:  1,
		Services: make(map[string]config.WeatherServiceConfig),
	}

	agg := NewAggregator(cfg, logger, tele)
	return agg
}

func TestAggregatorWorker_Creation(t *testing.T) {
	agg := createTestAggregator(t)
	worker := NewAggregatorWorker(agg, 1)

	if worker == nil {
		t.Fatal("Expected worker to be created")
	}

	if worker.workerID != 1 {
		t.Errorf("Expected worker ID 1, got %d", worker.workerID)
	}

	if worker.aggregator != agg {
		t.Error("Expected worker to reference the aggregator")
	}

	if worker.logger == nil {
		t.Error("Expected worker to have a logger")
	}
}

func TestAggregatorWorker_ProcessTask_CacheKeyGeneration(t *testing.T) {
	agg := createTestAggregator(t)
	worker := NewAggregatorWorker(agg, 1)

	task := &Task{
		ID:      "test-task",
		Lat:     40.7128,
		Lon:     -74.0060,
		Context: context.Background(),
	}

	initialStats := agg.GetCacheStats()
	if initialStats["cache_size"].(int) != 0 {
		t.Error("Expected empty cache initially")
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Worker panicked: %v", r)
		}
	}()

	worker.processTask(context.Background(), task)
}

func TestAggregatorWorker_StartStop_Lifecycle(t *testing.T) {
	agg := createTestAggregator(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := agg.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start aggregator: %v", err)
	}

	worker := NewAggregatorWorker(agg, 99) 

	if worker.workerID != 99 {
		t.Errorf("Expected worker ID 99, got %d", worker.workerID)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()

	err = agg.Stop(shutdownCtx)
	if err != nil {
		t.Errorf("Failed to stop aggregator: %v", err)
	}
}

func TestAggregatorWorker_ConcurrentCreation(t *testing.T) {
	agg := createTestAggregator(t)

	const numWorkers = 10
	workers := make([]*AggregatorWorker, numWorkers)
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			workers[id] = NewAggregatorWorker(agg, id)
		}(i)
	}

	wg.Wait()

	for i, worker := range workers {
		if worker == nil {
			t.Errorf("Worker %d was not created", i)
			continue
		}

		if worker.workerID != i {
			t.Errorf("Worker %d has incorrect ID: %d", i, worker.workerID)
		}

		if worker.aggregator != agg {
			t.Errorf("Worker %d has incorrect aggregator reference", i)
		}
	}
}

func TestAggregatorWorker_TaskStructure(t *testing.T) {
	task := &Task{
		ID:        "test-123",
		Lat:       51.5074,
		Lon:       -0.1278,
		Context:   context.Background(),
		ResultCh:  make(chan TaskResult, 1),
		CreatedAt: time.Now(),
	}

	if task.ID != "test-123" {
		t.Errorf("Expected task ID 'test-123', got '%s'", task.ID)
	}

	if task.Lat != 51.5074 {
		t.Errorf("Expected lat 51.5074, got %f", task.Lat)
	}

	if task.Lon != -0.1278 {
		t.Errorf("Expected lon -0.1278, got %f", task.Lon)
	}

	if task.Context == nil {
		t.Error("Expected task to have context")
	}

	if task.ResultCh == nil {
		t.Error("Expected task to have result channel")
	}

	result := TaskResult{
		Data:  &WeatherData{Services: map[string]interface{}{"test": "data"}},
		Error: nil,
	}

	if result.Data == nil {
		t.Error("Expected result to have data")
	}

	if result.Error != nil {
		t.Error("Expected no error in result")
	}
}

func TestAggregatorWorker_LoggerConfiguration(t *testing.T) {
	logger := zaptest.NewLogger(t)
	tele := &telemetry.Telemetry{}

	cfg := &config.WeatherConfig{
		CacheTTL: 300,
		Workers:  1,
		Services: make(map[string]config.WeatherServiceConfig),
	}

	agg := NewAggregator(cfg, logger, tele)
	worker := NewAggregatorWorker(agg, 42)

	if worker.logger == nil {
		t.Fatal("Expected worker to have a logger")
	}

	if worker.logger == agg.logger {
		t.Error("Expected worker logger to be different from aggregator logger (should include worker_id)")
	}
}
