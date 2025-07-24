package cmd

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/vzahanych/weather-demo-app/internal/config"
	"github.com/vzahanych/weather-demo-app/pkg/logger"
	"github.com/vzahanych/weather-demo-app/pkg/telemetry"
)

var (
	globalLogger    *logger.Logger
	globalTelemetry *telemetry.Telemetry
)

func rootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "weather",
		Short: "Weather aggregation service",
		Long:  `A production-ready service that aggregates weather data from multiple sources with caching, observability, and concurrent processing.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return initializeServices()
		},
	}

	cmd.AddCommand(serverCmd)

	return cmd
}

func Execute() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		if globalLogger != nil {
			globalLogger.Info("Received shutdown signal", "signal", sig.String())
		}
		cancel()
	}()

	return rootCmd().ExecuteContext(ctx)
}

func initializeServices() error {
	loggingConfig := config.LoggingConfig{
		Level:  "info",
		Format: "console",
	}

	var err error
	globalLogger, err = logger.New(loggingConfig)
	if err != nil {
		globalLogger = logger.NewDevelopment()
	}

	telemetryConfig := config.TelemetryConfig{
		Enabled:  false,
		Endpoint: "tempo:4317",
	}

	globalTelemetry, err = telemetry.New(telemetryConfig)
	if err != nil {
		globalLogger.Warn("Failed to initialize telemetry", "error", err)
	}

	return nil
}
