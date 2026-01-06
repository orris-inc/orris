package admin

import "errors"

// Domain errors for admin telegram binding
var (
	// ErrBindingNotFound is returned when an admin telegram binding is not found
	ErrBindingNotFound = errors.New("admin telegram binding not found")

	// ErrAlreadyBound is returned when user already has an admin telegram binding
	ErrAlreadyBound = errors.New("admin user already bound to telegram")

	// ErrTelegramAlreadyUsed is returned when telegram account is already used by another admin
	ErrTelegramAlreadyUsed = errors.New("telegram account already used by another admin")

	// ErrNotAdmin is returned when user is not an admin
	ErrNotAdmin = errors.New("user is not an admin")

	// ErrInvalidVerifyCode is returned when verify code is invalid or expired
	ErrInvalidVerifyCode = errors.New("invalid or expired verify code")
)
