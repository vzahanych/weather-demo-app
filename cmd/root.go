package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/vzahanych/weather-demo-app/internal/config"
	"github.com/vzahanych/weather-demo-app/pkg/logger"
	"github.com/vzahanych/weather-demo-app/pkg/telemetry"
	"go.uber.org/zap"
)

var (
	log        *logger.Logger
	tele       *telemetry.Telemetry
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
		if log != nil {
			log.Info("Received shutdown signal", zap.String("signal", sig.String()))
		}
		cancel()
	}()

	return rootCmd().ExecuteContext(ctx)
}

func initializeServices() error {
	// 1.Load config
	cfg, err := config.LoadConfig() // Load from env or file
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 2. Set config
	// Having config in atomic allows changing it during runtime
	config.SetConfig(cfg)

	// 3. Initialize logger
	log, err = logger.New(cfg.Logging)
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	tele, err = telemetry.New(context.Background(), cfg.Telemetry)
	if err != nil {
		log.Warn("Failed to initialize telemetry", zap.Error(err))
	}

	return nil
}
