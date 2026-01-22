package mappers

// Placeholder constants used for node entity reconstruction when actual values are not available.
// These placeholders are used when converting persistence models to domain value objects
// without subscription-derived credentials (UUID/password).
const (
	// PlaceholderUUID is used when UUID is required but not provided during model-to-VO conversion.
	PlaceholderUUID = "placeholder-uuid"

	// PlaceholderPassword is used when password is required but not provided during model-to-VO conversion.
	PlaceholderPassword = "placeholder"
)
