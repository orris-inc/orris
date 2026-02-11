package payment

import (
	"context"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/application/payment/blockchain"
	"github.com/orris-inc/orris/internal/application/payment/exchangerate"
	"github.com/orris-inc/orris/internal/application/payment/paymentgateway"
	"github.com/orris-inc/orris/internal/application/payment/suffixalloc"
	paymentUsecases "github.com/orris-inc/orris/internal/application/payment/usecases"
	"github.com/orris-inc/orris/internal/domain/payment"
	vo "github.com/orris-inc/orris/internal/domain/payment/valueobjects"
	"github.com/orris-inc/orris/internal/domain/subscription"
	infraBlockchain "github.com/orris-inc/orris/internal/infrastructure/blockchain"
	infraExchangerate "github.com/orris-inc/orris/internal/infrastructure/exchangerate"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/goroutine"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// USDTConfig holds all USDT-related configuration
type USDTConfig struct {
	Enabled               bool
	POLReceivingAddresses []string
	TRCReceivingAddresses []string
	PolygonScanAPIKey     string
	TronGridAPIKey        string
	PaymentTTLMinutes     int
	POLConfirmations      int
	TRCConfirmations      int
}

// USDTServiceManager manages all USDT-related services with hot-reload support
type USDTServiceManager struct {
	db               *gorm.DB
	paymentRepo      payment.PaymentRepository
	subscriptionRepo subscription.SubscriptionRepository
	logger           logger.Interface
	mu               sync.RWMutex

	// Current configuration
	config USDTConfig

	// Services
	exchangeService  exchangerate.ExchangeRateService
	suffixAllocator  suffixalloc.SuffixAllocator
	polygonMonitor   *infraBlockchain.PolygonMonitor
	tronMonitor      *infraBlockchain.TronMonitor
	compositeMonitor *infraBlockchain.CompositeMonitor
	usdtGateway      *paymentgateway.USDTGateway
	confirmUseCase   *paymentUsecases.ConfirmUSDTPaymentUseCase

	// Internal scheduler state
	stopChan         chan struct{}
	stopOnce         sync.Once
	wg               sync.WaitGroup
	running          bool
	cleanupInterval  time.Duration
	lastCleanup      time.Time
	cleanupRunning   bool
}

// NewUSDTServiceManager creates a new USDT service manager
func NewUSDTServiceManager(
	db *gorm.DB,
	paymentRepo payment.PaymentRepository,
	subscriptionRepo subscription.SubscriptionRepository,
	logger logger.Interface,
) *USDTServiceManager {
	return &USDTServiceManager{
		db:               db,
		paymentRepo:      paymentRepo,
		subscriptionRepo: subscriptionRepo,
		logger:           logger,
		stopChan:         make(chan struct{}),
		cleanupInterval:  5 * time.Minute,
	}
}

// Initialize initializes all USDT services with the given configuration
func (m *USDTServiceManager) Initialize(ctx context.Context, config USDTConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.config = config

	// Initialize exchange rate service (doesn't depend on config)
	m.exchangeService = infraExchangerate.NewCoinGeckoService(m.logger)

	// Initialize suffix allocator
	m.suffixAllocator = NewSuffixAllocator(m.db, m.logger)

	// Initialize blockchain monitors (always create, even without API key)
	// Without API key, requests will be rate-limited more aggressively
	m.polygonMonitor = infraBlockchain.NewPolygonMonitor(config.PolygonScanAPIKey, m.logger)
	m.tronMonitor = infraBlockchain.NewTronMonitor(config.TronGridAPIKey, m.logger)
	m.compositeMonitor = infraBlockchain.NewCompositeMonitor(m.polygonMonitor, m.tronMonitor, m.logger)

	// Log warning if API keys are not configured
	if config.PolygonScanAPIKey == "" {
		m.logger.Warnw("PolygonScan API key not configured, using conservative rate limiting (1 req/10s)")
	}
	if config.TronGridAPIKey == "" {
		m.logger.Warnw("TronGrid API key not configured, using conservative rate limiting (1 req/10s)")
	}

	// Initialize USDT gateway
	m.usdtGateway = paymentgateway.NewUSDTGateway(
		m.exchangeService,
		m.suffixAllocator,
		paymentgateway.USDTGatewayConfig{
			POLReceivingAddresses: config.POLReceivingAddresses,
			TRCReceivingAddresses: config.TRCReceivingAddresses,
			PaymentTTLMinutes:     config.PaymentTTLMinutes,
		},
		m.logger,
	)

	// Initialize confirm USDT payment use case
	m.confirmUseCase = paymentUsecases.NewConfirmUSDTPaymentUseCase(
		m.paymentRepo,
		m.subscriptionRepo,
		m.compositeMonitor,
		paymentUsecases.ConfirmUSDTPaymentConfig{
			RequiredConfirmationsPOL: config.POLConfirmations,
			RequiredConfirmationsTRC: config.TRCConfirmations,
		},
		m.logger,
	)

	m.logger.Infow("USDT services initialized",
		"enabled", config.Enabled,
		"pol_configured", len(config.POLReceivingAddresses) > 0 && config.PolygonScanAPIKey != "",
		"trc_configured", len(config.TRCReceivingAddresses) > 0 && config.TronGridAPIKey != "",
	)

	return nil
}

// OnSettingChange handles configuration changes (implements SettingChangeSubscriber)
func (m *USDTServiceManager) OnSettingChange(ctx context.Context, category string, changes map[string]any) error {
	if category != "usdt" {
		return nil
	}

	m.logger.Infow("USDT configuration changed, updating services")

	m.mu.Lock()
	defer m.mu.Unlock()

	// Update config from changes with validation
	if v, ok := changes["enabled"].(bool); ok {
		m.config.Enabled = v
	}
	if v, ok := changes["pol_receiving_addresses"].([]string); ok {
		// Validate all addresses
		validAddrs := make([]string, 0, len(v))
		for _, addr := range v {
			if addr == "" {
				continue
			}
			if err := vo.ChainTypePOL.ValidateAddress(addr); err != nil {
				m.logger.Warnw("skipping invalid POL receiving address in settings",
					"address", addr,
					"error", err,
				)
				continue
			}
			validAddrs = append(validAddrs, addr)
		}
		m.config.POLReceivingAddresses = validAddrs
	}
	if v, ok := changes["trc_receiving_addresses"].([]string); ok {
		// Validate all addresses
		validAddrs := make([]string, 0, len(v))
		for _, addr := range v {
			if addr == "" {
				continue
			}
			if err := vo.ChainTypeTRC.ValidateAddress(addr); err != nil {
				m.logger.Warnw("skipping invalid TRC receiving address in settings",
					"address", addr,
					"error", err,
				)
				continue
			}
			validAddrs = append(validAddrs, addr)
		}
		m.config.TRCReceivingAddresses = validAddrs
	}
	if v, ok := changes["polygonscan_api_key"].(string); ok {
		m.config.PolygonScanAPIKey = v
	}
	if v, ok := changes["trongrid_api_key"].(string); ok {
		m.config.TronGridAPIKey = v
	}
	if v, ok := changes["payment_ttl_minutes"].(int); ok {
		m.config.PaymentTTLMinutes = v
	}
	if v, ok := changes["pol_confirmations"].(int); ok {
		m.config.POLConfirmations = v
	}
	if v, ok := changes["trc_confirmations"].(int); ok {
		m.config.TRCConfirmations = v
	}

	// Update blockchain monitors (always update, even without API key)
	m.polygonMonitor = infraBlockchain.NewPolygonMonitor(m.config.PolygonScanAPIKey, m.logger)
	m.compositeMonitor.UpdatePolygonMonitor(m.polygonMonitor)
	m.tronMonitor = infraBlockchain.NewTronMonitor(m.config.TronGridAPIKey, m.logger)
	m.compositeMonitor.UpdateTronMonitor(m.tronMonitor)

	// Update USDT gateway config
	if m.usdtGateway != nil {
		m.usdtGateway.UpdateConfig(paymentgateway.USDTGatewayConfig{
			POLReceivingAddresses: m.config.POLReceivingAddresses,
			TRCReceivingAddresses: m.config.TRCReceivingAddresses,
			PaymentTTLMinutes:     m.config.PaymentTTLMinutes,
		})
	}

	// Update confirmation requirements
	if m.confirmUseCase != nil {
		m.confirmUseCase.UpdateConfig(paymentUsecases.ConfirmUSDTPaymentConfig{
			RequiredConfirmationsPOL: m.config.POLConfirmations,
			RequiredConfirmationsTRC: m.config.TRCConfirmations,
		})
	}

	m.logger.Infow("USDT services updated",
		"enabled", m.config.Enabled,
	)

	return nil
}

// IsEnabled returns whether USDT payments are enabled
func (m *USDTServiceManager) IsEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Enabled
}

