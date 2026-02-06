package admin

import (
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/id"
)

// Default configuration values
const (
	DefaultOfflineThresholdMinutes = 5
	MinOfflineThresholdMinutes     = 3
	MaxOfflineThresholdMinutes     = 30
	NotifyWindowHours              = 24

	// Resource expiring notification defaults
	DefaultResourceExpiringDays = 7
	MinResourceExpiringDays     = 1
	MaxResourceExpiringDays     = 30

	// Schedule configuration defaults and ranges
	DefaultDailySummaryHour     = 9
	DefaultWeeklySummaryHour    = 9
	DefaultWeeklySummaryWeekday = 1 // Monday
	MinSummaryHour              = 0
	MaxSummaryHour              = 23
	MinWeekday                  = 0 // Sunday
	MaxWeekday                  = 6 // Saturday

	DefaultOfflineCheckIntervalMinutes = 5
	MinOfflineCheckIntervalMinutes     = 1
	MaxOfflineCheckIntervalMinutes     = 30
)

// AdminTelegramBinding represents the admin telegram binding aggregate root
type AdminTelegramBinding struct {
	id               uint
	sid              string // Stripe-style ID: atg_bind_xxx
	userID           uint   // Internal user ID reference (must be admin role)
	telegramUserID   int64  // Telegram user_id
	telegramUsername string // @username (optional)
	language         string // User's preferred language (e.g., "zh", "en")

	// Notification preferences
	notifyNodeOffline    bool // Node offline alert
	notifyAgentOffline   bool // Forward agent offline alert
	notifyNewUser        bool // New user registration alert
	notifyPaymentSuccess bool // Payment success alert
	notifyDailySummary   bool // Daily business summary
	notifyWeeklySummary  bool // Weekly business summary

	// Thresholds
	offlineThresholdMinutes int // Minutes before considering offline

	// Resource expiring notification
	notifyResourceExpiring         bool       // Resource expiring alert
	resourceExpiringDays           int        // Days before expiration to start notifying
	lastResourceExpiringNotifyDate *time.Time // Date of last expiring notification (for daily deduplication)

	// Schedule configuration
	dailySummaryHour            int // Hour to send daily summary (0-23, business timezone)
	weeklySummaryHour           int // Hour to send weekly summary (0-23, business timezone)
	weeklySummaryWeekday        int // Weekday to send weekly summary (0=Sunday, 1=Monday...6=Saturday)
	offlineCheckIntervalMinutes int // Interval for offline re-check in minutes (1-30)

	// Time window deduplication
	lastNodeOfflineNotifyAt  *time.Time
	lastAgentOfflineNotifyAt *time.Time
	lastDailySummaryAt       *time.Time
	lastWeeklySummaryAt      *time.Time

	createdAt time.Time
	updatedAt time.Time
}

// NewAdminTelegramBinding creates a new admin telegram binding
func NewAdminTelegramBinding(userID uint, telegramUserID int64, telegramUsername, language string) (*AdminTelegramBinding, error) {
	if userID == 0 {
		return nil, fmt.Errorf("user ID is required")
	}
	if telegramUserID == 0 {
		return nil, fmt.Errorf("telegram user ID is required")
	}

	sid, err := id.NewAdminTelegramBindingID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate SID: %w", err)
	}

	if language == "" {
		language = "zh"
	}
	now := biztime.NowUTC()
	return &AdminTelegramBinding{
		sid:                         sid,
		userID:                      userID,
		telegramUserID:              telegramUserID,
		telegramUsername:            telegramUsername,
		language:                    language,
		notifyNodeOffline:           true, // Default enabled
		notifyAgentOffline:          true, // Default enabled
		notifyNewUser:               true, // Default enabled
		notifyPaymentSuccess:        true, // Default enabled
		notifyDailySummary:          true, // Default enabled
		notifyWeeklySummary:         true, // Default enabled
		offlineThresholdMinutes:     DefaultOfflineThresholdMinutes,
		notifyResourceExpiring:      true, // Default enabled
		resourceExpiringDays:        DefaultResourceExpiringDays,
		dailySummaryHour:            DefaultDailySummaryHour,
		weeklySummaryHour:           DefaultWeeklySummaryHour,
		weeklySummaryWeekday:        DefaultWeeklySummaryWeekday,
		offlineCheckIntervalMinutes: DefaultOfflineCheckIntervalMinutes,
		createdAt:                   now,
		updatedAt:                   now,
	}, nil
}

