package adapters

import (
	"context"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/application/node/usecases"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/node"
	nodevo "github.com/orris-inc/orris/internal/domain/node/valueobjects"
	"github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/adapters/nodeutil"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils/jsonutil"
	"github.com/orris-inc/orris/internal/shared/utils/logutil"
	"github.com/orris-inc/orris/internal/shared/utils/setutil"
)

// NodeRepository defines the interface for node persistence operations
type NodeRepository interface {
	GetByToken(ctx context.Context, tokenHash string) (*node.Node, error)
}

type NodeRepositoryAdapter struct {
	nodeRepo        NodeRepository
	forwardRuleRepo forward.Repository
	db              *gorm.DB
	logger          logger.Interface
	configLoader    *nodeutil.ConfigLoader
}

func NewNodeRepositoryAdapter(nodeRepo NodeRepository, forwardRuleRepo forward.Repository, db *gorm.DB, logger logger.Interface) *NodeRepositoryAdapter {
	return &NodeRepositoryAdapter{
		nodeRepo:        nodeRepo,
		forwardRuleRepo: forwardRuleRepo,
		db:              db,
		logger:          logger,
		configLoader:    nodeutil.NewConfigLoader(db, logger),
	}
}

func (r *NodeRepositoryAdapter) GetBySubscriptionToken(ctx context.Context, linkToken string, mode string) ([]*usecases.Node, error) {
	var subscriptionModel models.SubscriptionModel

	// Query subscription by link_token
	if err := r.db.WithContext(ctx).
		Where("link_token = ? AND status = ?", linkToken, valueobjects.StatusActive).
		First(&subscriptionModel).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			r.logger.Warnw("active subscription not found", "link_token_prefix", logutil.TruncateForLog(linkToken, 8))
			return []*usecases.Node{}, nil
		}
		r.logger.Errorw("failed to query subscription", "error", err)
		return nil, err
	}

	// Query plan to check plan type
	var planModel models.PlanModel
	if err := r.db.WithContext(ctx).
		Where("id = ?", subscriptionModel.PlanID).
		First(&planModel).Error; err != nil {
		r.logger.Errorw("failed to query plan", "error", err, "plan_id", subscriptionModel.PlanID)
		return nil, err
	}

	// For forward plan, return user's forward rules as nodes
	if planModel.PlanType == "forward" {
		return r.getForwardPlanNodes(ctx, subscriptionModel.ID, subscriptionModel.UserID, mode)
	}

	// For hybrid plan, return both resource group nodes AND user's forward rules
	if planModel.PlanType == "hybrid" {
		return r.getHybridPlanNodes(ctx, subscriptionModel.ID, subscriptionModel.UserID, subscriptionModel.PlanID, mode)
	}

	// For node plan, query nodes via resource_groups
	// Step 1: Get resource group IDs for this plan
	var groupIDs []uint
	if err := r.db.WithContext(ctx).
		Table("resource_groups").
		Where("plan_id = ? AND status = ?", subscriptionModel.PlanID, "active").
		Pluck("id", &groupIDs).Error; err != nil {
		r.logger.Errorw("failed to query resource groups", "error", err, "plan_id", subscriptionModel.PlanID)
		return nil, err
	}

	if len(groupIDs) == 0 {
		r.logger.Infow("no resource groups associated with plan", "plan_id", subscriptionModel.PlanID)
		return []*usecases.Node{}, nil
	}

	// Step 2: Get node IDs that belong to these resource groups
	// Use JSON_OVERLAPS to check if node's group_ids array contains any of the target group IDs
	var nodeIDs []uint
	groupIDsJSON := jsonutil.UintSliceToJSONArray(groupIDs)
	if err := r.db.WithContext(ctx).
		Table("nodes").
		Where("JSON_OVERLAPS(group_ids, ?)", groupIDsJSON).
		Pluck("id", &nodeIDs).Error; err != nil {
		r.logger.Errorw("failed to query nodes by group", "error", err, "group_ids", groupIDs)
		return nil, err
	}

	// Query nodes manually (no foreign key associations)
	// nodeIDs may be empty if resource groups only contain forward rules
	var originNodes []*usecases.Node
	nodeMap := make(map[uint]*usecases.Node)

	if len(nodeIDs) > 0 {
		nodeModels, _, err := r.queryActiveNodes(ctx, nodeIDs)
		if err != nil {
			return nil, err
		}

		// Build origin nodes with configs
		originNodes = r.buildNodesWithConfigs(ctx, nodeModels)

		// Build node map for quick lookup by getForwardedNodes
		for _, node := range originNodes {
			nodeMap[node.ID] = node
		}
	}

	// Query forward rules that belong to the resource groups to generate additional subscription entries
	// Rules are selected by group membership, target nodes can be outside the resource groups
	// This includes both regular forward rules and external forward rules (rule_type='external')
	forwardedNodes := r.getForwardedNodes(ctx, groupIDs, nodeMap)

	r.logger.Infow("retrieved nodes for subscription token",
		"subscription_id", subscriptionModel.ID,
		"plan_id", subscriptionModel.PlanID,
		"group_count", len(groupIDs),
		"node_count", len(originNodes),
		"forwarded_count", len(forwardedNodes),
		"mode", mode,
	)

	// Return nodes based on mode
	switch mode {
	case usecases.NodeModeForward:
		return forwardedNodes, nil
	case usecases.NodeModeOrigin:
		return originNodes, nil
	default: // NodeModeAll
		return append(originNodes, forwardedNodes...), nil
	}
}

