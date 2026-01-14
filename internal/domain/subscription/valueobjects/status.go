package valueobjects

type SubscriptionStatus string

const (
	StatusInactive       SubscriptionStatus = "inactive"
	StatusPendingPayment SubscriptionStatus = "pending_payment"
	StatusTrialing       SubscriptionStatus = "trialing"
	StatusActive         SubscriptionStatus = "active"
	StatusPastDue        SubscriptionStatus = "past_due"
	StatusSuspended      SubscriptionStatus = "suspended" // Suspended due to traffic limit or admin action
	StatusCancelled      SubscriptionStatus = "cancelled"
	StatusExpired        SubscriptionStatus = "expired"
)

func (s SubscriptionStatus) String() string {
	return string(s)
}

func (s SubscriptionStatus) CanUseService() bool {
	return s == StatusActive || s == StatusTrialing
}

func (s SubscriptionStatus) CanRenew() bool {
	return s == StatusActive || s == StatusPastDue || s == StatusExpired
}

func (s SubscriptionStatus) CanTransitionTo(target SubscriptionStatus) bool {
	transitions := map[SubscriptionStatus][]SubscriptionStatus{
		StatusInactive:       {StatusPendingPayment, StatusActive, StatusTrialing},
		StatusPendingPayment: {StatusActive, StatusInactive, StatusExpired},
		StatusTrialing:       {StatusActive, StatusCancelled, StatusExpired, StatusSuspended},
		StatusActive:         {StatusPastDue, StatusCancelled, StatusExpired, StatusSuspended},
		StatusPastDue:        {StatusActive, StatusCancelled, StatusExpired, StatusSuspended},
		StatusSuspended:      {StatusActive}, // Can be reactivated after resolving the issue
		StatusCancelled:      {},
		StatusExpired:        {StatusActive},
	}

	allowed, exists := transitions[s]
	if !exists {
		return false
	}

	for _, allowedStatus := range allowed {
		if allowedStatus == target {
			return true
		}
	}
	return false
}

var ValidStatuses = map[SubscriptionStatus]bool{
	StatusInactive:       true,
	StatusPendingPayment: true,
	StatusTrialing:       true,
	StatusActive:         true,
	StatusPastDue:        true,
	StatusSuspended:      true,
	StatusCancelled:      true,
	StatusExpired:        true,
}
