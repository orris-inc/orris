package telegram

import (
	"strings"
	"unicode/utf8"
)

const maxMessageLength = 4096

// splitMessage splits a long message into chunks that fit within Telegram's limit.
// The limit is measured in Unicode characters (runes), matching Telegram's API behavior.
// Splits at paragraph boundaries (\n\n), then line boundaries (\n),
// then hard-cuts at a rune boundary as last resort.
func splitMessage(text string, limit int) []string {
	if limit <= 0 {
		limit = maxMessageLength
	}
	if utf8.RuneCountInString(text) <= limit {
		return []string{text}
	}

	var chunks []string
	for utf8.RuneCountInString(text) > limit {
		// Find the byte position corresponding to the rune limit
		byteLimit := runeByteOffset(text, limit)
		cut := byteLimit

		// Try paragraph boundary
		if idx := strings.LastIndex(text[:byteLimit], "\n\n"); idx > 0 {
			cut = idx + 2 // include the double newline in current chunk
		} else if idx := strings.LastIndex(text[:byteLimit], "\n"); idx > 0 {
			// Try line boundary
			cut = idx + 1 // include the newline in current chunk
		}

		chunks = append(chunks, text[:cut])
		text = text[cut:]
	}
	if len(text) > 0 {
		chunks = append(chunks, text)
	}
	return chunks
}

// runeByteOffset returns the byte offset of the n-th rune in s.
// If s has fewer than n runes, returns len(s).
func runeByteOffset(s string, n int) int {
	offset := 0
	for i := 0; i < n && offset < len(s); i++ {
		_, size := utf8.DecodeRuneInString(s[offset:])
		offset += size
	}
	return offset
}
