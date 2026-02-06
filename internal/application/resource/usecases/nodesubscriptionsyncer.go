package usecases

import (
	"context"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils/setutil"
)

// NodeSubscriptionSyncer syncs subscriptions to a specific node agent.
// This is used to push subscription updates when resource group membership changes.
type NodeSubscriptionSyncer interface {
	SyncSubscriptionsToNode(ctx context.Context, nodeID uint) error
}

// collectGroupAffectedNodeIDs returns all node IDs affected by the given resource group.
// It merges nodes directly in the group with nodes targeted by the group's forward rules.
func collectGroupAffectedNodeIDs(
	ctx context.Context,
	groupID uint,
	nodeRepo node.NodeRepository,
	ruleRepo forward.Repository,
	log logger.Interface,
) []uint {
	nodeIDSet := setutil.NewUintSet()

	// Nodes directly in this group
	if nodeRepo != nil {
		ids, err := nodeRepo.GetIDsByGroupID(ctx, groupID)
		if err != nil {
			log.Warnw("failed to get node IDs for group sync", "error", err, "group_id", groupID)
		} else {
			for _, id := range ids {
				nodeIDSet.Add(id)
			}
		}
	}

	// Nodes targeted by forward rules in this group
	if ruleRepo != nil {
		rules, err := ruleRepo.ListSystemRulesByGroupIDs(ctx, []uint{groupID})
		if err != nil {
			log.Warnw("failed to list forward rules for group sync", "error", err, "group_id", groupID)
		} else {
			for _, rule := range rules {
				if nid := rule.TargetNodeID(); nid != nil {
					nodeIDSet.Add(*nid)
				}
			}
		}
	}

	return nodeIDSet.ToSlice()
}

// syncSubscriptionsToNodes triggers a subscription sync for each node ID.
// Errors are logged but not propagated.
func syncSubscriptionsToNodes(ctx context.Context, syncer NodeSubscriptionSyncer, nodeIDs []uint, log logger.Interface) {
	if syncer == nil || len(nodeIDs) == 0 {
		return
	}

	for _, nodeID := range nodeIDs {
		if err := syncer.SyncSubscriptionsToNode(ctx, nodeID); err != nil {
			log.Warnw("failed to sync subscriptions to node",
				"node_id", nodeID,
				"error", err,
			)
		}
	}
}
