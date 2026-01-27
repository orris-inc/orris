package dto

import (
	"time"
)

// SystemSettingResponse represents a single setting response
type SystemSettingResponse struct {
	SID         string    `json:"sid"`
	Category    string    `json:"category"`
	Key         string    `json:"key"`
	Value       any       `json:"value"`
	ValueType   string    `json:"value_type"`
	Description string    `json:"description"`
	IsSensitive bool      `json:"is_sensitive"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CategorySettingsResponse represents a category settings response
type CategorySettingsResponse struct {
	Category string                  `json:"category"`
	Settings []SystemSettingResponse `json:"settings"`
}

// UpdateSettingRequest represents the request to update a single setting
type UpdateSettingRequest struct {
	Value any `json:"value" binding:"required"`
}

// UpdateCategorySettingsRequest represents the request to batch update category settings
type UpdateCategorySettingsRequest struct {
	Settings map[string]any `json:"settings" binding:"required"`
}

// TelegramConfigResponse represents the Telegram configuration response
type TelegramConfigResponse struct {
	Enabled       bool   `json:"enabled"`
	BotToken      string `json:"bot_token"` // masked display
	WebhookURL    string `json:"webhook_url"`
	WebhookSecret string `json:"webhook_secret"` // masked display
	BotLink       string `json:"bot_link,omitempty"`
	Mode          string `json:"mode"` // "webhook" or "polling"
}

// UpdateTelegramConfigRequest represents the request to update Telegram configuration
type UpdateTelegramConfigRequest struct {
	Enabled       *bool   `json:"enabled"`
	BotToken      *string `json:"bot_token"`
	WebhookURL    *string `json:"webhook_url"`
	WebhookSecret *string `json:"webhook_secret"`
}

// TelegramTestResult represents the result of testing Telegram connection
type TelegramTestResult struct {
	Success     bool   `json:"success"`
	BotUsername string `json:"bot_username,omitempty"`
	Error       string `json:"error,omitempty"`
}

// MaskSensitiveValue masks a sensitive value for display
// Returns "***...***" format for non-empty values
func MaskSensitiveValue(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 6 {
		return "***"
	}
	return "***...***"
}

// SensitiveKeys defines keys that should be masked in responses
var SensitiveKeys = map[string]bool{
	"bot_token":          true,
	"webhook_secret":     true,
	"api_key":            true,
	"secret_key":         true,
	"password":           true,
	"client_secret":      true, // OAuth
	"smtp_password":      true, // Email
	"polygonscan_api_key": true, // USDT - PolygonScan API
	"trongrid_api_key":    true, // USDT - TronGrid API
}

// IsSensitiveKey checks if a key should be masked
func IsSensitiveKey(key string) bool {
	return SensitiveKeys[key]
}

// ConfigSource represents where a configuration value comes from
type ConfigSource string

const (
	SourceDatabase    ConfigSource = "database"
	SourceEnvironment ConfigSource = "environment"
	SourceDefault     ConfigSource = "default"
)

// SettingWithSource represents a setting value with its source
type SettingWithSource struct {
	Value       any          `json:"value"`
	Source      ConfigSource `json:"source"`
	IsSensitive bool         `json:"is_sensitive"`
	IsReadOnly  bool         `json:"is_read_only,omitempty"`
}

// SystemSettingsResponse represents system settings response
type SystemSettingsResponse struct {
	APIBaseURL          SettingWithSource `json:"api_base_url"`
	SubscriptionBaseURL SettingWithSource `json:"subscription_base_url"`
	FrontendURL         SettingWithSource `json:"frontend_url"`
	Timezone            SettingWithSource `json:"timezone"`
}

// UpdateSystemSettingsRequest represents the request to update system settings
// Note: api_base_url is read-only and can only be configured via environment variable
type UpdateSystemSettingsRequest struct {
	SubscriptionBaseURL *string `json:"subscription_base_url"`
	FrontendURL         *string `json:"frontend_url"`
}

// OAuthProviderSettings represents OAuth provider settings
type OAuthProviderSettings struct {
	Enabled      bool              `json:"enabled"`
	ClientID     SettingWithSource `json:"client_id"`
	ClientSecret SettingWithSource `json:"client_secret"`
	RedirectURL  SettingWithSource `json:"redirect_url"`
}

// OAuthSettingsResponse represents OAuth settings response
type OAuthSettingsResponse struct {
	Google OAuthProviderSettings `json:"google"`
	GitHub OAuthProviderSettings `json:"github"`
}

// UpdateOAuthSettingsRequest represents the request to update OAuth settings
type UpdateOAuthSettingsRequest struct {
	Google *UpdateOAuthProviderRequest `json:"google"`
	GitHub *UpdateOAuthProviderRequest `json:"github"`
}

// UpdateOAuthProviderRequest represents the request to update a single OAuth provider
type UpdateOAuthProviderRequest struct {
	ClientID     *string `json:"client_id"`
	ClientSecret *string `json:"client_secret"`
	RedirectURL  *string `json:"redirect_url"`
}

// EmailSettingsResponse represents email settings response
type EmailSettingsResponse struct {
	SMTPHost     SettingWithSource `json:"smtp_host"`
	SMTPPort     SettingWithSource `json:"smtp_port"`
	SMTPUser     SettingWithSource `json:"smtp_user"`
	SMTPPassword SettingWithSource `json:"smtp_password"`
	FromAddress  SettingWithSource `json:"from_address"`
	FromName     SettingWithSource `json:"from_name"`
}

// UpdateEmailSettingsRequest represents the request to update email settings
type UpdateEmailSettingsRequest struct {
	SMTPHost     *string `json:"smtp_host"`
	SMTPPort     *int    `json:"smtp_port"`
	SMTPUser     *string `json:"smtp_user"`
	SMTPPassword *string `json:"smtp_password"`
	FromAddress  *string `json:"from_address"`
	FromName     *string `json:"from_name"`
}

// SetupStatusResponse represents the setup status for first-time configuration
type SetupStatusResponse struct {
	IsConfigured    bool     `json:"is_configured"`
	RequiresSetup   bool     `json:"requires_setup"`
	MissingSettings []string `json:"missing_settings"`
}

// EmailTestRequest represents the request to test email connection
type EmailTestRequest struct {
	RecipientEmail string `json:"recipient_email" binding:"required,email,max=254"`
}

// EmailTestResponse represents the result of testing email connection
type EmailTestResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}
