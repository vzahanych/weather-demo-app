package service


type OpenMeteoService struct {
}

type OpenMeteoDay struct {
	Date           string  `json:"date"`
	MaxTemperature float64 `json:"max_temperature"`
	MinTemperature float64 `json:"min_temperature"`
	Precipitation  float64 `json:"precipitation"`
	WeatherCode    int     `json:"weather_code"`
}

func NewOpenMeteoService() *OpenMeteoService {
	return &OpenMeteoService{}
}

func (s *OpenMeteoService) Get5DayForecast(lat, lon float64) (map[string]*OpenMeteoDay, error) {
	return nil, nil
}
