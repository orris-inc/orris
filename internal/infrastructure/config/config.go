package config

import (
	"fmt"
	"strings"
	"sync"

	"github.com/spf13/viper"

	sharedConfig "github.com/orris-inc/orris/internal/shared/config"
)

type Config struct {
	Server       sharedConfig.ServerConfig       `mapstructure:"server"`
	Database     sharedConfig.DatabaseConfig     `mapstructure:"database"`
	Logger       sharedConfig.LoggerConfig       `mapstructure:"logger"`
	Auth         sharedConfig.AuthConfig         `mapstructure:"auth"`
	OAuth        sharedConfig.OAuthConfig        `mapstructure:"oauth"`
	Email        sharedConfig.EmailConfig        `mapstructure:"email"`
	Redis        sharedConfig.RedisConfig        `mapstructure:"redis"`
	Subscription sharedConfig.SubscriptionConfig `mapstructure:"subscription"`
	Forward      sharedConfig.ForwardConfig      `mapstructure:"forward"`
}

var (
	appConfig     *Config
	appConfigOnce sync.Once
	appConfigMu   sync.RWMutex
)

// Load loads configuration from file and environment variables
func Load(env string) (*Config, error) {
	// Load single config file
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath("../configs")
	viper.AddConfigPath("../../configs")

	// Set environment variable prefix and replacer
	viper.SetEnvPrefix("ORRIS")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Set default values
	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Allow env parameter to override server mode if provided
	if env != "" && env != "default" {
		viper.Set("server.mode", env)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	appConfigMu.Lock()
	appConfig = &config
	appConfigMu.Unlock()

	return &config, nil
}

// Get returns the loaded configuration
func Get() *Config {
	appConfigMu.RLock()
	defer appConfigMu.RUnlock()
	return appConfig
}

// setDefaults sets default configuration values
func setDefaults() {
	// Server defaults
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.mode", "debug")

	// Database defaults
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 3306)
	viper.SetDefault("database.username", "root")
	viper.SetDefault("database.password", "password")
	viper.SetDefault("database.database", "orris_dev")
	viper.SetDefault("database.max_idle_conns", 10)
	viper.SetDefault("database.max_open_conns", 100)
	viper.SetDefault("database.conn_max_lifetime", 60)

	// Logger defaults
	viper.SetDefault("logger.level", "info")
	viper.SetDefault("logger.format", "console")
	viper.SetDefault("logger.output_path", "stdout")

	// Auth defaults
	viper.SetDefault("auth.password.bcrypt_cost", 12)
	viper.SetDefault("auth.token.verification_expires_hours", 24)
	viper.SetDefault("auth.token.reset_expires_minutes", 30)
	viper.SetDefault("auth.jwt.secret", "change-me-in-production")
	viper.SetDefault("auth.jwt.access_exp_minutes", 15)
	viper.SetDefault("auth.jwt.refresh_exp_days", 7)
	viper.SetDefault("auth.session.default_exp_days", 1)
	viper.SetDefault("auth.session.remember_exp_days", 30)
	viper.SetDefault("auth.cookie.domain", "")
	viper.SetDefault("auth.cookie.path", "/")
	viper.SetDefault("auth.cookie.secure", false)
	viper.SetDefault("auth.cookie.same_site", "Lax")

	// OAuth defaults (empty by default, must be configured)
	viper.SetDefault("oauth.google.client_id", "")
	viper.SetDefault("oauth.google.client_secret", "")
	viper.SetDefault("oauth.google.redirect_url", "http://localhost:8080/api/auth/oauth/google/callback")
	viper.SetDefault("oauth.github.client_id", "")
	viper.SetDefault("oauth.github.client_secret", "")
	viper.SetDefault("oauth.github.redirect_url", "http://localhost:8080/api/auth/oauth/github/callback")

	// Email defaults
	viper.SetDefault("email.smtp_host", "localhost")
	viper.SetDefault("email.smtp_port", 1025)
	viper.SetDefault("email.smtp_user", "")
	viper.SetDefault("email.smtp_password", "")
	viper.SetDefault("email.from_address", "noreply@orris.local")
	viper.SetDefault("email.from_name", "Orris")

	// Redis defaults
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)

	// Forward defaults
	viper.SetDefault("forward.token_signing_secret", "change-me-in-production")
}