func (r *NodeRepositoryAdapter) GetByTokenHash(ctx context.Context, tokenHash string) (usecases.NodeData, error) {
	nodeEntity, err := r.nodeRepo.GetByToken(ctx, tokenHash)
	if err != nil {
		r.logger.Warnw("failed to get node by token hash",
			"token_hash", tokenHash,
			"error", err,
		)
		return usecases.NodeData{}, err
	}

	// Convert domain entity to use case NodeData
	return usecases.NodeData{
		ID:        nodeEntity.ID(),
		SID:       nodeEntity.SID(),
		Name:      nodeEntity.Name(),
		TokenHash: nodeEntity.TokenHash(),
		Status:    string(nodeEntity.Status()),
	}, nil
}

// getForwardedNodes queries forward rules by group membership and builds subscription nodes.
// Rules are selected based on resource group membership, regardless of whether their
// target nodes are in the same resource groups.
// Uses Repository method to ensure proper scope isolation (system rules only).
func (r *NodeRepositoryAdapter) getForwardedNodes(ctx context.Context, groupIDs []uint, originNodeMap map[uint]*usecases.Node) []*usecases.Node {
	if len(groupIDs) == 0 {
		return nil
	}

	// Query rules by group membership (not by target node)
	// This allows rules to be delivered even when their target nodes are not in the same resource groups
	forwardRules, err := r.forwardRuleRepo.ListSystemRulesByGroupIDs(ctx, groupIDs)
	if err != nil {
		r.logger.Warnw("failed to query system rules by groups", "error", err, "group_ids", groupIDs)
		return nil
	}

	if len(forwardRules) == 0 {
		return nil
	}

	// Collect target node IDs from rules
	nodeIDSet := setutil.NewUintSet()
	for _, rule := range forwardRules {
		if rule.TargetNodeID() != nil {
			nodeIDSet.Add(*rule.TargetNodeID())
		}
	}
	targetNodeIDs := nodeIDSet.ToSlice()

	// Build node map: first try origin nodes, then query missing ones
	nodeMap := make(map[uint]*usecases.Node)
	missingNodeIDs := make([]uint, 0)

	for _, id := range targetNodeIDs {
		if node, ok := originNodeMap[id]; ok {
			nodeMap[id] = node
		} else {
			missingNodeIDs = append(missingNodeIDs, id)
		}
	}

	// Query missing nodes (nodes not in resource groups but referenced by rules)
	if len(missingNodeIDs) > 0 {
		nodeModels, nodeModelMap, err := r.queryActiveNodes(ctx, missingNodeIDs)
		if err != nil {
			r.logger.Warnw("failed to query target nodes for forward rules", "error", err)
		} else {
			// Load protocol configs for these nodes
			configs := r.configLoader.LoadProtocolConfigs(ctx, nodeModels)
			for id, model := range nodeModelMap {
				source := nodeutil.NodeModelToSource(model)
				nodeMap[id] = nodeutil.BuildNode(source, configs)
			}
		}
	}

	// Collect agent IDs and load forward agents
	agentIDs := r.collectAgentIDsFromRules(forwardRules)
	agentMap := r.loadForwardAgents(ctx, agentIDs)

	// Build forwarded nodes using the combined node map
	builder := nodeutil.NewForwardedNodeBuilder(agentMap, nodeutil.NewProtocolConfigs())
	return builder.BuildForwardedNodesFromUsecaseNodes(forwardRules, nodeMap)
}

