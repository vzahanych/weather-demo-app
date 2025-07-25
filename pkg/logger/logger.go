package logger

import (
	"github.com/vzahanych/weather-demo-app/internal/config"
	"go.uber.org/zap"
)

type Logger struct {
	*zap.Logger
}

func New(cfg config.LoggingConfig) (*Logger, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}
	return &Logger{logger}, nil
}

func NewDevelopment() *Logger {
	logger, _ := zap.NewDevelopment()
	return &Logger{logger}
}

func NewProduction() *Logger {
	logger, _ := zap.NewProduction()
	return &Logger{logger}
}

func (l *Logger) Sync() error {
	return l.Logger.Sync()
}
