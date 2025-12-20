package dto

import (
	"fmt"
	"time"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/domain/user"
	"github.com/orris-inc/orris/internal/shared/mapper"
)

// SubscriptionUserDTO represents embedded user information in subscription responses
type SubscriptionUserDTO struct {
	SID   string `json:"id"`    // Stripe-style ID: usr_xxx
	Email string `json:"email"` // User's email address
	Name  string `json:"name"`  // User's display name
}

type SubscriptionDTO struct {
	SID                string               `json:"id"`             // Stripe-style ID: sub_xxx
	UserSID            string               `json:"user_id"`        // User's Stripe-style ID (kept for backward compatibility)
	User               *SubscriptionUserDTO `json:"user,omitempty"` // Embedded user information
	UUID               string               `json:"uuid"`           // UUID for internal use (kept for backward compatibility)
	LinkToken          string               `json:"link_token"`
	SubscribeURL       string               `json:"subscribe_url"`
	Plan               *PlanDTO             `json:"plan,omitempty"`
	Status             string               `json:"status"`
	StartDate          time.Time            `json:"start_date"`
	EndDate            time.Time            `json:"end_date"`
	AutoRenew          bool                 `json:"auto_renew"`
	CurrentPeriodStart time.Time            `json:"current_period_start"`
	CurrentPeriodEnd   time.Time            `json:"current_period_end"`
	IsExpired          bool                 `json:"is_expired"`
	IsActive           bool                 `json:"is_active"`
	CancelledAt        *time.Time           `json:"cancelled_at,omitempty"`
	CancelReason       *string              `json:"cancel_reason,omitempty"`
	CreatedAt          time.Time            `json:"created_at"`
	UpdatedAt          time.Time            `json:"updated_at"`
}

