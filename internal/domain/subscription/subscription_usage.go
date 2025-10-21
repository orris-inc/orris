package subscription

import (
	"errors"
	"time"
)

var (
	ErrInvalidPeriod = errors.New("period cannot be zero")
)

type SubscriptionUsage struct {
	id                uint
	subscriptionID    uint
	period            time.Time
	apiRequests       uint64
	apiDataOut        uint64
	apiDataIn         uint64
	storageUsed       uint64
	usersCount        uint
	projectsCount     uint
	webhookCalls      uint64
	emailsSent        uint64
	reportsGenerated  uint
	updatedAt         time.Time
}

func NewSubscriptionUsage(subscriptionID uint, period time.Time) (*SubscriptionUsage, error) {
	if subscriptionID == 0 {
		return nil, errors.New("subscription ID cannot be zero")
	}

	if period.IsZero() {
		return nil, ErrInvalidPeriod
	}

	return &SubscriptionUsage{
		subscriptionID:   subscriptionID,
		period:           period,
		apiRequests:      0,
		apiDataOut:       0,
		apiDataIn:        0,
		storageUsed:      0,
		usersCount:       0,
		projectsCount:    0,
		webhookCalls:     0,
		emailsSent:       0,
		reportsGenerated: 0,
		updatedAt:        time.Now(),
	}, nil
}

func ReconstructSubscriptionUsage(
	id uint,
	subscriptionID uint,
	period time.Time,
	apiRequests uint64,
	apiDataOut uint64,
	apiDataIn uint64,
	storageUsed uint64,
	usersCount uint,
	projectsCount uint,
	webhookCalls uint64,
	emailsSent uint64,
	reportsGenerated uint,
	updatedAt time.Time,
) (*SubscriptionUsage, error) {
	if id == 0 {
		return nil, errors.New("usage ID cannot be zero")
	}

	if subscriptionID == 0 {
		return nil, errors.New("subscription ID cannot be zero")
	}

	if period.IsZero() {
		return nil, ErrInvalidPeriod
	}

	return &SubscriptionUsage{
		id:               id,
		subscriptionID:   subscriptionID,
		period:           period,
		apiRequests:      apiRequests,
		apiDataOut:       apiDataOut,
		apiDataIn:        apiDataIn,
		storageUsed:      storageUsed,
		usersCount:       usersCount,
		projectsCount:    projectsCount,
		webhookCalls:     webhookCalls,
		emailsSent:       emailsSent,
		reportsGenerated: reportsGenerated,
		updatedAt:        updatedAt,
	}, nil
}

func (u *SubscriptionUsage) IncrementAPIRequests(count uint64) {
	u.apiRequests += count
	u.updatedAt = time.Now()
}

func (u *SubscriptionUsage) IncrementAPIDataOut(bytes uint64) {
	u.apiDataOut += bytes
	u.updatedAt = time.Now()
}

func (u *SubscriptionUsage) IncrementAPIDataIn(bytes uint64) {
	u.apiDataIn += bytes
	u.updatedAt = time.Now()
}

func (u *SubscriptionUsage) IncrementStorageUsed(bytes uint64) {
	u.storageUsed += bytes
	u.updatedAt = time.Now()
}

func (u *SubscriptionUsage) DecrementStorageUsed(bytes uint64) {
	if bytes > u.storageUsed {
		u.storageUsed = 0
	} else {
		u.storageUsed -= bytes
	}
	u.updatedAt = time.Now()
}

func (u *SubscriptionUsage) IncrementUsersCount() {
	u.usersCount++
	u.updatedAt = time.Now()
}

func (u *SubscriptionUsage) DecrementUsersCount() {
	if u.usersCount > 0 {
		u.usersCount--
		u.updatedAt = time.Now()
	}
}

func (u *SubscriptionUsage) IncrementProjectsCount() {
	u.projectsCount++
	u.updatedAt = time.Now()
}

func (u *SubscriptionUsage) DecrementProjectsCount() {
	if u.projectsCount > 0 {
		u.projectsCount--
		u.updatedAt = time.Now()
	}
}

func (u *SubscriptionUsage) IncrementWebhookCalls(count uint64) {
	u.webhookCalls += count
	u.updatedAt = time.Now()
}

func (u *SubscriptionUsage) IncrementEmailsSent(count uint64) {
	u.emailsSent += count
	u.updatedAt = time.Now()
}

func (u *SubscriptionUsage) IncrementReportsGenerated() {
	u.reportsGenerated++
	u.updatedAt = time.Now()
}

func (u *SubscriptionUsage) Reset() {
	u.apiRequests = 0
	u.apiDataOut = 0
	u.apiDataIn = 0
	u.storageUsed = 0
	u.usersCount = 0
	u.projectsCount = 0
	u.webhookCalls = 0
	u.emailsSent = 0
	u.reportsGenerated = 0
	u.updatedAt = time.Now()
}

func (u *SubscriptionUsage) ID() uint {
	return u.id
}

func (u *SubscriptionUsage) SubscriptionID() uint {
	return u.subscriptionID
}

func (u *SubscriptionUsage) Period() time.Time {
	return u.period
}

func (u *SubscriptionUsage) APIRequests() uint64 {
	return u.apiRequests
}

func (u *SubscriptionUsage) APIDataOut() uint64 {
	return u.apiDataOut
}

func (u *SubscriptionUsage) APIDataIn() uint64 {
	return u.apiDataIn
}

func (u *SubscriptionUsage) StorageUsed() uint64 {
	return u.storageUsed
}

func (u *SubscriptionUsage) UsersCount() uint {
	return u.usersCount
}

func (u *SubscriptionUsage) ProjectsCount() uint {
	return u.projectsCount
}

func (u *SubscriptionUsage) WebhookCalls() uint64 {
	return u.webhookCalls
}

func (u *SubscriptionUsage) EmailsSent() uint64 {
	return u.emailsSent
}

func (u *SubscriptionUsage) ReportsGenerated() uint {
	return u.reportsGenerated
}

func (u *SubscriptionUsage) UpdatedAt() time.Time {
	return u.updatedAt
}

func (u *SubscriptionUsage) GetTotalAPIData() uint64 {
	return u.apiDataOut + u.apiDataIn
}

func (u *SubscriptionUsage) GetTotalActivity() uint64 {
	return u.apiRequests + u.webhookCalls + u.emailsSent + uint64(u.reportsGenerated)
}

func (u *SubscriptionUsage) HasUsage() bool {
	return u.apiRequests > 0 ||
		u.apiDataOut > 0 ||
		u.apiDataIn > 0 ||
		u.storageUsed > 0 ||
		u.usersCount > 0 ||
		u.projectsCount > 0 ||
		u.webhookCalls > 0 ||
		u.emailsSent > 0 ||
		u.reportsGenerated > 0
}

func (u *SubscriptionUsage) SetID(id uint) error {
	if id == 0 {
		return errors.New("usage ID cannot be zero")
	}
	u.id = id
	return nil
}
