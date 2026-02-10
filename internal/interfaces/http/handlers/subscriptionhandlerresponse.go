package handlers

import (
	subdto "github.com/orris-inc/orris/internal/application/subscription/dto"
)

// SubscriptionResponse is a type alias for the application DTO.
// Use explicit struct if API contract diverges from internal DTO.
type SubscriptionResponse = subdto.SubscriptionDTO

// SubscriptionTokenResponse is a type alias for the application DTO.
type SubscriptionTokenResponse = subdto.SubscriptionTokenDTO

// CreateSubscriptionResponse represents the response for subscription creation
type CreateSubscriptionResponse struct {
	Subscription *subdto.SubscriptionDTO      `json:"subscription"`
	Token        *subdto.SubscriptionTokenDTO `json:"token"`
}
