package payment

import (
	"testing"
	"time"

	vo "github.com/orris-inc/orris/internal/domain/payment/valueobjects"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- helpers ---

func validMoney() vo.Money {
	return vo.NewMoney(1000, "CNY") // 10.00 CNY
}

func validPayment(t *testing.T) *Payment {
	t.Helper()
	p, err := NewPayment(1, 1, validMoney(), vo.PaymentMethodAlipay)
	require.NoError(t, err)
	return p
}

func validUSDTPayment(t *testing.T) *Payment {
	t.Helper()
	p, err := NewPayment(1, 1, validMoney(), vo.PaymentMethodUSDTPOL)
	require.NoError(t, err)
	return p
}

// reconstructPending builds a Payment in Pending state with an expiredAt that callers can control.
func reconstructPending(expiredAt time.Time) *Payment {
	return ReconstructPaymentWithParams(PaymentReconstructParams{
		ID:             10,
		OrderNo:        "PAY_test_123",
		SubscriptionID: 1,
		UserID:         1,
		Amount:         validMoney(),
		PaymentMethod:  vo.PaymentMethodAlipay,
		Status:         vo.PaymentStatusPending,
		ExpiredAt:      expiredAt,
		Metadata:       map[string]interface{}{},
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	})
}

// =============================================================================
// Constructor Tests
// =============================================================================

func TestNewPayment_ValidInput(t *testing.T) {
	tests := []struct {
		name           string
		subscriptionID uint
		userID         uint
		amount         vo.Money
		method         vo.PaymentMethod
	}{
		{
			name:           "alipay payment",
			subscriptionID: 1,
			userID:         10,
			amount:         vo.NewMoney(999, "CNY"),
			method:         vo.PaymentMethodAlipay,
		},
		{
			name:           "wechat payment",
			subscriptionID: 2,
			userID:         20,
			amount:         vo.NewMoney(1, "CNY"),
			method:         vo.PaymentMethodWechat,
		},
		{
			name:           "stripe payment",
			subscriptionID: 3,
			userID:         30,
			amount:         vo.NewMoney(50000, "USD"),
			method:         vo.PaymentMethodStripe,
		},
		{
			name:           "usdt pol payment",
			subscriptionID: 4,
			userID:         40,
			amount:         vo.NewMoney(10000, "CNY"),
			method:         vo.PaymentMethodUSDTPOL,
		},
		{
			name:           "usdt trc payment",
			subscriptionID: 5,
			userID:         50,
			amount:         vo.NewMoney(10000, "CNY"),
			method:         vo.PaymentMethodUSDTTRC,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, err := NewPayment(tc.subscriptionID, tc.userID, tc.amount, tc.method)
			require.NoError(t, err)
			require.NotNil(t, p)

			assert.Equal(t, uint(0), p.ID(), "new payment should have zero ID")
			assert.NotEmpty(t, p.OrderNo(), "order number should be generated")
			assert.Equal(t, tc.subscriptionID, p.SubscriptionID())
			assert.Equal(t, tc.userID, p.UserID())
			assert.True(t, p.Amount().Equals(tc.amount))
			assert.Equal(t, tc.method, p.PaymentMethod())
			assert.Equal(t, vo.PaymentStatusPending, p.Status())
			assert.Nil(t, p.TransactionID())
			assert.Nil(t, p.PaidAt())
			assert.NotNil(t, p.Metadata())
			assert.False(t, p.ExpiredAt().IsZero(), "expiredAt should be set")
			assert.False(t, p.CreatedAt().IsZero())
			assert.False(t, p.UpdatedAt().IsZero())
			assert.Equal(t, 0, p.Version())
		})
	}
}

