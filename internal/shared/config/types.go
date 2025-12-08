package config

import "fmt"

type ServerConfig struct {
	Host                string   `mapstructure:"host"`
	Port                int      `mapstructure:"port"`
	Mode                string   `mapstructure:"mode"`
	BaseURL             string   `mapstructure:"base_url"`
	AllowedOrigins      []string `mapstructure:"allowed_origins"`
	FrontendCallbackURL string   `mapstructure:"frontend_callback_url"`
}

func (s *ServerConfig) GetAddr() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// GetBaseURL returns the base URL, auto-generated if not explicitly set
func (s *ServerConfig) GetBaseURL() string {
	if s.BaseURL != "" {
		return s.BaseURL
	}
	// Auto-generate: use localhost for 0.0.0.0
	host := s.Host
	if host == "0.0.0.0" || host == "" {
		host = "localhost"
	}
	return fmt.Sprintf("http://%s:%d", host, s.Port)
}

type DatabaseConfig struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	Username        string `mapstructure:"username"`
	Password        string `mapstructure:"password"`
	Database        string `mapstructure:"database"`
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"`
}

func (d *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		d.Username, d.Password, d.Host, d.Port, d.Database)
}

type LoggerConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	OutputPath string `mapstructure:"output_path"`
}

type PasswordConfig struct {
	BcryptCost int `mapstructure:"bcrypt_cost"`
}

type TokenConfig struct {
	VerificationExpiresHours int `mapstructure:"verification_expires_hours"`
	ResetExpiresMinutes      int `mapstructure:"reset_expires_minutes"`
}

type JWTConfig struct {
	Secret           string `mapstructure:"secret"`
	AccessExpMinutes int    `mapstructure:"access_exp_minutes"`
	RefreshExpDays   int    `mapstructure:"refresh_exp_days"`
}

type SessionConfig struct {
	DefaultExpDays  int `mapstructure:"default_exp_days"`
	RememberExpDays int `mapstructure:"remember_exp_days"`
}

type CookieConfig struct {
	Domain   string `mapstructure:"domain"`
	Path     string `mapstructure:"path"`
	Secure   bool   `mapstructure:"secure"`
	SameSite string `mapstructure:"same_site"` // Strict, Lax, None
}

type AuthConfig struct {
	Password PasswordConfig `mapstructure:"password"`
	Token    TokenConfig    `mapstructure:"token"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Session  SessionConfig  `mapstructure:"session"`
	Cookie   CookieConfig   `mapstructure:"cookie"`
}

type GoogleOAuthConfig struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	RedirectURL  string `mapstructure:"redirect_url"`
}

// GetRedirectURL returns the redirect URL, auto-generated if not explicitly set
func (g *GoogleOAuthConfig) GetRedirectURL(baseURL string) string {
	if g.RedirectURL != "" {
		return g.RedirectURL
	}
	return fmt.Sprintf("%s/auth/oauth/google/callback", baseURL)
}

type GitHubOAuthConfig struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	RedirectURL  string `mapstructure:"redirect_url"`
}

// GetRedirectURL returns the redirect URL, auto-generated if not explicitly set
func (g *GitHubOAuthConfig) GetRedirectURL(baseURL string) string {
	if g.RedirectURL != "" {
		return g.RedirectURL
	}
	return fmt.Sprintf("%s/auth/oauth/github/callback", baseURL)
}

type OAuthConfig struct {
	Google GoogleOAuthConfig `mapstructure:"google"`
	GitHub GitHubOAuthConfig `mapstructure:"github"`
}

type EmailConfig struct {
	SMTPHost     string `mapstructure:"smtp_host"`
	SMTPPort     int    `mapstructure:"smtp_port"`
	SMTPUser     string `mapstructure:"smtp_user"`
	SMTPPassword string `mapstructure:"smtp_password"`
	FromAddress  string `mapstructure:"from_address"`
	FromName     string `mapstructure:"from_name"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

func (r *RedisConfig) GetAddr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// SubscriptionConfig holds subscription-related configuration
type SubscriptionConfig struct {
	// BaseURL is the base URL for subscription links (e.g., "https://sub.example.com")
	// If empty, falls back to server base URL
	BaseURL string `mapstructure:"base_url"`
}

// GetBaseURL returns the subscription base URL, falling back to server base URL if not set
func (s *SubscriptionConfig) GetBaseURL(serverBaseURL string) string {
	if s.BaseURL != "" {
		return s.BaseURL
	}
	return serverBaseURL
}
