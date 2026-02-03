package payment

import "context"

type PaymentRepository interface {
	Create(ctx context.Context, payment *Payment) error
	Update(ctx context.Context, payment *Payment) error
	GetByID(ctx context.Context, id uint) (*Payment, error)
	GetByOrderNo(ctx context.Context, orderNo string) (*Payment, error)
	GetByGatewayOrderNo(ctx context.Context, gatewayOrderNo string) (*Payment, error)
	GetBySubscriptionID(ctx context.Context, subscriptionID uint) ([]*Payment, error)
	GetPendingBySubscriptionID(ctx context.Context, subscriptionID uint) (*Payment, error)
	// HasPendingPaymentBySubscriptionID checks if there are any pending payments for a subscription
	HasPendingPaymentBySubscriptionID(ctx context.Context, subscriptionID uint) (bool, error)
	// GetSubscriptionIDsWithPendingPayments returns subscription IDs that have pending payments
	// from the given list of subscription IDs
	GetSubscriptionIDsWithPendingPayments(ctx context.Context, subscriptionIDs []uint) ([]uint, error)
	GetExpiredPayments(ctx context.Context) ([]*Payment, error)
	GetPendingUSDTPayments(ctx context.Context) ([]*Payment, error)
	// GetConfirmedUSDTPaymentsNeedingActivation returns confirmed USDT payments
	// that have subscription_activation_pending=true in metadata
	GetConfirmedUSDTPaymentsNeedingActivation(ctx context.Context) ([]*Payment, error)
	// GetPaidPaymentsNeedingActivation returns paid non-USDT payments
	// that have subscription_activation_pending=true in metadata
	GetPaidPaymentsNeedingActivation(ctx context.Context) ([]*Payment, error)
	// CountPendingUSDTPaymentsByUser returns the count of pending USDT payments for a user
	CountPendingUSDTPaymentsByUser(ctx context.Context, userID uint) (int, error)
}
