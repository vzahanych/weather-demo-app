package cmd

import (
	"context"
	"time"

	"github.com/spf13/cobra"
	"github.com/vzahanych/weather-demo-app/internal/config"
	"github.com/vzahanych/weather-demo-app/internal/server"
	"go.uber.org/zap"
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
	cfg := config.GetConfig()

	log.Info("Starting weather aggregation server",
		zap.String("config_path", configPath),
		zap.Bool("telemetry_enabled", cfg.Telemetry.Enabled),
		zap.Int("server_port", cfg.Server.Port))

	srv := server.NewServer()

	errChan := make(chan error, 1)
	go func() {
		if err := srv.Start(); err != nil {
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		log.Error("Server error", zap.Error(err))
		return err
	case <-cmd.Context().Done():
		log.Info("Shutting down server")

		_, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		if err := srv.Shutdown(); err != nil {
			log.Error("Error during server shutdown", zap.Error(err))
			return err
		}

		log.Info("Server shutdown complete")
		return nil
	}
}
