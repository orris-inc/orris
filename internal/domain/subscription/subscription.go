// Package subscription provides domain models and business logic for subscription management.
// It includes the Subscription aggregate root, subscription plans, tokens, and usage tracking.
package subscription

import (
	"fmt"
	"time"

	vo "github.com/orris-inc/orris/internal/domain/subscription/valueobjects"

	"github.com/google/uuid"
)

// Subscription represents the subscription aggregate root
type Subscription struct {
	id                 uint
	uuid               string // unique identifier used for node authentication
	userID             uint   // Deprecated: use subjectID instead. Kept for backward compatibility
	subjectType        string // Type of subject (user, user_group, etc.)
	subjectID          uint   // ID of the subject
	planID             uint
	status             vo.SubscriptionStatus
	startDate          time.Time
	endDate            time.Time
	autoRenew          bool
	currentPeriodStart time.Time
	currentPeriodEnd   time.Time
	cancelledAt        *time.Time
	cancelReason       *string
	metadata           map[string]interface{}
	version            int
	createdAt          time.Time
	updatedAt          time.Time
}

// NewSubscription creates a new subscription
// For backward compatibility, this defaults to user subject type
func NewSubscription(userID, planID uint, startDate, endDate time.Time, autoRenew bool) (*Subscription, error) {
	return NewSubscriptionWithSubject("user", userID, planID, startDate, endDate, autoRenew)
}

// NewSubscriptionWithSubject creates a new subscription with specific subject type
func NewSubscriptionWithSubject(subjectType string, subjectID, planID uint, startDate, endDate time.Time, autoRenew bool) (*Subscription, error) {
	if subjectType == "" {
		return nil, fmt.Errorf("subject type is required")
	}
	if subjectID == 0 {
		return nil, fmt.Errorf("subject ID is required")
	}
	if planID == 0 {
		return nil, fmt.Errorf("plan ID is required")
	}
	if endDate.Before(startDate) {
		return nil, fmt.Errorf("end date must be after start date")
	}

	// Generate unique UUID for this subscription
	subscriptionUUID := uuid.New().String()

	now := time.Now()
	s := &Subscription{
		uuid:               subscriptionUUID,
		userID:             subjectID, // For backward compatibility
		subjectType:        subjectType,
		subjectID:          subjectID,
		planID:             planID,
		status:             vo.StatusInactive,
		startDate:          startDate,
		endDate:            endDate,
		autoRenew:          autoRenew,
		currentPeriodStart: startDate,
		currentPeriodEnd:   endDate,
		metadata:           make(map[string]interface{}),
		version:            1,
		createdAt:          now,
		updatedAt:          now,
	}

	return s, nil
}

// ReconstructSubscription reconstructs a subscription from persistence
// Deprecated: Use ReconstructSubscriptionWithSubject for new code
func ReconstructSubscription(
	id, userID, planID uint,
	uuid string,
	status vo.SubscriptionStatus,
	startDate, endDate time.Time,
	autoRenew bool,
	currentPeriodStart, currentPeriodEnd time.Time,
	cancelledAt *time.Time,
	cancelReason *string,
	metadata map[string]interface{},
	version int,
	createdAt, updatedAt time.Time,
) (*Subscription, error) {
	return ReconstructSubscriptionWithSubject(
		id, userID, planID,
		"user", userID, // Default to user subject type
		uuid,
		status,
		startDate, endDate,
		autoRenew,
		currentPeriodStart, currentPeriodEnd,
		cancelledAt,
		cancelReason,
		metadata,
		version,
		createdAt, updatedAt,
	)
}

