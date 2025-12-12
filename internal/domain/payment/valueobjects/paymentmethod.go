package valueobjects

import "fmt"

type PaymentMethod string

const (
	PaymentMethodAlipay PaymentMethod = "alipay"
	PaymentMethodWechat PaymentMethod = "wechat"
	PaymentMethodStripe PaymentMethod = "stripe"
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
	case PaymentMethodAlipay, PaymentMethodWechat, PaymentMethodStripe:
		return true
	default:
		return false
	}
}

func (pm PaymentMethod) String() string {
	return string(pm)
}
