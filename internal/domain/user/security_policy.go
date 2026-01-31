package user

import "time"

// SecurityPolicy defines security-related configuration for user operations
type SecurityPolicy struct {
	MaxLoginAttempts       int
	LockoutDurationMinutes int
}

// DefaultSecurityPolicy returns the default security policy
func DefaultSecurityPolicy() *SecurityPolicy {
	return &SecurityPolicy{
		MaxLoginAttempts:       5,
		LockoutDurationMinutes: 15,
	}
}

// LockoutDuration returns the lockout duration as time.Duration
func (p *SecurityPolicy) LockoutDuration() time.Duration {
	return time.Duration(p.LockoutDurationMinutes) * time.Minute
}