// ReconstructSubscriptionWithSubject reconstructs a subscription from persistence with subject fields
func ReconstructSubscriptionWithSubject(
	id, userID, planID uint,
	subjectType string,
	subjectID uint,
	uuid string,
	status vo.SubscriptionStatus,
	startDate, endDate time.Time,
	autoRenew bool,
	currentPeriodStart, currentPeriodEnd time.Time,
	cancelledAt *time.Time,
	cancelReason *string,
	metadata map[string]interface{},
	version int,
	createdAt, updatedAt time.Time,
) (*Subscription, error) {
	if id == 0 {
		return nil, fmt.Errorf("subscription ID cannot be zero")
	}
	if uuid == "" {
		return nil, fmt.Errorf("subscription UUID is required")
	}
	if subjectType == "" {
		return nil, fmt.Errorf("subject type is required")
	}
	if subjectID == 0 {
		return nil, fmt.Errorf("subject ID is required")
	}
	if planID == 0 {
		return nil, fmt.Errorf("plan ID is required")
	}
	if !vo.ValidStatuses[status] {
		return nil, fmt.Errorf("invalid subscription status: %s", status)
	}

	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	return &Subscription{
		id:                 id,
		uuid:               uuid,
		userID:             userID,
		subjectType:        subjectType,
		subjectID:          subjectID,
		planID:             planID,
		status:             status,
		startDate:          startDate,
		endDate:            endDate,
		autoRenew:          autoRenew,
		currentPeriodStart: currentPeriodStart,
		currentPeriodEnd:   currentPeriodEnd,
		cancelledAt:        cancelledAt,
		cancelReason:       cancelReason,
		metadata:           metadata,
		version:            version,
		createdAt:          createdAt,
		updatedAt:          updatedAt,
	}, nil
}

// ID returns the subscription ID
func (s *Subscription) ID() uint {
	return s.id
}

// UUID returns the subscription UUID
func (s *Subscription) UUID() string {
	return s.uuid
}

// UserID returns the user ID
// Deprecated: use SubjectID instead
func (s *Subscription) UserID() uint {
	return s.userID
}

// SubjectType returns the subject type
func (s *Subscription) SubjectType() string {
	return s.subjectType
}

// SubjectID returns the subject ID
func (s *Subscription) SubjectID() uint {
	return s.subjectID
}

// PlanID returns the plan ID
func (s *Subscription) PlanID() uint {
	return s.planID
}

// Status returns the subscription status
func (s *Subscription) Status() vo.SubscriptionStatus {
	return s.status
}

// StartDate returns the subscription start date
func (s *Subscription) StartDate() time.Time {
	return s.startDate
}

// EndDate returns the subscription end date
func (s *Subscription) EndDate() time.Time {
	return s.endDate
}

// AutoRenew returns the auto-renew setting
func (s *Subscription) AutoRenew() bool {
	return s.autoRenew
}

// CurrentPeriodStart returns the current period start date
func (s *Subscription) CurrentPeriodStart() time.Time {
	return s.currentPeriodStart
}

// CurrentPeriodEnd returns the current period end date
func (s *Subscription) CurrentPeriodEnd() time.Time {
	return s.currentPeriodEnd
}

// CancelledAt returns when the subscription was cancelled
func (s *Subscription) CancelledAt() *time.Time {
	return s.cancelledAt
}

// CancelReason returns the cancellation reason
func (s *Subscription) CancelReason() *string {
	return s.cancelReason
}

// Metadata returns the subscription metadata
func (s *Subscription) Metadata() map[string]interface{} {
	return s.metadata
}

// Version returns the aggregate version for optimistic locking
func (s *Subscription) Version() int {
	return s.version
}

// CreatedAt returns when the subscription was created
func (s *Subscription) CreatedAt() time.Time {
	return s.createdAt
}

// UpdatedAt returns when the subscription was last updated
func (s *Subscription) UpdatedAt() time.Time {
	return s.updatedAt
}

// SetID sets the subscription ID (only for persistence layer use)
func (s *Subscription) SetID(id uint) error {
	if s.id != 0 {
		return fmt.Errorf("subscription ID is already set")
	}
	if id == 0 {
		return fmt.Errorf("subscription ID cannot be zero")
	}
	s.id = id
	return nil
}

// Activate activates a subscription
func (s *Subscription) Activate() error {
	if s.status == vo.StatusActive {
		return nil
	}

	if s.status != vo.StatusInactive && s.status != vo.StatusPendingPayment && s.status != vo.StatusTrialing {
		return fmt.Errorf("cannot activate subscription with status %s", s.status)
	}

	if !s.status.CanTransitionTo(vo.StatusActive) {
		return fmt.Errorf("invalid status transition from %s to active", s.status)
	}

	s.status = vo.StatusActive
	s.updatedAt = time.Now()
	s.version++

	return nil
}