// ReconstructAdminTelegramBinding reconstructs from persistence
func ReconstructAdminTelegramBinding(
	id uint,
	sid string,
	userID uint,
	telegramUserID int64,
	telegramUsername string,
	language string,
	notifyNodeOffline bool,
	notifyAgentOffline bool,
	notifyNewUser bool,
	notifyPaymentSuccess bool,
	notifyDailySummary bool,
	notifyWeeklySummary bool,
	offlineThresholdMinutes int,
	notifyResourceExpiring bool,
	resourceExpiringDays int,
	dailySummaryHour int,
	weeklySummaryHour int,
	weeklySummaryWeekday int,
	offlineCheckIntervalMinutes int,
	lastNodeOfflineNotifyAt *time.Time,
	lastAgentOfflineNotifyAt *time.Time,
	lastDailySummaryAt *time.Time,
	lastWeeklySummaryAt *time.Time,
	lastResourceExpiringNotifyDate *time.Time,
	createdAt, updatedAt time.Time,
) *AdminTelegramBinding {
	return &AdminTelegramBinding{
		id:                             id,
		sid:                            sid,
		userID:                         userID,
		telegramUserID:                 telegramUserID,
		telegramUsername:               telegramUsername,
		language:                       language,
		notifyNodeOffline:              notifyNodeOffline,
		notifyAgentOffline:             notifyAgentOffline,
		notifyNewUser:                  notifyNewUser,
		notifyPaymentSuccess:           notifyPaymentSuccess,
		notifyDailySummary:             notifyDailySummary,
		notifyWeeklySummary:            notifyWeeklySummary,
		offlineThresholdMinutes:        offlineThresholdMinutes,
		notifyResourceExpiring:         notifyResourceExpiring,
		resourceExpiringDays:           resourceExpiringDays,
		dailySummaryHour:               dailySummaryHour,
		weeklySummaryHour:              weeklySummaryHour,
		weeklySummaryWeekday:           weeklySummaryWeekday,
		offlineCheckIntervalMinutes:    offlineCheckIntervalMinutes,
		lastNodeOfflineNotifyAt:        lastNodeOfflineNotifyAt,
		lastAgentOfflineNotifyAt:       lastAgentOfflineNotifyAt,
		lastDailySummaryAt:             lastDailySummaryAt,
		lastWeeklySummaryAt:            lastWeeklySummaryAt,
		lastResourceExpiringNotifyDate: lastResourceExpiringNotifyDate,
		createdAt:                      createdAt,
		updatedAt:                      updatedAt,
	}
}

// Getters
func (b *AdminTelegramBinding) ID() uint                            { return b.id }
func (b *AdminTelegramBinding) SID() string                         { return b.sid }
func (b *AdminTelegramBinding) UserID() uint                        { return b.userID }
func (b *AdminTelegramBinding) TelegramUserID() int64               { return b.telegramUserID }
func (b *AdminTelegramBinding) TelegramUsername() string            { return b.telegramUsername }
func (b *AdminTelegramBinding) Language() string                    { return b.language }
func (b *AdminTelegramBinding) NotifyNodeOffline() bool             { return b.notifyNodeOffline }
func (b *AdminTelegramBinding) NotifyAgentOffline() bool            { return b.notifyAgentOffline }
func (b *AdminTelegramBinding) NotifyNewUser() bool                 { return b.notifyNewUser }
func (b *AdminTelegramBinding) NotifyPaymentSuccess() bool          { return b.notifyPaymentSuccess }
func (b *AdminTelegramBinding) NotifyDailySummary() bool            { return b.notifyDailySummary }
func (b *AdminTelegramBinding) NotifyWeeklySummary() bool           { return b.notifyWeeklySummary }
func (b *AdminTelegramBinding) OfflineThresholdMinutes() int        { return b.offlineThresholdMinutes }
func (b *AdminTelegramBinding) LastNodeOfflineNotifyAt() *time.Time { return b.lastNodeOfflineNotifyAt }
func (b *AdminTelegramBinding) LastAgentOfflineNotifyAt() *time.Time {
	return b.lastAgentOfflineNotifyAt
}
func (b *AdminTelegramBinding) LastDailySummaryAt() *time.Time  { return b.lastDailySummaryAt }
func (b *AdminTelegramBinding) LastWeeklySummaryAt() *time.Time { return b.lastWeeklySummaryAt }
func (b *AdminTelegramBinding) NotifyResourceExpiring() bool    { return b.notifyResourceExpiring }
func (b *AdminTelegramBinding) ResourceExpiringDays() int       { return b.resourceExpiringDays }
func (b *AdminTelegramBinding) DailySummaryHour() int           { return b.dailySummaryHour }
func (b *AdminTelegramBinding) WeeklySummaryHour() int          { return b.weeklySummaryHour }
func (b *AdminTelegramBinding) WeeklySummaryWeekday() int       { return b.weeklySummaryWeekday }
func (b *AdminTelegramBinding) OfflineCheckIntervalMinutes() int {
	return b.offlineCheckIntervalMinutes
}
func (b *AdminTelegramBinding) LastResourceExpiringNotifyDate() *time.Time {
	return b.lastResourceExpiringNotifyDate
}
func (b *AdminTelegramBinding) CreatedAt() time.Time { return b.createdAt }
func (b *AdminTelegramBinding) UpdatedAt() time.Time { return b.updatedAt }

