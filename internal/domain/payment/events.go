package payment

import (
	"time"

	vo "orris/internal/domain/payment/value_objects"
)

type PaymentSucceededEvent struct {
	PaymentID      uint
	SubscriptionID uint
	UserID         uint
	Amount         vo.Money
	OccurredAt     time.Time
}

func NewPaymentSucceededEvent(paymentID, subscriptionID, userID uint, amount vo.Money) *PaymentSucceededEvent {
	return &PaymentSucceededEvent{
		PaymentID:      paymentID,
		SubscriptionID: subscriptionID,
		UserID:         userID,
		Amount:         amount,
		OccurredAt:     time.Now(),
	}
}

type PaymentFailedEvent struct {
	PaymentID      uint
	SubscriptionID uint
	Reason         string
	OccurredAt     time.Time
}

func NewPaymentFailedEvent(paymentID, subscriptionID uint, reason string) *PaymentFailedEvent {
	return &PaymentFailedEvent{
		PaymentID:      paymentID,
		SubscriptionID: subscriptionID,
		Reason:         reason,
		OccurredAt:     time.Now(),
	}
}

type PaymentExpiredEvent struct {
	PaymentID      uint
	SubscriptionID uint
	OccurredAt     time.Time
}

func NewPaymentExpiredEvent(paymentID, subscriptionID uint) *PaymentExpiredEvent {
	return &PaymentExpiredEvent{
		PaymentID:      paymentID,
		SubscriptionID: subscriptionID,
		OccurredAt:     time.Now(),
	}
}
