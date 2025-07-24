package service

type WeatherAPIService struct {
}

type WeatherAPIDay struct {
	Date           string  `json:"date"`
	MaxTemperature float64 `json:"max_temperature"`
	MinTemperature float64 `json:"min_temperature"`
	Precipitation  float64 `json:"precipitation"`
	WeatherCode    int     `json:"weather_code"`
}

func NewWeatherAPIService() *WeatherAPIService {
	return &WeatherAPIService{}
}

func (s *WeatherAPIService) Get5DayForecast(lat, lon float64) (map[string]*WeatherAPIDay, error) {
	return nil, nil
}
