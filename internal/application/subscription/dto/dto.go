package dto

import (
	"time"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/mapper"
)

type SubscriptionDTO struct {
	ID                 uint                 `json:"id"`
	UserID             uint                 `json:"user_id"`
	Plan               *SubscriptionPlanDTO `json:"plan,omitempty"`
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

type SubscriptionPlanDTO struct {
	ID           uint                   `json:"id"`
	Name         string                 `json:"name"`
	Slug         string                 `json:"slug"`
	Description  string                 `json:"description"`
	Price        uint64                 `json:"price"`        // Deprecated: use Pricings array, kept for backward compatibility
	Currency     string                 `json:"currency"`     // Deprecated: use Pricings array, kept for backward compatibility
	BillingCycle string                 `json:"billing_cycle"` // Deprecated: use Pricings array, kept for backward compatibility
	TrialDays    int                    `json:"trial_days"`
	Status       string                 `json:"status"`
	Features     []string               `json:"features"`
	Limits       map[string]interface{} `json:"limits"`
	APIRateLimit uint                   `json:"api_rate_limit"`
	MaxUsers     uint                   `json:"max_users"`
	MaxProjects  uint                   `json:"max_projects"`
	IsPublic     bool                   `json:"is_public"`
	SortOrder    int                    `json:"sort_order"`
	Pricings     []*PricingOptionDTO    `json:"pricings,omitempty"` // Multiple pricing options for different billing cycles
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

type SubscriptionTokenDTO struct {
	ID             uint       `json:"id"`
	SubscriptionID uint       `json:"subscription_id"`
	Name           string     `json:"name"`
	Prefix         string     `json:"prefix"`
	Scope          string     `json:"scope"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	LastUsedAt     *time.Time `json:"last_used_at,omitempty"`
	UsageCount     uint64     `json:"usage_count"`
	IsActive       bool       `json:"is_active"`
	CreatedAt      time.Time  `json:"created_at"`
}

var (
	SubscriptionMapper = mapper.New(
		func(sub *subscription.Subscription) *SubscriptionDTO {
			return toSubscriptionDTOInternal(sub, nil)
		},
		func(dto *SubscriptionDTO) *subscription.Subscription {
			return nil
		},
	)

	SubscriptionPlanMapper = mapper.New(
		ToSubscriptionPlanDTO,
		func(dto *SubscriptionPlanDTO) *subscription.SubscriptionPlan {
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

func ToSubscriptionDTO(sub *subscription.Subscription, plan *subscription.SubscriptionPlan) *SubscriptionDTO {
	return toSubscriptionDTOInternal(sub, plan)
}

func toSubscriptionDTOInternal(sub *subscription.Subscription, plan *subscription.SubscriptionPlan) *SubscriptionDTO {
	if sub == nil {
		return nil
	}

	dto := &SubscriptionDTO{
		ID:                 sub.ID(),
		UserID:             sub.UserID(),
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
		dto.Plan = ToSubscriptionPlanDTO(plan)
	}

	return dto
}

func ToSubscriptionPlanDTO(plan *subscription.SubscriptionPlan) *SubscriptionPlanDTO {
	if plan == nil {
		return nil
	}

	var features []string
	var limits map[string]interface{}

	if plan.Features() != nil {
		features = plan.Features().Features
		limits = plan.Features().Limits
	}

	return &SubscriptionPlanDTO{
		ID:           plan.ID(),
		Name:         plan.Name(),
		Slug:         plan.Slug(),
		Description:  plan.Description(),
		Price:        plan.Price(),
		Currency:     plan.Currency(),
		BillingCycle: plan.BillingCycle().String(),
		TrialDays:    plan.TrialDays(),
		Status:       string(plan.Status()),
		Features:     features,
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

	return &SubscriptionTokenDTO{
		ID:             token.ID(),
		SubscriptionID: token.SubscriptionID(),
		Name:           token.Name(),
		Prefix:         token.Prefix(),
		Scope:          token.Scope().String(),
		ExpiresAt:      token.ExpiresAt(),
		LastUsedAt:     token.LastUsedAt(),
		UsageCount:     token.UsageCount(),
		IsActive:       token.IsActive(),
		CreatedAt:      token.CreatedAt(),
	}
}
