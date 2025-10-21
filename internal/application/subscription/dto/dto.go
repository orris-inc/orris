package dto

import (
	"time"

	"orris/internal/domain/subscription"
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
	ID             uint
	Name           string
	Slug           string
	Description    string
	Price          uint64
	Currency       string
	BillingCycle   string
	TrialDays      int
	Status         string
	Features       []string
	Limits         map[string]interface{}
	CustomEndpoint string
	APIRateLimit   uint
	MaxUsers       uint
	MaxProjects    uint
	StorageLimit   uint64
	IsPublic       bool
	SortOrder      int
	CreatedAt      time.Time
	UpdatedAt      time.Time
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

func ToSubscriptionDTO(sub *subscription.Subscription, plan *subscription.SubscriptionPlan) *SubscriptionDTO {
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
		ID:             plan.ID(),
		Name:           plan.Name(),
		Slug:           plan.Slug(),
		Description:    plan.Description(),
		Price:          plan.Price(),
		Currency:       plan.Currency(),
		BillingCycle:   plan.BillingCycle().String(),
		TrialDays:      plan.TrialDays(),
		Status:         string(plan.Status()),
		Features:       features,
		Limits:         limits,
		CustomEndpoint: plan.CustomEndpoint(),
		APIRateLimit:   plan.APIRateLimit(),
		MaxUsers:       plan.MaxUsers(),
		MaxProjects:    plan.MaxProjects(),
		StorageLimit:   plan.StorageLimit(),
		IsPublic:       plan.IsPublic(),
		SortOrder:      plan.SortOrder(),
		CreatedAt:      plan.CreatedAt(),
		UpdatedAt:      plan.UpdatedAt(),
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
