package adapters

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"gorm.io/gorm"

	"orris/internal/application/node/usecases"
	"orris/internal/domain/node"
	"orris/internal/infrastructure/persistence/models"
	"orris/internal/shared/logger"
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

	// Convert to use case Node structure
	nodeMap := make(map[uint]*usecases.Node)
	for _, nodeModel := range nodeModels {
		// Parse plugin opts from JSON
		pluginOpts := make(map[string]string)
		if len(nodeModel.PluginOpts) > 0 {
			var optsMap map[string]interface{}
			if err := json.Unmarshal(nodeModel.PluginOpts, &optsMap); err == nil {
				for key, val := range optsMap {
					if strVal, ok := val.(string); ok {
						pluginOpts[key] = strVal
					}
				}
			}
		}

		// Handle nil plugin
		plugin := ""
		if nodeModel.Plugin != nil {
			plugin = *nodeModel.Plugin
		}

		nodeMap[nodeModel.ID] = &usecases.Node{
			ID:               nodeModel.ID,
			Name:             nodeModel.Name,
			ServerAddress:    nodeModel.ServerAddress,
			ServerPort:       nodeModel.ServerPort,
			EncryptionMethod: nodeModel.EncryptionMethod,
			Password:         "", // Password is not stored at node level; will be filled with subscription UUID
			Plugin:           plugin,
			PluginOpts:       pluginOpts,
		}
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
