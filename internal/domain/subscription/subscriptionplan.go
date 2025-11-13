package subscription

import (
	"fmt"
	"time"

	vo "orris/internal/domain/subscription/value_objects"
)

type PlanStatus string

const (
	PlanStatusActive   PlanStatus = "active"
	PlanStatusInactive PlanStatus = "inactive"
)

var validCurrencies = map[string]bool{
	"CNY": true,
	"USD": true,
	"EUR": true,
	"GBP": true,
	"JPY": true,
}

type SubscriptionPlan struct {
	id           uint
	name         string
	slug         string
	description  string
	price        uint64
	currency     string
	billingCycle vo.BillingCycle
	trialDays    int
	status       PlanStatus
	features     *vo.PlanFeatures
	apiRateLimit uint
	maxUsers     uint
	maxProjects  uint
	isPublic     bool
	sortOrder    int
	metadata     map[string]interface{}
	createdAt    time.Time
	updatedAt    time.Time
}

func NewSubscriptionPlan(name, slug, description string, price uint64, currency string,
	billingCycle vo.BillingCycle, trialDays int) (*SubscriptionPlan, error) {

	if name == "" {
		return nil, fmt.Errorf("plan name is required")
	}
	if slug == "" {
		return nil, fmt.Errorf("plan slug is required")
	}
	if len(name) > 100 {
		return nil, fmt.Errorf("plan name too long (max 100 characters)")
	}
	if len(slug) > 100 {
		return nil, fmt.Errorf("plan slug too long (max 100 characters)")
	}
	if !validCurrencies[currency] {
		return nil, fmt.Errorf("invalid currency code: %s", currency)
	}
	if !billingCycle.IsValid() {
		return nil, fmt.Errorf("invalid billing cycle: %s", billingCycle)
	}
	if trialDays < 0 {
		return nil, fmt.Errorf("trial days cannot be negative")
	}

	now := time.Now()
	return &SubscriptionPlan{
		name:         name,
		slug:         slug,
		description:  description,
		price:        price,
		currency:     currency,
		billingCycle: billingCycle,
		trialDays:    trialDays,
		status:       PlanStatusActive,
		features:     nil,
		apiRateLimit: 60,
		maxUsers:     0,
		maxProjects:  0,
		isPublic:     true,
		sortOrder:    0,
		metadata:     make(map[string]interface{}),
		createdAt:    now,
		updatedAt:    now,
	}, nil
}

func ReconstructSubscriptionPlan(id uint, name, slug, description string, price uint64,
	currency string, billingCycle vo.BillingCycle, trialDays int, status string,
	features *vo.PlanFeatures, apiRateLimit, maxUsers, maxProjects uint,
	isPublic bool, sortOrder int, metadata map[string]interface{},
	createdAt, updatedAt time.Time) (*SubscriptionPlan, error) {

	if id == 0 {
		return nil, fmt.Errorf("plan ID cannot be zero")
	}

	planStatus := PlanStatus(status)
	if planStatus != PlanStatusActive && planStatus != PlanStatusInactive {
		return nil, fmt.Errorf("invalid plan status: %s", status)
	}

	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	return &SubscriptionPlan{
		id:           id,
		name:         name,
		slug:         slug,
		description:  description,
		price:        price,
		currency:     currency,
		billingCycle: billingCycle,
		trialDays:    trialDays,
		status:       planStatus,
		features:     features,
		apiRateLimit: apiRateLimit,
		maxUsers:     maxUsers,
		maxProjects:  maxProjects,
		isPublic:     isPublic,
		sortOrder:    sortOrder,
		metadata:     metadata,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
	}, nil
}

func (p *SubscriptionPlan) ID() uint {
	return p.id
}

func (p *SubscriptionPlan) SetID(id uint) error {
	if p.id != 0 {
		return fmt.Errorf("plan ID is already set")
	}
	if id == 0 {
		return fmt.Errorf("plan ID cannot be zero")
	}
	p.id = id
	return nil
}

func (p *SubscriptionPlan) Name() string {
	return p.name
}

func (p *SubscriptionPlan) Slug() string {
	return p.slug
}

func (p *SubscriptionPlan) Description() string {
	return p.description
}

func (p *SubscriptionPlan) Price() uint64 {
	return p.price
}

func (p *SubscriptionPlan) Currency() string {
	return p.currency
}

