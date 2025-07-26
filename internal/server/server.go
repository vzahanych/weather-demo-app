package server

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vzahanych/weather-demo-app/internal/aggregator"
	"github.com/vzahanych/weather-demo-app/internal/config"
	"github.com/vzahanych/weather-demo-app/internal/server/handlers"
	"github.com/vzahanych/weather-demo-app/internal/server/middlewares"
	"github.com/vzahanych/weather-demo-app/pkg/telemetry"
	"go.uber.org/zap"
)

type Server struct {
	engine *gin.Engine
	server *http.Server
	agg    *aggregator.Aggregator
	logger *zap.Logger
	tele   *telemetry.Telemetry
}

var (
	instance *Server
	once     sync.Once
)

func NewServer(logger *zap.Logger, tele *telemetry.Telemetry) *Server {
	once.Do(func() {
		cfg := config.GetConfig()

		agg := aggregator.NewAggregator(&cfg.Weather, logger, tele)

		gin.SetMode(gin.ReleaseMode)
		engine := gin.New()

		engine.Use(middlewares.RequestIDMiddleware(logger))
		engine.Use(middlewares.LoggingMiddleware(logger, time.RFC3339, true))
		engine.Use(middlewares.RecoveryMiddleware(logger, true))
		engine.Use(middlewares.TelemetryMiddleware(logger, tele))

		instance = &Server{
			engine: engine,
			agg:    agg,
			logger: logger,
			tele:   tele,
		}

		instance.setupRoutes()
	})

	return instance
}

func (s *Server) setupRoutes() {
	// Business endpoints
	s.engine.GET("/weather", handlers.NewWeatherHandler(s.agg, s.logger).GetWeather)

	// Health endpoints (Kubernetes friendly)
	s.engine.GET("/health", handlers.NewHealthHandler(s.logger).Health)
	s.engine.GET("/health/live", handlers.NewHealthHandler(s.logger).Liveness)
	s.engine.GET("/health/ready", handlers.NewHealthHandler(s.logger).Readiness)

	// Monitoring endpoints
	s.engine.GET("/metrics", handlers.NewMetricsHandler(s.logger).ServeMetrics)
}

func (s *Server) Start() error {
	cfg := config.GetConfig()

	s.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler: s.engine,
	}

	s.logger.Info("Starting server", zap.String("addr", s.server.Addr))
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown() error {
	if s.server == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return s.server.Shutdown(ctx)
}