func TestNewPayment_InvalidAmount(t *testing.T) {
	tests := []struct {
		name   string
		amount vo.Money
	}{
		{
			name:   "zero amount",
			amount: vo.NewMoney(0, "CNY"),
		},
		{
			name:   "negative amount",
			amount: vo.NewMoney(-100, "CNY"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, err := NewPayment(1, 1, tc.amount, vo.PaymentMethodAlipay)
			assert.Error(t, err)
			assert.Nil(t, p)
			assert.Contains(t, err.Error(), "amount must be positive")
		})
	}
}

func TestNewPayment_InvalidIDs(t *testing.T) {
	tests := []struct {
		name           string
		subscriptionID uint
		userID         uint
		expectErr      string
	}{
		{
			name:           "zero subscription ID",
			subscriptionID: 0,
			userID:         1,
			expectErr:      "subscription ID is required",
		},
		{
			name:           "zero user ID",
			subscriptionID: 1,
			userID:         0,
			expectErr:      "user ID is required",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, err := NewPayment(tc.subscriptionID, tc.userID, validMoney(), vo.PaymentMethodAlipay)
			assert.Error(t, err)
			assert.Nil(t, p)
			assert.Contains(t, err.Error(), tc.expectErr)
		})
	}
}

// =============================================================================
// State Transition Tests
// =============================================================================

func TestPayment_MarkAsPaid(t *testing.T) {
	t.Run("pending to paid", func(t *testing.T) {
		p := validPayment(t)
		txID := "tx_123456"

		err := p.MarkAsPaid(txID)
		require.NoError(t, err)

		assert.Equal(t, vo.PaymentStatusPaid, p.Status())
		require.NotNil(t, p.TransactionID())
		assert.Equal(t, txID, *p.TransactionID())
		require.NotNil(t, p.PaidAt())
		assert.Equal(t, 1, p.Version())
	})

	t.Run("idempotent when already paid", func(t *testing.T) {
		p := validPayment(t)
		require.NoError(t, p.MarkAsPaid("tx_first"))

		// Second call with different txID should be no-op (idempotent)
		err := p.MarkAsPaid("tx_second")
		assert.NoError(t, err)
		// Original transaction ID should be preserved
		assert.Equal(t, "tx_first", *p.TransactionID())
		assert.Equal(t, 1, p.Version(), "version should not change on idempotent call")
	})
}

func TestPayment_MarkAsFailed(t *testing.T) {
	t.Run("pending to failed", func(t *testing.T) {
		p := validPayment(t)
		reason := "insufficient funds"

		err := p.MarkAsFailed(reason)
		require.NoError(t, err)

		assert.Equal(t, vo.PaymentStatusFailed, p.Status())
		assert.Equal(t, reason, p.Metadata()["failure_reason"])
		assert.Equal(t, 1, p.Version())
	})

	t.Run("cannot fail an already paid payment", func(t *testing.T) {
		p := validPayment(t)
		require.NoError(t, p.MarkAsPaid("tx_abc"))

		err := p.MarkAsFailed("some reason")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot mark payment as failed")
		assert.Equal(t, vo.PaymentStatusPaid, p.Status(), "status should remain paid")
	})

	t.Run("cannot fail an already failed payment", func(t *testing.T) {
		p := validPayment(t)
		require.NoError(t, p.MarkAsFailed("first reason"))

		err := p.MarkAsFailed("second reason")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot mark payment as failed")
	})

	t.Run("cannot fail an expired payment", func(t *testing.T) {
		p := validPayment(t)
		require.NoError(t, p.MarkAsExpired())

		err := p.MarkAsFailed("too late")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot mark payment as failed")
	})
}