// Cancel cancels a subscription with a reason
func (s *Subscription) Cancel(reason string) error {
	if s.status == vo.StatusCancelled {
		return nil
	}

	if s.status != vo.StatusActive && s.status != vo.StatusTrialing {
		return fmt.Errorf("cannot cancel subscription with status %s", s.status)
	}

	if !s.status.CanTransitionTo(vo.StatusCancelled) {
		return fmt.Errorf("invalid status transition from %s to cancelled", s.status)
	}

	if reason == "" {
		return fmt.Errorf("cancel reason is required")
	}

	now := time.Now()
	s.status = vo.StatusCancelled
	s.cancelledAt = &now
	s.cancelReason = &reason
	s.updatedAt = now
	s.version++

	return nil
}

// Renew renews a subscription to a new end date
func (s *Subscription) Renew(endDate time.Time) error {
	if !s.status.CanRenew() {
		return fmt.Errorf("cannot renew subscription with status %s", s.status)
	}

	if endDate.Before(s.endDate) {
		return fmt.Errorf("new end date must be after current end date")
	}

	s.endDate = endDate
	s.currentPeriodStart = s.currentPeriodEnd
	s.currentPeriodEnd = endDate
	s.updatedAt = time.Now()
	s.version++

	if s.status == vo.StatusExpired {
		s.status = vo.StatusActive
	}

	return nil
}

func (s *Subscription) ChangePlan(newPlanID uint) error {
	if newPlanID == 0 {
		return fmt.Errorf("new plan ID is required")
	}

	if newPlanID == s.planID {
		return nil
	}

	if s.status != vo.StatusActive && s.status != vo.StatusTrialing {
		return fmt.Errorf("cannot change plan for subscription with status %s", s.status)
	}

	s.planID = newPlanID
	s.updatedAt = time.Now()
	s.version++

	return nil
}

// IsExpired checks if subscription is expired
func (s *Subscription) IsExpired() bool {
	return time.Now().After(s.endDate)
}

// IsActive checks if subscription is active and can be used
func (s *Subscription) IsActive() bool {
	return s.status.CanUseService() && !s.IsExpired()
}

// MarkAsExpired marks subscription as expired
func (s *Subscription) MarkAsExpired() error {
	if s.status == vo.StatusExpired {
		return nil
	}

	if !s.status.CanTransitionTo(vo.StatusExpired) {
		return fmt.Errorf("cannot mark subscription as expired with status %s", s.status)
	}

	s.status = vo.StatusExpired
	s.updatedAt = time.Now()
	s.version++

	return nil
}

// SetAutoRenew updates auto-renew setting
func (s *Subscription) SetAutoRenew(autoRenew bool) {
	if s.autoRenew == autoRenew {
		return
	}

	s.autoRenew = autoRenew
	s.updatedAt = time.Now()
	s.version++
}

// ResetUUID generates a new UUID for the subscription link
// This invalidates the old subscription link
func (s *Subscription) ResetUUID() {
	s.uuid = uuid.New().String()
	s.updatedAt = time.Now()
	s.version++
}

// UpdateCurrentPeriod updates the current billing period
func (s *Subscription) UpdateCurrentPeriod(start, end time.Time) error {
	if end.Before(start) {
		return fmt.Errorf("period end must be after period start")
	}

	s.currentPeriodStart = start
	s.currentPeriodEnd = end
	s.updatedAt = time.Now()
	s.version++

	return nil
}

// Validate performs domain-level validation
func (s *Subscription) Validate() error {
	if s.subjectType == "" {
		return fmt.Errorf("subject type is required")
	}
	if s.subjectID == 0 {
		return fmt.Errorf("subject ID is required")
	}
	if s.planID == 0 {
		return fmt.Errorf("plan ID is required")
	}
	if !vo.ValidStatuses[s.status] {
		return fmt.Errorf("invalid status: %s", s.status)
	}
	if s.endDate.Before(s.startDate) {
		return fmt.Errorf("end date must be after start date")
	}
	if s.currentPeriodEnd.Before(s.currentPeriodStart) {
		return fmt.Errorf("current period end must be after current period start")
	}
	return nil
}
