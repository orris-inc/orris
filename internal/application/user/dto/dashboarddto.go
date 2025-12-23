package dto

import (
	"time"
)

// UsageSummary represents aggregated traffic usage
type UsageSummary struct {
	Upload   uint64 `json:"upload"`
	Download uint64 `json:"download"`
	Total    uint64 `json:"total"`
}

// DashboardPlanDTO represents simplified plan info for dashboard
type DashboardPlanDTO struct {
	SID      string                 `json:"id"`
	Name     string                 `json:"name"`
	PlanType string                 `json:"plan_type"`
	Limits   map[string]interface{} `json:"limits,omitempty"`
}

// DashboardSubscriptionDTO represents subscription info with usage for dashboard
type DashboardSubscriptionDTO struct {
	SID                string            `json:"id"`
	Plan               *DashboardPlanDTO `json:"plan,omitempty"`
	Status             string            `json:"status"`
	CurrentPeriodStart time.Time         `json:"current_period_start"`
	CurrentPeriodEnd   time.Time         `json:"current_period_end"`
	IsActive           bool              `json:"is_active"`
	Usage              *UsageSummary     `json:"usage"`
}

// DashboardResponse represents the user dashboard response
type DashboardResponse struct {
	Subscriptions []*DashboardSubscriptionDTO `json:"subscriptions"`
	TotalUsage    *UsageSummary               `json:"total_usage"`
}
