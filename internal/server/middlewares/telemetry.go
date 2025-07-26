package middlewares

import (
	"github.com/gin-gonic/gin"
	"github.com/vzahanych/weather-demo-app/internal/server/utils"
	"github.com/vzahanych/weather-demo-app/pkg/telemetry"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func TelemetryMiddleware(logger *zap.Logger, tele *telemetry.Telemetry) gin.HandlerFunc {
	propagator := otel.GetTextMapPropagator()

	return gin.HandlerFunc(func(c *gin.Context) {
		tracer := tele.GetTracer()

		ctx := propagator.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		requestID := ""
		if reqID, exists := c.Get(RequestIDKey); exists {
			if id, ok := reqID.(string); ok {
				requestID = id
			}
		}

		spanName := c.Request.Method + " " + c.FullPath()
		ctx, span := tracer.Start(ctx, spanName,
			trace.WithAttributes(
				attribute.String("request.id", requestID),
				attribute.String("http.method", c.Request.Method),
				attribute.String("http.url", c.Request.URL.String()),
				attribute.String("http.route", c.FullPath()),
				attribute.String("user_agent", c.Request.UserAgent()),
				attribute.String("remote_addr", c.ClientIP()),
			),
		)

		c.Set(utils.SpanContextKey, ctx)
		c.Request = c.Request.WithContext(ctx)

		if tele.IsEnabled() {
			logger.Debug("Started tracing span",
				zap.String("span_name", spanName),
				zap.String("trace_id", span.SpanContext().TraceID().String()),
				zap.String("span_id", span.SpanContext().SpanID().String()))
		}

		defer func() {
			span.SetAttributes(
				attribute.Int("http.status_code", c.Writer.Status()),
				attribute.Int("http.response_size", c.Writer.Size()),
			)

			if c.Writer.Status() >= 400 {
				span.SetAttributes(attribute.Bool("error", true))
				if len(c.Errors) > 0 {
					span.SetAttributes(attribute.String("error.message", c.Errors.String()))
				}
			}

			span.End()

			if tele.IsEnabled() {
				logger.Debug("Ended tracing span",
					zap.String("span_name", spanName),
					zap.Int("status_code", c.Writer.Status()))
			}
		}()

		c.Next()
	})
}
