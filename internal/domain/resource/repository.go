package resource

import "context"

// Repository defines the interface for resource group persistence operations
type Repository interface {
	// Create creates a new resource group
	Create(ctx context.Context, group *ResourceGroup) error

	// Update updates an existing resource group
	Update(ctx context.Context, group *ResourceGroup) error

	// Delete soft deletes a resource group by ID
	Delete(ctx context.Context, id uint) error

	// GetByID retrieves a resource group by ID
	GetByID(ctx context.Context, id uint) (*ResourceGroup, error)

	// GetByIDs retrieves resource groups by their IDs
	GetByIDs(ctx context.Context, ids []uint) ([]*ResourceGroup, error)

	// GetBySID retrieves a resource group by Stripe-style ID
	GetBySID(ctx context.Context, sid string) (*ResourceGroup, error)

	// GetBySIDs retrieves resource groups by their Stripe-style IDs
	// Returns a map from SID to ResourceGroup for efficient lookup
	GetBySIDs(ctx context.Context, sids []string) (map[string]*ResourceGroup, error)

	// GetSIDsByIDs retrieves a map of resource group IDs to their SIDs
	GetSIDsByIDs(ctx context.Context, ids []uint) (map[uint]string, error)

	// GetByPlanID retrieves all resource groups for a plan
	GetByPlanID(ctx context.Context, planID uint) ([]*ResourceGroup, error)

	// GetByPlanIDs retrieves all resource groups for multiple plans
	// Returns a map from planID to list of ResourceGroups
	GetByPlanIDs(ctx context.Context, planIDs []uint) (map[uint][]*ResourceGroup, error)

	// List retrieves resource groups with optional filters
	List(ctx context.Context, filter ListFilter) ([]*ResourceGroup, int64, error)

	// ExistsByName checks if a resource group with the given name exists
	ExistsByName(ctx context.Context, name string) (bool, error)
}

// ListFilter defines the filter options for listing resource groups
type ListFilter struct {
	PlanID   *uint
	Status   *GroupStatus
	Search   string
	Page     int
	PageSize int
}
