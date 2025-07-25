package service

type WeatherService interface {
	Get5DayForecast(lat, lon float64) (map[string]interface{}, error)
	Name() string
}
