package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/domain/resource"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/errors"
)

// ValidateNodeAccess checks if the node belongs to the subscription's resource groups.
// This prevents users from accessing nodes outside their subscription scope.
func ValidateNodeAccess(
	ctx context.Context,
	subscriptionID uint,
	nodeEntity *node.Node,
	subscriptionRepo subscription.SubscriptionRepository,
	resourceGroupRepo resource.Repository,
) error {
	// Get subscription to find plan ID
	sub, err := subscriptionRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %w", err)
	}
	if sub == nil {
		return errors.NewNotFoundError("subscription", fmt.Sprintf("%d", subscriptionID))
	}

	// Get resource groups for the subscription's plan
	resourceGroups, err := resourceGroupRepo.GetByPlanID(ctx, sub.PlanID())
	if err != nil {
		return fmt.Errorf("failed to get resource groups: %w", err)
	}

	// If plan has no resource groups (e.g., forward plan), node_id cannot be specified
	if len(resourceGroups) == 0 {
		return errors.NewValidationError("node_id cannot be specified for this subscription plan")
	}

	// Build set of allowed group IDs
	allowedGroupIDs := make(map[uint]bool)
	for _, rg := range resourceGroups {
		allowedGroupIDs[rg.ID()] = true
	}

	// Check if node belongs to at least one of the allowed groups
	nodeGroupIDs := nodeEntity.GroupIDs()
	for _, gid := range nodeGroupIDs {
		if allowedGroupIDs[gid] {
			return nil // Node belongs to an allowed group
		}
	}

	return errors.NewForbiddenError("node does not belong to subscription's resource groups")
}
