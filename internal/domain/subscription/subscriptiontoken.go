package subscription

import (
	"crypto/subtle"
	"errors"
	"time"

	vo "github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
)

var (
	ErrInvalidTokenHash = errors.New("token hash cannot be empty")
	ErrInvalidPrefix    = errors.New("token prefix cannot be empty")
)

type SubscriptionToken struct {
	id             uint
	sid            string // Stripe-style ID: stoken_xxx
	subscriptionID uint
	name           string
	tokenHash      string
	prefix         string
	scope          vo.TokenScope
	expiresAt      *time.Time
	lastUsedAt     *time.Time
	lastUsedIP     *string
	usageCount     uint64
	isActive       bool
	createdAt      time.Time
	revokedAt      *time.Time
}

func NewSubscriptionToken(
	subscriptionID uint,
	name string,
	tokenHash string,
	prefix string,
	scope vo.TokenScope,
	expiresAt *time.Time,
) (*SubscriptionToken, error) {
	if subscriptionID == 0 {
		return nil, errors.New("subscription ID cannot be zero")
	}

	if name == "" {
		return nil, errors.New("token name cannot be empty")
	}

	if tokenHash == "" {
		return nil, ErrInvalidTokenHash
	}

	if prefix == "" {
		return nil, ErrInvalidPrefix
	}

	if !scope.IsValid() {
		return nil, errors.New("invalid token scope")
	}

	if expiresAt != nil && expiresAt.Before(time.Now()) {
		return nil, errors.New("expiration time cannot be in the past")
	}

	return &SubscriptionToken{
		subscriptionID: subscriptionID,
		name:           name,
		tokenHash:      tokenHash,
		prefix:         prefix,
		scope:          scope,
		expiresAt:      expiresAt,
		isActive:       true,
		createdAt:      time.Now(),
		usageCount:     0,
	}, nil
}

func ReconstructSubscriptionToken(
	id uint,
	subscriptionID uint,
	name string,
	tokenHash string,
	prefix string,
	scope vo.TokenScope,
	expiresAt *time.Time,
	lastUsedAt *time.Time,
	lastUsedIP *string,
	usageCount uint64,
	isActive bool,
	createdAt time.Time,
	revokedAt *time.Time,
) (*SubscriptionToken, error) {
	if id == 0 {
		return nil, errors.New("token ID cannot be zero")
	}

	if subscriptionID == 0 {
		return nil, errors.New("subscription ID cannot be zero")
	}

	if tokenHash == "" {
		return nil, ErrInvalidTokenHash
	}

	if prefix == "" {
		return nil, ErrInvalidPrefix
	}

	return &SubscriptionToken{
		id:             id,
		subscriptionID: subscriptionID,
		name:           name,
		tokenHash:      tokenHash,
		prefix:         prefix,
		scope:          scope,
		expiresAt:      expiresAt,
		lastUsedAt:     lastUsedAt,
		lastUsedIP:     lastUsedIP,
		usageCount:     usageCount,
		isActive:       isActive,
		createdAt:      createdAt,
		revokedAt:      revokedAt,
	}, nil
}

func (t *SubscriptionToken) Verify(plainToken string) bool {
	return subtle.ConstantTimeCompare([]byte(t.tokenHash), []byte(plainToken)) == 1
}

func (t *SubscriptionToken) IsExpired() bool {
	if t.expiresAt == nil {
		return false
	}
	return time.Now().After(*t.expiresAt)
}

func (t *SubscriptionToken) Revoke() error {
	if t.revokedAt != nil {
		return errors.New("token is already revoked")
	}

	now := time.Now()
	t.revokedAt = &now
	t.isActive = false

	return nil
}

func (t *SubscriptionToken) RecordUsage(ipAddress string) {
	now := time.Now()
	t.lastUsedAt = &now
	t.lastUsedIP = &ipAddress
	t.usageCount++
}

func (t *SubscriptionToken) HasScope(scope string) bool {
	return t.scope.CanPerform(scope)
}

func (t *SubscriptionToken) IsValid() bool {
	return t.isActive && !t.IsExpired()
}

func (t *SubscriptionToken) ID() uint {
	return t.id
}

// SID returns the Stripe-style ID
func (t *SubscriptionToken) SID() string {
	return t.sid
}

// SetSID sets the Stripe-style ID (only for persistence layer use)
func (t *SubscriptionToken) SetSID(sid string) {
	t.sid = sid
}

func (t *SubscriptionToken) SubscriptionID() uint {
	return t.subscriptionID
}

func (t *SubscriptionToken) Name() string {
	return t.name
}

func (t *SubscriptionToken) TokenHash() string {
	return t.tokenHash
}

func (t *SubscriptionToken) Prefix() string {
	return t.prefix
}

func (t *SubscriptionToken) Scope() vo.TokenScope {
	return t.scope
}

func (t *SubscriptionToken) ExpiresAt() *time.Time {
	return t.expiresAt
}

func (t *SubscriptionToken) LastUsedAt() *time.Time {
	return t.lastUsedAt
}

func (t *SubscriptionToken) LastUsedIP() *string {
	return t.lastUsedIP
}

func (t *SubscriptionToken) UsageCount() uint64 {
	return t.usageCount
}

func (t *SubscriptionToken) IsActive() bool {
	return t.isActive
}

func (t *SubscriptionToken) CreatedAt() time.Time {
	return t.createdAt
}

func (t *SubscriptionToken) RevokedAt() *time.Time {
	return t.revokedAt
}

func (t *SubscriptionToken) SetID(id uint) error {
	if id == 0 {
		return errors.New("token ID cannot be zero")
	}
	t.id = id
	return nil
}