// GetUSDTGateway returns the USDT gateway
func (m *USDTServiceManager) GetUSDTGateway() *paymentgateway.USDTGateway {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.usdtGateway
}

// GetExchangeService returns the exchange rate service
func (m *USDTServiceManager) GetExchangeService() exchangerate.ExchangeRateService {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.exchangeService
}

// GetSuffixAllocator returns the suffix allocator
func (m *USDTServiceManager) GetSuffixAllocator() suffixalloc.SuffixAllocator {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.suffixAllocator
}

// GetTransactionMonitor returns the composite transaction monitor
func (m *USDTServiceManager) GetTransactionMonitor() blockchain.TransactionMonitor {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.compositeMonitor
}

// GetConfirmUseCase returns the confirm USDT payment use case
func (m *USDTServiceManager) GetConfirmUseCase() *paymentUsecases.ConfirmUSDTPaymentUseCase {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.confirmUseCase
}

// StartScheduler starts the USDT monitor scheduler
func (m *USDTServiceManager) StartScheduler(ctx context.Context) {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return
	}
	m.running = true
	enabled := m.config.Enabled
	m.mu.Unlock()

	if !enabled {
		m.logger.Debugw("USDT scheduler not started (disabled)")
		return
	}

	m.logger.Infow("starting USDT monitor scheduler", "interval", "30s")

	m.wg.Add(1)
	goroutine.SafeGo(m.logger, "usdt-monitor-loop", func() {
		defer m.wg.Done()
		m.runMonitorLoop(ctx)
	})
}

