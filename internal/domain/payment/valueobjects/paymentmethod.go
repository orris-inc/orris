package valueobjects

import "fmt"

type PaymentMethod string

const (
	PaymentMethodAlipay  PaymentMethod = "alipay"
	PaymentMethodWechat  PaymentMethod = "wechat"
	PaymentMethodStripe  PaymentMethod = "stripe"
	PaymentMethodUSDTPOL PaymentMethod = "usdt_pol"
	PaymentMethodUSDTTRC PaymentMethod = "usdt_trc"
)

func NewPaymentMethod(method string) (PaymentMethod, error) {
	pm := PaymentMethod(method)
	if !pm.IsValid() {
		return "", fmt.Errorf("invalid payment method: %s", method)
	}
	return pm, nil
}

func (pm PaymentMethod) IsValid() bool {
	switch pm {
	case PaymentMethodAlipay, PaymentMethodWechat, PaymentMethodStripe,
		PaymentMethodUSDTPOL, PaymentMethodUSDTTRC:
		return true
	default:
		return false
	}
}

// IsUSDT returns true if this payment method is a USDT payment
func (pm PaymentMethod) IsUSDT() bool {
	return pm == PaymentMethodUSDTPOL || pm == PaymentMethodUSDTTRC
}

// ChainType returns the chain type for USDT payments
// Returns empty string for non-USDT payment methods
func (pm PaymentMethod) ChainType() string {
	switch pm {
	case PaymentMethodUSDTPOL:
		return "pol"
	case PaymentMethodUSDTTRC:
		return "trc"
	default:
		return ""
	}
}

func (pm PaymentMethod) String() string {
	return string(pm)
}
