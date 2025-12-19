package subscription

import (
	"errors"
	"fmt"
)

var (
	ErrSubscriptionNotFound    = errors.New("subscription not found")
	ErrSubscriptionExpired     = errors.New("subscription expired")
	ErrSubscriptionCancelled   = errors.New("subscription cancelled")
	ErrSubscriptionInactive    = errors.New("subscription inactive")
	ErrInvalidStatusTransition = errors.New("invalid status transition")
	ErrPlanNotFound            = errors.New("subscription plan not found")
	ErrPlanInactive            = errors.New("subscription plan inactive")
	ErrPlanSlugExists          = errors.New("plan slug already exists")
	ErrTokenNotFound           = errors.New("subscription token not found")
	ErrTokenExpired            = errors.New("subscription token expired")
	ErrTokenRevoked            = errors.New("subscription token revoked")
	ErrTokenInvalid            = errors.New("subscription token invalid")
	ErrUsageLimitExceeded      = errors.New("usage limit exceeded")
	ErrInvalidBillingCycle     = errors.New("invalid billing cycle")
	ErrInvalidPrice            = errors.New("invalid price")
)

func ErrInvalidTransition(from, to string) error {
	return fmt.Errorf("%w: from %s to %s", ErrInvalidStatusTransition, from, to)
}

func ErrLimitExceeded(limitType string, current, max uint64) error {
	return fmt.Errorf("%w: %s current=%d, max=%d", ErrUsageLimitExceeded, limitType, current, max)
}
