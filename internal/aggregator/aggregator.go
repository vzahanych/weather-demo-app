package aggregator

// AggregatedWeatherData represents the combined weather data from all sources
type AggregatedWeatherData struct {
	OpenMeteo  map[string]interface{} `json:"openMeteo,omitempty"`
	WeatherAPI map[string]interface{} `json:"weatherAPI,omitempty"`
	Timestamp  string                 `json:"timestamp"`
}

type Aggregator struct {
	// TODO: Add fields for cache, services, etc.
}

func NewAggregator() *Aggregator {
	return &Aggregator{}
}

func (a *Aggregator) GetWeatherData(lat, lon float64) (*AggregatedWeatherData, error) {
	// TODO: Implement weather data retrieval with caching
	return nil, nil
}