// SetID sets the binding ID (only for persistence layer use)
func (b *AdminTelegramBinding) SetID(id uint) {
	b.id = id
}

// UpdatePreferences updates notification preferences
func (b *AdminTelegramBinding) UpdatePreferences(
	notifyNodeOffline *bool,
	notifyAgentOffline *bool,
	notifyNewUser *bool,
	notifyPaymentSuccess *bool,
	notifyDailySummary *bool,
	notifyWeeklySummary *bool,
	offlineThresholdMinutes *int,
	notifyResourceExpiring *bool,
	resourceExpiringDays *int,
	dailySummaryHour *int,
	weeklySummaryHour *int,
	weeklySummaryWeekday *int,
	offlineCheckIntervalMinutes *int,
) error {
	if offlineThresholdMinutes != nil {
		if *offlineThresholdMinutes < MinOfflineThresholdMinutes || *offlineThresholdMinutes > MaxOfflineThresholdMinutes {
			return fmt.Errorf("offline threshold must be between %d and %d minutes", MinOfflineThresholdMinutes, MaxOfflineThresholdMinutes)
		}
		b.offlineThresholdMinutes = *offlineThresholdMinutes
	}

	if resourceExpiringDays != nil {
		if *resourceExpiringDays < MinResourceExpiringDays || *resourceExpiringDays > MaxResourceExpiringDays {
			return fmt.Errorf("resource expiring days must be between %d and %d", MinResourceExpiringDays, MaxResourceExpiringDays)
		}
		b.resourceExpiringDays = *resourceExpiringDays
	}

	if dailySummaryHour != nil {
		if *dailySummaryHour < MinSummaryHour || *dailySummaryHour > MaxSummaryHour {
			return fmt.Errorf("daily summary hour must be between %d and %d", MinSummaryHour, MaxSummaryHour)
		}
		b.dailySummaryHour = *dailySummaryHour
	}

	if weeklySummaryHour != nil {
		if *weeklySummaryHour < MinSummaryHour || *weeklySummaryHour > MaxSummaryHour {
			return fmt.Errorf("weekly summary hour must be between %d and %d", MinSummaryHour, MaxSummaryHour)
		}
		b.weeklySummaryHour = *weeklySummaryHour
	}

	if weeklySummaryWeekday != nil {
		if *weeklySummaryWeekday < MinWeekday || *weeklySummaryWeekday > MaxWeekday {
			return fmt.Errorf("weekly summary weekday must be between %d and %d", MinWeekday, MaxWeekday)
		}
		b.weeklySummaryWeekday = *weeklySummaryWeekday
	}

	if offlineCheckIntervalMinutes != nil {
		if *offlineCheckIntervalMinutes < MinOfflineCheckIntervalMinutes || *offlineCheckIntervalMinutes > MaxOfflineCheckIntervalMinutes {
			return fmt.Errorf("offline check interval must be between %d and %d minutes", MinOfflineCheckIntervalMinutes, MaxOfflineCheckIntervalMinutes)
		}
		b.offlineCheckIntervalMinutes = *offlineCheckIntervalMinutes
	}

	if notifyNodeOffline != nil {
		b.notifyNodeOffline = *notifyNodeOffline
	}
	if notifyAgentOffline != nil {
		b.notifyAgentOffline = *notifyAgentOffline
	}
	if notifyNewUser != nil {
		b.notifyNewUser = *notifyNewUser
	}
	if notifyPaymentSuccess != nil {
		b.notifyPaymentSuccess = *notifyPaymentSuccess
	}
	if notifyDailySummary != nil {
		b.notifyDailySummary = *notifyDailySummary
	}
	if notifyWeeklySummary != nil {
		b.notifyWeeklySummary = *notifyWeeklySummary
	}
	if notifyResourceExpiring != nil {
		b.notifyResourceExpiring = *notifyResourceExpiring
	}

	b.updatedAt = biztime.NowUTC()
	return nil
}

// CanNotifyNodeOffline checks if node offline notification can be sent (deduplication)
func (b *AdminTelegramBinding) CanNotifyNodeOffline() bool {
	if !b.notifyNodeOffline {
		return false
	}
	if b.lastNodeOfflineNotifyAt == nil {
		return true
	}
	return biztime.NowUTC().Sub(*b.lastNodeOfflineNotifyAt).Hours() >= NotifyWindowHours
}