func TestPayment_MarkAsExpired(t *testing.T) {
	t.Run("pending to expired", func(t *testing.T) {
		p := validPayment(t)

		err := p.MarkAsExpired()
		require.NoError(t, err)

		assert.Equal(t, vo.PaymentStatusExpired, p.Status())
		assert.Equal(t, 1, p.Version())
	})

	t.Run("idempotent on already expired", func(t *testing.T) {
		p := validPayment(t)
		require.NoError(t, p.MarkAsExpired())

		err := p.MarkAsExpired()
		assert.NoError(t, err, "should be no-op on already expired payment")
		assert.Equal(t, 1, p.Version(), "version should not change on idempotent call")
	})

	t.Run("idempotent on paid payment", func(t *testing.T) {
		p := validPayment(t)
		require.NoError(t, p.MarkAsPaid("tx_abc"))

		err := p.MarkAsExpired()
		assert.NoError(t, err, "should be no-op on paid payment")
		assert.Equal(t, vo.PaymentStatusPaid, p.Status(), "status should remain paid")
	})

	t.Run("idempotent on failed payment", func(t *testing.T) {
		p := validPayment(t)
		require.NoError(t, p.MarkAsFailed("reason"))

		err := p.MarkAsExpired()
		assert.NoError(t, err, "should be no-op on failed payment")
		assert.Equal(t, vo.PaymentStatusFailed, p.Status(), "status should remain failed")
	})
}

func TestPayment_AlreadyPaid(t *testing.T) {
	p := validPayment(t)
	require.NoError(t, p.MarkAsPaid("tx_original"))

	t.Run("cannot mark as failed after paid", func(t *testing.T) {
		err := p.MarkAsFailed("reason")
		assert.Error(t, err)
		assert.Equal(t, vo.PaymentStatusPaid, p.Status())
	})
}

func TestPayment_AlreadyFailed(t *testing.T) {
	p := validPayment(t)
	require.NoError(t, p.MarkAsFailed("original reason"))

	t.Run("cannot mark as paid after failed", func(t *testing.T) {
		err := p.MarkAsPaid("tx_late")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot mark payment as paid")
		assert.Equal(t, vo.PaymentStatusFailed, p.Status())
	})
}

// =============================================================================
// Business Logic Tests
// =============================================================================

func TestPayment_ValidateCallbackAmount(t *testing.T) {
	amount := vo.NewMoney(1000, "CNY")
	p, err := NewPayment(1, 1, amount, vo.PaymentMethodAlipay)
	require.NoError(t, err)

	t.Run("matching amount and currency", func(t *testing.T) {
		err := p.ValidateCallbackAmount(1000, "CNY")
		assert.NoError(t, err)
	})

	t.Run("amount mismatch", func(t *testing.T) {
		err := p.ValidateCallbackAmount(999, "CNY")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "amount mismatch")
	})

	t.Run("currency mismatch", func(t *testing.T) {
		err := p.ValidateCallbackAmount(1000, "USD")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "currency mismatch")
	})

	t.Run("both amount and currency mismatch", func(t *testing.T) {
		err := p.ValidateCallbackAmount(500, "USD")
		assert.Error(t, err)
		// Amount is checked first, so the error should be about amount
		assert.Contains(t, err.Error(), "amount mismatch")
	})
}

func TestPayment_SetGatewayInfo(t *testing.T) {
	p := validPayment(t)
	gatewayOrderNo := "GW_ORDER_123"
	paymentURL := "https://pay.example.com/order/123"
	qrCode := "https://pay.example.com/qr/123"

	p.SetGatewayInfo(gatewayOrderNo, paymentURL, qrCode)

	require.NotNil(t, p.GatewayOrderNo())
	assert.Equal(t, gatewayOrderNo, *p.GatewayOrderNo())
	require.NotNil(t, p.PaymentURL())
	assert.Equal(t, paymentURL, *p.PaymentURL())
	require.NotNil(t, p.QRCode())
	assert.Equal(t, qrCode, *p.QRCode())
}

