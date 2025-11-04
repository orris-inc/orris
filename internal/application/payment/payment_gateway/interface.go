package payment_gateway

import (
	"context"
	"net/http"
	"time"
)

type PaymentGateway interface {
	CreatePayment(ctx context.Context, req CreatePaymentRequest) (*CreatePaymentResponse, error)
	VerifyCallback(req *http.Request) (*CallbackData, error)
}

type CreatePaymentRequest struct {
	OrderNo   string
	Amount    int64
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

type CallbackData struct {
	GatewayOrderNo string
	TransactionID  string
	Amount         int64
	Currency       string
	Status         string
	PaidAt         time.Time
	RawData        map[string]string
}
