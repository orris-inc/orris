package payment

import (
	"fmt"
	"time"

	vo "github.com/orris-inc/orris/internal/domain/payment/valueobjects"
	"github.com/orris-inc/orris/internal/domain/shared/services"
	"github.com/orris-inc/orris/internal/shared/biztime"
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

	// USDT-specific fields
	chainType        *vo.ChainType
	usdtAmountRaw    *uint64 // USDT amount in smallest unit (1 USDT = 1000000)
	receivingAddress *string
	exchangeRate     *float64 // Exchange rate at time of payment (for reference only)
	txHash           *string
	blockNumber      *uint64
	confirmedAt      *time.Time

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
	now := biztime.NowUTC()
	expiredAt := now.Add(30 * time.Minute)

	return &Payment{
		orderNo:        orderNo,
		subscriptionID: subscriptionID,
		userID:         userID,
		amount:         amount,
		paymentMethod:  method,
		status:         vo.PaymentStatusPending,
		expiredAt:      expiredAt,
		metadata:       make(map[string]interface{}),
		createdAt:      now,
		updatedAt:      now,
	}, nil
}

func (p *Payment) MarkAsPaid(transactionID string) error {
	if p.status == vo.PaymentStatusPaid {
		return nil
	}

	if p.status != vo.PaymentStatusPending {
		return fmt.Errorf("cannot mark payment as paid with status %s", p.status)
	}

	now := biztime.NowUTC()
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
	p.updatedAt = biztime.NowUTC()
	p.version++

	return nil
}

func (p *Payment) MarkAsExpired() error {
	if p.status.IsFinal() {
		return nil
	}

	p.status = vo.PaymentStatusExpired
	p.updatedAt = biztime.NowUTC()
	p.version++

	return nil
}

func (p *Payment) SetGatewayInfo(gatewayOrderNo, paymentURL, qrCode string) {
	p.gatewayOrderNo = &gatewayOrderNo
	p.paymentURL = &paymentURL
	p.qrCode = &qrCode
	p.updatedAt = biztime.NowUTC()
}

func (p *Payment) IsExpired() bool {
	return biztime.NowUTC().After(p.expiredAt) && p.status == vo.PaymentStatusPending
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

// SetMetadata sets a metadata key-value pair
func (p *Payment) SetMetadata(key string, value interface{}) {
	if p.metadata == nil {
		p.metadata = make(map[string]interface{})
	}
	p.metadata[key] = value
	p.updatedAt = biztime.NowUTC()
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

// USDT-specific getters

func (p *Payment) ChainType() *vo.ChainType {
	return p.chainType
}

func (p *Payment) USDTAmountRaw() *uint64 {
	return p.usdtAmountRaw
}

func (p *Payment) ReceivingAddress() *string {
	return p.receivingAddress
}

func (p *Payment) ExchangeRate() *float64 {
	return p.exchangeRate
}

func (p *Payment) TxHash() *string {
	return p.txHash
}

func (p *Payment) BlockNumber() *uint64 {
	return p.blockNumber
}

func (p *Payment) ConfirmedAt() *time.Time {
	return p.confirmedAt
}

// SetUSDTInfo sets USDT-specific payment information
// usdtAmountRaw is the USDT amount in smallest unit (1 USDT = 1000000)
func (p *Payment) SetUSDTInfo(chainType vo.ChainType, usdtAmountRaw uint64, receivingAddress string, exchangeRate float64) {
	p.chainType = &chainType
	p.usdtAmountRaw = &usdtAmountRaw
	p.receivingAddress = &receivingAddress
	p.exchangeRate = &exchangeRate
	p.updatedAt = biztime.NowUTC()
}

// ConfirmUSDTTransaction marks the USDT payment as confirmed with blockchain transaction details
func (p *Payment) ConfirmUSDTTransaction(txHash string, blockNumber uint64) error {
	if p.status == vo.PaymentStatusPaid {
		return nil
	}

	if p.status != vo.PaymentStatusPending {
		return fmt.Errorf("cannot confirm USDT transaction with status %s", p.status)
	}

	if !p.paymentMethod.IsUSDT() {
		return fmt.Errorf("cannot confirm USDT transaction for non-USDT payment method")
	}

	now := biztime.NowUTC()
	p.status = vo.PaymentStatusPaid
	p.txHash = &txHash
	p.blockNumber = &blockNumber
	p.confirmedAt = &now
	p.paidAt = &now
	p.transactionID = &txHash
	p.updatedAt = now
	p.version++

	return nil
}

// IsUSDTPayment returns true if this is a USDT payment
func (p *Payment) IsUSDTPayment() bool {
	return p.paymentMethod.IsUSDT()
}

// ValidateCallbackAmount validates that the callback amount and currency match the payment
func (p *Payment) ValidateCallbackAmount(amount int64, currency string) error {
	if p.amount.AmountInCents() != amount {
		return fmt.Errorf("amount mismatch: expected %d, got %d", p.amount.AmountInCents(), amount)
	}
	if p.amount.Currency() != currency {
		return fmt.Errorf("currency mismatch: expected %s, got %s", p.amount.Currency(), currency)
	}
	return nil
}

// SetID sets the payment ID after persistence (used by repository after Create)
func (p *Payment) SetID(id uint) {
	p.id = id
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

// ReconstructPaymentWithUSDT creates a Payment instance with USDT-specific fields from persistence
func ReconstructPaymentWithUSDT(
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
	// USDT-specific fields
	chainType *vo.ChainType,
	usdtAmountRaw *uint64, // USDT amount in smallest unit (1 USDT = 1000000)
	receivingAddress *string,
	exchangeRate *float64,
	txHash *string,
	blockNumber *uint64,
	confirmedAt *time.Time,
) *Payment {
	return &Payment{
		id:               id,
		orderNo:          orderNo,
		subscriptionID:   subscriptionID,
		userID:           userID,
		amount:           amount,
		paymentMethod:    paymentMethod,
		status:           status,
		gatewayOrderNo:   gatewayOrderNo,
		transactionID:    transactionID,
		paymentURL:       paymentURL,
		qrCode:           qrCode,
		paidAt:           paidAt,
		expiredAt:        expiredAt,
		chainType:        chainType,
		usdtAmountRaw:    usdtAmountRaw,
		receivingAddress: receivingAddress,
		exchangeRate:     exchangeRate,
		txHash:           txHash,
		blockNumber:      blockNumber,
		confirmedAt:      confirmedAt,
		metadata:         metadata,
		version:          version,
		createdAt:        createdAt,
		updatedAt:        updatedAt,
	}
}
