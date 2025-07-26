package service

import "context"

type WeatherService interface {
	Get5DayForecast(ctx context.Context, lat, lon float64) (map[string]interface{}, error)
	Name() string
}
