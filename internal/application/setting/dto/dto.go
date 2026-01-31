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
	"bot_token":           true,
	"webhook_secret":      true,
	"api_key":             true,
	"secret_key":          true,
	"password":            true,
	"client_secret":       true, // OAuth
	"smtp_password":       true, // Email
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

// SubscriptionSettingsResponse represents subscription settings response
// Controls how subscription output is displayed to end users
type SubscriptionSettingsResponse struct {
	// ShowInfoNodes controls whether to show info nodes (expire/traffic) in subscription output
	// When enabled, subscription clients will see two info nodes at the top:
	// - "📅 到期: YYYY-MM-DD" (expiration date)
	// - "📊 流量: X.XXG / Y.YYG" (traffic usage)
	ShowInfoNodes SettingWithSource `json:"show_info_nodes"`
}

// UpdateSubscriptionSettingsRequest represents the request to update subscription settings
type UpdateSubscriptionSettingsRequest struct {
	// ShowInfoNodes enables/disables info nodes in subscription output
	ShowInfoNodes *bool `json:"show_info_nodes"`
}

// BrandingSettingsResponse represents branding settings response (admin)
type BrandingSettingsResponse struct {
	AppName    SettingWithSource `json:"app_name"`
	LogoURL    SettingWithSource `json:"logo_url"`
	FaviconURL SettingWithSource `json:"favicon_url"`
}

// UpdateBrandingSettingsRequest represents the request to update branding settings
type UpdateBrandingSettingsRequest struct {
	AppName    *string `json:"app_name" binding:"omitempty,max=50"`
	LogoURL    *string `json:"logo_url" binding:"omitempty,max=500"`
	FaviconURL *string `json:"favicon_url" binding:"omitempty,max=500"`
}

// PublicBrandingResponse represents public branding response (simplified, no source tracking)
type PublicBrandingResponse struct {
	AppName    string `json:"app_name"`
	LogoURL    string `json:"logo_url"`
	FaviconURL string `json:"favicon_url"`
}

// BrandingUploadResponse represents the response after uploading a branding image
type BrandingUploadResponse struct {
	URL string `json:"url"`
}

// SecuritySettingsResponse represents security settings response
type SecuritySettingsResponse struct {
	// Password policy
	PasswordMinLength        SettingWithSource `json:"password_min_length"`        // Minimum password length (default: 8)
	PasswordRequireUppercase SettingWithSource `json:"password_require_uppercase"` // Require uppercase letter (default: false)
	PasswordRequireLowercase SettingWithSource `json:"password_require_lowercase"` // Require lowercase letter (default: false)
	PasswordRequireNumber    SettingWithSource `json:"password_require_number"`    // Require number (default: false)
	PasswordRequireSpecial   SettingWithSource `json:"password_require_special"`   // Require special character (default: false)
	PasswordExpiryDays       SettingWithSource `json:"password_expiry_days"`       // Password expiry in days, 0 = never (default: 0)

	// Session settings
	SessionTimeoutMinutes SettingWithSource `json:"session_timeout_minutes"` // Session timeout in minutes (default: 1440 = 24 hours)

	// Login protection
	MaxLoginAttempts       SettingWithSource `json:"max_login_attempts"`       // Max login attempts before lockout (default: 5)
	LockoutDurationMinutes SettingWithSource `json:"lockout_duration_minutes"` // Lockout duration after max attempts (default: 15)
}

// UpdateSecuritySettingsRequest represents the request to update security settings
type UpdateSecuritySettingsRequest struct {
	// Password policy
	PasswordMinLength        *int  `json:"password_min_length" binding:"omitempty,min=8,max=32"`
	PasswordRequireUppercase *bool `json:"password_require_uppercase"`
	PasswordRequireLowercase *bool `json:"password_require_lowercase"`
	PasswordRequireNumber    *bool `json:"password_require_number"`
	PasswordRequireSpecial   *bool `json:"password_require_special"`
	PasswordExpiryDays       *int  `json:"password_expiry_days" binding:"omitempty,min=0,max=365"`

	// Session settings
	SessionTimeoutMinutes *int `json:"session_timeout_minutes" binding:"omitempty,min=5,max=43200"`

	// Login protection
	MaxLoginAttempts       *int `json:"max_login_attempts" binding:"omitempty,min=3,max=20"`
	LockoutDurationMinutes *int `json:"lockout_duration_minutes" binding:"omitempty,min=1,max=1440"`
}

// RegistrationSettingsResponse represents registration settings response
type RegistrationSettingsResponse struct {
	// RegistrationEnabled controls whether new user registration is allowed
	RegistrationEnabled SettingWithSource `json:"registration_enabled"`
	// EmailVerificationRequired controls whether email verification is required
	EmailVerificationRequired SettingWithSource `json:"email_verification_required"`
}

// UpdateRegistrationSettingsRequest represents the request to update registration settings
type UpdateRegistrationSettingsRequest struct {
	RegistrationEnabled       *bool `json:"registration_enabled"`
	EmailVerificationRequired *bool `json:"email_verification_required"`
}

// LegalSettingsResponse represents legal settings response
type LegalSettingsResponse struct {
	// TermsOfServiceURL is the URL to the terms of service page
	TermsOfServiceURL SettingWithSource `json:"terms_of_service_url"`
	// PrivacyPolicyURL is the URL to the privacy policy page
	PrivacyPolicyURL SettingWithSource `json:"privacy_policy_url"`
}

// UpdateLegalSettingsRequest represents the request to update legal settings
type UpdateLegalSettingsRequest struct {
	TermsOfServiceURL *string `json:"terms_of_service_url" binding:"omitempty,max=500"`
	PrivacyPolicyURL  *string `json:"privacy_policy_url" binding:"omitempty,max=500"`
}

// PublicLegalResponse represents public legal URLs (no auth required)
type PublicLegalResponse struct {
	TermsOfServiceURL string `json:"terms_of_service_url"`
	PrivacyPolicyURL  string `json:"privacy_policy_url"`
}

// PublicRegistrationResponse represents public registration settings (no auth required)
type PublicRegistrationResponse struct {
	RegistrationEnabled       bool `json:"registration_enabled"`
	EmailVerificationRequired bool `json:"email_verification_required"`
}

// PasswordPolicyRule represents a single password policy rule for frontend display
type PasswordPolicyRule struct {
	Type     string `json:"type"`               // Rule type: min_length, max_length, uppercase, lowercase, number, special
	Value    any    `json:"value,omitempty"`    // Value for length rules (int)
	Required bool   `json:"required,omitempty"` // Whether this rule is required (for character rules)
	Message  string `json:"message"`            // Human-readable description
}

// PublicPasswordPolicyResponse represents public password policy (no auth required)
// Used by frontend to display password requirements during registration/password change
type PublicPasswordPolicyResponse struct {
	MinLength         int                  `json:"min_length"`
	MaxLength         int                  `json:"max_length"`
	RequireUppercase  bool                 `json:"require_uppercase"`
	RequireLowercase  bool                 `json:"require_lowercase"`
	RequireNumber     bool                 `json:"require_number"`
	RequireSpecial    bool                 `json:"require_special"`
	Rules             []PasswordPolicyRule `json:"rules"`
	SpecialCharacters string               `json:"special_characters"`
}
