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
	"bot_token":      true,
	"webhook_secret": true,
	"api_key":        true,
	"secret_key":     true,
	"password":       true,
}

// IsSensitiveKey checks if a key should be masked
func IsSensitiveKey(key string) bool {
	return SensitiveKeys[key]
}
