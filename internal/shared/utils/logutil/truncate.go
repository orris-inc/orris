package logutil

// TruncateForLog truncates a string to maxLen characters for safe logging.
// If the string is longer than maxLen, it appends "..." to indicate truncation.
// This is useful for logging sensitive data like tokens where only a prefix
// should be visible in logs.
func TruncateForLog(s string, maxLen int) string {
	if maxLen <= 0 {
		return "..."
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
