package config

type Config struct {
	Server    ServerConfig    `mapstructure:"server"`
	Weather   WeatherConfig   `mapstructure:"weather"`
	Logging   LoggingConfig   `mapstructure:"logging"`
	Telemetry TelemetryConfig `mapstructure:"telemetry"`
}

type ServerConfig struct {
	Port         int    `mapstructure:"port"`
	Host         string `mapstructure:"host"`
	ReadTimeout  int    `mapstructure:"read_timeout"`
	WriteTimeout int    `mapstructure:"write_timeout"`
	IdleTimeout  int    `mapstructure:"idle_timeout"`
}

type WeatherConfig struct {
	OpenMeteo  OpenMeteoConfig  `mapstructure:"open_meteo"`
	WeatherAPI WeatherAPIConfig `mapstructure:"weather_api"`
	Timeout    int              `mapstructure:"timeout"`
	Retries    int              `mapstructure:"retries"`
	CacheTTL   int              `mapstructure:"cache_ttl"`
}

type OpenMeteoConfig struct {
	BaseURL string `mapstructure:"base_url"`
	Enabled bool   `mapstructure:"enabled"`
}

type WeatherAPIConfig struct {
	BaseURL string `mapstructure:"base_url"`
	APIKey  string `mapstructure:"api_key"`
	Enabled bool   `mapstructure:"enabled"`
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

func Load(configPath string) (*Config, error) {
	return nil, nil
}