func (p *SubscriptionPlan) BillingCycle() vo.BillingCycle {
	return p.billingCycle
}

func (p *SubscriptionPlan) TrialDays() int {
	return p.trialDays
}

func (p *SubscriptionPlan) Status() PlanStatus {
	return p.status
}

func (p *SubscriptionPlan) Features() *vo.PlanFeatures {
	return p.features
}

func (p *SubscriptionPlan) APIRateLimit() uint {
	return p.apiRateLimit
}

func (p *SubscriptionPlan) MaxUsers() uint {
	return p.maxUsers
}

func (p *SubscriptionPlan) MaxProjects() uint {
	return p.maxProjects
}

func (p *SubscriptionPlan) IsPublic() bool {
	return p.isPublic
}

func (p *SubscriptionPlan) SortOrder() int {
	return p.sortOrder
}

func (p *SubscriptionPlan) Metadata() map[string]interface{} {
	return p.metadata
}

func (p *SubscriptionPlan) CreatedAt() time.Time {
	return p.createdAt
}

func (p *SubscriptionPlan) UpdatedAt() time.Time {
	return p.updatedAt
}

func (p *SubscriptionPlan) Activate() error {
	if p.status == PlanStatusActive {
		return nil
	}
	p.status = PlanStatusActive
	p.updatedAt = time.Now()
	return nil
}

func (p *SubscriptionPlan) Deactivate() error {
	if p.status == PlanStatusInactive {
		return nil
	}
	p.status = PlanStatusInactive
	p.updatedAt = time.Now()
	return nil
}

func (p *SubscriptionPlan) UpdatePrice(price uint64, currency string) error {
	if !validCurrencies[currency] {
		return fmt.Errorf("invalid currency code: %s", currency)
	}
	p.price = price
	p.currency = currency
	p.updatedAt = time.Now()
	return nil
}

func (p *SubscriptionPlan) UpdateDescription(description string) {
	p.description = description
	p.updatedAt = time.Now()
}

func (p *SubscriptionPlan) UpdateFeatures(features *vo.PlanFeatures) error {
	if features == nil {
		return fmt.Errorf("features cannot be nil")
	}
	p.features = features
	p.updatedAt = time.Now()
	return nil
}

func (p *SubscriptionPlan) SetAPIRateLimit(limit uint) error {
	if limit == 0 {
		return fmt.Errorf("API rate limit must be greater than 0")
	}
	p.apiRateLimit = limit
	p.updatedAt = time.Now()
	return nil
}

func (p *SubscriptionPlan) SetMaxUsers(max uint) {
	p.maxUsers = max
	p.updatedAt = time.Now()
}

func (p *SubscriptionPlan) SetMaxProjects(max uint) {
	p.maxProjects = max
	p.updatedAt = time.Now()
}

func (p *SubscriptionPlan) SetSortOrder(order int) {
	p.sortOrder = order
	p.updatedAt = time.Now()
}

func (p *SubscriptionPlan) SetPublic(isPublic bool) {
	p.isPublic = isPublic
	p.updatedAt = time.Now()
}

func (p *SubscriptionPlan) HasFeature(feature string) bool {
	if p.features == nil {
		return false
	}
	return p.features.HasFeature(feature)
}

func (p *SubscriptionPlan) GetLimit(key string) (interface{}, bool) {
	if p.features == nil {
		return nil, false
	}
	return p.features.GetLimit(key)
}

func (p *SubscriptionPlan) IsActive() bool {
	return p.status == PlanStatusActive
}

// GetTrafficLimit returns the monthly traffic limit in bytes from features
// Returns 0 if unlimited or features not set
func (p *SubscriptionPlan) GetTrafficLimit() (uint64, error) {
	if p.features == nil {
		return 0, nil // unlimited if no features configured
	}
	return p.features.GetTrafficLimit()
}

// IsUnlimitedTraffic checks if the plan has unlimited traffic
func (p *SubscriptionPlan) IsUnlimitedTraffic() bool {
	if p.features == nil {
		return true // unlimited if no features configured
	}
	return p.features.IsUnlimitedTraffic()
}

// HasTrafficRemaining checks if the used traffic is within the plan limit
func (p *SubscriptionPlan) HasTrafficRemaining(usedBytes uint64) (bool, error) {
	if p.features == nil {
		return true, nil // unlimited if no features configured
	}
	return p.features.HasTrafficRemaining(usedBytes)
}
