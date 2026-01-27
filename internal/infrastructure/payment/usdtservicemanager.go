package payment

import (
	"context"
	"sync"

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
	"github.com/orris-inc/orris/internal/infrastructure/scheduler"
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
	monitorScheduler *scheduler.USDTMonitorScheduler
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

	// Initialize monitor scheduler
	m.monitorScheduler = scheduler.NewUSDTMonitorScheduler(m.confirmUseCase, m.logger)
	m.monitorScheduler.SetSuffixAllocator(m.suffixAllocator)

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
	m.mu.RLock()
	scheduler := m.monitorScheduler
	enabled := m.config.Enabled
	m.mu.RUnlock()

	if scheduler != nil && enabled {
		scheduler.Start(ctx)
	}
}

// StopScheduler stops the USDT monitor scheduler
func (m *USDTServiceManager) StopScheduler() {
	m.mu.RLock()
	scheduler := m.monitorScheduler
	m.mu.RUnlock()

	if scheduler != nil {
		scheduler.Stop()
	}
}

// GetConfig returns the current USDT configuration
func (m *USDTServiceManager) GetConfig() USDTConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}
