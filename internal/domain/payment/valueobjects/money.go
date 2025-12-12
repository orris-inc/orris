package valueobjects

import "fmt"

type Money struct {
	amountInCents int64
	currency      string
}

func NewMoney(amountInCents int64, currency string) Money {
	if currency == "" {
		currency = "CNY"
	}
	return Money{
		amountInCents: amountInCents,
		currency:      currency,
	}
}

func (m Money) AmountInCents() int64 {
	return m.amountInCents
}

func (m Money) Currency() string {
	return m.currency
}

func (m Money) AmountInYuan() float64 {
	return float64(m.amountInCents) / 100.0
}

func (m Money) Equals(other Money) bool {
	return m.amountInCents == other.amountInCents && m.currency == other.currency
}

func (m Money) IsPositive() bool {
	return m.amountInCents > 0
}

func (m Money) String() string {
	return fmt.Sprintf("%.2f %s", m.AmountInYuan(), m.currency)
}
