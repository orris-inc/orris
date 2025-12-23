package telegram

import "errors"

var (
	// ErrBindingNotFound is returned when a telegram binding is not found
	ErrBindingNotFound = errors.New("telegram binding not found")
	// ErrAlreadyBound is returned when user already has a telegram binding
	ErrAlreadyBound = errors.New("user already has telegram binding")
	// ErrTelegramAlreadyUsed is returned when telegram account is already bound to another user
	ErrTelegramAlreadyUsed = errors.New("telegram account already bound to another user")
	// ErrInvalidVerifyCode is returned when verification code is invalid
	ErrInvalidVerifyCode = errors.New("invalid verification code")
	// ErrVerifyCodeExpired is returned when verification code has expired
	ErrVerifyCodeExpired = errors.New("verification code expired")
)
