package subscription

import "time"

// SubscriptionCreatedEvent represents subscription creation
type SubscriptionCreatedEvent struct {
	SubscriptionID uint
	UserID         uint
	PlanID         uint
	Status         string
	StartDate      time.Time
	EndDate        time.Time
	Timestamp      time.Time
}

func NewSubscriptionCreatedEvent(subscriptionID, userID, planID uint, status string, startDate, endDate time.Time) *SubscriptionCreatedEvent {
	return &SubscriptionCreatedEvent{
		SubscriptionID: subscriptionID,
		UserID:         userID,
		PlanID:         planID,
		Status:         status,
		StartDate:      startDate,
		EndDate:        endDate,
		Timestamp:      time.Now(),
	}
}

func (e *SubscriptionCreatedEvent) GetEventType() string {
	return "subscription.created"
}

func (e *SubscriptionCreatedEvent) GetTimestamp() time.Time {
	return e.Timestamp
}

func (e *SubscriptionCreatedEvent) GetAggregateID() uint {
	return e.SubscriptionID
}

// SubscriptionActivatedEvent represents subscription activation
type SubscriptionActivatedEvent struct {
	SubscriptionID uint
	UserID         uint
	PlanID         uint
	ActivatedAt    time.Time
	Timestamp      time.Time
}

func NewSubscriptionActivatedEvent(subscriptionID, userID, planID uint, activatedAt time.Time) *SubscriptionActivatedEvent {
	return &SubscriptionActivatedEvent{
		SubscriptionID: subscriptionID,
		UserID:         userID,
		PlanID:         planID,
		ActivatedAt:    activatedAt,
		Timestamp:      time.Now(),
	}
}

func (e *SubscriptionActivatedEvent) GetEventType() string {
	return "subscription.activated"
}

func (e *SubscriptionActivatedEvent) GetTimestamp() time.Time {
	return e.Timestamp
}

func (e *SubscriptionActivatedEvent) GetAggregateID() uint {
	return e.SubscriptionID
}

// SubscriptionCancelledEvent represents subscription cancellation
type SubscriptionCancelledEvent struct {
	SubscriptionID uint
	UserID         uint
	PlanID         uint
	Reason         string
	CancelledAt    time.Time
	Timestamp      time.Time
}

func NewSubscriptionCancelledEvent(subscriptionID, userID, planID uint, reason string, cancelledAt time.Time) *SubscriptionCancelledEvent {
	return &SubscriptionCancelledEvent{
		SubscriptionID: subscriptionID,
		UserID:         userID,
		PlanID:         planID,
		Reason:         reason,
		CancelledAt:    cancelledAt,
		Timestamp:      time.Now(),
	}
}

func (e *SubscriptionCancelledEvent) GetEventType() string {
	return "subscription.cancelled"
}

func (e *SubscriptionCancelledEvent) GetTimestamp() time.Time {
	return e.Timestamp
}

func (e *SubscriptionCancelledEvent) GetAggregateID() uint {
	return e.SubscriptionID
}

// SubscriptionRenewedEvent represents subscription renewal
type SubscriptionRenewedEvent struct {
	SubscriptionID uint
	UserID         uint
	PlanID         uint
	OldEndDate     time.Time
	NewEndDate     time.Time
	RenewedAt      time.Time
	Timestamp      time.Time
}

func NewSubscriptionRenewedEvent(subscriptionID, userID, planID uint, oldEndDate, newEndDate, renewedAt time.Time) *SubscriptionRenewedEvent {
	return &SubscriptionRenewedEvent{
		SubscriptionID: subscriptionID,
		UserID:         userID,
		PlanID:         planID,
		OldEndDate:     oldEndDate,
		NewEndDate:     newEndDate,
		RenewedAt:      renewedAt,
		Timestamp:      time.Now(),
	}
}

func (e *SubscriptionRenewedEvent) GetEventType() string {
	return "subscription.renewed"
}

func (e *SubscriptionRenewedEvent) GetTimestamp() time.Time {
	return e.Timestamp
}

func (e *SubscriptionRenewedEvent) GetAggregateID() uint {
	return e.SubscriptionID
}

// SubscriptionPlanChangedEvent represents subscription plan change
type SubscriptionPlanChangedEvent struct {
	SubscriptionID uint
	UserID         uint
	OldPlanID      uint
	NewPlanID      uint
	ChangedAt      time.Time
	Timestamp      time.Time
}

func NewSubscriptionPlanChangedEvent(subscriptionID, userID, oldPlanID, newPlanID uint, changedAt time.Time) *SubscriptionPlanChangedEvent {
	return &SubscriptionPlanChangedEvent{
		SubscriptionID: subscriptionID,
		UserID:         userID,
		OldPlanID:      oldPlanID,
		NewPlanID:      newPlanID,
		ChangedAt:      changedAt,
		Timestamp:      time.Now(),
	}
}

