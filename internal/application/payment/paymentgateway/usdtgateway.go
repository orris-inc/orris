package paymentgateway

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/orris-inc/orris/internal/application/payment/exchangerate"
	"github.com/orris-inc/orris/internal/application/payment/suffixalloc"
	vo "github.com/orris-inc/orris/internal/domain/payment/valueobjects"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// USDTGatewayConfig holds the configuration for USDT gateway
type USDTGatewayConfig struct {
	POLReceivingAddresses []string
	TRCReceivingAddresses []string
	PaymentTTLMinutes     int
}

// USDTPaymentInfo contains the information needed to complete a USDT payment
type USDTPaymentInfo struct {
	ChainType        vo.ChainType
	USDTAmountRaw    uint64 // USDT amount in smallest unit (1 USDT = 1000000)
	ReceivingAddress string
	ExchangeRate     float64 // Exchange rate at time of payment (for display only)
	ExpiredAt        time.Time
	// Internal fields for suffix management
	BaseAmountRaw uint64 // Base amount in smallest unit before suffix
	Suffix        uint   // Allocated suffix
}

// USDTAmountFloat returns the USDT amount as float64 for display purposes
func (info *USDTPaymentInfo) USDTAmountFloat() float64 {
	return float64(info.USDTAmountRaw) / float64(suffixalloc.USDTUnit)
}

// USDTGateway handles USDT payment creation
type USDTGateway struct {
	exchangeService exchangerate.ExchangeRateService
	suffixAllocator suffixalloc.SuffixAllocator
	config          USDTGatewayConfig
	configMu        sync.RWMutex // Protects config for concurrent access
	logger          logger.Interface
}

// NewUSDTGateway creates a new USDT payment gateway
func NewUSDTGateway(
	exchangeService exchangerate.ExchangeRateService,
	suffixAllocator suffixalloc.SuffixAllocator,
	config USDTGatewayConfig,
	logger logger.Interface,
) *USDTGateway {
	return &USDTGateway{
		exchangeService: exchangeService,
		suffixAllocator: suffixAllocator,
		config:          config,
		logger:          logger,
	}
}

// Ensure USDTGateway implements PaymentGateway
var _ PaymentGateway = (*USDTGateway)(nil)

// CreatePayment creates a USDT payment (implements PaymentGateway interface)
func (g *USDTGateway) CreatePayment(ctx context.Context, req CreatePaymentRequest) (*CreatePaymentResponse, error) {
	// This method should not be used directly for USDT payments
	// Use CreateUSDTPayment instead
	return nil, fmt.Errorf("use CreateUSDTPayment for USDT payments")
}

// VerifyCallback verifies a payment callback (not used for USDT)
func (g *USDTGateway) VerifyCallback(req *http.Request) (*CallbackData, error) {
	// USDT payments don't use callbacks, they are monitored on-chain
	return nil, fmt.Errorf("USDT payments don't use callbacks")
}

// CreateUSDTPayment creates a new USDT payment
// cnyAmount is in cents (e.g., 7250 for 72.50 CNY)
func (g *USDTGateway) CreateUSDTPayment(ctx context.Context, paymentID uint, cnyAmount int64, paymentMethod vo.PaymentMethod) (*USDTPaymentInfo, error) {
	// Determine chain type from payment method
	chainType, err := vo.NewChainType(paymentMethod.ChainType())
	if err != nil {
		return nil, fmt.Errorf("invalid chain type for payment method: %w", err)
	}

	// Get receiving addresses for this chain (multi-wallet support)
	addresses := g.getReceivingAddresses(chainType)
	if len(addresses) == 0 {
		return nil, fmt.Errorf("no receiving addresses configured for %s", chainType)
	}

	// Validate all receiving addresses
	validAddresses := make([]string, 0, len(addresses))
	for _, addr := range addresses {
		if err := chainType.ValidateAddress(addr); err != nil {
			g.logger.Warnw("skipping invalid receiving address",
				"chain_type", chainType,
				"address", addr,
				"error", err,
			)
			continue
		}
		validAddresses = append(validAddresses, addr)
	}

	if len(validAddresses) == 0 {
		return nil, fmt.Errorf("no valid receiving addresses configured for %s", chainType)
	}

	// Convert CNY amount (in cents) to USDT (in smallest unit)
	usdtAmountRaw, err := g.exchangeService.ConvertCNYToUSDTRaw(ctx, cnyAmount)
	if err != nil {
		return nil, fmt.Errorf("failed to convert CNY to USDT: %w", err)
	}

	// Round down to nearest 0.01 USDT for base amount (to leave room for suffix)
	baseAmountRaw := suffixalloc.RoundToBase(usdtAmountRaw)

	// Get TTL from config (with locking)
	ttl := g.getPaymentTTL()

	// Allocate a unique suffix from the address pool
	allocation, err := g.suffixAllocator.Allocate(ctx, chainType, validAddresses, baseAmountRaw, paymentID, ttl)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate suffix: %w", err)
	}

	// Get exchange rate for reference (display only)
	rate, err := g.exchangeService.GetUSDTRate(ctx)
	if err != nil {
		// Don't fail if we can't get the rate again, we already have the amount
		g.logger.Warnw("failed to get exchange rate for record", "error", err)
		rate = 0
	}

	g.logger.Infow("created USDT payment",
		"payment_id", paymentID,
		"chain_type", chainType,
		"cny_amount_cents", cnyAmount,
		"usdt_amount_raw", allocation.FullAmountRaw,
		"usdt_amount", float64(allocation.FullAmountRaw)/float64(suffixalloc.USDTUnit),
		"receiving_address", allocation.ReceivingAddress,
		"exchange_rate", rate,
	)

	return &USDTPaymentInfo{
		ChainType:        chainType,
		USDTAmountRaw:    allocation.FullAmountRaw,
		ReceivingAddress: allocation.ReceivingAddress,
		ExchangeRate:     rate,
		ExpiredAt:        biztime.NowUTC().Add(ttl),
		BaseAmountRaw:    baseAmountRaw,
		Suffix:           allocation.Suffix,
	}, nil
}

// ReleaseSuffix releases an allocated suffix when payment creation fails
func (g *USDTGateway) ReleaseSuffix(ctx context.Context, info *USDTPaymentInfo) error {
	if info == nil {
		return nil
	}
	return g.suffixAllocator.Release(ctx, info.ChainType, info.ReceivingAddress, info.BaseAmountRaw, info.Suffix)
}

// getReceivingAddresses returns the receiving addresses for a chain type (multi-wallet support)
func (g *USDTGateway) getReceivingAddresses(chainType vo.ChainType) []string {
	g.configMu.RLock()
	defer g.configMu.RUnlock()

	switch chainType {
	case vo.ChainTypePOL:
		return g.config.POLReceivingAddresses
	case vo.ChainTypeTRC:
		return g.config.TRCReceivingAddresses
	default:
		return nil
	}
}

// getPaymentTTL returns the payment TTL from config
func (g *USDTGateway) getPaymentTTL() time.Duration {
	g.configMu.RLock()
	defer g.configMu.RUnlock()

	ttl := time.Duration(g.config.PaymentTTLMinutes) * time.Minute
	if ttl == 0 {
		ttl = 10 * time.Minute // Default 10 minutes per flow diagram
	}
	return ttl
}

// UpdateConfig updates the gateway configuration
func (g *USDTGateway) UpdateConfig(config USDTGatewayConfig) {
	g.configMu.Lock()
	defer g.configMu.Unlock()
	g.config = config
}
