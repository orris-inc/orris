package usecases

import (
	"fmt"
	"html"
)

// EscapeHTML escapes HTML special characters
// to prevent format injection from user-provided data
func EscapeHTML(s string) string {
	return html.EscapeString(s)
}

// FormatAmount formats amount to display string
// amount is in main currency unit (e.g., 99.00), not cents
func FormatAmount(amount float64, currency string) string {
	if currency == "" {
		currency = "CNY"
	}

	symbol := "¥"
	switch currency {
	case "USD":
		symbol = "$"
	case "EUR":
		symbol = "€"
	case "GBP":
		symbol = "£"
	case "JPY":
		return fmt.Sprintf("¥%.0f", amount) // JPY doesn't use decimals
	}

	return fmt.Sprintf("%s%.2f", symbol, amount)
}

// FormatBytes formats bytes into human readable format
// e.g., 1024 -> "1.00 KB", 1048576 -> "1.00 MB"
func FormatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatPercentChange formats percent change with emoji indicator
// e.g., 10.5 -> "📈 +10.5%", -5.2 -> "📉 -5.2%", 0 -> "➡️ 0%"
func FormatPercentChange(percent float64) string {
	if percent > 0 {
		return fmt.Sprintf("📈 +%.1f%%", percent)
	} else if percent < 0 {
		return fmt.Sprintf("📉 %.1f%%", percent)
	}
	return "➡️ 0%"
}

// FormatPercentChangeCompact formats percent change in compact inline format
// e.g., 10.5 -> "(📈+10.5%)", -5.2 -> "(📉-5.2%)", 0 -> "(--)"
func FormatPercentChangeCompact(percent float64) string {
	if percent == 0 {
		return "(--)"
	}
	if percent > 0 {
		return fmt.Sprintf("(📈+%.1f%%)", percent)
	}
	return fmt.Sprintf("(📉%.1f%%)", percent)
}

// CalculateChangePercent calculates the percentage change between two values
func CalculateChangePercent(current, previous int64) float64 {
	if previous == 0 {
		if current == 0 {
			return 0
		}
		return 100 // New from zero
	}
	return float64(current-previous) / float64(previous) * 100
}

// CalculateChangePercentUint64 calculates the percentage change for uint64 values
// Uses safe conversion to prevent integer overflow when values exceed math.MaxInt64
func CalculateChangePercentUint64(current, previous uint64) float64 {
	if previous == 0 {
		if current == 0 {
			return 0
		}
		return 100
	}
	// Safe conversion: use float64 directly to avoid int64 overflow
	// when current or previous exceeds math.MaxInt64
	return (float64(current) - float64(previous)) / float64(previous) * 100
}
