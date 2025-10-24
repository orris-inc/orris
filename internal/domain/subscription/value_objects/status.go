package value_objects

type SubscriptionStatus string

const (
	StatusInactive  SubscriptionStatus = "inactive"
	StatusTrialing  SubscriptionStatus = "trialing"
	StatusActive    SubscriptionStatus = "active"
	StatusPastDue   SubscriptionStatus = "past_due"
	StatusCancelled SubscriptionStatus = "cancelled"
	StatusExpired   SubscriptionStatus = "expired"
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
		StatusInactive:  {StatusActive, StatusTrialing},
		StatusTrialing:  {StatusActive, StatusCancelled, StatusExpired},
		StatusActive:    {StatusPastDue, StatusCancelled, StatusExpired},
		StatusPastDue:   {StatusActive, StatusCancelled, StatusExpired},
		StatusCancelled: {},
		StatusExpired:   {StatusActive},
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
	StatusInactive:  true,
	StatusTrialing:  true,
	StatusActive:    true,
	StatusPastDue:   true,
	StatusCancelled: true,
	StatusExpired:   true,
}