// CanNotifyAgentOffline checks if agent offline notification can be sent (deduplication)
func (b *AdminTelegramBinding) CanNotifyAgentOffline() bool {
	if !b.notifyAgentOffline {
		return false
	}
	if b.lastAgentOfflineNotifyAt == nil {
		return true
	}
	return biztime.NowUTC().Sub(*b.lastAgentOfflineNotifyAt).Hours() >= NotifyWindowHours
}

// CanSendDailySummary checks if daily summary can be sent (once per business day).
// Uses calendar-based dedup: compares business dates instead of elapsed time
// to avoid timing races between cron trigger and recording.
func (b *AdminTelegramBinding) CanSendDailySummary() bool {
	if !b.notifyDailySummary {
		return false
	}
	if b.lastDailySummaryAt == nil {
		return true
	}
	lastBizDate := biztime.ToBizTimezone(*b.lastDailySummaryAt).Format("2006-01-02")
	todayBizDate := biztime.ToBizTimezone(biztime.NowUTC()).Format("2006-01-02")
	return lastBizDate != todayBizDate
}

// CanSendWeeklySummary checks if weekly summary can be sent (once per weekly period).
// Uses calendar-based dedup: the period starts at midnight of the configured weekday
// in business timezone. If last sent time is before the current period start, allow sending.
func (b *AdminTelegramBinding) CanSendWeeklySummary() bool {
	if !b.notifyWeeklySummary {
		return false
	}
	if b.lastWeeklySummaryAt == nil {
		return true
	}
	bizNow := biztime.ToBizTimezone(biztime.NowUTC())
	periodStart := mostRecentWeekdayMidnight(bizNow, b.weeklySummaryWeekday)
	lastSentBiz := biztime.ToBizTimezone(*b.lastWeeklySummaryAt)
	return lastSentBiz.Before(periodStart)
}

// RecordNodeOfflineNotification records that a node offline notification was sent
func (b *AdminTelegramBinding) RecordNodeOfflineNotification() {
	now := biztime.NowUTC()
	b.lastNodeOfflineNotifyAt = &now
	b.updatedAt = now
}

// RecordAgentOfflineNotification records that an agent offline notification was sent
func (b *AdminTelegramBinding) RecordAgentOfflineNotification() {
	now := biztime.NowUTC()
	b.lastAgentOfflineNotifyAt = &now
	b.updatedAt = now
}

// RecordDailySummary records that a daily summary was sent
func (b *AdminTelegramBinding) RecordDailySummary() {
	now := biztime.NowUTC()
	b.lastDailySummaryAt = &now
	b.updatedAt = now
}

// RecordWeeklySummary records that a weekly summary was sent
func (b *AdminTelegramBinding) RecordWeeklySummary() {
	now := biztime.NowUTC()
	b.lastWeeklySummaryAt = &now
	b.updatedAt = now
}

// CanNotifyResourceExpiring checks if resource expiring notification can be sent (daily deduplication)
// Uses date-based deduplication to ensure at most one notification per calendar day
func (b *AdminTelegramBinding) CanNotifyResourceExpiring() bool {
	if !b.notifyResourceExpiring {
		return false
	}
	if b.lastResourceExpiringNotifyDate == nil {
		return true
	}
	// Check if last notification was on a different date (using business timezone date comparison)
	now := biztime.NowUTC()
	lastBizDate := biztime.ToBizTimezone(*b.lastResourceExpiringNotifyDate).Format("2006-01-02")
	todayBizDate := biztime.ToBizTimezone(now).Format("2006-01-02")
	return lastBizDate != todayBizDate
}

// RecordResourceExpiringNotification records that a resource expiring notification was sent
func (b *AdminTelegramBinding) RecordResourceExpiringNotification() {
	now := biztime.NowUTC()
	b.lastResourceExpiringNotifyDate = &now
	b.updatedAt = now
}

// mostRecentWeekdayMidnight returns midnight of the most recent occurrence
// of the given weekday (0=Sunday..6=Saturday) in the same timezone as t.
// If today IS the target weekday, returns today's midnight.
func mostRecentWeekdayMidnight(t time.Time, targetWeekday int) time.Time {
	daysSince := int(t.Weekday()) - targetWeekday
	if daysSince < 0 {
		daysSince += 7
	}
	d := t.AddDate(0, 0, -daysSince)
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, t.Location())
}

// UpdateTelegramUsername updates the telegram username
func (b *AdminTelegramBinding) UpdateTelegramUsername(username string) {
	b.telegramUsername = username
	b.updatedAt = biztime.NowUTC()
}

// UpdateLanguage updates the user's preferred language
func (b *AdminTelegramBinding) UpdateLanguage(lang string) {
	b.language = lang
	b.updatedAt = biztime.NowUTC()
}
