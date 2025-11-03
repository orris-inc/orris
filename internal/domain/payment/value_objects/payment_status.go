package value_objects

type PaymentStatus string

const (
	PaymentStatusPending  PaymentStatus = "pending"
	PaymentStatusPaid     PaymentStatus = "paid"
	PaymentStatusFailed   PaymentStatus = "failed"
	PaymentStatusExpired  PaymentStatus = "expired"
	PaymentStatusRefunded PaymentStatus = "refunded"
)

func (s PaymentStatus) IsValid() bool {
	switch s {
	case PaymentStatusPending, PaymentStatusPaid, PaymentStatusFailed, PaymentStatusExpired, PaymentStatusRefunded:
		return true
	default:
		return false
	}
}

func (s PaymentStatus) IsPaid() bool {
	return s == PaymentStatusPaid
}

func (s PaymentStatus) IsPending() bool {
	return s == PaymentStatusPending
}

func (s PaymentStatus) IsFinal() bool {
	return s == PaymentStatusPaid || s == PaymentStatusFailed || s == PaymentStatusExpired || s == PaymentStatusRefunded
}

func (s PaymentStatus) String() string {
	return string(s)
}
