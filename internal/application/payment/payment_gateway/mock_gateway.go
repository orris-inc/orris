package payment_gateway

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type MockGateway struct {
	shouldSucceed bool
}

func NewMockGateway(shouldSucceed bool) *MockGateway {
	return &MockGateway{
		shouldSucceed: shouldSucceed,
	}
}

func (m *MockGateway) CreatePayment(ctx context.Context, req CreatePaymentRequest) (*CreatePaymentResponse, error) {
	gatewayOrderNo := fmt.Sprintf("MOCK_%s", req.OrderNo)
	paymentURL := fmt.Sprintf("https://mock-payment.example.com/pay?order=%s", gatewayOrderNo)
	qrCode := fmt.Sprintf("https://mock-payment.example.com/qr?order=%s", gatewayOrderNo)

	return &CreatePaymentResponse{
		GatewayOrderNo: gatewayOrderNo,
		PaymentURL:     paymentURL,
		QRCode:         qrCode,
	}, nil
}

func (m *MockGateway) VerifyCallback(req *http.Request) (*CallbackData, error) {
	if err := req.ParseForm(); err != nil {
		return nil, fmt.Errorf("failed to parse form: %w", err)
	}

	gatewayOrderNo := req.FormValue("gateway_order_no")
	if gatewayOrderNo == "" {
		return nil, fmt.Errorf("missing gateway_order_no")
	}

	status := "TRADE_SUCCESS"
	if !m.shouldSucceed {
		status = "TRADE_FAILED"
	}

	return &CallbackData{
		GatewayOrderNo: gatewayOrderNo,
		TransactionID:  fmt.Sprintf("TXN_%d", time.Now().Unix()),
		Amount:         9900,
		Currency:       "CNY",
		Status:         status,
		PaidAt:         time.Now(),
		RawData:        map[string]string{},
	}, nil
}

func (m *MockGateway) QueryPaymentStatus(ctx context.Context, gatewayOrderNo string) (*PaymentStatusResponse, error) {
	status := "TRADE_SUCCESS"
	if !m.shouldSucceed {
		status = "WAIT_BUYER_PAY"
	}

	paidAt := time.Now()

	return &PaymentStatusResponse{
		GatewayOrderNo: gatewayOrderNo,
		TransactionID:  fmt.Sprintf("TXN_%d", time.Now().Unix()),
		Status:         status,
		Amount:         9900,
		Currency:       "CNY",
		PaidAt:         &paidAt,
	}, nil
}
