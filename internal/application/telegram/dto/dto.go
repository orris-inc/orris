package dto

import "time"

// TelegramBindingResponse represents the telegram binding response
type TelegramBindingResponse struct {
	SID              string    `json:"sid"`
	TelegramUserID   int64     `json:"telegram_user_id"`
	TelegramUsername string    `json:"telegram_username,omitempty"`
	NotifyExpiring   bool      `json:"notify_expiring"`
	NotifyTraffic    bool      `json:"notify_traffic"`
	ExpiringDays     int       `json:"expiring_days"`
	TrafficThreshold int       `json:"traffic_threshold"`
	CreatedAt        time.Time `json:"created_at"`
}

// BindingStatusResponse represents the binding status response
type BindingStatusResponse struct {
	IsBound    bool                     `json:"is_bound"`
	Binding    *TelegramBindingResponse `json:"binding,omitempty"`
	VerifyCode string                   `json:"verify_code,omitempty"` // Shown when not bound
}

// UpdatePreferencesRequest represents the request to update notification preferences
type UpdatePreferencesRequest struct {
	NotifyExpiring   *bool `json:"notify_expiring"`
	NotifyTraffic    *bool `json:"notify_traffic"`
	ExpiringDays     *int  `json:"expiring_days"`
	TrafficThreshold *int  `json:"traffic_threshold"`
}

// WebhookUpdate represents a Telegram webhook update
type WebhookUpdate struct {
	UpdateID int64           `json:"update_id"`
	Message  *WebhookMessage `json:"message,omitempty"`
}

// WebhookMessage represents a Telegram message
type WebhookMessage struct {
	MessageID int64         `json:"message_id"`
	From      *TelegramUser `json:"from,omitempty"`
	Chat      *TelegramChat `json:"chat,omitempty"`
	Text      string        `json:"text,omitempty"`
}

// TelegramUser represents a Telegram user
type TelegramUser struct {
	ID        int64  `json:"id"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}

// TelegramChat represents a Telegram chat
type TelegramChat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

// ReminderResult represents the result of a reminder check
type ReminderResult struct {
	ExpiringNotified int `json:"expiring_notified"`
	TrafficNotified  int `json:"traffic_notified"`
	Errors           int `json:"errors"`
}
