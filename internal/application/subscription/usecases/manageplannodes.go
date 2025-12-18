package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// ManagePlanNodesUseCase handles plan-node entitlement management
type ManagePlanNodesUseCase struct {
	planRepo        subscription.PlanRepository
	entitlementRepo subscription.EntitlementRepository
	nodeRepo        node.NodeRepository
	logger          logger.Interface
}

// NewManagePlanNodesUseCase creates a new ManagePlanNodesUseCase
func NewManagePlanNodesUseCase(
	planRepo subscription.PlanRepository,
	entitlementRepo subscription.EntitlementRepository,
	nodeRepo node.NodeRepository,
) *ManagePlanNodesUseCase {
	return &ManagePlanNodesUseCase{
		planRepo:        planRepo,
		entitlementRepo: entitlementRepo,
		nodeRepo:        nodeRepo,
		logger:          logger.NewLogger(),
	}
}

// BindNodesCommand represents the command to bind nodes to a plan
type BindNodesCommand struct {
	PlanID  uint
	NodeIDs []uint
}

// UnbindNodesCommand represents the command to unbind nodes from a plan
type UnbindNodesCommand struct {
	PlanID  uint
	NodeIDs []uint
}

// PlanNodeDTO represents a node associated with a plan
type PlanNodeDTO struct {
	ID            uint   `json:"id"`
	Name          string `json:"name"`
	ServerAddress string `json:"server_address"`
	Protocol      string `json:"protocol"`
	Status        string `json:"status"`
}

// GetPlanNodesResult represents the result of getting plan nodes
type GetPlanNodesResult struct {
	Nodes []PlanNodeDTO `json:"nodes"`
	Total int           `json:"total"`
}

// BindNodes binds nodes to a plan
func (uc *ManagePlanNodesUseCase) BindNodes(ctx context.Context, cmd BindNodesCommand) error {
	// Validate plan exists and is node type
	plan, err := uc.planRepo.GetByID(ctx, cmd.PlanID)
	if err != nil {
		uc.logger.Errorw("failed to get plan", "plan_id", cmd.PlanID, "error", err)
		return fmt.Errorf("failed to get plan: %w", err)
	}
	if plan == nil {
		return fmt.Errorf("plan not found: %d", cmd.PlanID)
	}
	if !plan.PlanType().IsNode() {
		return fmt.Errorf("plan %d is type '%s', can only bind nodes to 'node' type plans", cmd.PlanID, plan.PlanType())
	}

	// Validate all nodes exist
	for _, nodeID := range cmd.NodeIDs {
		nodeEntity, err := uc.nodeRepo.GetByID(ctx, nodeID)
		if err != nil {
			uc.logger.Errorw("failed to get node", "node_id", nodeID, "error", err)
			return fmt.Errorf("failed to get node %d: %w", nodeID, err)
		}
		if nodeEntity == nil {
			return fmt.Errorf("node not found: %d", nodeID)
		}
	}

	// Create entitlements for each node
	var entitlements []*subscription.Entitlement
	for _, nodeID := range cmd.NodeIDs {
		// Check if already exists
		exists, err := uc.entitlementRepo.Exists(ctx, cmd.PlanID, subscription.EntitlementResourceTypeNode, nodeID)
		if err != nil {
			uc.logger.Errorw("failed to check entitlement existence", "plan_id", cmd.PlanID, "node_id", nodeID, "error", err)
			return fmt.Errorf("failed to check entitlement: %w", err)
		}
		if exists {
			uc.logger.Infow("entitlement already exists, skipping", "plan_id", cmd.PlanID, "node_id", nodeID)
			continue
		}

		entitlement, err := subscription.NewEntitlement(cmd.PlanID, subscription.EntitlementResourceTypeNode, nodeID)
		if err != nil {
			return fmt.Errorf("failed to create entitlement: %w", err)
		}
		entitlements = append(entitlements, entitlement)
	}

	if len(entitlements) == 0 {
		uc.logger.Infow("no new entitlements to create", "plan_id", cmd.PlanID)
		return nil
	}

	// Batch create entitlements
	if err := uc.entitlementRepo.BatchCreate(ctx, entitlements); err != nil {
		uc.logger.Errorw("failed to create entitlements", "plan_id", cmd.PlanID, "count", len(entitlements), "error", err)
		return fmt.Errorf("failed to create entitlements: %w", err)
	}

	uc.logger.Infow("bound nodes to plan", "plan_id", cmd.PlanID, "node_count", len(entitlements))
	return nil
}

// UnbindNodes unbinds nodes from a plan
func (uc *ManagePlanNodesUseCase) UnbindNodes(ctx context.Context, cmd UnbindNodesCommand) error {
	// Validate plan exists
	plan, err := uc.planRepo.GetByID(ctx, cmd.PlanID)
	if err != nil {
		uc.logger.Errorw("failed to get plan", "plan_id", cmd.PlanID, "error", err)
		return fmt.Errorf("failed to get plan: %w", err)
	}
	if plan == nil {
		return fmt.Errorf("plan not found: %d", cmd.PlanID)
	}

	// Delete entitlements for each node
	for _, nodeID := range cmd.NodeIDs {
		if err := uc.entitlementRepo.DeleteByPlanAndResource(ctx, cmd.PlanID, subscription.EntitlementResourceTypeNode, nodeID); err != nil {
			uc.logger.Errorw("failed to delete entitlement", "plan_id", cmd.PlanID, "node_id", nodeID, "error", err)
			return fmt.Errorf("failed to delete entitlement: %w", err)
		}
	}

	uc.logger.Infow("unbound nodes from plan", "plan_id", cmd.PlanID, "node_count", len(cmd.NodeIDs))
	return nil
}

// GetPlanNodes returns all nodes associated with a plan
func (uc *ManagePlanNodesUseCase) GetPlanNodes(ctx context.Context, planID uint) (*GetPlanNodesResult, error) {
	// Validate plan exists
	plan, err := uc.planRepo.GetByID(ctx, planID)
	if err != nil {
		uc.logger.Errorw("failed to get plan", "plan_id", planID, "error", err)
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}
	if plan == nil {
		return nil, fmt.Errorf("plan not found: %d", planID)
	}

	// Get node IDs from entitlements
	nodeIDs, err := uc.entitlementRepo.GetResourceIDs(ctx, planID, subscription.EntitlementResourceTypeNode)
	if err != nil {
		uc.logger.Errorw("failed to get entitlements", "plan_id", planID, "error", err)
		return nil, fmt.Errorf("failed to get entitlements: %w", err)
	}

	if len(nodeIDs) == 0 {
		return &GetPlanNodesResult{
			Nodes: []PlanNodeDTO{},
			Total: 0,
		}, nil
	}

	// Fetch node details
	var nodes []PlanNodeDTO
	for _, nodeID := range nodeIDs {
		nodeEntity, err := uc.nodeRepo.GetByID(ctx, nodeID)
		if err != nil {
			uc.logger.Warnw("failed to get node, skipping", "node_id", nodeID, "error", err)
			continue
		}
		if nodeEntity == nil {
			continue
		}

		nodes = append(nodes, PlanNodeDTO{
			ID:            nodeEntity.ID(),
			Name:          nodeEntity.Name(),
			ServerAddress: nodeEntity.ServerAddress().Value(),
			Protocol:      nodeEntity.Protocol().String(),
			Status:        string(nodeEntity.Status()),
		})
	}

	return &GetPlanNodesResult{
		Nodes: nodes,
		Total: len(nodes),
	}, nil
}