// StopScheduler stops the USDT monitor scheduler
func (m *USDTServiceManager) StopScheduler() {
	m.stopOnce.Do(func() {
		m.mu.Lock()
		m.running = false
		m.mu.Unlock()

		m.logger.Infow("stopping USDT monitor scheduler")
		close(m.stopChan)
		m.wg.Wait()
		m.logger.Infow("USDT monitor scheduler stopped")
	})
}

// IsRunning returns whether the scheduler is running
func (m *USDTServiceManager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

func (m *USDTServiceManager) runMonitorLoop(ctx context.Context) {
	// Run immediately on startup
	m.processUSDTPayments(ctx)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			m.logger.Infow("USDT monitor scheduler stopped due to context cancellation")
			return
		case <-m.stopChan:
			m.logger.Infow("USDT monitor scheduler stopped")
			return
		case <-ticker.C:
			m.processUSDTPayments(ctx)
		}
	}
}

func (m *USDTServiceManager) processUSDTPayments(ctx context.Context) {
	m.mu.RLock()
	confirmUC := m.confirmUseCase
	allocator := m.suffixAllocator
	m.mu.RUnlock()

	if confirmUC == nil {
		return
	}

	m.logger.Debugw("checking USDT payments")

	startTime := biztime.NowUTC()
	results, err := confirmUC.Execute(ctx)
	if err != nil {
		m.logger.Errorw("failed to check USDT payments",
			"error", err,
			"duration", time.Since(startTime),
		)
		return
	}

	if len(results) > 0 {
		confirmedCount := 0
		pendingCount := 0
		activationFailedCount := 0
		for _, r := range results {
			if r.Confirmed {
				confirmedCount++
				if r.SubscriptionActivation == "failed" {
					activationFailedCount++
				}
				m.logger.Infow("USDT payment confirmed",
					"payment_id", r.PaymentID,
					"tx_hash", r.TxHash,
					"confirmations", r.Confirmations,
					"subscription_activated", r.SubscriptionActivated,
				)
			} else {
				pendingCount++
			}
		}

		if confirmedCount > 0 || pendingCount > 0 {
			m.logger.Infow("USDT payment check completed",
				"confirmed", confirmedCount,
				"pending_confirmations", pendingCount,
				"activation_failed", activationFailedCount,
				"duration", time.Since(startTime),
			)
		}
	}

	// Retry pending subscription activations
	m.retryPendingActivations(ctx, confirmUC)

	// Cleanup expired suffix allocations periodically
	m.cleanupExpiredSuffixes(ctx, allocator)
}

func (m *USDTServiceManager) retryPendingActivations(ctx context.Context, confirmUC *paymentUsecases.ConfirmUSDTPaymentUseCase) {
	successCount, err := confirmUC.RetryPendingSubscriptionActivations(ctx)
	if err != nil {
		m.logger.Warnw("failed to retry pending activations", "error", err)
		return
	}
	if successCount > 0 {
		m.logger.Infow("retried pending subscription activations", "success_count", successCount)
	}
}

func (m *USDTServiceManager) cleanupExpiredSuffixes(ctx context.Context, allocator suffixalloc.SuffixAllocator) {
	if allocator == nil {
		return
	}

	m.mu.Lock()
	// Only cleanup every cleanupInterval
	if time.Since(m.lastCleanup) < m.cleanupInterval {
		m.mu.Unlock()
		return
	}

	// Check if cleanup is already running to prevent concurrent executions
	if m.cleanupRunning {
		m.mu.Unlock()
		return
	}

	// Mark cleanup as started
	m.cleanupRunning = true
	m.lastCleanup = biztime.NowUTC()
	m.mu.Unlock()

	// Perform cleanup outside of lock to avoid blocking other operations
	if err := allocator.CleanupExpired(ctx); err != nil {
		m.logger.Warnw("failed to cleanup expired suffixes", "error", err)
	}

	// Mark cleanup as finished
	m.mu.Lock()
	m.cleanupRunning = false
	m.mu.Unlock()
}

// GetConfig returns the current USDT configuration
func (m *USDTServiceManager) GetConfig() USDTConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}
