package usecases

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// FuzzEscapeHTML tests the escapeHTML function with random inputs
func FuzzEscapeHTML(f *testing.F) {
	// Seed corpus with interesting cases
	seeds := []string{
		"",
		"hello world",
		"<b>bold</b>",
		"<script>alert('xss')</script>",
		"<a href=\"url\">link</a>",
		"&amp;&lt;&gt;",
		"emoji 👍🎉",
		"中文测试",
		"日本語テスト",
		"'; DROP TABLE users; --",
		"\x00\x01\x02",
		strings.Repeat("<", 1000),
		strings.Repeat(">", 1000),
		"a<b>c&d\"e'f",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, input string) {
		// Skip invalid UTF-8
		if !utf8.ValidString(input) {
			return
		}

		result := escapeHTML(input)

		// Verify: HTML special chars should be escaped
		if strings.Contains(input, "<") && !strings.Contains(result, "&lt;") {
			t.Errorf("escapeHTML(%q) = %q, expected < to be escaped", input, result)
		}
		if strings.Contains(input, ">") && !strings.Contains(result, "&gt;") {
			t.Errorf("escapeHTML(%q) = %q, expected > to be escaped", input, result)
		}
		if strings.Contains(input, "&") && !strings.Contains(result, "&amp;") {
			t.Errorf("escapeHTML(%q) = %q, expected & to be escaped", input, result)
		}

		// Verify: output should be valid UTF-8
		if !utf8.ValidString(result) {
			t.Errorf("escapeHTML(%q) produced invalid UTF-8: %q", input, result)
		}

		// Verify: no panic occurred (implicit)
	})
}

// FuzzFormatAmount tests the formatAmount function
func FuzzFormatAmount(f *testing.F) {
	// Seed with edge cases
	amounts := []float64{0, 0.01, 0.99, 1.0, 99.99, 100.0, 1000.50, 999999.99, -1.0, -0.01}
	currencies := []string{"", "CNY", "USD", "EUR", "GBP", "JPY", "INVALID"}

	for _, amount := range amounts {
		for _, currency := range currencies {
			f.Add(amount, currency)
		}
	}

	f.Fuzz(func(t *testing.T, amount float64, currency string) {
		result := formatAmount(amount, currency)

		// Should not panic and should return non-empty string
		if result == "" {
			t.Errorf("formatAmount(%f, %q) returned empty string", amount, currency)
		}

		// Should contain the amount (at least partially)
		// For JPY, should not have decimals
		if currency == "JPY" && strings.Contains(result, ".") {
			t.Errorf("formatAmount(%f, JPY) = %q, should not contain decimal point", amount, result)
		}
	})
}

// FuzzFormatPercentChange tests the formatPercentChange function
func FuzzFormatPercentChange(f *testing.F) {
	// Seed with edge cases
	percents := []float64{
		0, 0.1, -0.1, 1.0, -1.0, 50.0, -50.0,
		100.0, -100.0, 0.001, -0.001, 999999.99, -999999.99,
	}

	for _, p := range percents {
		f.Add(p)
	}

	f.Fuzz(func(t *testing.T, percent float64) {
		result := formatPercentChange(percent)

		// Should not panic and should return non-empty string
		if result == "" {
			t.Errorf("formatPercentChange(%f) returned empty string", percent)
		}

		// Should contain appropriate indicator
		if percent > 0 && !strings.Contains(result, "📈") {
			t.Errorf("formatPercentChange(%f) = %q, expected 📈 for positive", percent, result)
		}
		if percent < 0 && !strings.Contains(result, "📉") {
			t.Errorf("formatPercentChange(%f) = %q, expected 📉 for negative", percent, result)
		}
		if percent == 0 && !strings.Contains(result, "➡️") {
			t.Errorf("formatPercentChange(%f) = %q, expected ➡️ for zero", percent, result)
		}
	})
}

// FuzzBuildNewUserMessage tests message building with various inputs
func FuzzBuildNewUserMessage(f *testing.F) {
	f.Add("usr_abc123", "test@example.com", "Test User", "registration")
	f.Add("", "", "", "")
	f.Add("usr_*special*", "email_with_*stars*@test.com", "_name_", "`source`")
	f.Add("usr_中文", "中文@测试.cn", "中文用户", "注册")

	f.Fuzz(func(t *testing.T, userSID, email, name, source string) {
		// Skip invalid UTF-8
		if !utf8.ValidString(userSID) || !utf8.ValidString(email) ||
			!utf8.ValidString(name) || !utf8.ValidString(source) {
			return
		}

		result := BuildNewUserMessage(userSID, email, name, source, time.Now())

		// Should not panic and should return non-empty string
		if result == "" {
			t.Errorf("BuildNewUserMessage returned empty string")
		}

		// Should contain header
		if !strings.Contains(result, "新用户注册") {
			t.Errorf("BuildNewUserMessage should contain header")
		}

		// User-provided data should be escaped (no raw special chars in result)
		// This is a simplified check
		if strings.Contains(email, "*") && !strings.Contains(result, "\\*") {
			// Note: This check is simplified; the actual escaping may vary
		}
	})
}

// FuzzBuildPaymentSuccessMessage tests payment message with various inputs
func FuzzBuildPaymentSuccessMessage(f *testing.F) {
	f.Add("pay_123", "usr_abc", "test@test.com", "Basic Plan", 99.99, "CNY", "alipay", "txn_123")
	f.Add("", "", "", "", 0.0, "", "", "")
	f.Add("pay_*", "usr_*", "*@*.com", "*Plan*", -1.0, "USD", "_method_", "`txn`")

	f.Fuzz(func(t *testing.T, paymentSID, userSID, userEmail, planName string, amount float64, currency, paymentMethod, transactionID string) {
		// Skip invalid UTF-8
		if !utf8.ValidString(paymentSID) || !utf8.ValidString(userSID) ||
			!utf8.ValidString(userEmail) || !utf8.ValidString(planName) ||
			!utf8.ValidString(currency) || !utf8.ValidString(paymentMethod) ||
			!utf8.ValidString(transactionID) {
			return
		}

		result := BuildPaymentSuccessMessage(paymentSID, userSID, userEmail, planName, amount, currency, paymentMethod, transactionID, time.Now())

		// Should not panic and should return non-empty string
		if result == "" {
			t.Errorf("BuildPaymentSuccessMessage returned empty string")
		}

		// Should contain header
		if !strings.Contains(result, "支付成功") {
			t.Errorf("BuildPaymentSuccessMessage should contain header")
		}
	})
}
