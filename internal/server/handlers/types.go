package handlers

// WeatherRequest represents the incoming request for weather data
type WeatherRequest struct {
	Lat float64 `form:"lat" binding:"required,min=-90,max=90" json:"lat"`
	Lon float64 `form:"lon" binding:"required,min=-180,max=180" json:"lon"`
}

// WeatherResponse represents the aggregated weather response according to requirements
type WeatherResponse map[string]ServiceData

// ServiceData represents weather data from a specific service
type ServiceData struct {
	Day1 interface{} `json:"day1,omitempty"`
	Day2 interface{} `json:"day2,omitempty"`
	Day3 interface{} `json:"day3,omitempty"`
	Day4 interface{} `json:"day4,omitempty"`
	Day5 interface{} `json:"day5,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// HealthResponse represents health check response
type HealthResponse struct {
	Status    string `json:"status"`
	Uptime    string `json:"uptime"`
	Timestamp string `json:"timestamp,omitempty"`
}
