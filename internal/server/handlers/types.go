package handlers

// WeatherRequest represents the weather API request with comprehensive validation
type WeatherRequest struct {
	Lat float64 `form:"lat" json:"lat" validate:"required,latitude" binding:"required"`
	Lon float64 `form:"lon" json:"lon" validate:"required,longitude" binding:"required"`
}

// WeatherResponse represents the aggregated weather response according to requirements
type WeatherResponse map[string]ServiceData

// ServiceData represents weather data from a specific service with validation
type ServiceData struct {
	Day1 interface{} `json:"day1,omitempty" validate:"omitempty"`
	Day2 interface{} `json:"day2,omitempty" validate:"omitempty"`
	Day3 interface{} `json:"day3,omitempty" validate:"omitempty"`
	Day4 interface{} `json:"day4,omitempty" validate:"omitempty"`
	Day5 interface{} `json:"day5,omitempty" validate:"omitempty"`
}

// ErrorResponse represents an error response with validation
type ErrorResponse struct {
	Error   string `json:"error" validate:"required,min=1,max=500"`
	Code    string `json:"code,omitempty" validate:"omitempty,min=1,max=50"`
	Details string `json:"details,omitempty" validate:"omitempty,max=1000"`
}

// HealthResponse represents health check response with validation
type HealthResponse struct {
	Status    string `json:"status" validate:"required,oneof=ok degraded unavailable"`
	Uptime    string `json:"uptime" validate:"required"`
	Timestamp string `json:"timestamp,omitempty" validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
}
