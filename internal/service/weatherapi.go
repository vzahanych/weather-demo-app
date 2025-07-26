package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"

	"github.com/vzahanych/weather-demo-app/internal/config"
	"github.com/vzahanych/weather-demo-app/pkg/telemetry"
	"go.uber.org/zap"
)

type WeatherAPIService struct {
	baseURL string
	apiKey  string
	client  *http.Client
	params  map[string]string
	logger  *zap.Logger
	tele    *telemetry.Telemetry
}

type WeatherAPIDay struct {
	Date           string  `json:"date"`
	MaxTemperature float64 `json:"max_temperature"`
	MinTemperature float64 `json:"min_temperature"`
	Precipitation  float64 `json:"precipitation"`
	WeatherCode    int     `json:"weather_code"`
}

func NewWeatherAPIServiceWithConfig(cfg config.WeatherServiceConfig, logger *zap.Logger, tele *telemetry.Telemetry) *WeatherAPIService {
	return &WeatherAPIService{
		baseURL: cfg.BaseURL,
		apiKey:  cfg.APIKey,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		params: cfg.Params,
		logger: logger,
		tele:   tele,
	}
}

func (s *WeatherAPIService) Name() string {
	return "weather-api"
}

func (s *WeatherAPIService) Get5DayForecast(ctx context.Context, lat, lon float64) (map[string]interface{}, error) {
	tracer := s.tele.GetTracer()
	ctx, span := tracer.Start(ctx, "weather-api.Get5DayForecast")
	defer span.End()

	span.SetAttributes(
		attribute.Float64("lat", lat),
		attribute.Float64("lon", lon),
		attribute.String("service", "weather-api"),
	)

	if s.apiKey == "" {
		s.logger.Warn("WeatherAPI service called without API key",
			zap.Float64("lat", lat),
			zap.Float64("lon", lon))
		span.SetAttributes(
			attribute.Bool("success", false),
			attribute.String("error", "API key not configured"),
		)
		return map[string]interface{}{
			"error": "WeatherAPI requires API key to be configured",
		}, nil
	}

	s.logger.Debug("Fetching 5-day forecast from WeatherAPI",
		zap.Float64("lat", lat),
		zap.Float64("lon", lon))

	results := make(map[string]interface{})
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(dayOffset int) {
			defer wg.Done()

			date := time.Now().AddDate(0, 0, dayOffset).Format("2006-01-02")

			// Create child span for each day fetch
			tracer := s.tele.GetTracer()
			_, daySpan := tracer.Start(ctx, "weather-api.fetchDayForecast")
			daySpan.SetAttributes(
				attribute.String("date", date),
				attribute.Int("day", dayOffset+1),
			)

			dayData, err := s.fetchDayForecast(lat, lon, date)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				s.logger.Warn("Failed to fetch day forecast from WeatherAPI",
					zap.String("date", date),
					zap.Error(err))
				daySpan.SetAttributes(
					attribute.Bool("success", false),
					attribute.String("error", err.Error()),
				)
				results[fmt.Sprintf("day%d", dayOffset+1)] = map[string]interface{}{
					"error": err.Error(),
					"date":  date,
				}
			} else {
				daySpan.SetAttributes(attribute.Bool("success", true))
				results[fmt.Sprintf("day%d", dayOffset+1)] = dayData
			}
			daySpan.End()
		}(i)
	}

	wg.Wait()

	span.SetAttributes(attribute.Int("days_fetched", len(results)))

	s.logger.Info("WeatherAPI forecast completed",
		zap.Int("days_fetched", len(results)),
		zap.Float64("lat", lat),
		zap.Float64("lon", lon))

	return results, nil
}

func (s *WeatherAPIService) fetchDayForecast(lat, lon float64, date string) (map[string]interface{}, error) {
	u, err := url.Parse(fmt.Sprintf("%s/forecast.json", s.baseURL))
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("key", s.apiKey)
	q.Set("q", fmt.Sprintf("%.6f,%.6f", lat, lon))
	q.Set("date", date)

	for key, value := range s.params {
		q.Set(key, value)
	}

	u.RawQuery = q.Encode()

	resp, err := s.client.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}
