package dto

import "time"

// AdminTelegramBindingResponse represents the admin telegram binding response
type AdminTelegramBindingResponse struct {
	SID                     string    `json:"sid"`
	TelegramUserID          int64     `json:"telegram_user_id"`
	TelegramUsername        string    `json:"telegram_username,omitempty"`
	NotifyNodeOffline       bool      `json:"notify_node_offline"`
	NotifyAgentOffline      bool      `json:"notify_agent_offline"`
	NotifyNewUser           bool      `json:"notify_new_user"`
	NotifyPaymentSuccess    bool      `json:"notify_payment_success"`
	NotifyDailySummary      bool      `json:"notify_daily_summary"`
	NotifyWeeklySummary     bool      `json:"notify_weekly_summary"`
	OfflineThresholdMinutes int       `json:"offline_threshold_minutes"`
	NotifyResourceExpiring  bool      `json:"notify_resource_expiring"`
	ResourceExpiringDays    int       `json:"resource_expiring_days"`
	CreatedAt               time.Time `json:"created_at"`
}

// AdminBindingStatusResponse represents the admin binding status response
type AdminBindingStatusResponse struct {
	IsBound    bool                          `json:"is_bound"`
	Binding    *AdminTelegramBindingResponse `json:"binding,omitempty"`
	VerifyCode string                        `json:"verify_code,omitempty"` // Shown when not bound
	BotLink    string                        `json:"bot_link,omitempty"`    // Telegram bot link (https://t.me/username)
	ExpiresAt  *time.Time                    `json:"expires_at,omitempty"`  // Verify code expiration time
}

// UpdateAdminPreferencesRequest represents the request to update admin notification preferences
type UpdateAdminPreferencesRequest struct {
	NotifyNodeOffline       *bool `json:"notify_node_offline"`
	NotifyAgentOffline      *bool `json:"notify_agent_offline"`
	NotifyNewUser           *bool `json:"notify_new_user"`
	NotifyPaymentSuccess    *bool `json:"notify_payment_success"`
	NotifyDailySummary      *bool `json:"notify_daily_summary"`
	NotifyWeeklySummary     *bool `json:"notify_weekly_summary"`
	OfflineThresholdMinutes *int  `json:"offline_threshold_minutes"`
	NotifyResourceExpiring  *bool `json:"notify_resource_expiring"`
	ResourceExpiringDays    *int  `json:"resource_expiring_days"`
}
