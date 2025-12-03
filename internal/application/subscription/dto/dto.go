package dto

import (
	"time"

	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/mapper"
)

type SubscriptionDTO struct {
	ID                 uint
	UserID             uint
	Plan               *SubscriptionPlanDTO
	Status             string
	StartDate          time.Time
	EndDate            time.Time
	AutoRenew          bool
	CurrentPeriodStart time.Time
	CurrentPeriodEnd   time.Time
	IsExpired          bool
	IsActive           bool
	CancelledAt        *time.Time
	CancelReason       *string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type SubscriptionPlanDTO struct {
	ID           uint
	Name         string
	Slug         string
	Description  string
	Price        uint64 // Deprecated: use Pricings array, kept for backward compatibility
	Currency     string
	BillingCycle string // Deprecated: use Pricings array, kept for backward compatibility
	TrialDays    int
	Status       string
	Features     []string
	Limits       map[string]interface{}
	APIRateLimit uint
	MaxUsers     uint
	MaxProjects  uint
	IsPublic     bool
	SortOrder    int
	Pricings     []*PricingOptionDTO `json:"pricings,omitempty"` // Multiple pricing options for different billing cycles
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// PricingOptionDTO represents a single pricing option for a specific billing cycle
type PricingOptionDTO struct {
	BillingCycle string `json:"billing_cycle"` // weekly, monthly, quarterly, semi_annual, yearly, lifetime
	Price        uint64 `json:"price"`         // Price in smallest currency unit (cents)
	Currency     string `json:"currency"`      // Currency code: CNY, USD, EUR, GBP, JPY
	IsActive     bool   `json:"is_active"`     // Whether this pricing option is currently available
}

type SubscriptionTokenDTO struct {
	ID             uint
	SubscriptionID uint
	Name           string
	Prefix         string
	Scope          string
	ExpiresAt      *time.Time
	LastUsedAt     *time.Time
	UsageCount     uint64
	IsActive       bool
	CreatedAt      time.Time
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
