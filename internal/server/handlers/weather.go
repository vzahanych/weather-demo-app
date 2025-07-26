package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/vzahanych/weather-demo-app/internal/aggregator"
	"github.com/vzahanych/weather-demo-app/internal/server/utils"
	"go.uber.org/zap"
)

type WeatherHandler struct {
	aggregator *aggregator.Aggregator
	logger     *zap.Logger
}

func NewWeatherHandler(agg *aggregator.Aggregator, logger *zap.Logger) *WeatherHandler {
	return &WeatherHandler{
		aggregator: agg,
		logger:     logger,
	}
}

func (h *WeatherHandler) GetWeather(c *gin.Context) {
	ctx := utils.GetContextFromGinContext(c)
	requestID := utils.GetRequestIDFromGinContext(c)

	// Create logger with request ID for this request
	reqLogger := h.logger.With(zap.String("request_id", requestID))

	var req WeatherRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		reqLogger.Warn("Invalid request parameters", zap.Error(err))
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request parameters",
			Code:    "INVALID_PARAMS",
			Details: err.Error(),
		})
		return
	}

	reqLogger.Info("Processing weather request",
		zap.Float64("lat", req.Lat),
		zap.Float64("lon", req.Lon))

	data, err := h.aggregator.GetWeatherData(ctx, req.Lat, req.Lon)
	if err != nil {
		reqLogger.Error("Failed to get weather data", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to fetch weather data",
			Code:    "AGGREGATION_ERROR",
			Details: err.Error(),
		})
		return
	}

	response := h.transformToWeatherResponse(data, req.Lat, req.Lon)
	reqLogger.Info("Weather request completed successfully",
		zap.Int("services_count", len(response)))

	c.JSON(http.StatusOK, response)
}

func (h *WeatherHandler) transformToWeatherResponse(data *aggregator.WeatherData, lat, lon float64) WeatherResponse {
	response := make(WeatherResponse)

	for serviceName, serviceData := range data.Services {
		if serviceMap, ok := serviceData.(map[string]interface{}); ok {
			serviceStruct := ServiceData{}

			if day1, exists := serviceMap["day1"]; exists {
				serviceStruct.Day1 = day1
			}
			if day2, exists := serviceMap["day2"]; exists {
				serviceStruct.Day2 = day2
			}
			if day3, exists := serviceMap["day3"]; exists {
				serviceStruct.Day3 = day3
			}
			if day4, exists := serviceMap["day4"]; exists {
				serviceStruct.Day4 = day4
			}
			if day5, exists := serviceMap["day5"]; exists {
				serviceStruct.Day5 = day5
			}

			response[serviceName] = serviceStruct
		}
	}

	return response
}
