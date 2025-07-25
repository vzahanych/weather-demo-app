package telemetry

import (
	"context"
	"fmt"
	"time"

	"github.com/vzahanych/weather-demo-app/internal/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Telemetry struct {
	enabled  bool
	tracer   trace.Tracer
	provider *sdktrace.TracerProvider
}

func New(ctx context.Context, cfg config.TelemetryConfig) (*Telemetry, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	
	t := &Telemetry{
		enabled: cfg.Enabled,
	}	

	if err := t.initTracer(ctx,cfg.Endpoint); err != nil {
		return nil, fmt.Errorf("failed to initialize tracer: %w", err)
	}

	return t, nil
}

func (t *Telemetry) initTracer(ctx context.Context, endpoint string) error {
	conn, err := grpc.DialContext(ctx, endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return fmt.Errorf("failed to create gRPC connection: %w", err)
	}

	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName("weather-app"),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	t.provider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(t.provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	t.tracer = otel.Tracer("weather-app")

	return nil
}

func (t *Telemetry) StartSpan(ctx context.Context, name string) (context.Context, func()) {
	if !t.enabled || t.tracer == nil {
		return ctx, func() {}
	}

	ctx, span := t.tracer.Start(ctx, name)
	return ctx, func() {
		span.End()
	}
}

func (t *Telemetry) StartSpanWithAttributes(ctx context.Context, name string, attrs map[string]interface{}) (context.Context, func()) {
	if !t.enabled || t.tracer == nil {
		return ctx, func() {}
	}

	ctx, span := t.tracer.Start(ctx, name)

	return ctx, func() {
		span.End()
	}
}

func (t *Telemetry) RecordMetric(name string, value float64, labels map[string]string) {
	if !t.enabled {
		return
	}
}

func (t *Telemetry) RecordError(err error, ctx context.Context, contextData map[string]interface{}) {
	if !t.enabled {
		return
	}

	if span := trace.SpanFromContext(ctx); span != nil {
		span.RecordError(err)
		for k, v := range contextData {
			span.SetAttributes(attribute.String(k, fmt.Sprintf("%v", v)))
		}
	}
}

func (t *Telemetry) Shutdown(ctx context.Context) error {
	if !t.enabled {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if t.provider != nil {
		if err := t.provider.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown tracer provider: %w", err)
		}
	}

	return nil
}
