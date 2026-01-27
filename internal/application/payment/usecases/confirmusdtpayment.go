package usecases

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/orris-inc/orris/internal/application/payment/blockchain"
	"github.com/orris-inc/orris/internal/domain/payment"
	vo "github.com/orris-inc/orris/internal/domain/payment/valueobjects"
	"github.com/orris-inc/orris/internal/domain/subscription"
	subscriptionVO "github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// ConfirmUSDTPaymentUseCase handles the confirmation of USDT payments
type ConfirmUSDTPaymentUseCase struct {
	paymentRepo         payment.PaymentRepository
	subscriptionRepo    subscription.SubscriptionRepository
	transactionMonitor  blockchain.TransactionMonitor
	requiredConfirmsPOL int
	requiredConfirmsTRC int
	requestDelay        time.Duration // Delay between API requests to avoid rate limiting
	configMu            sync.RWMutex  // Protects requiredConfirms* and requestDelay fields
	executeMu           sync.Mutex    // Prevents concurrent Execute calls to avoid double confirmation
	logger              logger.Interface
}

// ConfirmUSDTPaymentConfig holds configuration for USDT payment confirmation
type ConfirmUSDTPaymentConfig struct {
	RequiredConfirmationsPOL int
	RequiredConfirmationsTRC int
}

const (
	// Default confirmation requirements
	defaultConfirmsPOL = 12
	defaultConfirmsTRC = 19
	// Maximum allowed confirmations (prevent misconfiguration)
	maxConfirmations = 100
	// Grace period after payment expiration to still accept transactions
	// This accounts for blockchain confirmation delays
	expirationGracePeriod = 1 * time.Hour

	// Request delay between payment checks
	// Use conservative 10s delay to avoid API rate limiting
	requestDelay = 10 * time.Second
)

// NewConfirmUSDTPaymentUseCase creates a new ConfirmUSDTPaymentUseCase
func NewConfirmUSDTPaymentUseCase(
	paymentRepo payment.PaymentRepository,
	subscriptionRepo subscription.SubscriptionRepository,
	transactionMonitor blockchain.TransactionMonitor,
	config ConfirmUSDTPaymentConfig,
	logger logger.Interface,
) *ConfirmUSDTPaymentUseCase {
	requiredConfirmsPOL := validateConfirmations(config.RequiredConfirmationsPOL, defaultConfirmsPOL)
	requiredConfirmsTRC := validateConfirmations(config.RequiredConfirmationsTRC, defaultConfirmsTRC)

	return &ConfirmUSDTPaymentUseCase{
		paymentRepo:         paymentRepo,
		subscriptionRepo:    subscriptionRepo,
		transactionMonitor:  transactionMonitor,
		requiredConfirmsPOL: requiredConfirmsPOL,
		requiredConfirmsTRC: requiredConfirmsTRC,
		requestDelay:        requestDelay,
		logger:              logger,
	}
}

// validateConfirmations validates and normalizes confirmation count
// Returns defaultVal if value is <= 0, caps at maxConfirmations if too high
func validateConfirmations(value, defaultVal int) int {
	if value <= 0 {
		return defaultVal
	}
	if value > maxConfirmations {
		return maxConfirmations
	}
	return value
}

// ConfirmUSDTPaymentResult contains the result of a payment confirmation check
type ConfirmUSDTPaymentResult struct {
	PaymentID              uint
	Confirmed              bool
	TxHash                 string
	Confirmations          int
	SubscriptionActivated  bool
	SubscriptionActivation string // "success", "failed", "skipped"
}

// Execute checks and confirms pending USDT payments
// Uses mutex to prevent concurrent execution and avoid double confirmation
func (uc *ConfirmUSDTPaymentUseCase) Execute(ctx context.Context) ([]ConfirmUSDTPaymentResult, error) {
	// Prevent concurrent execution to avoid double confirmation of the same payment
	uc.executeMu.Lock()
	defer uc.executeMu.Unlock()

	// Get all pending USDT payments
	pendingPayments, err := uc.paymentRepo.GetPendingUSDTPayments(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending USDT payments: %w", err)
	}

	if len(pendingPayments) == 0 {
		return nil, nil
	}

	uc.configMu.RLock()
	delay := uc.requestDelay
	uc.configMu.RUnlock()

	uc.logger.Infow("checking pending USDT payments",
		"count", len(pendingPayments),
		"request_delay", delay,
	)

	var results []ConfirmUSDTPaymentResult

	for i, p := range pendingPayments {
		result, err := uc.checkPayment(ctx, p)
		if err != nil {
			uc.logger.Warnw("failed to check payment",
				"payment_id", p.ID(),
				"error", err,
			)
			continue
		}

		if result != nil {
			results = append(results, *result)
		}

		// Apply rate limiting delay between requests (skip after last payment)
		if i < len(pendingPayments)-1 {
			select {
			case <-ctx.Done():
				return results, ctx.Err()
			case <-time.After(delay):
			}
		}
	}

	return results, nil
}

