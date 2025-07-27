package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/vzahanych/weather-demo-app/internal/config"
	"github.com/vzahanych/weather-demo-app/pkg/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"
)

type OpenMeteoService struct {
	baseURL string
	client  *http.Client
	logger  *zap.Logger
	tele    *telemetry.Telemetry
}

func NewOpenMeteoServiceWithConfig(cfg config.WeatherServiceConfig, logger *zap.Logger, tele *telemetry.Telemetry) *OpenMeteoService {
	return &OpenMeteoService{
		baseURL: cfg.BaseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
		tele:   tele,
	}
}

func (s *OpenMeteoService) Name() string {
	return "open-meteo"
}

func (s *OpenMeteoService) Get5DayForecast(ctx context.Context, lat, lon float64) (map[string]interface{}, error) {
	tracer := s.tele.GetTracer()
	ctx, span := tracer.Start(ctx, "open-meteo.Get5DayForecast")
	defer span.End()

	span.SetAttributes(
		attribute.Float64("lat", lat),
		attribute.Float64("lon", lon),
		attribute.String("service", "open-meteo"),
	)

	s.logger.Debug("Fetching 5-day forecast from Open-Meteo",
		zap.Float64("lat", lat),
		zap.Float64("lon", lon))

	var wg sync.WaitGroup
	results := make(map[string]interface{})
	mu := sync.Mutex{}

	now := time.Now()

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(day int) {
			defer wg.Done()

			date := now.AddDate(0, 0, day)
			dateStr := date.Format("2006-01-02")

			// Create child span for each day fetch
			tracer := s.tele.GetTracer()
			_, daySpan := tracer.Start(ctx, "open-meteo.fetchDayForecast")
			daySpan.SetAttributes(
				attribute.String("date", dateStr),
				attribute.Int("day", day+1),
			)

			s.logger.Debug("Fetching day forecast",
				zap.String("date", dateStr),
				zap.Int("day", day+1),
			)

			data, err := s.fetchDayForecast(lat, lon, dateStr)
			if err == nil {
				mu.Lock()
				results[fmt.Sprintf("day%d", day+1)] = data
				mu.Unlock()
				daySpan.SetAttributes(attribute.Bool("success", true))
			} else {
				s.logger.Warn("Failed to fetch day forecast",
					zap.String("date", dateStr),
					zap.Error(err))
				daySpan.SetAttributes(
					attribute.Bool("success", false),
					attribute.String("error", err.Error()),
				)
			}
			daySpan.End()
		}(i)
	}

	wg.Wait()

	span.SetAttributes(attribute.Int("days_fetched", len(results)))

	s.logger.Info("Open-Meteo forecast completed",
		zap.Int("days_fetched", len(results)),
		zap.Float64("lat", lat),
		zap.Float64("lon", lon))

	return results, nil
}

func (s *OpenMeteoService) fetchDayForecast(lat, lon float64, date string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/forecast?latitude=%.6f&longitude=%.6f&start_date=%s&end_date=%s&daily=temperature_2m_max,temperature_2m_min,precipitation_sum,weathercode",
		s.baseURL, lat, lon, date, date)

	resp, err := s.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}