func TestPayment_IsExpired(t *testing.T) {
	t.Run("not expired when expiredAt is in the future", func(t *testing.T) {
		p := reconstructPending(time.Now().UTC().Add(1 * time.Hour))
		assert.False(t, p.IsExpired())
	})

	t.Run("expired when expiredAt is in the past and status is pending", func(t *testing.T) {
		p := reconstructPending(time.Now().UTC().Add(-1 * time.Hour))
		assert.True(t, p.IsExpired())
	})

	t.Run("not expired when status is paid even if past expiredAt", func(t *testing.T) {
		p := reconstructPending(time.Now().UTC().Add(-1 * time.Hour))
		require.NoError(t, p.MarkAsPaid("tx_123"))
		assert.False(t, p.IsExpired(), "paid payment should not be considered expired")
	})

	t.Run("not expired when status is failed even if past expiredAt", func(t *testing.T) {
		p := reconstructPending(time.Now().UTC().Add(-1 * time.Hour))
		require.NoError(t, p.MarkAsFailed("reason"))
		assert.False(t, p.IsExpired(), "failed payment should not be considered expired")
	})
}

func TestPayment_IsUSDTPayment(t *testing.T) {
	tests := []struct {
		name     string
		method   vo.PaymentMethod
		expected bool
	}{
		{"alipay is not USDT", vo.PaymentMethodAlipay, false},
		{"wechat is not USDT", vo.PaymentMethodWechat, false},
		{"stripe is not USDT", vo.PaymentMethodStripe, false},
		{"usdt_pol is USDT", vo.PaymentMethodUSDTPOL, true},
		{"usdt_trc is USDT", vo.PaymentMethodUSDTTRC, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, err := NewPayment(1, 1, validMoney(), tc.method)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, p.IsUSDTPayment())
		})
	}
}

func TestPayment_SetUSDTInfo(t *testing.T) {
	p := validUSDTPayment(t)

	chainType := vo.ChainTypePOL
	var usdtAmountRaw uint64 = 15000000 // 15 USDT
	receivingAddr := "0x1234567890abcdef1234567890abcdef12345678"
	exchangeRate := 7.25

	p.SetUSDTInfo(chainType, usdtAmountRaw, receivingAddr, exchangeRate)

	require.NotNil(t, p.ChainType())
	assert.Equal(t, chainType, *p.ChainType())
	require.NotNil(t, p.USDTAmountRaw())
	assert.Equal(t, usdtAmountRaw, *p.USDTAmountRaw())
	require.NotNil(t, p.ReceivingAddress())
	assert.Equal(t, receivingAddr, *p.ReceivingAddress())
	require.NotNil(t, p.ExchangeRate())
	assert.Equal(t, exchangeRate, *p.ExchangeRate())
}

func TestPayment_ConfirmUSDTTransaction(t *testing.T) {
	t.Run("successful confirmation", func(t *testing.T) {
		p := validUSDTPayment(t)
		txHash := "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
		var blockNumber uint64 = 12345678

		err := p.ConfirmUSDTTransaction(txHash, blockNumber)
		require.NoError(t, err)

		assert.Equal(t, vo.PaymentStatusPaid, p.Status())
		require.NotNil(t, p.TxHash())
		assert.Equal(t, txHash, *p.TxHash())
		require.NotNil(t, p.BlockNumber())
		assert.Equal(t, blockNumber, *p.BlockNumber())
		require.NotNil(t, p.ConfirmedAt())
		require.NotNil(t, p.PaidAt())
		require.NotNil(t, p.TransactionID())
		assert.Equal(t, txHash, *p.TransactionID(), "transactionID should be set to txHash")
		assert.Equal(t, 1, p.Version())
	})

	t.Run("idempotent when already paid", func(t *testing.T) {
		p := validUSDTPayment(t)
		require.NoError(t, p.ConfirmUSDTTransaction("tx_first", 100))

		err := p.ConfirmUSDTTransaction("tx_second", 200)
		assert.NoError(t, err)
		assert.Equal(t, "tx_first", *p.TxHash(), "original tx hash should be preserved")
		assert.Equal(t, 1, p.Version(), "version should not change on idempotent call")
	})

	t.Run("cannot confirm non-USDT payment", func(t *testing.T) {
		p := validPayment(t) // alipay payment
		err := p.ConfirmUSDTTransaction("tx_hash", 100)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "non-USDT payment method")
		assert.Equal(t, vo.PaymentStatusPending, p.Status())
	})

	t.Run("cannot confirm failed payment", func(t *testing.T) {
		p := validUSDTPayment(t)
		require.NoError(t, p.MarkAsFailed("timeout"))

		err := p.ConfirmUSDTTransaction("tx_hash", 100)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot confirm USDT transaction")
		assert.Equal(t, vo.PaymentStatusFailed, p.Status())
	})

	t.Run("cannot confirm expired payment", func(t *testing.T) {
		p := validUSDTPayment(t)
		require.NoError(t, p.MarkAsExpired())

		err := p.ConfirmUSDTTransaction("tx_hash", 100)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot confirm USDT transaction")
	})
}

