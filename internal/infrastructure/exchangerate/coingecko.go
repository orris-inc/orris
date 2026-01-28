package exchangerate

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/orris-inc/orris/internal/application/payment/exchangerate"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

const (
	// CoinGecko API endpoint for USDT price
	coingeckoAPIURL = "https://api.coingecko.com/api/v3/simple/price?ids=tether&vs_currencies=cny"
	// Cache duration for exchange rate
	cacheDuration = 5 * time.Minute
	// Maximum cache age for fallback (15 minutes)
	// If cache is older than this, we refuse to use it even if API fails
	// Shorter duration reduces risk of stale rates during price volatility
	maxCacheAge = 15 * time.Minute
	// HTTP request timeout
	requestTimeout = 10 * time.Second
	// Maximum response body size for exchange rate API (64KB)
	maxExchangeRateResponseSize = 64 << 10
	// Reasonable USDT/CNY rate range (USDT typically trades around 7.0-7.5 CNY)
	minReasonableRate = 5.0
	maxReasonableRate = 10.0
	// Maximum allowed rate change percentage (10%)
	maxRateChangePercent = 0.10
)

// coingeckoResponse represents the CoinGecko API response
type coingeckoResponse struct {
	Tether struct {
		CNY float64 `json:"cny"`
	} `json:"tether"`
}

// CoinGeckoService implements ExchangeRateService using CoinGecko API
type CoinGeckoService struct {
	httpClient *http.Client
	logger     logger.Interface

	// Cache
	mu         sync.RWMutex
	cachedRate float64
	cachedAt   time.Time
}

// NewCoinGeckoService creates a new CoinGecko exchange rate service
func NewCoinGeckoService(logger logger.Interface) *CoinGeckoService {
	return &CoinGeckoService{
		httpClient: &http.Client{
			Timeout: requestTimeout,
		},
		logger: logger,
	}
}

// Ensure CoinGeckoService implements ExchangeRateService
var _ exchangerate.ExchangeRateService = (*CoinGeckoService)(nil)

// GetUSDTRate returns the current USDT to CNY exchange rate
func (s *CoinGeckoService) GetUSDTRate(ctx context.Context) (float64, error) {
	now := biztime.NowUTC()

	// Check cache first
	s.mu.RLock()
	if s.cachedRate > 0 && now.Sub(s.cachedAt) < cacheDuration {
		rate := s.cachedRate
		s.mu.RUnlock()
		return rate, nil
	}
	cachedRateSnapshot := s.cachedRate
	s.mu.RUnlock()

	// Fetch fresh rate
	rate, err := s.fetchRate(ctx)
	if err != nil {
		// Return cached rate if available, but only if not too old
		s.mu.RLock()
		cacheAge := now.Sub(s.cachedAt)
		if s.cachedRate > 0 && cacheAge < maxCacheAge {
			cachedRate := s.cachedRate
			s.mu.RUnlock()
			s.logger.Warnw("failed to fetch exchange rate, using cached value",
				"error", err,
				"cached_rate", cachedRate,
				"cache_age", cacheAge,
			)
			return cachedRate, nil
		}
		s.mu.RUnlock()
		return 0, fmt.Errorf("failed to get USDT rate: %w", err)
	}

	// Rate change validation: reject if new rate differs from cached rate by more than maxRateChangePercent
	// Skip this check if there's no cached rate (first fetch)
	if cachedRateSnapshot > 0 {
		changePercent := math.Abs(rate-cachedRateSnapshot) / cachedRateSnapshot
		if changePercent > maxRateChangePercent {
			// Check if cache is still valid before falling back to it
			s.mu.RLock()
			cacheAge := now.Sub(s.cachedAt)
			s.mu.RUnlock()

			if cacheAge >= maxCacheAge {
				// Cache is too old, cannot use stale rate - reject to prevent incorrect pricing
				s.logger.Errorw("exchange rate change exceeds threshold and cache expired, refusing to use stale rate",
					"new_rate", rate,
					"cached_rate", cachedRateSnapshot,
					"change_percent", changePercent,
					"max_allowed_percent", maxRateChangePercent,
					"cache_age", cacheAge,
					"max_cache_age", maxCacheAge,
				)
				return 0, fmt.Errorf("rate change %.2f%% exceeds threshold and cache expired (age: %v)", changePercent*100, cacheAge)
			}

			s.logger.Warnw("exchange rate change exceeds threshold, using cached value",
				"new_rate", rate,
				"cached_rate", cachedRateSnapshot,
				"change_percent", changePercent,
				"max_allowed_percent", maxRateChangePercent,
				"cache_age", cacheAge,
			)
			// Return cached rate instead of the abnormal new rate
			return cachedRateSnapshot, nil
		}
	}

	// Update cache
	s.mu.Lock()
	s.cachedRate = rate
	s.cachedAt = now
	s.mu.Unlock()

	return rate, nil
}

// ConvertCNYToUSDTRaw converts a CNY amount (in cents) to USDT (in smallest unit)
// Input: cnyAmountCents - CNY amount in cents (e.g., 7250 for 72.50 CNY)
// Output: USDT amount in smallest unit (e.g., 10000000 for 10 USDT)
func (s *CoinGeckoService) ConvertCNYToUSDTRaw(ctx context.Context, cnyAmountCents int64) (uint64, error) {
	rate, err := s.GetUSDTRate(ctx)
	if err != nil {
		return 0, err
	}

	if rate <= 0 {
		return 0, fmt.Errorf("invalid exchange rate: %f", rate)
	}

	// Convert cents to CNY, divide by rate, then convert to smallest unit
	// CNY amount / rate = USDT amount
	// For example: 7250 cents = 72.50 CNY / 7.25 = 10 USDT = 10000000 units
	//
	// Formula: (cnyAmountCents / CNYUnit) / rate * USDTUnit
	// Rearranged to minimize floating point operations:
	// = cnyAmountCents * USDTUnit / (CNYUnit * rate)
	usdtRaw := float64(cnyAmountCents) * float64(exchangerate.USDTUnit) / (float64(exchangerate.CNYUnit) * rate)

	// Round to nearest integer (add 0.5 before truncating)
	return uint64(usdtRaw + 0.5), nil
}

// fetchRate fetches the current USDT/CNY rate from CoinGecko
func (s *CoinGeckoService) fetchRate(ctx context.Context) (float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, coingeckoAPIURL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch exchange rate: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var data coingeckoResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxExchangeRateResponseSize)).Decode(&data); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	if data.Tether.CNY <= 0 {
		return 0, fmt.Errorf("invalid rate from API: %f", data.Tether.CNY)
	}

	// Validate rate is within reasonable range
	if data.Tether.CNY < minReasonableRate || data.Tether.CNY > maxReasonableRate {
		return 0, fmt.Errorf("rate %f outside reasonable range [%f, %f]", data.Tether.CNY, minReasonableRate, maxReasonableRate)
	}

	s.logger.Infow("fetched USDT exchange rate",
		"rate", data.Tether.CNY,
	)

	return data.Tether.CNY, nil
}
