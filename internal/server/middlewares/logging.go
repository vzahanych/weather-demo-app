package middlewares

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func LoggingMiddleware(logger *zap.Logger, timeFormat string, utc bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		requestID := ""
		if reqID, exists := c.Get(RequestIDKey); exists {
			if id, ok := reqID.(string); ok {
				requestID = id
			}
		}

		c.Next()

		param := gin.LogFormatterParams{
			Request:      c.Request,
			TimeStamp:    time.Now(),
			Latency:      time.Since(start),
			ClientIP:     c.ClientIP(),
			Method:       c.Request.Method,
			StatusCode:   c.Writer.Status(),
			ErrorMessage: c.Errors.ByType(gin.ErrorTypePrivate).String(),
			BodySize:     c.Writer.Size(),
		}

		if utc {
			param.TimeStamp = param.TimeStamp.UTC()
		}

		if raw != "" {
			path = path + "?" + raw
		}

		fields := []zap.Field{
			zap.String("method", param.Method),
			zap.String("path", path),
			zap.Int("status", param.StatusCode),
			zap.Duration("latency", param.Latency),
			zap.String("client_ip", param.ClientIP),
			zap.Int("body_size", param.BodySize),
		}

		if requestID != "" {
			fields = append(fields, zap.String("request_id", requestID))
		}

		if userAgent := c.Request.UserAgent(); userAgent != "" {
			fields = append(fields, zap.String("user_agent", userAgent))
		}

		if param.ErrorMessage != "" {
			fields = append(fields, zap.String("error", param.ErrorMessage))
		}

		switch {
		case param.StatusCode >= 400 && param.StatusCode < 500:
			logger.Warn("HTTP request", fields...)
		case param.StatusCode >= 500:
			logger.Error("HTTP request", fields...)
		default:
			logger.Info("HTTP request", fields...)
		}
	}
}

func RecoveryMiddleware(logger *zap.Logger, stack bool) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		requestID := ""
		if reqID, exists := c.Get(RequestIDKey); exists {
			if id, ok := reqID.(string); ok {
				requestID = id
			}
		}

		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("client_ip", c.ClientIP()),
			zap.Any("recovered", recovered),
		}

		if requestID != "" {
			fields = append(fields, zap.String("request_id", requestID))
		}

		if stack {
			fields = append(fields, zap.Stack("stack"))
		}

		logger.Error("HTTP panic recovered", fields...)
		c.AbortWithStatus(500)
	})
}