// =============================================================================
// State Machine Tests — valid and invalid transitions
// =============================================================================

func TestPayment_StateMachine(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T) *Payment
		act       func(p *Payment) error
		wantErr   bool
		wantState vo.PaymentStatus
	}{
		// From Pending
		{
			name:      "pending -> paid via MarkAsPaid",
			setup:     func(t *testing.T) *Payment { return validPayment(t) },
			act:       func(p *Payment) error { return p.MarkAsPaid("tx") },
			wantErr:   false,
			wantState: vo.PaymentStatusPaid,
		},
		{
			name:      "pending -> failed via MarkAsFailed",
			setup:     func(t *testing.T) *Payment { return validPayment(t) },
			act:       func(p *Payment) error { return p.MarkAsFailed("reason") },
			wantErr:   false,
			wantState: vo.PaymentStatusFailed,
		},
		{
			name:      "pending -> expired via MarkAsExpired",
			setup:     func(t *testing.T) *Payment { return validPayment(t) },
			act:       func(p *Payment) error { return p.MarkAsExpired() },
			wantErr:   false,
			wantState: vo.PaymentStatusExpired,
		},
		// From Paid
		{
			name: "paid -> paid via MarkAsPaid (idempotent)",
			setup: func(t *testing.T) *Payment {
				p := validPayment(t)
				require.NoError(t, p.MarkAsPaid("tx"))
				return p
			},
			act:       func(p *Payment) error { return p.MarkAsPaid("tx2") },
			wantErr:   false,
			wantState: vo.PaymentStatusPaid,
		},
		{
			name: "paid -> failed rejected",
			setup: func(t *testing.T) *Payment {
				p := validPayment(t)
				require.NoError(t, p.MarkAsPaid("tx"))
				return p
			},
			act:       func(p *Payment) error { return p.MarkAsFailed("reason") },
			wantErr:   true,
			wantState: vo.PaymentStatusPaid,
		},
		{
			name: "paid -> expired (no-op, stays paid)",
			setup: func(t *testing.T) *Payment {
				p := validPayment(t)
				require.NoError(t, p.MarkAsPaid("tx"))
				return p
			},
			act:       func(p *Payment) error { return p.MarkAsExpired() },
			wantErr:   false,
			wantState: vo.PaymentStatusPaid,
		},
		// From Failed
		{
			name: "failed -> paid rejected",
			setup: func(t *testing.T) *Payment {
				p := validPayment(t)
				require.NoError(t, p.MarkAsFailed("reason"))
				return p
			},
			act:       func(p *Payment) error { return p.MarkAsPaid("tx") },
			wantErr:   true,
			wantState: vo.PaymentStatusFailed,
		},
		{
			name: "failed -> failed rejected",
			setup: func(t *testing.T) *Payment {
				p := validPayment(t)
				require.NoError(t, p.MarkAsFailed("reason"))
				return p
			},
			act:       func(p *Payment) error { return p.MarkAsFailed("reason2") },
			wantErr:   true,
			wantState: vo.PaymentStatusFailed,
		},
		{
			name: "failed -> expired (no-op, stays failed)",
			setup: func(t *testing.T) *Payment {
				p := validPayment(t)
				require.NoError(t, p.MarkAsFailed("reason"))
				return p
			},
			act:       func(p *Payment) error { return p.MarkAsExpired() },
			wantErr:   false,
			wantState: vo.PaymentStatusFailed,
		},
		// From Expired
		{
			name: "expired -> paid rejected",
			setup: func(t *testing.T) *Payment {
				p := validPayment(t)
				require.NoError(t, p.MarkAsExpired())
				return p
			},
			act:       func(p *Payment) error { return p.MarkAsPaid("tx") },
			wantErr:   true,
			wantState: vo.PaymentStatusExpired,
		},
		{
			name: "expired -> failed rejected",
			setup: func(t *testing.T) *Payment {
				p := validPayment(t)
				require.NoError(t, p.MarkAsExpired())
				return p
			},
			act:       func(p *Payment) error { return p.MarkAsFailed("reason") },
			wantErr:   true,
			wantState: vo.PaymentStatusExpired,
		},
		{
			name: "expired -> expired (no-op, stays expired)",
			setup: func(t *testing.T) *Payment {
				p := validPayment(t)
				require.NoError(t, p.MarkAsExpired())
				return p
			},
			act:       func(p *Payment) error { return p.MarkAsExpired() },
			wantErr:   false,
			wantState: vo.PaymentStatusExpired,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := tc.setup(t)
			err := tc.act(p)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.wantState, p.Status())
		})
	}
}

