package config

import (
	"sync/atomic"
)

var configValue atomic.Value

func GetConfig() *Config {
	return configValue.Load().(*Config)
}

func SetConfig(cfg *Config) {
	configValue.Store(cfg)
}

type Config struct {
	Version     string          `mapstructure:"version"`
	Environment string          `mapstructure:"environment"`
	Server      ServerConfig    `mapstructure:"server"`
	Weather     WeatherConfig   `mapstructure:"weather"`
	Logging     LoggingConfig   `mapstructure:"logging"`
	Telemetry   TelemetryConfig `mapstructure:"telemetry"`
}

type ServerConfig struct {
	Port         int    `mapstructure:"port"`
	Host         string `mapstructure:"host"`
	ReadTimeout  int    `mapstructure:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout"`
	IdleTimeout  int    `mapstructure:"idle_timeout"`
}

type WeatherConfig struct {
	Services map[string]WeatherServiceConfig `mapstructure:"services"`
	Timeout  int                             `mapstructure:"timeout"`
	Retries  int                             `mapstructure:"retries"`
	CacheTTL int                             `mapstructure:"cache_ttl"`
}

type WeatherServiceConfig struct {
	Type    string            `mapstructure:"type"`
	Enabled bool              `mapstructure:"enabled"`
	BaseURL string            `mapstructure:"base_url"`
	APIKey  string            `mapstructure:"api_key"`
	Params  map[string]string `mapstructure:"params"`
}

type LoggingConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	OutputPath string `mapstructure:"output_path"`
}

type TelemetryConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Endpoint string `mapstructure:"endpoint"`
}

func NewDefaultConfig() *Config {
	return &Config{
		Version:     "1.0.0",
		Environment: "development",
		Server: ServerConfig{
			Port:         8080,
			Host:         "0.0.0.0",
			ReadTimeout:  30,
			WriteTimeout: 30,
			IdleTimeout:  60,
		},
		Weather: WeatherConfig{
			Services: map[string]WeatherServiceConfig{
				"open-meteo": {
					Type:    "open-meteo",
					Enabled: true,
					BaseURL: "https://api.open-meteo.com/v1",
					Params: map[string]string{
						"daily": "temperature_2m_max,temperature_2m_min,precipitation_sum,weathercode",
					},
				},
				"weather-api": {
					Type:    "weather-api",
					Enabled: false,
					BaseURL: "https://api.weatherapi.com/v1",
					APIKey:  "",
					Params: map[string]string{
						"format": "json",
					},
				},
			},
			Timeout:  10,
			Retries:  3,
			CacheTTL: 300,
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "json",
			OutputPath: "",
		},
		Telemetry: TelemetryConfig{
			Enabled:  false,
			Endpoint: "tempo:4317",
		},
	}
}
