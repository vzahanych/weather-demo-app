package aggregator

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

type AggregatorWorker struct {
	aggregator *Aggregator
	workerID   int
	logger     *zap.Logger
}

func NewAggregatorWorker(aggregator *Aggregator, workerID int) *AggregatorWorker {
	return &AggregatorWorker{
		aggregator: aggregator,
		workerID:   workerID,
		logger:     aggregator.logger.With(zap.Int("worker_id", workerID)),
	}
}

func (w *AggregatorWorker) Start(ctx context.Context) {
	defer w.aggregator.workerWg.Done()

	w.logger.Info("Worker started", zap.Int("worker_id", w.workerID))

	for {
		select {
		case task, ok := <-w.aggregator.taskQueue:
			if !ok {
				w.logger.Info("Task queue closed, worker stopping", zap.Int("worker_id", w.workerID))
				return
			}

			w.logger.Debug("Processing task", zap.String("task_id", task.ID))
			w.processTask(ctx, task)

		case <-w.aggregator.shutdownCh:
			w.logger.Info("Shutdown signal received, worker stopping", zap.Int("worker_id", w.workerID))
			return
		case <-ctx.Done():
			w.logger.Info("Context cancelled, worker stopping", zap.Int("worker_id", w.workerID))
			return
		}
	}
}

// Task context looks better here as we could transfer request_id from the server to the worker
// TODO: implement request_id usage here for logging and tracing
func (w *AggregatorWorker) processTask(ctx context.Context, task *Task) {
	tracer := w.aggregator.tele.GetTracer()
	ctx, span := tracer.Start(task.Context, "aggregator.processTask")
	defer span.End()

	span.SetAttributes(
		attribute.String("task_id", task.ID),
		attribute.Float64("lat", task.Lat),
		attribute.Float64("lon", task.Lon),
		attribute.Int("worker_id", w.workerID),
	)

	data, err := w.aggregator.fetchWeatherData(ctx, task.Lat, task.Lon)

	result := TaskResult{
		Data:  data,
		Error: err,
	}

	if err == nil {
		cacheKey := fmt.Sprintf("%.6f,%.6f", task.Lat, task.Lon)
		w.aggregator.setCache(cacheKey, data)
		w.logger.Debug("Task completed successfully", zap.String("task_id", task.ID), zap.Int("worker_id", w.workerID))
	} else {
		w.logger.Error("Task failed", zap.String("task_id", task.ID), zap.Int("worker_id", w.workerID), zap.Error(err))
	}

	w.aggregator.notifyPendingTasks(task.ID, result)
}