// =============================================================================
// Edge Cases
// =============================================================================

func TestPayment_AmountComparisonPrecision(t *testing.T) {
	// Verify that integer-cent comparison avoids floating-point issues
	amount := vo.NewMoney(333, "CNY") // 3.33 CNY
	p, err := NewPayment(1, 1, amount, vo.PaymentMethodAlipay)
	require.NoError(t, err)

	assert.NoError(t, p.ValidateCallbackAmount(333, "CNY"))
	assert.Error(t, p.ValidateCallbackAmount(334, "CNY"))
	assert.Error(t, p.ValidateCallbackAmount(332, "CNY"))
}

func TestPayment_ExpirationTiming(t *testing.T) {
	t.Run("exactly at boundary", func(t *testing.T) {
		// A payment with expiredAt exactly now should not be expired yet
		// because time.After is strictly greater than
		now := time.Now().UTC()
		p := reconstructPending(now.Add(1 * time.Second))
		assert.False(t, p.IsExpired(), "payment with expiredAt slightly in the future should not be expired")
	})
}

func TestPayment_SetMetadata(t *testing.T) {
	p := validPayment(t)

	p.SetMetadata("key1", "value1")
	p.SetMetadata("key2", 42)

	assert.Equal(t, "value1", p.Metadata()["key1"])
	assert.Equal(t, 42, p.Metadata()["key2"])
}

func TestPayment_SetID(t *testing.T) {
	p := validPayment(t)
	assert.Equal(t, uint(0), p.ID())

	p.SetID(100)
	assert.Equal(t, uint(100), p.ID())
}