// getForwardPlanNodes returns nodes for forward plan subscriptions
// For forward plans, users see their own forward rules as subscription nodes
// Forward plans have no "origin" nodes - all nodes are forwarded by nature
// Uses Repository method to ensure proper scope isolation (user's own rules only).
func (r *NodeRepositoryAdapter) getForwardPlanNodes(ctx context.Context, subscriptionID uint, userID uint, mode string) ([]*usecases.Node, error) {
	// Forward plans have no origin nodes, return empty for origin mode
	if mode == usecases.NodeModeOrigin {
		r.logger.Infow("forward plan has no origin nodes", "user_id", userID, "mode", mode)
		return []*usecases.Node{}, nil
	}

	forwardedNodes, err := r.getUserForwardNodes(ctx, userID)
	if err != nil {
		return nil, err
	}

	r.logger.Infow("retrieved forward plan nodes for user",
		"user_id", userID,
		"subscription_id", subscriptionID,
		"forwarded_node_count", len(forwardedNodes),
		"mode", mode,
	)

	return forwardedNodes, nil
}

// getHybridPlanNodes returns nodes for hybrid plan subscriptions
// For hybrid plans, users see both resource group nodes AND their own forward rules
func (r *NodeRepositoryAdapter) getHybridPlanNodes(ctx context.Context, subscriptionID uint, userID uint, planID uint, mode string) ([]*usecases.Node, error) {
	// Step 1: Get resource group nodes (same as node plan logic)
	// Query resource group IDs for this plan
	var groupIDs []uint
	if err := r.db.WithContext(ctx).
		Table("resource_groups").
		Where("plan_id = ? AND status = ?", planID, "active").
		Pluck("id", &groupIDs).Error; err != nil {
		r.logger.Errorw("failed to query resource groups", "error", err, "plan_id", planID)
		return nil, err
	}

	var resourceGroupNodes []*usecases.Node
	if len(groupIDs) > 0 {
		// Get node IDs that belong to these resource groups
		var nodeIDs []uint
		groupIDsJSON := jsonutil.UintSliceToJSONArray(groupIDs)
		if err := r.db.WithContext(ctx).
			Table("nodes").
			Where("JSON_OVERLAPS(group_ids, ?)", groupIDsJSON).
			Pluck("id", &nodeIDs).Error; err != nil {
			r.logger.Errorw("failed to query nodes by group", "error", err, "group_ids", groupIDs)
			return nil, err
		}

		// Build node map for quick lookup by getForwardedNodes
		// nodeIDs may be empty if resource groups only contain forward rules
		nodeMap := make(map[uint]*usecases.Node)

		if len(nodeIDs) > 0 {
			// Query nodes manually (no foreign key associations)
			nodeModels, _, err := r.queryActiveNodes(ctx, nodeIDs)
			if err != nil {
				return nil, err
			}

			// Load protocol configs for resource group nodes
			resourceGroupNodes = r.buildNodesWithConfigs(ctx, nodeModels)

			for _, node := range resourceGroupNodes {
				nodeMap[node.ID] = node
			}
		}

		// Generate forwarded nodes for resource group nodes if needed
		// Rules are selected by group membership, target nodes can be outside the resource groups
		if mode == usecases.NodeModeForward || mode == usecases.NodeModeAll {
			forwardedResourceGroupNodes := r.getForwardedNodes(ctx, groupIDs, nodeMap)
			if mode == usecases.NodeModeForward {
				resourceGroupNodes = forwardedResourceGroupNodes
			} else {
				resourceGroupNodes = append(resourceGroupNodes, forwardedResourceGroupNodes...)
			}
		}
	}

	// Step 2: Get user's forward rules as nodes (same as forward plan logic, but only target_node_id NOT NULL)
	// User forward rules are only included in "forward" and "all" modes (they are forwarded nodes by nature)
	var userForwardNodes []*usecases.Node
	if mode == usecases.NodeModeForward || mode == usecases.NodeModeAll {
		var err error
		userForwardNodes, err = r.getUserForwardNodes(ctx, userID)
		if err != nil {
			return nil, err
		}
	}

	// Combine both types of nodes
	// External forward rules are already included via:
	// - getForwardedNodes for resource group nodes (system external rules)
	// - getUserForwardNodes for user's own external rules
	allNodes := append(resourceGroupNodes, userForwardNodes...)

	r.logger.Infow("retrieved hybrid plan nodes",
		"user_id", userID,
		"subscription_id", subscriptionID,
		"plan_id", planID,
		"resource_group_node_count", len(resourceGroupNodes),
		"user_forward_node_count", len(userForwardNodes),
		"total_count", len(allNodes),
		"mode", mode,
	)

	return allNodes, nil
}

