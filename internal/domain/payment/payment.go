package payment

import (
	"fmt"
	"time"

	vo "github.com/orris-inc/orris/internal/domain/payment/valueobjects"
	"github.com/orris-inc/orris/internal/domain/shared/services"
)

type Payment struct {
	id             uint
	orderNo        string
	subscriptionID uint
	userID         uint
	amount         vo.Money
	paymentMethod  vo.PaymentMethod
	status         vo.PaymentStatus

	gatewayOrderNo *string
	transactionID  *string
	paymentURL     *string
	qrCode         *string

	paidAt    *time.Time
	expiredAt time.Time

	metadata map[string]interface{}

	version   int
	createdAt time.Time
	updatedAt time.Time
}

func NewPayment(subscriptionID, userID uint, amount vo.Money, method vo.PaymentMethod) (*Payment, error) {
	if subscriptionID == 0 {
		return nil, fmt.Errorf("subscription ID is required")
	}
	if userID == 0 {
		return nil, fmt.Errorf("user ID is required")
	}
	if !amount.IsPositive() {
		return nil, fmt.Errorf("amount must be positive")
	}

	orderNoGen := services.NewOrderNumberGenerator()
	orderNo := orderNoGen.Generate("PAY")
	expiredAt := time.Now().Add(30 * time.Minute)

	return &Payment{
		orderNo:        orderNo,
		subscriptionID: subscriptionID,
		userID:         userID,
		amount:         amount,
		paymentMethod:  method,
		status:         vo.PaymentStatusPending,
		expiredAt:      expiredAt,
		metadata:       make(map[string]interface{}),
		createdAt:      time.Now(),
		updatedAt:      time.Now(),
	}, nil
}

func (p *Payment) MarkAsPaid(transactionID string) error {
	if p.status == vo.PaymentStatusPaid {
		return nil
	}

	if p.status != vo.PaymentStatusPending {
		return fmt.Errorf("cannot mark payment as paid with status %s", p.status)
	}

	now := time.Now()
	p.status = vo.PaymentStatusPaid
	p.transactionID = &transactionID
	p.paidAt = &now
	p.updatedAt = now
	p.version++

	return nil
}

func (p *Payment) MarkAsFailed(reason string) error {
	if p.status.IsFinal() {
		return fmt.Errorf("cannot mark payment as failed with final status %s", p.status)
	}

	p.status = vo.PaymentStatusFailed
	p.metadata["failure_reason"] = reason
	p.updatedAt = time.Now()
	p.version++

	return nil
}

func (p *Payment) MarkAsExpired() error {
	if p.status.IsFinal() {
		return nil
	}

	p.status = vo.PaymentStatusExpired
	p.updatedAt = time.Now()
	p.version++

	return nil
}

func (p *Payment) SetGatewayInfo(gatewayOrderNo, paymentURL, qrCode string) {
	p.gatewayOrderNo = &gatewayOrderNo
	p.paymentURL = &paymentURL
	p.qrCode = &qrCode
	p.updatedAt = time.Now()
}

func (p *Payment) IsExpired() bool {
	return time.Now().After(p.expiredAt) && p.status == vo.PaymentStatusPending
}

func (p *Payment) ID() uint {
	return p.id
}

func (p *Payment) OrderNo() string {
	return p.orderNo
}

func (p *Payment) SubscriptionID() uint {
	return p.subscriptionID
}

func (p *Payment) UserID() uint {
	return p.userID
}

func (p *Payment) Amount() vo.Money {
	return p.amount
}

func (p *Payment) PaymentMethod() vo.PaymentMethod {
	return p.paymentMethod
}

func (p *Payment) Status() vo.PaymentStatus {
	return p.status
}

func (p *Payment) GatewayOrderNo() *string {
	return p.gatewayOrderNo
}

func (p *Payment) TransactionID() *string {
	return p.transactionID
}

func (p *Payment) PaymentURL() *string {
	return p.paymentURL
}

func (p *Payment) QRCode() *string {
	return p.qrCode
}

func (p *Payment) PaidAt() *time.Time {
	return p.paidAt
}

func (p *Payment) ExpiredAt() time.Time {
	return p.expiredAt
}

func (p *Payment) Metadata() map[string]interface{} {
	return p.metadata
}

func (p *Payment) Version() int {
	return p.version
}

func (p *Payment) CreatedAt() time.Time {
	return p.createdAt
}

func (p *Payment) UpdatedAt() time.Time {
	return p.updatedAt
}

func ReconstructPayment(
	id uint,
	orderNo string,
	subscriptionID, userID uint,
	amount vo.Money,
	paymentMethod vo.PaymentMethod,
	status vo.PaymentStatus,
	gatewayOrderNo, transactionID, paymentURL, qrCode *string,
	paidAt *time.Time,
	expiredAt time.Time,
	metadata map[string]interface{},
	version int,
	createdAt, updatedAt time.Time,
) *Payment {
	return &Payment{
		id:             id,
		orderNo:        orderNo,
		subscriptionID: subscriptionID,
		userID:         userID,
		amount:         amount,
		paymentMethod:  paymentMethod,
		status:         status,
		gatewayOrderNo: gatewayOrderNo,
		transactionID:  transactionID,
		paymentURL:     paymentURL,
		qrCode:         qrCode,
		paidAt:         paidAt,
		expiredAt:      expiredAt,
		metadata:       metadata,
		version:        version,
		createdAt:      createdAt,
		updatedAt:      updatedAt,
	}
}
