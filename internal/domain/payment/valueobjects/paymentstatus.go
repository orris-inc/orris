package valueobjects

type PaymentStatus string

const (
	PaymentStatusPending PaymentStatus = "pending"
	PaymentStatusPaid    PaymentStatus = "paid"
	PaymentStatusFailed  PaymentStatus = "failed"
	PaymentStatusExpired PaymentStatus = "expired"
)

func (s PaymentStatus) IsValid() bool {
	switch s {
	case PaymentStatusPending, PaymentStatusPaid, PaymentStatusFailed, PaymentStatusExpired:
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
	return s == PaymentStatusPaid || s == PaymentStatusFailed || s == PaymentStatusExpired
}

func (s PaymentStatus) String() string {
	return string(s)
}
