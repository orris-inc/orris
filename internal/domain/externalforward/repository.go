package externalforward

import "context"

// Repository defines the interface for external forward rule persistence.
type Repository interface {
	// Create persists a new external forward rule.
	Create(ctx context.Context, rule *ExternalForwardRule) error

	// GetByID retrieves an external forward rule by ID.
	GetByID(ctx context.Context, id uint) (*ExternalForwardRule, error)

	// GetBySID retrieves an external forward rule by SID.
	GetBySID(ctx context.Context, sid string) (*ExternalForwardRule, error)

	// Update updates an existing external forward rule.
	Update(ctx context.Context, rule *ExternalForwardRule) error

	// Delete removes an external forward rule.
	Delete(ctx context.Context, id uint) error

	// ListBySubscriptionID returns all external forward rules for a specific subscription.
	ListBySubscriptionID(ctx context.Context, subscriptionID uint) ([]*ExternalForwardRule, error)

	// ListBySubscriptionIDWithPagination returns external forward rules for a subscription with filtering and pagination.
	ListBySubscriptionIDWithPagination(ctx context.Context, subscriptionID uint, filter ListFilter) ([]*ExternalForwardRule, int64, error)

	// ListEnabledBySubscriptionID returns all enabled external forward rules for a specific subscription.
	ListEnabledBySubscriptionID(ctx context.Context, subscriptionID uint) ([]*ExternalForwardRule, error)

	// ListByUserID returns external forward rules for a specific user with filtering and pagination.
	ListByUserID(ctx context.Context, userID uint, filter ListFilter) ([]*ExternalForwardRule, int64, error)

	// CountBySubscriptionID returns the total count of external forward rules for a specific subscription.
	CountBySubscriptionID(ctx context.Context, subscriptionID uint) (int64, error)

	// UpdateSortOrders batch updates sort_order for multiple rules.
	UpdateSortOrders(ctx context.Context, ruleOrders map[uint]int) error

	// ListWithPagination returns external forward rules with optional filters and pagination (for admin use).
	ListWithPagination(ctx context.Context, filter AdminListFilter) ([]*ExternalForwardRule, int64, error)

	// ListByGroupID returns all external forward rules that belong to the specified resource group.
	// Uses JSON_CONTAINS to check if group_ids array contains the given group ID.
	// Supports pagination when page > 0 and pageSize > 0.
	ListByGroupID(ctx context.Context, groupID uint, page, pageSize int) ([]*ExternalForwardRule, int64, error)

	// ListEnabledByGroupIDs returns all enabled external forward rules for the given resource groups.
	ListEnabledByGroupIDs(ctx context.Context, groupIDs []uint) ([]*ExternalForwardRule, error)

	// GetBySIDs returns external forward rules for the given SIDs.
	// Returns a map of SID -> rule for easy lookup. Missing rules are not included.
	GetBySIDs(ctx context.Context, sids []string) (map[string]*ExternalForwardRule, error)

	// AddGroupIDAtomically adds a group ID to a rule's group_ids array atomically using JSON_ARRAY_APPEND.
	// Returns true if the group ID was added, false if it already exists.
	AddGroupIDAtomically(ctx context.Context, ruleID, groupID uint) (bool, error)

	// RemoveGroupIDAtomically removes a group ID from a rule's group_ids array atomically using JSON_REMOVE.
	// Returns true if the group ID was removed, false if it didn't exist.
	RemoveGroupIDAtomically(ctx context.Context, ruleID, groupID uint) (bool, error)
}

// ListFilter defines the filtering options for listing external forward rules.
type ListFilter struct {
	Page     int
	PageSize int
	Status   string
	OrderBy  string
	Order    string
}

// AdminListFilter defines the filtering options for admin listing external forward rules.
type AdminListFilter struct {
	Page           int
	PageSize       int
	SubscriptionID *uint
	UserID         *uint
	Status         string
	ExternalSource string
	OrderBy        string
	Order          string
}
