package valueobjects

import (
	"errors"
	"time"
)

// PlanPricing represents the price for a specific billing cycle
// It's a value object that encapsulates pricing details for a subscription plan
type PlanPricing struct {
	id           uint
	planID       uint
	billingCycle BillingCycle
	price        uint64
	currency     string
	isActive     bool
	createdAt    time.Time
	updatedAt    time.Time
}

var (
	// ErrInvalidPrice is returned when price is zero or negative
	ErrInvalidPrice = errors.New("price must be greater than zero")
	// ErrInvalidCurrency is returned when currency code is invalid
	ErrInvalidCurrency = errors.New("invalid currency code")
	// ErrInvalidPlanID is returned when plan ID is zero
	ErrInvalidPlanID = errors.New("plan ID must be greater than zero")
)

// Valid currency codes
var validCurrencies = map[string]bool{
	"CNY": true,
	"USD": true,
	"EUR": true,
	"GBP": true,
	"JPY": true,
}

// NewPlanPricing creates a new PlanPricing value object
// Returns error if validation fails
func NewPlanPricing(planID uint, cycle BillingCycle, price uint64, currency string) (*PlanPricing, error) {
	if planID == 0 {
		return nil, ErrInvalidPlanID
	}

	if price == 0 {
		return nil, ErrInvalidPrice
	}

	if !validCurrencies[currency] {
		return nil, ErrInvalidCurrency
	}

	if !cycle.IsValid() {
		return nil, ErrInvalidBillingCycle
	}

	now := time.Now()
	return &PlanPricing{
		planID:       planID,
		billingCycle: cycle,
		price:        price,
		currency:     currency,
		isActive:     true,
		createdAt:    now,
		updatedAt:    now,
	}, nil
}

// ReconstructPlanPricing reconstructs a PlanPricing from persistence layer
// Used by repository when loading from database
func ReconstructPlanPricing(id, planID uint, cycle BillingCycle, price uint64, currency string, isActive bool, createdAt, updatedAt time.Time) *PlanPricing {
	return &PlanPricing{
		id:           id,
		planID:       planID,
		billingCycle: cycle,
		price:        price,
		currency:     currency,
		isActive:     isActive,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
	}
}

// Getters

// ID returns the pricing ID
func (p *PlanPricing) ID() uint {
	return p.id
}

// PlanID returns the associated plan ID
func (p *PlanPricing) PlanID() uint {
	return p.planID
}

// BillingCycle returns the billing cycle
func (p *PlanPricing) BillingCycle() BillingCycle {
	return p.billingCycle
}

// Price returns the price in smallest currency unit (cents)
func (p *PlanPricing) Price() uint64 {
	return p.price
}

// Currency returns the currency code
func (p *PlanPricing) Currency() string {
	return p.currency
}

// IsActive returns whether this pricing is active
func (p *PlanPricing) IsActive() bool {
	return p.isActive
}

// CreatedAt returns the creation timestamp
func (p *PlanPricing) CreatedAt() time.Time {
	return p.createdAt
}

// UpdatedAt returns the last update timestamp
func (p *PlanPricing) UpdatedAt() time.Time {
	return p.updatedAt
}

// Business methods

// UpdatePrice updates the price
// Returns error if new price is invalid
func (p *PlanPricing) UpdatePrice(newPrice uint64) error {
	if newPrice == 0 {
		return ErrInvalidPrice
	}
	p.price = newPrice
	p.updatedAt = time.Now()
	return nil
}

// Activate activates this pricing option
func (p *PlanPricing) Activate() {
	p.isActive = true
	p.updatedAt = time.Now()
}

// Deactivate deactivates this pricing option
func (p *PlanPricing) Deactivate() {
	p.isActive = false
	p.updatedAt = time.Now()
}

// Equals checks if two PlanPricing objects are equal
// Two pricings are equal if they have the same plan ID, billing cycle, price, and currency
func (p *PlanPricing) Equals(other *PlanPricing) bool {
	if other == nil {
		return false
	}
	return p.planID == other.planID &&
		p.billingCycle == other.billingCycle &&
		p.price == other.price &&
		p.currency == other.currency &&
		p.isActive == other.isActive
}

// Validate validates the pricing invariants
func (p *PlanPricing) Validate() error {
	if p.planID == 0 {
		return ErrInvalidPlanID
	}
	if p.price == 0 {
		return ErrInvalidPrice
	}
	if !validCurrencies[p.currency] {
		return ErrInvalidCurrency
	}
	if !p.billingCycle.IsValid() {
		return ErrInvalidBillingCycle
	}
	return nil
}
