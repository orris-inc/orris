package config

import "fmt"

type ServerConfig struct {
	Host                  string   `mapstructure:"host"`
	Port                  int      `mapstructure:"port"`
	Mode                  string   `mapstructure:"mode"`
	BaseURL               string   `mapstructure:"base_url"`
	AllowedOrigins        []string `mapstructure:"allowed_origins"`
	FrontendCallbackURL   string   `mapstructure:"frontend_callback_url"`
}

func (s *ServerConfig) GetAddr() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
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

type AuthConfig struct {
	Password PasswordConfig `mapstructure:"password"`
	Token    TokenConfig    `mapstructure:"token"`
	JWT      JWTConfig      `mapstructure:"jwt"`
}

type GoogleOAuthConfig struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	RedirectURL  string `mapstructure:"redirect_url"`
}

type GitHubOAuthConfig struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
	RedirectURL  string `mapstructure:"redirect_url"`
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
