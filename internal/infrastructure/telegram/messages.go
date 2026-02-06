package telegram

import "html"

// EscapeHTML escapes HTML special characters for safe Telegram message formatting
func EscapeHTML(s string) string {
	return html.EscapeString(s)
}
