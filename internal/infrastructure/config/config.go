package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config represents application configuration
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Logger   LoggerConfig   `mapstructure:"logger"`
	Auth     AuthConfig     `mapstructure:"auth"`
	OAuth    OAuthConfig    `mapstructure:"oauth"`
	Email    EmailConfig    `mapstructure:"email"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"` // debug, release, test
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	Username        string `mapstructure:"username"`
	Password        string `mapstructure:"password"`
	Database        string `mapstructure:"database"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"` // in minutes
}

// LoggerConfig represents logger configuration
type LoggerConfig struct {
	Level      string `mapstructure:"level"`       // debug, info, warn, error
	Format     string `mapstructure:"format"`      // json, console
	OutputPath string `mapstructure:"output_path"` // stdout, stderr, or file path
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	Password PasswordConfig `mapstructure:"password"`
	Token    TokenConfig    `mapstructure:"token"`
	JWT      JWTConfig      `mapstructure:"jwt"`
}

// PasswordConfig represents password hashing configuration
type PasswordConfig struct {
	BcryptCost int `mapstructure:"bcrypt_cost"`
}

// TokenConfig represents token expiration configuration
type TokenConfig struct {
	VerificationExpiresHours int `mapstructure:"verification_expires_hours"`
	ResetExpiresMinutes      int `mapstructure:"reset_expires_minutes"`
}

// JWTConfig represents JWT configuration
type JWTConfig struct {
	Secret            string `mapstructure:"secret"`
	AccessExpMinutes  int    `mapstructure:"access_exp_minutes"`
	RefreshExpDays    int    `mapstructure:"refresh_exp_days"`
}

// OAuthConfig represents OAuth providers configuration
type OAuthConfig struct {
	Google GoogleOAuthConfig `mapstructure:"google"`
	GitHub GitHubOAuthConfig `mapstructure:"github"`
}

// GoogleOAuthConfig represents Google OAuth configuration
type GoogleOAuthConfig struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	RedirectURL  string `mapstructure:"redirect_url"`
}

// GitHubOAuthConfig represents GitHub OAuth configuration
type GitHubOAuthConfig struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	RedirectURL  string `mapstructure:"redirect_url"`
}

// EmailConfig represents email service configuration
type EmailConfig struct {
	SMTPHost     string `mapstructure:"smtp_host"`
	SMTPPort     int    `mapstructure:"smtp_port"`
	SMTPUser     string `mapstructure:"smtp_user"`
	SMTPPassword string `mapstructure:"smtp_password"`
	FromAddress  string `mapstructure:"from_address"`
	FromName     string `mapstructure:"from_name"`
}

// GetAddr returns the server address
func (s *ServerConfig) GetAddr() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// GetDSN returns the MySQL DSN (Data Source Name)
func (d *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		d.Username, d.Password, d.Host, d.Port, d.Database)
}

var appConfig *Config

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

	appConfig = &config
	return &config, nil
}

// Get returns the loaded configuration
func Get() *Config {
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
}