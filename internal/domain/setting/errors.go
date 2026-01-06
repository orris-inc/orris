package setting

import "errors"

var (
	// ErrSettingNotFound is returned when a setting is not found
	ErrSettingNotFound = errors.New("setting not found")

	// ErrInvalidSettingKey is returned when the setting key is invalid
	ErrInvalidSettingKey = errors.New("invalid setting key")

	// ErrInvalidValueType is returned when the value type is invalid or mismatched
	ErrInvalidValueType = errors.New("invalid value type")
)
