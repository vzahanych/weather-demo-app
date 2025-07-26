package utils

import (
	"context"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/trace"
)

const (
	SpanContextKey = "span_context"
	RequestIDKey   = "request_id"
)

// GetSpanFromGinContext extracts the span context from Gin context
func GetSpanFromGinContext(c *gin.Context) trace.Span {
	if spanCtx, exists := c.Get(SpanContextKey); exists {
		if ctx, ok := spanCtx.(context.Context); ok {
			return trace.SpanFromContext(ctx)
		}
	}
	return trace.SpanFromContext(c.Request.Context())
}

// GetContextFromGinContext extracts the context with span from Gin context
func GetContextFromGinContext(c *gin.Context) context.Context {
	if spanCtx, exists := c.Get(SpanContextKey); exists {
		if ctx, ok := spanCtx.(context.Context); ok {
			return ctx
		}
	}
	return c.Request.Context()
}

// GetRequestIDFromGinContext extracts request ID from Gin context
func GetRequestIDFromGinContext(c *gin.Context) string {
	if requestID, exists := c.Get(RequestIDKey); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return ""
}
