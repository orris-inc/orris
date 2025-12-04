package adapters

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"gorm.io/gorm"

	"github.com/orris-inc/orris/internal/application/node/usecases"
	"github.com/orris-inc/orris/internal/domain/node"
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

func (r *NodeRepositoryAdapter) GetBySubscriptionToken(ctx context.Context, token string) ([]*usecases.Node, error) {
	// Hash the token
	tokenHash := hashSubscriptionToken(token)

	// Query subscription token
	var tokenModel models.SubscriptionTokenModel
	if err := r.db.WithContext(ctx).
		Where("token_hash = ? AND is_active = ?", tokenHash, true).
		First(&tokenModel).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			r.logger.Warnw("subscription token not found", "token_hash", tokenHash)
			return []*usecases.Node{}, nil
		}
		r.logger.Errorw("failed to query subscription token", "error", err)
		return nil, err
	}

	// Query subscription
	var subscriptionModel models.SubscriptionModel
	if err := r.db.WithContext(ctx).
		Where("id = ? AND status = ?", tokenModel.SubscriptionID, "active").
		First(&subscriptionModel).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			r.logger.Warnw("active subscription not found", "subscription_id", tokenModel.SubscriptionID)
			return []*usecases.Node{}, nil
		}
		r.logger.Errorw("failed to query subscription", "error", err)
		return nil, err
	}

	// Query node groups associated with this plan
	var nodeGroupPlanModels []models.NodeGroupPlanModel
	if err := r.db.WithContext(ctx).
		Where("subscription_plan_id = ?", subscriptionModel.PlanID).
		Find(&nodeGroupPlanModels).Error; err != nil {
		r.logger.Errorw("failed to query node group plans", "error", err, "plan_id", subscriptionModel.PlanID)
		return nil, err
	}

	if len(nodeGroupPlanModels) == 0 {
		r.logger.Infow("no node groups associated with plan", "plan_id", subscriptionModel.PlanID)
		return []*usecases.Node{}, nil
	}

	// Extract node group IDs
	nodeGroupIDs := make([]uint, len(nodeGroupPlanModels))
	for i, gp := range nodeGroupPlanModels {
		nodeGroupIDs[i] = gp.NodeGroupID
	}

	// Query node group associations
	var nodeGroupNodeModels []models.NodeGroupNodeModel
	if err := r.db.WithContext(ctx).
		Where("node_group_id IN ?", nodeGroupIDs).
		Find(&nodeGroupNodeModels).Error; err != nil {
		r.logger.Errorw("failed to query node group nodes", "error", err)
		return nil, err
	}

	if len(nodeGroupNodeModels) == 0 {
		r.logger.Infow("no nodes found in node groups", "node_group_ids", nodeGroupIDs)
		return []*usecases.Node{}, nil
	}

	// Extract unique node IDs
	nodeIDSet := make(map[uint]bool)
	for _, ngn := range nodeGroupNodeModels {
		nodeIDSet[ngn.NodeID] = true
	}

	nodeIDs := make([]uint, 0, len(nodeIDSet))
	for nodeID := range nodeIDSet {
		nodeIDs = append(nodeIDs, nodeID)
	}

	// Query nodes manually (no foreign key associations)
	var nodeModels []models.NodeModel
	if err := r.db.WithContext(ctx).
		Where("id IN ?", nodeIDs).
		Where("status = ?", "active").
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

		ucNode := &usecases.Node{
			ID:            nodeModel.ID,
			Name:          nodeModel.Name,
			ServerAddress: nodeModel.ServerAddress,
			ServerPort:    nodeModel.ServerPort,
			Protocol:      protocol,
			Password:      "", // Password is not stored at node level; will be filled with subscription UUID
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

	// Convert map to slice
	nodes := make([]*usecases.Node, 0, len(nodeMap))
	for _, node := range nodeMap {
		nodes = append(nodes, node)
	}

	r.logger.Infow("retrieved nodes for subscription token",
		"subscription_id", subscriptionModel.ID,
		"plan_id", subscriptionModel.PlanID,
		"node_count", len(nodes),
	)

	return nodes, nil
}

func hashSubscriptionToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
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
		Name:      nodeEntity.Name(),
		TokenHash: nodeEntity.TokenHash(),
		Status:    string(nodeEntity.Status()),
	}, nil
}