// checkPayment checks a single payment for blockchain confirmation
func (uc *ConfirmUSDTPaymentUseCase) checkPayment(ctx context.Context, p *payment.Payment) (*ConfirmUSDTPaymentResult, error) {
	if !p.IsUSDTPayment() {
		return nil, fmt.Errorf("not a USDT payment")
	}

	chainType := p.ChainType()
	if chainType == nil {
		return nil, fmt.Errorf("payment has no chain type")
	}

	usdtAmountRaw := p.USDTAmountRaw()
	if usdtAmountRaw == nil {
		return nil, fmt.Errorf("payment has no USDT amount")
	}

	receivingAddress := p.ReceivingAddress()
	if receivingAddress == nil {
		return nil, fmt.Errorf("payment has no receiving address")
	}

	// Search for matching transaction on chain using exact integer amount
	// Pass payment creation time to filter out transactions before payment was created
	// This prevents attackers from using pre-existing transactions to claim payments
	tx, err := uc.transactionMonitor.FindTransaction(ctx, *chainType, *receivingAddress, *usdtAmountRaw, p.CreatedAt())
	if err != nil {
		return nil, fmt.Errorf("failed to search for transaction: %w", err)
	}

	if tx == nil {
		// No matching transaction found yet
		return nil, nil
	}

	// Validate transaction timestamp is within acceptable window
	// Transaction must be after payment creation (with buffer) and before expiration + grace period
	// This prevents accepting very old transactions that happen to match
	txDeadline := p.ExpiredAt().Add(expirationGracePeriod)
	if tx.Timestamp.After(txDeadline) {
		uc.logger.Warnw("transaction timestamp after deadline, ignoring",
			"payment_id", p.ID(),
			"tx_hash", tx.TxHash,
			"tx_time", tx.Timestamp,
			"deadline", txDeadline,
		)
		return nil, nil
	}

	// Check if transaction has enough confirmations
	requiredConfirms := uc.getRequiredConfirmations(*chainType)
	if tx.Confirmations < requiredConfirms {
		uc.logger.Infow("transaction found but waiting for confirmations",
			"payment_id", p.ID(),
			"tx_hash", tx.TxHash,
			"confirmations", tx.Confirmations,
			"required", requiredConfirms,
		)
		return &ConfirmUSDTPaymentResult{
			PaymentID:     p.ID(),
			Confirmed:     false,
			TxHash:        tx.TxHash,
			Confirmations: tx.Confirmations,
		}, nil
	}

	// Transaction confirmed - update payment
	if err := p.ConfirmUSDTTransaction(tx.TxHash, tx.BlockNumber); err != nil {
		return nil, fmt.Errorf("failed to confirm payment: %w", err)
	}

	if err := uc.paymentRepo.Update(ctx, p); err != nil {
		return nil, fmt.Errorf("failed to update payment: %w", err)
	}

	uc.logger.Infow("USDT payment confirmed",
		"payment_id", p.ID(),
		"tx_hash", tx.TxHash,
		"confirmations", tx.Confirmations,
	)

	// Activate subscription with retry
	subscriptionActivation := "success"
	subscriptionActivated := false
	if err := uc.activateSubscription(ctx, p); err != nil {
		subscriptionActivation = "failed"
		uc.logger.Errorw("failed to activate subscription after payment confirmation",
			"payment_id", p.ID(),
			"subscription_id", p.SubscriptionID(),
			"error", err,
		)
		// Mark payment as needing subscription activation retry
		p.SetMetadata("subscription_activation_pending", true)
		p.SetMetadata("subscription_activation_error", err.Error())
		if updateErr := uc.paymentRepo.Update(ctx, p); updateErr != nil {
			uc.logger.Errorw("failed to update payment metadata",
				"payment_id", p.ID(),
				"error", updateErr,
			)
		}
	} else {
		subscriptionActivated = true
		// Clear any pending activation flag
		p.SetMetadata("subscription_activation_pending", false)
		if updateErr := uc.paymentRepo.Update(ctx, p); updateErr != nil {
			uc.logger.Warnw("failed to clear activation pending flag",
				"payment_id", p.ID(),
				"error", updateErr,
			)
		}
	}

	return &ConfirmUSDTPaymentResult{
		PaymentID:              p.ID(),
		Confirmed:              true,
		TxHash:                 tx.TxHash,
		Confirmations:          tx.Confirmations,
		SubscriptionActivated:  subscriptionActivated,
		SubscriptionActivation: subscriptionActivation,
	}, nil
}

