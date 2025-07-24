package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/vzahanych/weather-demo-app/internal/config"
	"github.com/vzahanych/weather-demo-app/internal/server"
	"github.com/vzahanych/weather-demo-app/pkg/logger"
	"github.com/vzahanych/weather-demo-app/pkg/telemetry"
)

var (
	configPath string
	serverCmd  = &cobra.Command{
		Use:   "server",
		Short: "Start the weather aggregation server",
		Long:  `Start the HTTP server that aggregates weather data from multiple sources with caching and observability.`,
		RunE:  runServer,
	}
)

func init() {
	serverCmd.Flags().StringVarP(&configPath, "config", "c", "", "path to configuration file (default: ./config.yaml)")
}

func runServer(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if globalLogger != nil {
		globalLogger.Sync()
	}

	logger, err := logger.New(cfg.Logging)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer logger.Sync()

	if globalTelemetry != nil {
		globalTelemetry.Shutdown(context.Background())
	}

	tel, err := telemetry.New(cfg.Telemetry)
	if err != nil {
		logger.Warn("Failed to initialize telemetry", "error", err)
	} else {
		defer func() {
			if err := tel.Shutdown(context.Background()); err != nil {
				logger.Error("Failed to shutdown telemetry", "error", err)
			}
		}()
	}

	logger.Info("Starting weather aggregation server",
		"config_path", configPath,
		"telemetry_enabled", cfg.Telemetry.Enabled,
		"server_port", cfg.Server.Port)

	srv := server.NewServer()

	errChan := make(chan error, 1)
	go func() {
		if err := srv.Start(); err != nil {
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		logger.Error("Server error", "error", err)
		return err
	case <-cmd.Context().Done():
		logger.Info("Shutting down server")

		_, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		if err := srv.Shutdown(); err != nil {
			logger.Error("Error during server shutdown", "error", err)
			return err
		}

		logger.Info("Server shutdown complete")
		return nil
	}
}