type PlanDTO struct {
	SID          string                 `json:"id"` // Stripe-style ID: plan_xxx
	Name         string                 `json:"name"`
	Slug         string                 `json:"slug"`
	Description  string                 `json:"description"`
	Status       string                 `json:"status"`
	PlanType     string                 `json:"plan_type"` // Plan type: node or forward
	Limits       map[string]interface{} `json:"limits"`
	APIRateLimit uint                   `json:"api_rate_limit"`
	MaxUsers     uint                   `json:"max_users"`
	MaxProjects  uint                   `json:"max_projects"`
	IsPublic     bool                   `json:"is_public"`
	SortOrder    int                    `json:"sort_order"`
	Pricings     []*PricingOptionDTO    `json:"pricings"` // Multiple pricing options for different billing cycles
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// PricingOptionDTO represents a single pricing option for a specific billing cycle
type PricingOptionDTO struct {
	BillingCycle string `json:"billing_cycle"` // weekly, monthly, quarterly, semi_annual, yearly, lifetime
	Price        uint64 `json:"price"`         // Price in smallest currency unit (cents)
	Currency     string `json:"currency"`      // Currency code: CNY, USD, EUR, GBP, JPY
	IsActive     bool   `json:"is_active"`     // Whether this pricing option is currently available
}

// PricingOptionInput represents input for creating/updating a pricing option
type PricingOptionInput struct {
	BillingCycle string `json:"billing_cycle" binding:"required"`
	Price        uint64 `json:"price" binding:"required"`
	Currency     string `json:"currency" binding:"required"`
	IsActive     bool   `json:"is_active"`
}

type SubscriptionTokenDTO struct {
	SID             string     `json:"id"`              // Stripe-style ID: stoken_xxx
	SubscriptionSID string     `json:"subscription_id"` // Subscription's Stripe-style ID
	Name            string     `json:"name"`
	Prefix          string     `json:"prefix"`
	Scope           string     `json:"scope"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	LastUsedAt      *time.Time `json:"last_used_at,omitempty"`
	UsageCount      uint64     `json:"usage_count"`
	IsActive        bool       `json:"is_active"`
	CreatedAt       time.Time  `json:"created_at"`
}

var (
	SubscriptionMapper = mapper.New(
		func(sub *subscription.Subscription) *SubscriptionDTO {
			return toSubscriptionDTOInternal(sub, nil, nil, "")
		},
		func(dto *SubscriptionDTO) *subscription.Subscription {
			return nil
		},
	)

	PlanMapper = mapper.New(
		ToPlanDTO,
		func(dto *PlanDTO) *subscription.Plan {
			return nil
		},
	)

	SubscriptionTokenMapper = mapper.New(
		ToSubscriptionTokenDTO,
		func(dto *SubscriptionTokenDTO) *subscription.SubscriptionToken {
			return nil
		},
	)
)

// ToSubscriptionUserDTO converts a domain user to embedded user DTO for subscription responses.
func ToSubscriptionUserDTO(u *user.User) *SubscriptionUserDTO {
	if u == nil {
		return nil
	}

	return &SubscriptionUserDTO{
		SID:   u.SID(),
		Email: u.Email().String(),
		Name:  u.Name().DisplayName(),
	}
}

// ToSubscriptionDTO converts a domain subscription to DTO with subscribe URL.
// baseURL is used to construct the full subscribe URL (e.g., "https://api.example.com").
// u is the subscription owner, can be nil if user info is not available.
func ToSubscriptionDTO(sub *subscription.Subscription, plan *subscription.Plan, u *user.User, baseURL string) *SubscriptionDTO {
	return toSubscriptionDTOInternal(sub, plan, u, baseURL)
}

func toSubscriptionDTOInternal(sub *subscription.Subscription, plan *subscription.Plan, u *user.User, baseURL string) *SubscriptionDTO {
	if sub == nil {
		return nil
	}

	// Build subscribe URL: {baseURL}/s/{link_token}
	// Using LinkToken instead of UUID for better security (256 bits vs 122 bits)
	subscribeURL := ""
	if baseURL != "" && sub.LinkToken() != "" {
		subscribeURL = fmt.Sprintf("%s/s/%s", baseURL, sub.LinkToken())
	}

	// Set user SID and embedded user info from the user object
	userSID := ""
	var userDTO *SubscriptionUserDTO
	if u != nil {
		userSID = u.SID()
		userDTO = ToSubscriptionUserDTO(u)
	}

	dto := &SubscriptionDTO{
		SID:                sub.SID(),
		UserSID:            userSID,
		User:               userDTO,
		UUID:               sub.UUID(),
		LinkToken:          sub.LinkToken(),
		SubscribeURL:       subscribeURL,
		Status:             sub.Status().String(),
		StartDate:          sub.StartDate(),
		EndDate:            sub.EndDate(),
		AutoRenew:          sub.AutoRenew(),
		CurrentPeriodStart: sub.CurrentPeriodStart(),
		CurrentPeriodEnd:   sub.CurrentPeriodEnd(),
		IsExpired:          sub.IsExpired(),
		IsActive:           sub.IsActive(),
		CancelledAt:        sub.CancelledAt(),
		CancelReason:       sub.CancelReason(),
		CreatedAt:          sub.CreatedAt(),
		UpdatedAt:          sub.UpdatedAt(),
	}

	if plan != nil {
		dto.Plan = ToPlanDTO(plan)
	}

	return dto
}

// ToPlanDTO converts a domain plan to DTO
func ToPlanDTO(plan *subscription.Plan) *PlanDTO {
	if plan == nil {
		return nil
	}

	var limits map[string]interface{}
	if plan.Features() != nil {
		limits = plan.Features().Limits
	}

	return &PlanDTO{
		SID:          plan.SID(),
		Name:         plan.Name(),
		Slug:         plan.Slug(),
		Description:  plan.Description(),
		Status:       string(plan.Status()),
		PlanType:     plan.PlanType().String(),
		Limits:       limits,
		APIRateLimit: plan.APIRateLimit(),
		MaxUsers:     plan.MaxUsers(),
		MaxProjects:  plan.MaxProjects(),
		IsPublic:     plan.IsPublic(),
		SortOrder:    plan.SortOrder(),
		CreatedAt:    plan.CreatedAt(),
		UpdatedAt:    plan.UpdatedAt(),
	}
}

func ToSubscriptionTokenDTO(token *subscription.SubscriptionToken) *SubscriptionTokenDTO {
	if token == nil {
		return nil
	}

	// Subscription SID will be set by the use case layer
	subscriptionSID := ""

	return &SubscriptionTokenDTO{
		SID:             token.SID(),
		SubscriptionSID: subscriptionSID,
		Name:            token.Name(),
		Prefix:          token.Prefix(),
		Scope:           token.Scope().String(),
		ExpiresAt:       token.ExpiresAt(),
		LastUsedAt:      token.LastUsedAt(),
		UsageCount:      token.UsageCount(),
		IsActive:        token.IsActive(),
		CreatedAt:       token.CreatedAt(),
	}
}
