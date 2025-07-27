package handlers

import (
	"context"
	"fmt"
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
	ctx = context.WithValue(ctx, "request_id", requestID)

	reqLogger := h.logger.With(zap.String("request_id", requestID))

	var req WeatherRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		reqLogger.Warn("Failed to bind request parameters", zap.Error(err), zap.String("request_id", requestID))
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request format",
			Code:    "BIND_ERROR",
			Details: err.Error(),
		})
		return
	}

	// Validate request using go-playground/validator
	// Looks like we don't need it for demo, but let's keep it for future
	if validationErrors := utils.ValidateStruct(req); validationErrors != nil {
		reqLogger.Warn("Request validation failed",
			zap.String("request_id", requestID),
			zap.Any("validation_errors", validationErrors),
			zap.Float64("lat", req.Lat),
			zap.Float64("lon", req.Lon))

		c.JSON(http.StatusBadRequest, gin.H{
			"error":             "Invalid request parameters",
			"code":              "VALIDATION_ERROR",
			"details":           "Request parameters failed validation",
			"validation_errors": validationErrors,
		})
		return
	}

	reqLogger.Info("Processing weather request",
		zap.String("request_id", requestID),
		zap.Float64("lat", req.Lat),
		zap.Float64("lon", req.Lon))

	data, err := h.aggregator.GetWeatherData(ctx, req.Lat, req.Lon)
	if err != nil {
		// Check for throttling errors and return appropriate status codes
		if timeoutErr, ok := err.(*aggregator.HandlerTimeoutError); ok {
			reqLogger.Warn("Request timeout due to server overload",
				zap.Error(err),
				zap.String("request_id", requestID),
				zap.Duration("timeout", timeoutErr.Timeout))
			c.JSON(http.StatusServiceUnavailable, ErrorResponse{
				Error:   timeoutErr.Message,
				Code:    "REQUEST_TIMEOUT",
				Details: fmt.Sprintf("Request timed out after %v", timeoutErr.Timeout),
			})
			return
		}

		if queueErr, ok := err.(*aggregator.QueueFullError); ok {
			reqLogger.Warn("Request rejected due to queue full", zap.Error(err), zap.String("request_id", requestID))
			c.JSON(http.StatusServiceUnavailable, ErrorResponse{
				Error:   queueErr.Message,
				Code:    "QUEUE_FULL",
				Details: "Server is currently overloaded, please retry with exponential backoff",
			})
			return
		}

		// All other errors are internal server errors
		reqLogger.Error("Failed to get weather data", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to fetch weather data",
			Code:    "AGGREGATION_ERROR",
			Details: err.Error(),
		})
		return
	}

	response := h.transformToWeatherResponse(data, req.Lat, req.Lon)

	// Validate response structure to ensure API contract compliance
	if validationErrors := h.validateWeatherResponse(response); validationErrors != nil {
		reqLogger.Error("Response validation failed",
			zap.Any("validation_errors", validationErrors))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Internal server error: invalid response format",
			Code:    "RESPONSE_VALIDATION_ERROR",
			Details: "Response failed internal validation checks",
		})
		return
	}

	reqLogger.Info("Weather request completed successfully",
		zap.String("request_id", requestID),
		zap.Int("services_count", len(response)))

	c.JSON(http.StatusOK, response)
}

// It is a simple transformation from the aggregator data to the response
// But there is no requerements for detailed response structure
// So just keep what we get from the aggregator
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

func (h *WeatherHandler) validateWeatherResponse(response WeatherResponse) []utils.ValidationError {
	for serviceName, serviceData := range response {
		if validationErrors := utils.ValidateStruct(serviceData); validationErrors != nil {
			h.logger.Warn("Service data validation failed",
				zap.String("service", serviceName),
				zap.Any("validation_errors", validationErrors))
			return validationErrors
		}
	}
	return nil
}
