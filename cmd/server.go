package cmd

import (
	"context"
	"time"

	"github.com/spf13/cobra"
	"github.com/vzahanych/weather-demo-app/internal/config"
	"github.com/vzahanych/weather-demo-app/internal/server"
	"github.com/vzahanych/weather-demo-app/pkg/logger"
	"github.com/vzahanych/weather-demo-app/pkg/telemetry"
	"go.uber.org/zap"
)

var (
	configPath string
	serverCmd  = &cobra.Command{
		Use:   "server",
		Short: "Start the weather server",
		RunE:  runServer,
	}
)

func init() {
	serverCmd.Flags().StringVarP(&configPath, "config", "c", "", "config file path")
}

func runServer(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	
	// config is set to atomic to be able to change it without downtime
	config.SetConfig(cfg)

	logger, err := logger.NewZapLogger(cfg.Logging)
	if err != nil {
		return err
	}
	defer logger.Sync()

	tele, err := telemetry.New(cmd.Context(), cfg.Telemetry)
	if err != nil {
		logger.Error("Failed to initialize telemetry", zap.Error(err))
		return err
	}
	defer tele.Shutdown(cmd.Context())

	logger.Info("Starting server", zap.Int("port", cfg.Server.Port))

	srv := server.NewServer(logger, tele)

	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.Start()
	}()

	select {
	case err := <-errChan:
		return err
	case <-cmd.Context().Done():
		logger.Info("Shutting down")

		_, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(); err != nil {
			logger.Error("Shutdown error", zap.Error(err))
		}
		return nil
	}
}
