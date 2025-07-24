package logger

import (
	"github.com/vzahanych/weather-demo-app/internal/config"
	"go.uber.org/zap"
)

type Logger struct {
	*zap.SugaredLogger
}

func New(cfg config.LoggingConfig) (*Logger, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}
	return &Logger{logger.Sugar()}, nil
}

func NewDevelopment() *Logger {
	logger, _ := zap.NewDevelopment()
	return &Logger{logger.Sugar()}
}

func NewProduction() *Logger {
	logger, _ := zap.NewProduction()
	return &Logger{logger.Sugar()}
}

func (l *Logger) Sync() error {
	return l.SugaredLogger.Sync()
}
