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

type OpenMeteoService struct {
	baseURL string
	client  *http.Client
	params  map[string]string
}

type OpenMeteoDay struct {
	Date           string  `json:"date"`
	MaxTemperature float64 `json:"max_temperature"`
	MinTemperature float64 `json:"min_temperature"`
	Precipitation  float64 `json:"precipitation"`
	WeatherCode    int     `json:"weather_code"`
}

func NewOpenMeteoServiceWithConfig(cfg config.WeatherServiceConfig) *OpenMeteoService {
	return &OpenMeteoService{
		baseURL: cfg.BaseURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		params: cfg.Params,
	}
}

func (s *OpenMeteoService) Name() string {
	return "open-meteo"
}

func (s *OpenMeteoService) Get5DayForecast(lat, lon float64) (map[string]interface{}, error) {
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

func (s *OpenMeteoService) fetchDayForecast(lat, lon float64, date string) (map[string]interface{}, error) {
	u, err := url.Parse(fmt.Sprintf("%s/forecast", s.baseURL))
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("latitude", fmt.Sprintf("%.6f", lat))
	q.Set("longitude", fmt.Sprintf("%.6f", lon))
	q.Set("start_date", date)
	q.Set("end_date", date)

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
