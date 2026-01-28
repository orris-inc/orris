package paymentgateway

import (
	"context"
	"net/http"
	"time"
)

// PaymentGateway defines the interface for payment gateway integrations
type PaymentGateway interface {
	CreatePayment(ctx context.Context, req CreatePaymentRequest) (*CreatePaymentResponse, error)
	// VerifyCallback verifies and parses the payment callback from the gateway.
	// The returned CallbackData.Amount MUST be in the smallest currency unit (e.g., cents for CNY/USD).
	VerifyCallback(req *http.Request) (*CallbackData, error)
}

// CreatePaymentRequest contains the data needed to create a payment
type CreatePaymentRequest struct {
	OrderNo   string
	Amount    int64 // Amount in smallest currency unit (e.g., cents: 100 = 1 CNY)
	Currency  string
	Subject   string
	Body      string
	ReturnURL string
	NotifyURL string
}

type CreatePaymentResponse struct {
	GatewayOrderNo string
	PaymentURL     string
	QRCode         string
}

// CallbackData contains the parsed payment callback data from the gateway.
// IMPORTANT: Amount must be in the smallest currency unit (e.g., cents for CNY/USD)
// to match the Payment.Amount stored in the database.
type CallbackData struct {
	GatewayOrderNo string
	TransactionID  string
	Amount         int64 // Amount in smallest currency unit (e.g., cents: 100 = 1 CNY)
	Currency       string
	Status         string
	PaidAt         time.Time
	RawData        map[string]string
}
