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
	GetExpiredPayments(ctx context.Context) ([]*Payment, error)
}