func TestPayment_ReconstructPaymentWithParams(t *testing.T) {
	now := time.Now().UTC()
	txID := "tx_reconstructed"
	gwOrder := "GW_123"
	payURL := "https://example.com/pay"
	qr := "qr_code_data"
	chainType := vo.ChainTypeTRC
	var usdtAmt uint64 = 5000000
	recvAddr := "TReceiverAddress12345678901234567"
	rate := 7.1
	txHash := "tx_hash_abc"
	var blockNum uint64 = 999
	confirmedAt := now.Add(-5 * time.Minute)

	params := PaymentReconstructParams{
		ID:               42,
		OrderNo:          "PAY_20240101_ABC",
		SubscriptionID:   10,
		UserID:           20,
		Amount:           vo.NewMoney(5000, "CNY"),
		PaymentMethod:    vo.PaymentMethodUSDTTRC,
		Status:           vo.PaymentStatusPaid,
		GatewayOrderNo:   &gwOrder,
		TransactionID:    &txID,
		PaymentURL:       &payURL,
		QRCode:           &qr,
		PaidAt:           &now,
		ExpiredAt:        now.Add(30 * time.Minute),
		ChainType:        &chainType,
		USDTAmountRaw:    &usdtAmt,
		ReceivingAddress: &recvAddr,
		ExchangeRate:     &rate,
		TxHash:           &txHash,
		BlockNumber:      &blockNum,
		ConfirmedAt:      &confirmedAt,
		Metadata:         map[string]interface{}{"key": "value"},
		Version:          3,
		CreatedAt:        now.Add(-1 * time.Hour),
		UpdatedAt:        now,
	}

	p := ReconstructPaymentWithParams(params)
	require.NotNil(t, p)

	assert.Equal(t, uint(42), p.ID())
	assert.Equal(t, "PAY_20240101_ABC", p.OrderNo())
	assert.Equal(t, uint(10), p.SubscriptionID())
	assert.Equal(t, uint(20), p.UserID())
	assert.True(t, p.Amount().Equals(vo.NewMoney(5000, "CNY")))
	assert.Equal(t, vo.PaymentMethodUSDTTRC, p.PaymentMethod())
	assert.Equal(t, vo.PaymentStatusPaid, p.Status())
	require.NotNil(t, p.GatewayOrderNo())
	assert.Equal(t, gwOrder, *p.GatewayOrderNo())
	require.NotNil(t, p.TransactionID())
	assert.Equal(t, txID, *p.TransactionID())
	require.NotNil(t, p.PaymentURL())
	assert.Equal(t, payURL, *p.PaymentURL())
	require.NotNil(t, p.QRCode())
	assert.Equal(t, qr, *p.QRCode())
	require.NotNil(t, p.PaidAt())
	assert.Equal(t, now, *p.PaidAt())
	assert.Equal(t, now.Add(30*time.Minute), p.ExpiredAt())
	require.NotNil(t, p.ChainType())
	assert.Equal(t, chainType, *p.ChainType())
	require.NotNil(t, p.USDTAmountRaw())
	assert.Equal(t, usdtAmt, *p.USDTAmountRaw())
	require.NotNil(t, p.ReceivingAddress())
	assert.Equal(t, recvAddr, *p.ReceivingAddress())
	require.NotNil(t, p.ExchangeRate())
	assert.Equal(t, rate, *p.ExchangeRate())
	require.NotNil(t, p.TxHash())
	assert.Equal(t, txHash, *p.TxHash())
	require.NotNil(t, p.BlockNumber())
	assert.Equal(t, blockNum, *p.BlockNumber())
	require.NotNil(t, p.ConfirmedAt())
	assert.Equal(t, confirmedAt, *p.ConfirmedAt())
	assert.Equal(t, "value", p.Metadata()["key"])
	assert.Equal(t, 3, p.Version())
}

func TestPayment_VersionIncrementsCorrectly(t *testing.T) {
	p := validPayment(t)
	assert.Equal(t, 0, p.Version())

	// Each state transition should increment version by 1
	require.NoError(t, p.MarkAsPaid("tx_1"))
	assert.Equal(t, 1, p.Version())

	// Already paid - idempotent, version should not change
	require.NoError(t, p.MarkAsPaid("tx_2"))
	assert.Equal(t, 1, p.Version())
}

func TestPayment_USDTConfirmSetsTransactionID(t *testing.T) {
	// Verify that ConfirmUSDTTransaction sets transactionID to the txHash
	p := validUSDTPayment(t)
	txHash := "0xdeadbeef"

	require.NoError(t, p.ConfirmUSDTTransaction(txHash, 42))

	require.NotNil(t, p.TransactionID())
	assert.Equal(t, txHash, *p.TransactionID())
}
