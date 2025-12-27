package telegram

import (
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/id"
)

// NotifyWindowHours defines the deduplication window (24 hours)
const NotifyWindowHours = 24

// TelegramBinding represents the telegram binding aggregate root
type TelegramBinding struct {
	id               uint
	sid              string // Stripe-style ID: tg_bind_xxx
	userID           uint   // Internal user ID reference
	telegramUserID   int64  // Telegram user_id
	telegramUsername string // @username (optional)

	// Notification preferences
	notifyExpiring   bool // Subscription expiring reminder
	notifyTraffic    bool // Traffic usage reminder
	expiringDays     int  // Days before expiry to notify (default: 3)
	trafficThreshold int  // Traffic threshold percentage (default: 80)

	// Time window deduplication
	lastExpiringNotifyAt *time.Time // Last expiring notification time
	lastTrafficNotifyAt  *time.Time // Last traffic notification time

	createdAt time.Time
	updatedAt time.Time
}

// NewTelegramBinding creates a new telegram binding
func NewTelegramBinding(userID uint, telegramUserID int64, telegramUsername string) (*TelegramBinding, error) {
	if userID == 0 {
		return nil, fmt.Errorf("user ID is required")
	}
	if telegramUserID == 0 {
		return nil, fmt.Errorf("telegram user ID is required")
	}

	sid, err := id.NewTelegramBindingID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate SID: %w", err)
	}

	now := biztime.NowUTC()
	return &TelegramBinding{
		sid:              sid,
		userID:           userID,
		telegramUserID:   telegramUserID,
		telegramUsername: telegramUsername,
		notifyExpiring:   true, // Default enabled
		notifyTraffic:    true, // Default enabled
		expiringDays:     3,    // Default 3 days
		trafficThreshold: 80,   // Default 80%
		createdAt:        now,
		updatedAt:        now,
	}, nil
}

// ReconstructTelegramBinding reconstructs from persistence
func ReconstructTelegramBinding(
	id uint,
	sid string,
	userID uint,
	telegramUserID int64,
	telegramUsername string,
	notifyExpiring bool,
	notifyTraffic bool,
	expiringDays int,
	trafficThreshold int,
	lastExpiringNotifyAt *time.Time,
	lastTrafficNotifyAt *time.Time,
	createdAt, updatedAt time.Time,
) *TelegramBinding {
	return &TelegramBinding{
		id:                   id,
		sid:                  sid,
		userID:               userID,
		telegramUserID:       telegramUserID,
		telegramUsername:     telegramUsername,
		notifyExpiring:       notifyExpiring,
		notifyTraffic:        notifyTraffic,
		expiringDays:         expiringDays,
		trafficThreshold:     trafficThreshold,
		lastExpiringNotifyAt: lastExpiringNotifyAt,
		lastTrafficNotifyAt:  lastTrafficNotifyAt,
		createdAt:            createdAt,
		updatedAt:            updatedAt,
	}
}

// Getters
func (b *TelegramBinding) ID() uint                         { return b.id }
func (b *TelegramBinding) SID() string                      { return b.sid }
func (b *TelegramBinding) UserID() uint                     { return b.userID }
func (b *TelegramBinding) TelegramUserID() int64            { return b.telegramUserID }
func (b *TelegramBinding) TelegramUsername() string         { return b.telegramUsername }
func (b *TelegramBinding) NotifyExpiring() bool             { return b.notifyExpiring }
func (b *TelegramBinding) NotifyTraffic() bool              { return b.notifyTraffic }
func (b *TelegramBinding) ExpiringDays() int                { return b.expiringDays }
func (b *TelegramBinding) TrafficThreshold() int            { return b.trafficThreshold }
func (b *TelegramBinding) LastExpiringNotifyAt() *time.Time { return b.lastExpiringNotifyAt }
func (b *TelegramBinding) LastTrafficNotifyAt() *time.Time  { return b.lastTrafficNotifyAt }
func (b *TelegramBinding) CreatedAt() time.Time             { return b.createdAt }
func (b *TelegramBinding) UpdatedAt() time.Time             { return b.updatedAt }

// SetID sets the binding ID (only for persistence layer use)
func (b *TelegramBinding) SetID(id uint) {
	b.id = id
}

// UpdatePreferences updates notification preferences
func (b *TelegramBinding) UpdatePreferences(notifyExpiring, notifyTraffic bool, expiringDays, trafficThreshold int) error {
	if expiringDays < 1 || expiringDays > 30 {
		return fmt.Errorf("expiring days must be between 1 and 30")
	}
	if trafficThreshold < 50 || trafficThreshold > 99 {
		return fmt.Errorf("traffic threshold must be between 50 and 99")
	}

	b.notifyExpiring = notifyExpiring
	b.notifyTraffic = notifyTraffic
	b.expiringDays = expiringDays
	b.trafficThreshold = trafficThreshold
	b.updatedAt = biztime.NowUTC()
	return nil
}

// CanNotifyExpiring checks if expiring notification can be sent (deduplication)
func (b *TelegramBinding) CanNotifyExpiring() bool {
	if !b.notifyExpiring {
		return false
	}
	if b.lastExpiringNotifyAt == nil {
		return true
	}
	// Use UTC time for consistent comparison
	return biztime.NowUTC().Sub(*b.lastExpiringNotifyAt).Hours() >= NotifyWindowHours
}

// CanNotifyTraffic checks if traffic notification can be sent (deduplication)
func (b *TelegramBinding) CanNotifyTraffic() bool {
	if !b.notifyTraffic {
		return false
	}
	if b.lastTrafficNotifyAt == nil {
		return true
	}
	// Use UTC time for consistent comparison
	return biztime.NowUTC().Sub(*b.lastTrafficNotifyAt).Hours() >= NotifyWindowHours
}

// RecordExpiringNotification records that an expiring notification was sent
func (b *TelegramBinding) RecordExpiringNotification() {
	now := biztime.NowUTC()
	b.lastExpiringNotifyAt = &now
	b.updatedAt = now
}

// RecordTrafficNotification records that a traffic notification was sent
func (b *TelegramBinding) RecordTrafficNotification() {
	now := biztime.NowUTC()
	b.lastTrafficNotifyAt = &now
	b.updatedAt = now
}

// UpdateTelegramUsername updates the telegram username
func (b *TelegramBinding) UpdateTelegramUsername(username string) {
	b.telegramUsername = username
	b.updatedAt = biztime.NowUTC()
}