func (e *SubscriptionPlanChangedEvent) GetEventType() string {
	return "subscription.plan_changed"
}

func (e *SubscriptionPlanChangedEvent) GetTimestamp() time.Time {
	return e.Timestamp
}

func (e *SubscriptionPlanChangedEvent) GetAggregateID() uint {
	return e.SubscriptionID
}

// SubscriptionExpiredEvent represents subscription expiration
type SubscriptionExpiredEvent struct {
	SubscriptionID uint
	UserID         uint
	PlanID         uint
	ExpiredAt      time.Time
	Timestamp      time.Time
}

func NewSubscriptionExpiredEvent(subscriptionID, userID, planID uint, expiredAt time.Time) *SubscriptionExpiredEvent {
	return &SubscriptionExpiredEvent{
		SubscriptionID: subscriptionID,
		UserID:         userID,
		PlanID:         planID,
		ExpiredAt:      expiredAt,
		Timestamp:      time.Now(),
	}
}

func (e *SubscriptionExpiredEvent) GetEventType() string {
	return "subscription.expired"
}

func (e *SubscriptionExpiredEvent) GetTimestamp() time.Time {
	return e.Timestamp
}

func (e *SubscriptionExpiredEvent) GetAggregateID() uint {
	return e.SubscriptionID
}

// SubscriptionTokenGeneratedEvent represents subscription token generation
type SubscriptionTokenGeneratedEvent struct {
	TokenID        uint
	SubscriptionID uint
	UserID         uint
	TokenType      string
	ExpiresAt      *time.Time
	Timestamp      time.Time
}

func NewSubscriptionTokenGeneratedEvent(tokenID, subscriptionID, userID uint, tokenType string, expiresAt *time.Time) *SubscriptionTokenGeneratedEvent {
	return &SubscriptionTokenGeneratedEvent{
		TokenID:        tokenID,
		SubscriptionID: subscriptionID,
		UserID:         userID,
		TokenType:      tokenType,
		ExpiresAt:      expiresAt,
		Timestamp:      time.Now(),
	}
}

func (e *SubscriptionTokenGeneratedEvent) GetEventType() string {
	return "subscription.token_generated"
}

func (e *SubscriptionTokenGeneratedEvent) GetTimestamp() time.Time {
	return e.Timestamp
}

func (e *SubscriptionTokenGeneratedEvent) GetAggregateID() uint {
	return e.SubscriptionID
}

// SubscriptionTokenRevokedEvent represents subscription token revocation
type SubscriptionTokenRevokedEvent struct {
	TokenID        uint
	SubscriptionID uint
	UserID         uint
	Reason         string
	RevokedAt      time.Time
	Timestamp      time.Time
}

func NewSubscriptionTokenRevokedEvent(tokenID, subscriptionID, userID uint, reason string, revokedAt time.Time) *SubscriptionTokenRevokedEvent {
	return &SubscriptionTokenRevokedEvent{
		TokenID:        tokenID,
		SubscriptionID: subscriptionID,
		UserID:         userID,
		Reason:         reason,
		RevokedAt:      revokedAt,
		Timestamp:      time.Now(),
	}
}

func (e *SubscriptionTokenRevokedEvent) GetEventType() string {
	return "subscription.token_revoked"
}

func (e *SubscriptionTokenRevokedEvent) GetTimestamp() time.Time {
	return e.Timestamp
}

func (e *SubscriptionTokenRevokedEvent) GetAggregateID() uint {
	return e.SubscriptionID
}

// SubscriptionTokenExpiredEvent represents subscription token expiration
type SubscriptionTokenExpiredEvent struct {
	TokenID        uint
	SubscriptionID uint
	UserID         uint
	ExpiredAt      time.Time
	Timestamp      time.Time
}

func NewSubscriptionTokenExpiredEvent(tokenID, subscriptionID, userID uint, expiredAt time.Time) *SubscriptionTokenExpiredEvent {
	return &SubscriptionTokenExpiredEvent{
		TokenID:        tokenID,
		SubscriptionID: subscriptionID,
		UserID:         userID,
		ExpiredAt:      expiredAt,
		Timestamp:      time.Now(),
	}
}

func (e *SubscriptionTokenExpiredEvent) GetEventType() string {
	return "subscription.token_expired"
}

func (e *SubscriptionTokenExpiredEvent) GetTimestamp() time.Time {
	return e.Timestamp
}

func (e *SubscriptionTokenExpiredEvent) GetAggregateID() uint {
	return e.SubscriptionID
}
