package utils

import "strings"

// MaskEmail masks an email address for safe logging.
// Example: "user@example.com" -> "u***@example.com"
func MaskEmail(email string) string {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 {
		return "***"
	}
	local := parts[0]
	if len(local) <= 1 {
		return local + "***@" + parts[1]
	}
	return string(local[0]) + "***@" + parts[1]
}
