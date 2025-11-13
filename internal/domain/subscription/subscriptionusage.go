package subscription

import (
	"errors"
	"time"
)

var (
	ErrInvalidPeriod = errors.New("period cannot be zero")
)

type SubscriptionUsage struct {
	id             uint
	subscriptionID uint
	period         time.Time
	storageUsed    uint64
	usersCount     uint
	updatedAt      time.Time
}

func NewSubscriptionUsage(subscriptionID uint, period time.Time) (*SubscriptionUsage, error) {
	if subscriptionID == 0 {
		return nil, errors.New("subscription ID cannot be zero")
	}

	if period.IsZero() {
		return nil, ErrInvalidPeriod
	}

	return &SubscriptionUsage{
		subscriptionID: subscriptionID,
		period:         period,
		storageUsed:    0,
		usersCount:     0,
		updatedAt:      time.Now(),
	}, nil
}

func ReconstructSubscriptionUsage(
	id uint,
	subscriptionID uint,
	period time.Time,
	storageUsed uint64,
	usersCount uint,
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
		id:             id,
		subscriptionID: subscriptionID,
		period:         period,
		storageUsed:    storageUsed,
		usersCount:     usersCount,
		updatedAt:      updatedAt,
	}, nil
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

func (u *SubscriptionUsage) Reset() {
	u.storageUsed = 0
	u.usersCount = 0
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

func (u *SubscriptionUsage) StorageUsed() uint64 {
	return u.storageUsed
}

func (u *SubscriptionUsage) UsersCount() uint {
	return u.usersCount
}

func (u *SubscriptionUsage) UpdatedAt() time.Time {
	return u.updatedAt
}

func (u *SubscriptionUsage) HasUsage() bool {
	return u.storageUsed > 0 || u.usersCount > 0
}

func (u *SubscriptionUsage) SetID(id uint) error {
	if id == 0 {
		return errors.New("usage ID cannot be zero")
	}
	u.id = id
	return nil
}
