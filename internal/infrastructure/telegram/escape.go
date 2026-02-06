package telegram

import "strings"

// EscapeMarkdownV1 escapes Telegram Markdown V1 special characters.
// Special chars: _ * ` [
func EscapeMarkdownV1(s string) string {
	replacer := strings.NewReplacer(
		`_`, `\_`,
		`*`, `\*`,
		"`", "\\`",
		`[`, `\[`,
	)
	return replacer.Replace(s)
}
