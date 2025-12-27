package subscription

import (
	"fmt"
	"time"

	vo "github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/id"
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
	status       PlanStatus
	planType     vo.PlanType
	features     *vo.PlanFeatures
	apiRateLimit uint
	maxUsers     uint
	maxProjects  uint
	nodeLimit    *int // maximum number of user nodes (nil or 0 = unlimited)
	isPublic     bool
	sortOrder    int
	metadata     map[string]interface{}
	version      int
	createdAt    time.Time
	updatedAt    time.Time
}

func NewPlan(name, slug, description string, planType vo.PlanType) (*Plan, error) {

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
	if !planType.IsValid() {
		return nil, fmt.Errorf("invalid plan type: %s", planType)
	}

	// Generate Stripe-style SID
	sid, err := id.NewPlanID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate SID: %w", err)
	}

	now := biztime.NowUTC()
	return &Plan{
		sid:          sid,
		name:         name,
		slug:         slug,
		description:  description,
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

func ReconstructPlan(id uint, sid string, name, slug, description string,
	status string, planType string, features *vo.PlanFeatures,
	apiRateLimit, maxUsers, maxProjects uint, nodeLimit *int, isPublic bool, sortOrder int,
	metadata map[string]interface{}, version int,
	createdAt, updatedAt time.Time) (*Plan, error) {

	if id == 0 {
		return nil, fmt.Errorf("plan ID cannot be zero")
	}
	if sid == "" {
		return nil, fmt.Errorf("plan SID is required")
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
		sid:          sid,
		name:         name,
		slug:         slug,
		description:  description,
		status:       planStatus,
		planType:     pt,
		features:     features,
		apiRateLimit: apiRateLimit,
		maxUsers:     maxUsers,
		maxProjects:  maxProjects,
		nodeLimit:    nodeLimit,
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

// NodeLimit returns the maximum number of user nodes (nil or 0 = unlimited)
func (p *Plan) NodeLimit() *int {
	return p.nodeLimit
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
	p.updatedAt = biztime.NowUTC()
	p.version++
	return nil
}

func (p *Plan) Deactivate() error {
	if p.status == PlanStatusInactive {
		return nil
	}
	p.status = PlanStatusInactive
	p.updatedAt = biztime.NowUTC()
	p.version++
	return nil
}

func (p *Plan) UpdateDescription(description string) {
	p.description = description
	p.updatedAt = biztime.NowUTC()
	p.version++
}

func (p *Plan) UpdateFeatures(features *vo.PlanFeatures) error {
	if features == nil {
		return fmt.Errorf("features cannot be nil")
	}
	p.features = features
	p.updatedAt = biztime.NowUTC()
	p.version++
	return nil
}

func (p *Plan) SetAPIRateLimit(limit uint) error {
	if limit == 0 {
		return fmt.Errorf("API rate limit must be greater than 0")
	}
	p.apiRateLimit = limit
	p.updatedAt = biztime.NowUTC()
	p.version++
	return nil
}

func (p *Plan) SetMaxUsers(max uint) {
	p.maxUsers = max
	p.updatedAt = biztime.NowUTC()
	p.version++
}

func (p *Plan) SetMaxProjects(max uint) {
	p.maxProjects = max
	p.updatedAt = biztime.NowUTC()
	p.version++
}

// SetNodeLimit sets the maximum number of user nodes
func (p *Plan) SetNodeLimit(limit *int) {
	p.nodeLimit = limit
	p.updatedAt = biztime.NowUTC()
	p.version++
}

func (p *Plan) SetSortOrder(order int) {
	p.sortOrder = order
	p.updatedAt = biztime.NowUTC()
	p.version++
}

func (p *Plan) SetPublic(isPublic bool) {
	p.isPublic = isPublic
	p.updatedAt = biztime.NowUTC()
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

// HasNodeLimit returns true if this plan has a node limit
func (p *Plan) HasNodeLimit() bool {
	return p.nodeLimit != nil && *p.nodeLimit > 0
}

// GetNodeLimit returns the node limit, or 0 if unlimited
func (p *Plan) GetNodeLimit() int {
	if p.nodeLimit == nil {
		return 0
	}
	return *p.nodeLimit
}
