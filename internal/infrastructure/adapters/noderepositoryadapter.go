package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/application/node/usecases"
	"github.com/orris-inc/orris/internal/domain/node"
	nodevo "github.com/orris-inc/orris/internal/domain/node/valueobjects"
	"github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/persistence/models"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// NodeRepository defines the interface for node persistence operations
type NodeRepository interface {
	GetByToken(ctx context.Context, tokenHash string) (*node.Node, error)
}

type NodeRepositoryAdapter struct {
	nodeRepo NodeRepository
	db       *gorm.DB
	logger   logger.Interface
}

func NewNodeRepositoryAdapter(nodeRepo NodeRepository, db *gorm.DB, logger logger.Interface) *NodeRepositoryAdapter {
	return &NodeRepositoryAdapter{
		nodeRepo: nodeRepo,
		db:       db,
		logger:   logger,
	}
}

func (r *NodeRepositoryAdapter) GetBySubscriptionToken(ctx context.Context, linkToken string, mode string) ([]*usecases.Node, error) {
	var subscriptionModel models.SubscriptionModel

	// Query subscription by link_token
	if err := r.db.WithContext(ctx).
		Where("link_token = ? AND status = ?", linkToken, valueobjects.StatusActive).
		First(&subscriptionModel).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			r.logger.Warnw("active subscription not found", "link_token_prefix", truncateTokenForLog(linkToken))
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
		return r.getForwardPlanNodes(ctx, subscriptionModel.UserID, mode)
	}

	// For hybrid plan, return both resource group nodes AND user's forward rules
	if planModel.PlanType == "hybrid" {
		return r.getHybridPlanNodes(ctx, subscriptionModel.UserID, subscriptionModel.PlanID, mode)
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
	groupIDsJSON := uintSliceToJSONArray(groupIDs)
	if err := r.db.WithContext(ctx).
		Table("nodes").
		Where("JSON_OVERLAPS(group_ids, ?) AND deleted_at IS NULL", groupIDsJSON).
		Pluck("id", &nodeIDs).Error; err != nil {
		r.logger.Errorw("failed to query nodes by group", "error", err, "group_ids", groupIDs)
		return nil, err
	}

	if len(nodeIDs) == 0 {
		r.logger.Infow("no nodes in resource groups for plan", "plan_id", subscriptionModel.PlanID)
		return []*usecases.Node{}, nil
	}

	// Query nodes manually (no foreign key associations)
	var nodeModels []models.NodeModel
	if err := r.db.WithContext(ctx).
		Where("id IN ?", nodeIDs).
		Where("status = ?", string(nodevo.NodeStatusActive)).
		Find(&nodeModels).Error; err != nil {
		r.logger.Errorw("failed to query nodes", "error", err)
		return nil, err
	}

	// Collect trojan node IDs for batch loading
	var trojanNodeIDs []uint
	for _, nodeModel := range nodeModels {
		if nodeModel.Protocol == "trojan" {
			trojanNodeIDs = append(trojanNodeIDs, nodeModel.ID)
		}
	}

	// Load trojan configs from trojan_configs table
	trojanConfigMap := make(map[uint]*models.TrojanConfigModel)
	if len(trojanNodeIDs) > 0 {
		var trojanConfigs []models.TrojanConfigModel
		if err := r.db.WithContext(ctx).
			Where("node_id IN ?", trojanNodeIDs).
			Find(&trojanConfigs).Error; err != nil {
			r.logger.Errorw("failed to query trojan configs", "error", err)
			return nil, err
		}
		for i := range trojanConfigs {
			trojanConfigMap[trojanConfigs[i].NodeID] = &trojanConfigs[i]
		}
	}

	// Collect shadowsocks node IDs for batch loading
	var ssNodeIDs []uint
	for _, nodeModel := range nodeModels {
		if nodeModel.Protocol == "shadowsocks" || nodeModel.Protocol == "" {
			ssNodeIDs = append(ssNodeIDs, nodeModel.ID)
		}
	}

	// Load shadowsocks configs from shadowsocks_configs table
	ssConfigMap := make(map[uint]*models.ShadowsocksConfigModel)
	if len(ssNodeIDs) > 0 {
		var ssConfigs []models.ShadowsocksConfigModel
		if err := r.db.WithContext(ctx).
			Where("node_id IN ?", ssNodeIDs).
			Find(&ssConfigs).Error; err != nil {
			r.logger.Errorw("failed to query shadowsocks configs", "error", err)
			return nil, err
		}
		for i := range ssConfigs {
			ssConfigMap[ssConfigs[i].NodeID] = &ssConfigs[i]
		}
	}

	// Convert to use case Node structure
	nodeMap := make(map[uint]*usecases.Node)
	for _, nodeModel := range nodeModels {
		// Default protocol to shadowsocks if not specified
		protocol := nodeModel.Protocol
		if protocol == "" {
			protocol = "shadowsocks"
		}

		// Calculate effective subscription port (use SubscriptionPort if set, otherwise AgentPort)
		subscriptionPort := nodeModel.AgentPort
		if nodeModel.SubscriptionPort != nil {
			subscriptionPort = *nodeModel.SubscriptionPort
		}

		ucNode := &usecases.Node{
			ID:               nodeModel.ID,
			Name:             nodeModel.Name,
			ServerAddress:    resolveServerAddress(nodeModel.ServerAddress, nodeModel.PublicIPv4, nodeModel.PublicIPv6),
			SubscriptionPort: subscriptionPort,
			Protocol:         protocol,
			Password:         "", // Password is not stored at node level; will be filled with subscription UUID
		}

		// Load Shadowsocks config from shadowsocks_configs table
		if protocol == "shadowsocks" {
			if sc, ok := ssConfigMap[nodeModel.ID]; ok {
				ucNode.EncryptionMethod = sc.EncryptionMethod
				if sc.Plugin != nil {
					ucNode.Plugin = *sc.Plugin
				}
				// Parse plugin opts from JSON
				if len(sc.PluginOpts) > 0 {
					pluginOpts := make(map[string]string)
					var optsMap map[string]interface{}
					if err := json.Unmarshal(sc.PluginOpts, &optsMap); err == nil {
						for key, val := range optsMap {
							if strVal, ok := val.(string); ok {
								pluginOpts[key] = strVal
							}
						}
					}
					ucNode.PluginOpts = pluginOpts
				}
			}
		}

		// Load Trojan config from trojan_configs table
		if protocol == "trojan" {
			if tc, ok := trojanConfigMap[nodeModel.ID]; ok {
				ucNode.TransportProtocol = tc.TransportProtocol
				ucNode.Host = tc.Host
				ucNode.Path = tc.Path
				ucNode.SNI = tc.SNI
				ucNode.AllowInsecure = tc.AllowInsecure
			} else {
				// Default transport protocol if no config found
				ucNode.TransportProtocol = "tcp"
			}
		}

		nodeMap[nodeModel.ID] = ucNode
	}

	// Convert map to slice (origin nodes)
	originNodes := make([]*usecases.Node, 0, len(nodeMap))
	for _, node := range nodeMap {
		originNodes = append(originNodes, node)
	}

	// Query forward rules that target these nodes to generate additional subscription entries
	forwardedNodes := r.getForwardedNodes(ctx, nodeIDs, nodeMap)

	r.logger.Infow("retrieved nodes for subscription token",
		"subscription_id", subscriptionModel.ID,
		"plan_id", subscriptionModel.PlanID,
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

// getForwardedNodes queries forward rules that target the given nodes and generates
// additional subscription entries using the forward agent's public address.
func (r *NodeRepositoryAdapter) getForwardedNodes(ctx context.Context, nodeIDs []uint, nodeMap map[uint]*usecases.Node) []*usecases.Node {
	if len(nodeIDs) == 0 {
		return nil
	}

	// Query forward rules that target these nodes (all types with target node, enabled status)
	var forwardRules []models.ForwardRuleModel
	if err := r.db.WithContext(ctx).
		Where("target_node_id IN ?", nodeIDs).
		Where("status = ?", "enabled").
		Where("rule_type IN ?", []string{"direct", "entry", "chain", "direct_chain"}).
		Find(&forwardRules).Error; err != nil {
		r.logger.Warnw("failed to query forward rules for nodes", "error", err)
		return nil
	}

	if len(forwardRules) == 0 {
		return nil
	}

	// Collect agent IDs
	agentIDSet := make(map[uint]bool)
	for _, rule := range forwardRules {
		agentIDSet[rule.AgentID] = true
	}

	agentIDs := make([]uint, 0, len(agentIDSet))
	for agentID := range agentIDSet {
		agentIDs = append(agentIDs, agentID)
	}

	// Query forward agents
	var agents []models.ForwardAgentModel
	if err := r.db.WithContext(ctx).
		Where("id IN ?", agentIDs).
		Where("status = ?", "enabled").
		Find(&agents).Error; err != nil {
		r.logger.Warnw("failed to query forward agents", "error", err)
		return nil
	}

	// Build agent map
	agentMap := make(map[uint]*models.ForwardAgentModel)
	for i := range agents {
		agentMap[agents[i].ID] = &agents[i]
	}

	// Generate forwarded node entries
	var forwardedNodes []*usecases.Node
	for _, rule := range forwardRules {
		agent, ok := agentMap[rule.AgentID]
		if !ok || agent.PublicAddress == "" {
			// Skip if agent not found or has no public address
			continue
		}

		if rule.TargetNodeID == nil {
			continue
		}

		originalNode, ok := nodeMap[*rule.TargetNodeID]
		if !ok {
			continue
		}

		// Create a forwarded node entry with agent's public address
		forwardedNode := &usecases.Node{
			ID:                originalNode.ID,
			Name:              rule.Name, // Use forward rule name
			ServerAddress:     agent.PublicAddress,
			SubscriptionPort:  rule.ListenPort,
			Protocol:          originalNode.Protocol,
			EncryptionMethod:  originalNode.EncryptionMethod,
			Password:          originalNode.Password,
			Plugin:            originalNode.Plugin,
			PluginOpts:        originalNode.PluginOpts,
			TransportProtocol: originalNode.TransportProtocol,
			Host:              originalNode.Host,
			Path:              originalNode.Path,
			SNI:               originalNode.SNI,
			AllowInsecure:     originalNode.AllowInsecure,
		}

		forwardedNodes = append(forwardedNodes, forwardedNode)
	}

	return forwardedNodes
}

// getForwardPlanNodes returns nodes for forward plan subscriptions
// For forward plans, users see their own forward rules as subscription nodes
// Forward plans have no "origin" nodes - all nodes are forwarded by nature
func (r *NodeRepositoryAdapter) getForwardPlanNodes(ctx context.Context, userID uint, mode string) ([]*usecases.Node, error) {
	// Forward plans have no origin nodes, return empty for origin mode
	if mode == usecases.NodeModeOrigin {
		r.logger.Infow("forward plan has no origin nodes", "user_id", userID, "mode", mode)
		return []*usecases.Node{}, nil
	}

	// Query user's enabled forward rules with target nodes
	var forwardRules []models.ForwardRuleModel
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Where("status = ?", "enabled").
		Where("target_node_id IS NOT NULL").
		Where("rule_type IN ?", []string{"direct", "entry", "chain", "direct_chain"}).
		Find(&forwardRules).Error; err != nil {
		r.logger.Errorw("failed to query user forward rules", "user_id", userID, "error", err)
		return nil, err
	}

	if len(forwardRules) == 0 {
		r.logger.Infow("no forward rules found for user", "user_id", userID)
		return []*usecases.Node{}, nil
	}

	// Collect target node IDs and agent IDs
	nodeIDSet := make(map[uint]bool)
	agentIDSet := make(map[uint]bool)
	for _, rule := range forwardRules {
		if rule.TargetNodeID != nil {
			nodeIDSet[*rule.TargetNodeID] = true
		}
		agentIDSet[rule.AgentID] = true
	}

	// Convert to slices
	nodeIDs := make([]uint, 0, len(nodeIDSet))
	for id := range nodeIDSet {
		nodeIDs = append(nodeIDs, id)
	}
	agentIDs := make([]uint, 0, len(agentIDSet))
	for id := range agentIDSet {
		agentIDs = append(agentIDs, id)
	}

	// Query target nodes
	var nodeModels []models.NodeModel
	if len(nodeIDs) > 0 {
		if err := r.db.WithContext(ctx).
			Where("id IN ?", nodeIDs).
			Where("status = ?", string(nodevo.NodeStatusActive)).
			Find(&nodeModels).Error; err != nil {
			r.logger.Errorw("failed to query target nodes", "error", err)
			return nil, err
		}
	}

	// Build node map
	nodeMap := make(map[uint]*models.NodeModel)
	for i := range nodeModels {
		nodeMap[nodeModels[i].ID] = &nodeModels[i]
	}

	// Load trojan configs for target nodes
	var trojanNodeIDs []uint
	for _, nm := range nodeModels {
		if nm.Protocol == "trojan" {
			trojanNodeIDs = append(trojanNodeIDs, nm.ID)
		}
	}
	trojanConfigMap := make(map[uint]*models.TrojanConfigModel)
	if len(trojanNodeIDs) > 0 {
		var trojanConfigs []models.TrojanConfigModel
		if err := r.db.WithContext(ctx).
			Where("node_id IN ?", trojanNodeIDs).
			Find(&trojanConfigs).Error; err != nil {
			r.logger.Warnw("failed to query trojan configs", "error", err)
		} else {
			for i := range trojanConfigs {
				trojanConfigMap[trojanConfigs[i].NodeID] = &trojanConfigs[i]
			}
		}
	}

	// Load shadowsocks configs for target nodes
	var ssNodeIDs []uint
	for _, nm := range nodeModels {
		if nm.Protocol == "shadowsocks" || nm.Protocol == "" {
			ssNodeIDs = append(ssNodeIDs, nm.ID)
		}
	}
	ssConfigMap := make(map[uint]*models.ShadowsocksConfigModel)
	if len(ssNodeIDs) > 0 {
		var ssConfigs []models.ShadowsocksConfigModel
		if err := r.db.WithContext(ctx).
			Where("node_id IN ?", ssNodeIDs).
			Find(&ssConfigs).Error; err != nil {
			r.logger.Warnw("failed to query shadowsocks configs", "error", err)
		} else {
			for i := range ssConfigs {
				ssConfigMap[ssConfigs[i].NodeID] = &ssConfigs[i]
			}
		}
	}

	// Query forward agents
	var agents []models.ForwardAgentModel
	if len(agentIDs) > 0 {
		if err := r.db.WithContext(ctx).
			Where("id IN ?", agentIDs).
			Where("status = ?", "enabled").
			Find(&agents).Error; err != nil {
			r.logger.Warnw("failed to query forward agents", "error", err)
			return nil, err
		}
	}

	// Build agent map
	agentMap := make(map[uint]*models.ForwardAgentModel)
	for i := range agents {
		agentMap[agents[i].ID] = &agents[i]
	}

	// Generate forwarded node entries
	var forwardedNodes []*usecases.Node
	for _, rule := range forwardRules {
		agent, ok := agentMap[rule.AgentID]
		if !ok || agent.PublicAddress == "" {
			continue
		}

		if rule.TargetNodeID == nil {
			continue
		}

		targetNode, ok := nodeMap[*rule.TargetNodeID]
		if !ok {
			continue
		}

		// Determine protocol
		protocol := targetNode.Protocol
		if protocol == "" {
			protocol = "shadowsocks"
		}

		// Build forwarded node
		forwardedNode := &usecases.Node{
			ID:               targetNode.ID,
			Name:             rule.Name,
			ServerAddress:    agent.PublicAddress,
			SubscriptionPort: rule.ListenPort,
			Protocol:         protocol,
			Password:         "", // Will be filled with subscription UUID
		}

		// Load Shadowsocks config
		if protocol == "shadowsocks" {
			if sc, ok := ssConfigMap[targetNode.ID]; ok {
				forwardedNode.EncryptionMethod = sc.EncryptionMethod
				if sc.Plugin != nil {
					forwardedNode.Plugin = *sc.Plugin
				}
				if len(sc.PluginOpts) > 0 {
					pluginOpts := make(map[string]string)
					var optsMap map[string]interface{}
					if err := json.Unmarshal(sc.PluginOpts, &optsMap); err == nil {
						for key, val := range optsMap {
							if strVal, ok := val.(string); ok {
								pluginOpts[key] = strVal
							}
						}
					}
					forwardedNode.PluginOpts = pluginOpts
				}
			}
		}

		// Load Trojan config
		if protocol == "trojan" {
			if tc, ok := trojanConfigMap[targetNode.ID]; ok {
				forwardedNode.TransportProtocol = tc.TransportProtocol
				forwardedNode.Host = tc.Host
				forwardedNode.Path = tc.Path
				forwardedNode.SNI = tc.SNI
				forwardedNode.AllowInsecure = tc.AllowInsecure
			} else {
				forwardedNode.TransportProtocol = "tcp"
			}
		}

		forwardedNodes = append(forwardedNodes, forwardedNode)
	}

	r.logger.Infow("retrieved forward plan nodes for user",
		"user_id", userID,
		"forward_rule_count", len(forwardRules),
		"forwarded_node_count", len(forwardedNodes),
		"mode", mode,
	)

	return forwardedNodes, nil
}

// getHybridPlanNodes returns nodes for hybrid plan subscriptions
// For hybrid plans, users see both resource group nodes AND their own forward rules
func (r *NodeRepositoryAdapter) getHybridPlanNodes(ctx context.Context, userID uint, planID uint, mode string) ([]*usecases.Node, error) {
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
		groupIDsJSON := uintSliceToJSONArray(groupIDs)
		if err := r.db.WithContext(ctx).
			Table("nodes").
			Where("JSON_OVERLAPS(group_ids, ?) AND deleted_at IS NULL", groupIDsJSON).
			Pluck("id", &nodeIDs).Error; err != nil {
			r.logger.Errorw("failed to query nodes by group", "error", err, "group_ids", groupIDs)
			return nil, err
		}

		if len(nodeIDs) > 0 {
			// Query nodes manually (no foreign key associations)
			var nodeModels []models.NodeModel
			if err := r.db.WithContext(ctx).
				Where("id IN ?", nodeIDs).
				Where("status = ?", string(nodevo.NodeStatusActive)).
				Find(&nodeModels).Error; err != nil {
				r.logger.Errorw("failed to query nodes", "error", err)
				return nil, err
			}

			// Load protocol configs for resource group nodes
			resourceGroupNodes = r.buildNodesWithConfigs(ctx, nodeModels)

			// Generate forwarded nodes for resource group nodes if needed
			if mode == usecases.NodeModeForward || mode == usecases.NodeModeAll {
				nodeMap := make(map[uint]*usecases.Node)
				for _, node := range resourceGroupNodes {
					nodeMap[node.ID] = node
				}
				forwardedResourceGroupNodes := r.getForwardedNodes(ctx, nodeIDs, nodeMap)
				if mode == usecases.NodeModeForward {
					resourceGroupNodes = forwardedResourceGroupNodes
				} else {
					resourceGroupNodes = append(resourceGroupNodes, forwardedResourceGroupNodes...)
				}
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
	allNodes := append(resourceGroupNodes, userForwardNodes...)

	r.logger.Infow("retrieved hybrid plan nodes",
		"user_id", userID,
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
	// Collect trojan node IDs for batch loading
	var trojanNodeIDs []uint
	for _, nodeModel := range nodeModels {
		if nodeModel.Protocol == "trojan" {
			trojanNodeIDs = append(trojanNodeIDs, nodeModel.ID)
		}
	}

	// Load trojan configs from trojan_configs table
	trojanConfigMap := make(map[uint]*models.TrojanConfigModel)
	if len(trojanNodeIDs) > 0 {
		var trojanConfigs []models.TrojanConfigModel
		if err := r.db.WithContext(ctx).
			Where("node_id IN ?", trojanNodeIDs).
			Find(&trojanConfigs).Error; err != nil {
			r.logger.Errorw("failed to query trojan configs", "error", err)
		} else {
			for i := range trojanConfigs {
				trojanConfigMap[trojanConfigs[i].NodeID] = &trojanConfigs[i]
			}
		}
	}

	// Collect shadowsocks node IDs for batch loading
	var ssNodeIDs []uint
	for _, nodeModel := range nodeModels {
		if nodeModel.Protocol == "shadowsocks" || nodeModel.Protocol == "" {
			ssNodeIDs = append(ssNodeIDs, nodeModel.ID)
		}
	}

	// Load shadowsocks configs from shadowsocks_configs table
	ssConfigMap := make(map[uint]*models.ShadowsocksConfigModel)
	if len(ssNodeIDs) > 0 {
		var ssConfigs []models.ShadowsocksConfigModel
		if err := r.db.WithContext(ctx).
			Where("node_id IN ?", ssNodeIDs).
			Find(&ssConfigs).Error; err != nil {
			r.logger.Errorw("failed to query shadowsocks configs", "error", err)
		} else {
			for i := range ssConfigs {
				ssConfigMap[ssConfigs[i].NodeID] = &ssConfigs[i]
			}
		}
	}

	// Convert to use case Node structure
	nodes := make([]*usecases.Node, 0, len(nodeModels))
	for _, nodeModel := range nodeModels {
		// Default protocol to shadowsocks if not specified
		protocol := nodeModel.Protocol
		if protocol == "" {
			protocol = "shadowsocks"
		}

		// Calculate effective subscription port (use SubscriptionPort if set, otherwise AgentPort)
		subscriptionPort := nodeModel.AgentPort
		if nodeModel.SubscriptionPort != nil {
			subscriptionPort = *nodeModel.SubscriptionPort
		}

		ucNode := &usecases.Node{
			ID:               nodeModel.ID,
			Name:             nodeModel.Name,
			ServerAddress:    resolveServerAddress(nodeModel.ServerAddress, nodeModel.PublicIPv4, nodeModel.PublicIPv6),
			SubscriptionPort: subscriptionPort,
			Protocol:         protocol,
			Password:         "", // Password is not stored at node level; will be filled with subscription UUID
		}

		// Load Shadowsocks config from shadowsocks_configs table
		if protocol == "shadowsocks" {
			if sc, ok := ssConfigMap[nodeModel.ID]; ok {
				ucNode.EncryptionMethod = sc.EncryptionMethod
				if sc.Plugin != nil {
					ucNode.Plugin = *sc.Plugin
				}
				// Parse plugin opts from JSON
				if len(sc.PluginOpts) > 0 {
					pluginOpts := make(map[string]string)
					var optsMap map[string]interface{}
					if err := json.Unmarshal(sc.PluginOpts, &optsMap); err == nil {
						for key, val := range optsMap {
							if strVal, ok := val.(string); ok {
								pluginOpts[key] = strVal
							}
						}
					}
					ucNode.PluginOpts = pluginOpts
				}
			}
		}

		// Load Trojan config from trojan_configs table
		if protocol == "trojan" {
			if tc, ok := trojanConfigMap[nodeModel.ID]; ok {
				ucNode.TransportProtocol = tc.TransportProtocol
				ucNode.Host = tc.Host
				ucNode.Path = tc.Path
				ucNode.SNI = tc.SNI
				ucNode.AllowInsecure = tc.AllowInsecure
			} else {
				// Default transport protocol if no config found
				ucNode.TransportProtocol = "tcp"
			}
		}

		nodes = append(nodes, ucNode)
	}

	return nodes
}

// getUserForwardNodes retrieves user's forward rules with target nodes as subscription nodes
// Only returns forward rules where target_node_id is NOT NULL
func (r *NodeRepositoryAdapter) getUserForwardNodes(ctx context.Context, userID uint) ([]*usecases.Node, error) {
	// Query user's enabled forward rules with target nodes
	var forwardRules []models.ForwardRuleModel
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Where("status = ?", "enabled").
		Where("target_node_id IS NOT NULL").
		Where("rule_type IN ?", []string{"direct", "entry", "chain", "direct_chain"}).
		Find(&forwardRules).Error; err != nil {
		r.logger.Errorw("failed to query user forward rules", "user_id", userID, "error", err)
		return nil, err
	}

	if len(forwardRules) == 0 {
		return []*usecases.Node{}, nil
	}

	// Collect target node IDs and agent IDs
	nodeIDSet := make(map[uint]bool)
	agentIDSet := make(map[uint]bool)
	for _, rule := range forwardRules {
		if rule.TargetNodeID != nil {
			nodeIDSet[*rule.TargetNodeID] = true
		}
		agentIDSet[rule.AgentID] = true
	}

	// Convert to slices
	nodeIDs := make([]uint, 0, len(nodeIDSet))
	for id := range nodeIDSet {
		nodeIDs = append(nodeIDs, id)
	}
	agentIDs := make([]uint, 0, len(agentIDSet))
	for id := range agentIDSet {
		agentIDs = append(agentIDs, id)
	}

	// Query target nodes
	var nodeModels []models.NodeModel
	if len(nodeIDs) > 0 {
		if err := r.db.WithContext(ctx).
			Where("id IN ?", nodeIDs).
			Where("status = ?", string(nodevo.NodeStatusActive)).
			Find(&nodeModels).Error; err != nil {
			r.logger.Errorw("failed to query target nodes", "error", err)
			return nil, err
		}
	}

	// Build node map
	nodeMap := make(map[uint]*models.NodeModel)
	for i := range nodeModels {
		nodeMap[nodeModels[i].ID] = &nodeModels[i]
	}

	// Load trojan configs for target nodes
	var trojanNodeIDs []uint
	for _, nm := range nodeModels {
		if nm.Protocol == "trojan" {
			trojanNodeIDs = append(trojanNodeIDs, nm.ID)
		}
	}
	trojanConfigMap := make(map[uint]*models.TrojanConfigModel)
	if len(trojanNodeIDs) > 0 {
		var trojanConfigs []models.TrojanConfigModel
		if err := r.db.WithContext(ctx).
			Where("node_id IN ?", trojanNodeIDs).
			Find(&trojanConfigs).Error; err != nil {
			r.logger.Warnw("failed to query trojan configs", "error", err)
		} else {
			for i := range trojanConfigs {
				trojanConfigMap[trojanConfigs[i].NodeID] = &trojanConfigs[i]
			}
		}
	}

	// Load shadowsocks configs for target nodes
	var ssNodeIDs []uint
	for _, nm := range nodeModels {
		if nm.Protocol == "shadowsocks" || nm.Protocol == "" {
			ssNodeIDs = append(ssNodeIDs, nm.ID)
		}
	}
	ssConfigMap := make(map[uint]*models.ShadowsocksConfigModel)
	if len(ssNodeIDs) > 0 {
		var ssConfigs []models.ShadowsocksConfigModel
		if err := r.db.WithContext(ctx).
			Where("node_id IN ?", ssNodeIDs).
			Find(&ssConfigs).Error; err != nil {
			r.logger.Warnw("failed to query shadowsocks configs", "error", err)
		} else {
			for i := range ssConfigs {
				ssConfigMap[ssConfigs[i].NodeID] = &ssConfigs[i]
			}
		}
	}

	// Query forward agents
	var agents []models.ForwardAgentModel
	if len(agentIDs) > 0 {
		if err := r.db.WithContext(ctx).
			Where("id IN ?", agentIDs).
			Where("status = ?", "enabled").
			Find(&agents).Error; err != nil {
			r.logger.Warnw("failed to query forward agents", "error", err)
			return nil, err
		}
	}

	// Build agent map
	agentMap := make(map[uint]*models.ForwardAgentModel)
	for i := range agents {
		agentMap[agents[i].ID] = &agents[i]
	}

	// Generate forwarded node entries
	var forwardedNodes []*usecases.Node
	for _, rule := range forwardRules {
		agent, ok := agentMap[rule.AgentID]
		if !ok || agent.PublicAddress == "" {
			continue
		}

		if rule.TargetNodeID == nil {
			continue
		}

		targetNode, ok := nodeMap[*rule.TargetNodeID]
		if !ok {
			continue
		}

		// Determine protocol
		protocol := targetNode.Protocol
		if protocol == "" {
			protocol = "shadowsocks"
		}

		// Build forwarded node
		forwardedNode := &usecases.Node{
			ID:               targetNode.ID,
			Name:             rule.Name,
			ServerAddress:    agent.PublicAddress,
			SubscriptionPort: rule.ListenPort,
			Protocol:         protocol,
			Password:         "", // Will be filled with subscription UUID
		}

		// Load Shadowsocks config
		if protocol == "shadowsocks" {
			if sc, ok := ssConfigMap[targetNode.ID]; ok {
				forwardedNode.EncryptionMethod = sc.EncryptionMethod
				if sc.Plugin != nil {
					forwardedNode.Plugin = *sc.Plugin
				}
				if len(sc.PluginOpts) > 0 {
					pluginOpts := make(map[string]string)
					var optsMap map[string]interface{}
					if err := json.Unmarshal(sc.PluginOpts, &optsMap); err == nil {
						for key, val := range optsMap {
							if strVal, ok := val.(string); ok {
								pluginOpts[key] = strVal
							}
						}
					}
					forwardedNode.PluginOpts = pluginOpts
				}
			}
		}

		// Load Trojan config
		if protocol == "trojan" {
			if tc, ok := trojanConfigMap[targetNode.ID]; ok {
				forwardedNode.TransportProtocol = tc.TransportProtocol
				forwardedNode.Host = tc.Host
				forwardedNode.Path = tc.Path
				forwardedNode.SNI = tc.SNI
				forwardedNode.AllowInsecure = tc.AllowInsecure
			} else {
				forwardedNode.TransportProtocol = "tcp"
			}
		}

		forwardedNodes = append(forwardedNodes, forwardedNode)
	}

	return forwardedNodes, nil
}

// resolveServerAddress returns the effective server address for subscription
// If server address is configured, use it; otherwise fall back to agent's reported public IP
func resolveServerAddress(configuredAddr string, publicIPv4, publicIPv6 *string) string {
	// If server address is explicitly configured, always use it
	if configuredAddr != "" {
		return configuredAddr
	}

	// Fall back to agent's reported public IP
	// Prefer IPv4 over IPv6 for better compatibility
	if publicIPv4 != nil && *publicIPv4 != "" {
		return *publicIPv4
	}

	if publicIPv6 != nil && *publicIPv6 != "" {
		return *publicIPv6
	}

	// Return empty if no address available (should not happen in practice)
	// The value object layer allows empty addresses for this fallback scenario
	return ""
}

// uintSliceToJSONArray converts a slice of uint to a JSON array string
// Used for JSON_OVERLAPS query parameter
func uintSliceToJSONArray(ids []uint) string {
	if len(ids) == 0 {
		return "[]"
	}
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = fmt.Sprintf("%d", id)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

// truncateTokenForLog safely truncates a token for logging purposes
// Returns first 8 characters followed by "..." or the full token if shorter
func truncateTokenForLog(token string) string {
	const maxLen = 8
	if len(token) <= maxLen {
		return token
	}
	return token[:maxLen] + "..."
}
