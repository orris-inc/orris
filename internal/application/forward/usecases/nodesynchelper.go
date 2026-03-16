package usecases

import (
	"context"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// collectAffectedNodeIDs returns all node IDs affected by a forward rule change.
// This includes nodes in the rule's resource groups (they serve subscriptions that
// reference this rule) and the rule's target node (if set).
func collectAffectedNodeIDs(ctx context.Context, rule *forward.ForwardRule, nodeRepo node.NodeRepository, log logger.Interface) []uint {
	seen := make(map[uint]struct{})
	var result []uint

	addNode := func(id uint) {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			result = append(result, id)
		}
	}

	// Collect nodes from each resource group the rule belongs to
	if nodeRepo != nil {
		for _, groupID := range rule.GroupIDs() {
			ids, err := nodeRepo.GetIDsByGroupID(ctx, groupID)
			if err != nil {
				log.Warnw("failed to get node IDs for group during rule sync",
					"group_id", groupID,
					"error", err,
				)
				continue
			}
			for _, id := range ids {
				addNode(id)
			}
		}
	}

	// Add the target node
	if nid := rule.TargetNodeID(); nid != nil {
		addNode(*nid)
	}

	return result
}

// mergeUniqueUints merges two uint slices and returns only unique values.
func mergeUniqueUints(a, b []uint) []uint {
	seen := make(map[uint]struct{}, len(a)+len(b))
	var result []uint
	for _, v := range a {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}
	for _, v := range b {
		if _, ok := seen[v]; !ok {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}
	return result
}
