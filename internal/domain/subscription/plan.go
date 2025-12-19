package subscription

import (
	"fmt"
	"time"

	vo "github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
)

type PlanStatus string

const (
	PlanStatusActive   PlanStatus = "active"
	PlanStatusInactive PlanStatus = "inactive"
)

type Plan struct {
	id           uint
	sid          string // Stripe-style ID: plan_xxx
	name         string
	slug         string
	description  string
	trialDays    int
	status       PlanStatus
	planType     vo.PlanType
	features     *vo.PlanFeatures
	apiRateLimit uint
	maxUsers     uint
	maxProjects  uint
	isPublic     bool
	sortOrder    int
	metadata     map[string]interface{}
	version      int
	createdAt    time.Time
	updatedAt    time.Time
}

func NewPlan(name, slug, description string, trialDays int, planType vo.PlanType) (*Plan, error) {

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
	if trialDays < 0 {
		return nil, fmt.Errorf("trial days cannot be negative")
	}
	if !planType.IsValid() {
		return nil, fmt.Errorf("invalid plan type: %s", planType)
	}

	now := time.Now()
	return &Plan{
		name:         name,
		slug:         slug,
		description:  description,
		trialDays:    trialDays,
		status:       PlanStatusActive,
		planType:     planType,
		features:     nil,
		apiRateLimit: 60,
		maxUsers:     0,
		maxProjects:  0,
		isPublic:     true,
		sortOrder:    0,
		metadata:     make(map[string]interface{}),
		version:      1,
		createdAt:    now,
		updatedAt:    now,
	}, nil
}

func ReconstructPlan(id uint, name, slug, description string,
	trialDays int, status string, planType string, features *vo.PlanFeatures,
	apiRateLimit, maxUsers, maxProjects uint, isPublic bool, sortOrder int,
	metadata map[string]interface{}, version int,
	createdAt, updatedAt time.Time) (*Plan, error) {

	if id == 0 {
		return nil, fmt.Errorf("plan ID cannot be zero")
	}

	planStatus := PlanStatus(status)
	if planStatus != PlanStatusActive && planStatus != PlanStatusInactive {
		return nil, fmt.Errorf("invalid plan status: %s", status)
	}

	pt, err := vo.NewPlanType(planType)
	if err != nil {
		return nil, fmt.Errorf("invalid plan type: %w", err)
	}

	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	return &Plan{
		id:           id,
		name:         name,
		slug:         slug,
		description:  description,
		trialDays:    trialDays,
		status:       planStatus,
		planType:     pt,
		features:     features,
		apiRateLimit: apiRateLimit,
		maxUsers:     maxUsers,
		maxProjects:  maxProjects,
		isPublic:     isPublic,
		sortOrder:    sortOrder,
		metadata:     metadata,
		version:      version,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
	}, nil
}

func (p *Plan) ID() uint {
	return p.id
}

// SID returns the Stripe-style ID
func (p *Plan) SID() string {
	return p.sid
}

// SetSID sets the Stripe-style ID (only for persistence layer use)
func (p *Plan) SetSID(sid string) {
	p.sid = sid
}

func (p *Plan) SetID(id uint) error {
	if p.id != 0 {
		return fmt.Errorf("plan ID is already set")
	}
	if id == 0 {
		return fmt.Errorf("plan ID cannot be zero")
	}
	p.id = id
	return nil
}

func (p *Plan) Name() string {
	return p.name
}

func (p *Plan) Slug() string {
	return p.slug
}

func (p *Plan) Description() string {
	return p.description
}

func (p *Plan) TrialDays() int {
	return p.trialDays
}

func (p *Plan) Status() PlanStatus {
	return p.status
}

func (p *Plan) PlanType() vo.PlanType {
	return p.planType
}

func (p *Plan) Features() *vo.PlanFeatures {
	return p.features
}

func (p *Plan) APIRateLimit() uint {
	return p.apiRateLimit
}

func (p *Plan) MaxUsers() uint {
	return p.maxUsers
}

func (p *Plan) MaxProjects() uint {
	return p.maxProjects
}

func (p *Plan) IsPublic() bool {
	return p.isPublic
}

func (p *Plan) SortOrder() int {
	return p.sortOrder
}

func (p *Plan) Metadata() map[string]interface{} {
	return p.metadata
}

func (p *Plan) CreatedAt() time.Time {
	return p.createdAt
}

func (p *Plan) UpdatedAt() time.Time {
	return p.updatedAt
}

// Version returns the aggregate version for optimistic locking
func (p *Plan) Version() int {
	return p.version
}

// IncrementVersion increments the version for optimistic locking
func (p *Plan) IncrementVersion() {
	p.version++
}

func (p *Plan) Activate() error {
	if p.status == PlanStatusActive {
		return nil
	}
	p.status = PlanStatusActive
	p.updatedAt = time.Now()
	p.version++
	return nil
}

func (p *Plan) Deactivate() error {
	if p.status == PlanStatusInactive {
		return nil
	}
	p.status = PlanStatusInactive
	p.updatedAt = time.Now()
	p.version++
	return nil
}

func (p *Plan) UpdateDescription(description string) {
	p.description = description
	p.updatedAt = time.Now()
	p.version++
}

func (p *Plan) UpdateFeatures(features *vo.PlanFeatures) error {
	if features == nil {
		return fmt.Errorf("features cannot be nil")
	}
	p.features = features
	p.updatedAt = time.Now()
	p.version++
	return nil
}

func (p *Plan) SetAPIRateLimit(limit uint) error {
	if limit == 0 {
		return fmt.Errorf("API rate limit must be greater than 0")
	}
	p.apiRateLimit = limit
	p.updatedAt = time.Now()
	p.version++
	return nil
}

func (p *Plan) SetMaxUsers(max uint) {
	p.maxUsers = max
	p.updatedAt = time.Now()
	p.version++
}

func (p *Plan) SetMaxProjects(max uint) {
	p.maxProjects = max
	p.updatedAt = time.Now()
	p.version++
}

func (p *Plan) SetSortOrder(order int) {
	p.sortOrder = order
	p.updatedAt = time.Now()
	p.version++
}

func (p *Plan) SetPublic(isPublic bool) {
	p.isPublic = isPublic
	p.updatedAt = time.Now()
	p.version++
}

func (p *Plan) GetLimit(key string) (interface{}, bool) {
	if p.features == nil {
		return nil, false
	}
	return p.features.GetLimit(key)
}

func (p *Plan) IsActive() bool {
	return p.status == PlanStatusActive
}

// GetTrafficLimit returns the monthly traffic limit in bytes from features
// Returns 0 if unlimited or features not set
func (p *Plan) GetTrafficLimit() (uint64, error) {
	if p.features == nil {
		return 0, nil // unlimited if no features configured
	}
	return p.features.GetTrafficLimit()
}

// IsUnlimitedTraffic checks if the plan has unlimited traffic
func (p *Plan) IsUnlimitedTraffic() bool {
	if p.features == nil {
		return true // unlimited if no features configured
	}
	return p.features.IsUnlimitedTraffic()
}

// HasTrafficRemaining checks if the used traffic is within the plan limit
func (p *Plan) HasTrafficRemaining(usedBytes uint64) (bool, error) {
	if p.features == nil {
		return true, nil // unlimited if no features configured
	}
	return p.features.HasTrafficRemaining(usedBytes)
}