// buildNodesWithConfigs builds use case nodes from node models with protocol configs loaded
func (r *NodeRepositoryAdapter) buildNodesWithConfigs(ctx context.Context, nodeModels []models.NodeModel) []*usecases.Node {
	configs := r.configLoader.LoadProtocolConfigs(ctx, nodeModels)

	nodes := make([]*usecases.Node, 0, len(nodeModels))
	for _, nodeModel := range nodeModels {
		source := nodeutil.NodeModelToSource(&nodeModel)
		node := nodeutil.BuildNode(source, configs)
		nodes = append(nodes, node)
	}

	return nodes
}

// getUserForwardNodes retrieves user's forward rules with target nodes as subscription nodes
// Only returns forward rules where target_node_id is NOT NULL
// Uses Repository method to ensure proper scope isolation (user's own rules only).
func (r *NodeRepositoryAdapter) getUserForwardNodes(ctx context.Context, userID uint) ([]*usecases.Node, error) {
	forwardRules, err := r.forwardRuleRepo.ListUserRulesForDelivery(ctx, userID)
	if err != nil {
		r.logger.Errorw("failed to query user forward rules", "user_id", userID, "error", err)
		return nil, err
	}

	if len(forwardRules) == 0 {
		return []*usecases.Node{}, nil
	}

	// Collect target node IDs and agent IDs
	nodeIDs, agentIDs := r.collectIDsFromRules(forwardRules)

	// Query target nodes
	nodeModels, nodeMap, err := r.queryActiveNodes(ctx, nodeIDs)
	if err != nil {
		return nil, err
	}

	// Load protocol configs and agents
	configs := r.configLoader.LoadProtocolConfigs(ctx, nodeModels)
	agentMap := r.loadForwardAgents(ctx, agentIDs)

	// Use ForwardedNodeBuilder to generate forwarded nodes
	builder := nodeutil.NewForwardedNodeBuilder(agentMap, configs)
	return builder.BuildForwardedNodesFromModels(forwardRules, nodeMap), nil
}

// collectIDsFromRules extracts node IDs and agent IDs from forward rules
func (r *NodeRepositoryAdapter) collectIDsFromRules(rules []*forward.ForwardRule) (nodeIDs, agentIDs []uint) {
	nodeIDSet := setutil.NewUintSet()
	agentIDSet := setutil.NewUintSet()

	for _, rule := range rules {
		if rule.TargetNodeID() != nil {
			nodeIDSet.Add(*rule.TargetNodeID())
		}
		if rule.AgentID() > 0 {
			agentIDSet.Add(rule.AgentID())
		}
	}

	return nodeIDSet.ToSlice(), agentIDSet.ToSlice()
}

// collectAgentIDsFromRules extracts agent IDs from forward rules (skipping external rules).
func (r *NodeRepositoryAdapter) collectAgentIDsFromRules(rules []*forward.ForwardRule) []uint {
	agentIDSet := setutil.NewUintSet()
	for _, rule := range rules {
		if rule.AgentID() > 0 {
			agentIDSet.Add(rule.AgentID())
		}
	}
	return agentIDSet.ToSlice()
}

// queryActiveNodes queries active nodes by IDs and returns both slice and map
func (r *NodeRepositoryAdapter) queryActiveNodes(ctx context.Context, nodeIDs []uint) ([]models.NodeModel, map[uint]*models.NodeModel, error) {
	nodeMap := make(map[uint]*models.NodeModel)
	if len(nodeIDs) == 0 {
		return nil, nodeMap, nil
	}

	var nodeModels []models.NodeModel
	if err := r.db.WithContext(ctx).
		Where("id IN ?", nodeIDs).
		Where("status = ?", string(nodevo.NodeStatusActive)).
		Order("sort_order ASC").
		Find(&nodeModels).Error; err != nil {
		r.logger.Errorw("failed to query target nodes", "error", err)
		return nil, nil, err
	}

	for i := range nodeModels {
		nodeMap[nodeModels[i].ID] = &nodeModels[i]
	}

	return nodeModels, nodeMap, nil
}

// loadForwardAgents loads forward agents by IDs and returns a map
func (r *NodeRepositoryAdapter) loadForwardAgents(ctx context.Context, agentIDs []uint) map[uint]*models.ForwardAgentModel {
	agentMap := make(map[uint]*models.ForwardAgentModel)
	if len(agentIDs) == 0 {
		return agentMap
	}

	var agents []models.ForwardAgentModel
	if err := r.db.WithContext(ctx).
		Where("id IN ?", agentIDs).
		Where("status = ?", "enabled").
		Order("sort_order ASC").
		Find(&agents).Error; err != nil {
		r.logger.Warnw("failed to query forward agents", "error", err)
		return agentMap
	}

	for i := range agents {
		agentMap[agents[i].ID] = &agents[i]
	}
	return agentMap
}
