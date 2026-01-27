package exchangerate

import "context"

const (
	// USDTUnit is the multiplier to convert USDT to smallest unit (1 USDT = 1000000 units)
	USDTUnit = 1000000
	// CNYUnit is the multiplier to convert CNY to cents (1 CNY = 100 cents)
	CNYUnit = 100
)

// ExchangeRateService provides currency exchange rate functionality
type ExchangeRateService interface {
	// GetUSDTRate returns the current USDT to CNY exchange rate (1 USDT = X CNY)
	// The rate is returned as float64 since it's an external API value
	GetUSDTRate(ctx context.Context) (float64, error)

	// ConvertCNYToUSDTRaw converts a CNY amount (in cents) to USDT (in smallest unit)
	// Input: cnyAmountCents - CNY amount in cents (e.g., 7250 for 72.50 CNY)
	// Output: USDT amount in smallest unit (e.g., 10000000 for 10 USDT)
	ConvertCNYToUSDTRaw(ctx context.Context, cnyAmountCents int64) (uint64, error)
}
