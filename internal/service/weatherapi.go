package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/vzahanych/weather-demo-app/internal/config"
)

type WeatherAPIService struct {
	baseURL string
	apiKey  string
	client  *http.Client
	params  map[string]string
}

type WeatherAPIDay struct {
	Date           string  `json:"date"`
	MaxTemperature float64 `json:"max_temperature"`
	MinTemperature float64 `json:"min_temperature"`
	Precipitation  float64 `json:"precipitation"`
	WeatherCode    int     `json:"weather_code"`
}

func NewWeatherAPIServiceWithConfig(cfg config.WeatherServiceConfig) *WeatherAPIService {
	return &WeatherAPIService{
		baseURL: cfg.BaseURL,
		apiKey:  cfg.APIKey,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		params: cfg.Params,
	}
}

func (s *WeatherAPIService) Name() string {
	return "weather-api"
}

func (s *WeatherAPIService) Get5DayForecast(lat, lon float64) (map[string]interface{}, error) {
	if s.apiKey == "" {
		return map[string]interface{}{
			"error": "WeatherAPI requires API key to be configured",
		}, nil
	}

	results := make(map[string]interface{})
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(dayOffset int) {
			defer wg.Done()

			date := time.Now().AddDate(0, 0, dayOffset).Format("2006-01-02")
			dayData, err := s.fetchDayForecast(lat, lon, date)

			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				results[fmt.Sprintf("day%d", dayOffset+1)] = map[string]interface{}{
					"error": err.Error(),
					"date":  date,
				}
			} else {
				results[fmt.Sprintf("day%d", dayOffset+1)] = dayData
			}
		}(i)
	}

	wg.Wait()
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
