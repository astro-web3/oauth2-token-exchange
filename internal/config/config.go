package config

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server struct {
		Addr         string        `mapstructure:"addr"`
		Mode         string        `mapstructure:"mode"`
		ReadTimeout  time.Duration `mapstructure:"read_timeout"`
		WriteTimeout time.Duration `mapstructure:"write_timeout"`
	} `mapstructure:"server"`

	Redis struct {
		URL      string `mapstructure:"url"`
		PoolSize int    `mapstructure:"pool_size"`
	} `mapstructure:"redis"`

	Auth struct {
		AdminMachineUser struct {
			PAT string `mapstructure:"pat"`
		} `mapstructure:"admin_machine_user"`
		Zitadel struct {
			Issuer         string `mapstructure:"issuer"`
			ClientID       string `mapstructure:"client_id"`
			ClientSecret   string `mapstructure:"client_secret"`
			OrganizationID string `mapstructure:"organization_id"`
		} `mapstructure:"zitadel"`
		CacheTTL   time.Duration `mapstructure:"cache_ttl"`
		HeaderKeys struct {
			UserID                string `mapstructure:"user_id"`
			UserEmail             string `mapstructure:"user_email"`
			UserGroups            string `mapstructure:"user_groups"`
			UserPreferredUsername string `mapstructure:"user_preferred_username"`
			UserJWT               string `mapstructure:"user_jwt"`
		} `mapstructure:"header_keys"`
	} `mapstructure:"auth"`

	Observability struct {
		MetricsEnabled     bool   `mapstructure:"metrics_enabled"`
		TraceEnabled       bool   `mapstructure:"trace_enabled"`
		TracingEndpointURL string `mapstructure:"tracing_endpoint_url"`
		LogLevel           string `mapstructure:"log_level"`
		Format             string `mapstructure:"log_format"`
		LogSource          bool   `mapstructure:"log_source"`
	} `mapstructure:"observability"`

	CORS struct {
		AllowedOrigins []string `mapstructure:"allowed_origins"`
	} `mapstructure:"cors"`
}

func MustLoad() *Config {
	v := viper.New()

	logger := slog.Default()

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("./config")
	v.AddConfigPath(".")

	v.AutomaticEnv()
	v.SetEnvPrefix("OAUTH2_TOKEN_EXCHANGE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		logger.Error("Failed to read config", slog.Any("error", err))
		os.Exit(1)
	}

	if env := os.Getenv("APP_ENV"); env != "" {
		v.SetConfigName(fmt.Sprintf("config.%s", env))
		if err := v.MergeInConfig(); err != nil {
			logger.Info("No environment-specific config (optional)", slog.String("env", env))
		}
		logger.Info("Environment-specific config loaded", slog.String("env", env))
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		logger.Error("Failed to unmarshal config", slog.Any("error", err))
		os.Exit(1)
	}

	return &cfg
}
