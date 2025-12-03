package utils

import (
	"fmt"
	"strings"

	vo "github.com/orris-inc/orris/internal/domain/subscription/value_objects"
)

// ValidateBillingCycle validates the billing cycle string and returns an error if invalid
func ValidateBillingCycle(cycle string) error {
	if strings.TrimSpace(cycle) == "" {
		return fmt.Errorf("billing cycle cannot be empty")
	}

	normalized := strings.ToLower(strings.TrimSpace(cycle))
	billingCycle := vo.BillingCycle(normalized)

	if !billingCycle.IsValid() {
		return fmt.Errorf("invalid billing cycle: %s", cycle)
	}

	return nil
}

// FormatPrice formats the price for display based on currency code
// The price parameter is expected to be in the smallest currency unit (cents)
// For example, 1000 would be $10.00, €10,00, etc.
func FormatPrice(price uint64, currency string) string {
	// Convert cents to decimal representation
	dollars := price / 100
	cents := price % 100

	// Define currency symbols and formatting rules
	currencyFormats := map[string]struct {
		symbol    string
		separator string
		position  string // "before" or "after"
	}{
		"CNY": {symbol: "¥", separator: "", position: "before"},
		"USD": {symbol: "$", separator: ".", position: "before"},
		"EUR": {symbol: "€", separator: ",", position: "after"},
		"GBP": {symbol: "£", separator: ".", position: "before"},
		"JPY": {symbol: "¥", separator: "", position: "before"},
	}

	format, exists := currencyFormats[currency]
	if !exists {
		// Default format if currency is not found
		return fmt.Sprintf("%s %.2f", currency, float64(price)/100.0)
	}

	var priceStr string
	if format.separator == "" {
		// For currencies without decimal separator (like JPY, CNY)
		priceStr = fmt.Sprintf("%d", dollars)
	} else {
		// For currencies with decimal separator
		priceStr = fmt.Sprintf("%d%s%02d", dollars, format.separator, cents)
	}

	if format.position == "before" {
		return fmt.Sprintf("%s%s", format.symbol, priceStr)
	}
	return fmt.Sprintf("%s%s", priceStr, format.symbol)
}

// CalculateSavingRate calculates the discount rate when purchasing a long-term plan
// monthlyPrice: the regular monthly price in smallest currency unit (cents)
// totalPrice: the total price for the entire period in smallest currency unit (cents)
// months: the number of months in the billing cycle
// Returns the saving rate as a percentage (0.0 to 100.0)
// For example, if monthlyPrice is 1000 (¥10) and totalPrice is 8000 (¥80) for 12 months,
// the regular price would be 12000 (¥120), so the saving rate is 33.33%
func CalculateSavingRate(monthlyPrice, totalPrice uint64, months int) float32 {
	if months <= 0 {
		return 0
	}

	if monthlyPrice == 0 {
		return 0
	}

	// Calculate what the total price would be at monthly rate
	expectedTotalPrice := monthlyPrice * uint64(months)

	// Avoid division by zero
	if expectedTotalPrice == 0 {
		return 0
	}

	// Calculate the savings
	savings := expectedTotalPrice - totalPrice

	// Calculate the saving rate as percentage
	savingRate := (float32(savings) / float32(expectedTotalPrice)) * 100.0

	// Ensure the result is between 0 and 100
	if savingRate < 0 {
		return 0
	}
	if savingRate > 100 {
		return 100
	}

	return savingRate
}