// getRequiredConfirmations returns the required confirmations for a chain
func (uc *ConfirmUSDTPaymentUseCase) getRequiredConfirmations(chainType vo.ChainType) int {
	uc.configMu.RLock()
	defer uc.configMu.RUnlock()

	switch chainType {
	case vo.ChainTypePOL:
		return uc.requiredConfirmsPOL
	case vo.ChainTypeTRC:
		return uc.requiredConfirmsTRC
	default:
		return 20 // Safe default
	}
}

// activateSubscription activates the subscription after payment confirmation
func (uc *ConfirmUSDTPaymentUseCase) activateSubscription(ctx context.Context, p *payment.Payment) error {
	sub, err := uc.subscriptionRepo.GetByID(ctx, p.SubscriptionID())
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}

	// Only activate if subscription is in correct status
	if sub.Status() != subscriptionVO.StatusPendingPayment && sub.Status() != subscriptionVO.StatusInactive {
		uc.logger.Warnw("subscription not in activatable status",
			"subscription_id", sub.ID(),
			"status", sub.Status(),
		)
		return nil
	}

	// Activate the subscription
	if err := sub.Activate(); err != nil {
		return fmt.Errorf("failed to activate subscription: %w", err)
	}

	if err := uc.subscriptionRepo.Update(ctx, sub); err != nil {
		return fmt.Errorf("failed to update subscription: %w", err)
	}

	uc.logger.Infow("subscription activated after USDT payment",
		"subscription_id", sub.ID(),
		"payment_id", p.ID(),
	)

	return nil
}

// UpdateConfig updates the confirmation requirements
func (uc *ConfirmUSDTPaymentUseCase) UpdateConfig(config ConfirmUSDTPaymentConfig) {
	uc.configMu.Lock()
	defer uc.configMu.Unlock()

	if config.RequiredConfirmationsPOL > 0 {
		uc.requiredConfirmsPOL = validateConfirmations(config.RequiredConfirmationsPOL, uc.requiredConfirmsPOL)
	}
	if config.RequiredConfirmationsTRC > 0 {
		uc.requiredConfirmsTRC = validateConfirmations(config.RequiredConfirmationsTRC, uc.requiredConfirmsTRC)
	}
}

// RetryPendingSubscriptionActivations retries subscription activation for confirmed payments
// that failed to activate their subscriptions previously
func (uc *ConfirmUSDTPaymentUseCase) RetryPendingSubscriptionActivations(ctx context.Context) (int, error) {
	// Get confirmed USDT payments with pending subscription activation
	confirmedPayments, err := uc.paymentRepo.GetConfirmedUSDTPaymentsNeedingActivation(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get payments needing activation: %w", err)
	}

	if len(confirmedPayments) == 0 {
		return 0, nil
	}

	uc.logger.Infow("retrying subscription activations", "count", len(confirmedPayments))

	successCount := 0
	for _, p := range confirmedPayments {
		if err := uc.activateSubscription(ctx, p); err != nil {
			uc.logger.Warnw("retry activation failed",
				"payment_id", p.ID(),
				"subscription_id", p.SubscriptionID(),
				"error", err,
			)
			continue
		}

		// Clear the pending flag
		p.SetMetadata("subscription_activation_pending", false)
		if updateErr := uc.paymentRepo.Update(ctx, p); updateErr != nil {
			uc.logger.Warnw("failed to clear activation pending flag after retry",
				"payment_id", p.ID(),
				"error", updateErr,
			)
		}
		successCount++
	}

	if successCount > 0 {
		uc.logger.Infow("subscription activations retried",
			"success", successCount,
			"total", len(confirmedPayments),
		)
	}

	return successCount, nil
}
