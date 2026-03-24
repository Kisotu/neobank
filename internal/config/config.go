package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server      ServerConfig
	Database    DatabaseConfig
	JWT         JWTConfig
	Logging     LoggingConfig
	RateLimiter RateLimiterConfig
}

type ServerConfig struct {
	Host string
	Port string
}

type DatabaseConfig struct {
	URL               string
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
}

type JWTConfig struct {
	Secret     string
	Expiry     time.Duration
	RefreshTTL time.Duration
}

type LoggingConfig struct {
	Level  string
	Format string
}

type RateLimiterConfig struct {
	GeneralPerMinute  int
	LoginPerMinute    int
	TransferPerMinute int
}

func Load() (*Config, error) {
	v := viper.New()
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", "8080")

	v.SetDefault("database.max_conns", 20)
	v.SetDefault("database.min_conns", 2)
	v.SetDefault("database.max_conn_lifetime", "30m")
	v.SetDefault("database.max_conn_idle_time", "10m")
	v.SetDefault("database.health_check_period", "1m")

	v.SetDefault("jwt.secret", "change-me")
	v.SetDefault("jwt.expiry", "15m")
	v.SetDefault("jwt.refresh_ttl", "168h")

	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")

	v.SetDefault("rate_limiter.general_per_minute", 100)
	v.SetDefault("rate_limiter.login_per_minute", 5)
	v.SetDefault("rate_limiter.transfer_per_minute", 10)

	v.BindEnv("database.url", "DATABASE_URL")

	maxConnLifetime, err := time.ParseDuration(v.GetString("database.max_conn_lifetime"))
	if err != nil {
		return nil, fmt.Errorf("invalid database.max_conn_lifetime: %w", err)
	}

	maxConnIdleTime, err := time.ParseDuration(v.GetString("database.max_conn_idle_time"))
	if err != nil {
		return nil, fmt.Errorf("invalid database.max_conn_idle_time: %w", err)
	}

	healthCheckPeriod, err := time.ParseDuration(v.GetString("database.health_check_period"))
	if err != nil {
		return nil, fmt.Errorf("invalid database.health_check_period: %w", err)
	}

	jwtExpiry, err := time.ParseDuration(v.GetString("jwt.expiry"))
	if err != nil {
		return nil, fmt.Errorf("invalid jwt.expiry: %w", err)
	}

	refreshTTL, err := time.ParseDuration(v.GetString("jwt.refresh_ttl"))
	if err != nil {
		return nil, fmt.Errorf("invalid jwt.refresh_ttl: %w", err)
	}

	cfg := &Config{
		Server: ServerConfig{
			Host: v.GetString("server.host"),
			Port: v.GetString("server.port"),
		},
		Database: DatabaseConfig{
			URL:               strings.TrimSpace(v.GetString("database.url")),
			MaxConns:          int32(v.GetInt("database.max_conns")),
			MinConns:          int32(v.GetInt("database.min_conns")),
			MaxConnLifetime:   maxConnLifetime,
			MaxConnIdleTime:   maxConnIdleTime,
			HealthCheckPeriod: healthCheckPeriod,
		},
		JWT: JWTConfig{
			Secret:     v.GetString("jwt.secret"),
			Expiry:     jwtExpiry,
			RefreshTTL: refreshTTL,
		},
		Logging: LoggingConfig{
			Level:  strings.ToLower(v.GetString("logging.level")),
			Format: strings.ToLower(v.GetString("logging.format")),
		},
		RateLimiter: RateLimiterConfig{
			GeneralPerMinute:  v.GetInt("rate_limiter.general_per_minute"),
			LoginPerMinute:    v.GetInt("rate_limiter.login_per_minute"),
			TransferPerMinute: v.GetInt("rate_limiter.transfer_per_minute"),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if strings.TrimSpace(c.Server.Port) == "" {
		return fmt.Errorf("server port is required")
	}

	if strings.TrimSpace(c.Database.URL) == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}

	if c.Database.MaxConns <= 0 {
		return fmt.Errorf("database max connections must be positive")
	}

	if c.Database.MinConns < 0 || c.Database.MinConns > c.Database.MaxConns {
		return fmt.Errorf("database min connections must be between 0 and max connections")
	}

	if strings.TrimSpace(c.JWT.Secret) == "" {
		return fmt.Errorf("jwt secret is required")
	}

	return nil
}
