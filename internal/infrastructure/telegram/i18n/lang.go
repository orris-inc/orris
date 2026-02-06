package i18n

import "strings"

// Lang represents a supported language
type Lang string

const (
	ZH Lang = "zh"
	EN Lang = "en"
)

// DetectLang detects language from Telegram's language_code field
func DetectLang(languageCode string) Lang {
	if strings.HasPrefix(languageCode, "zh") {
		return ZH
	}
	return EN
}

// ParseLang parses a stored language string into Lang, defaulting to ZH
func ParseLang(s string) Lang {
	if Lang(s) == EN {
		return EN
	}
	return ZH
}
