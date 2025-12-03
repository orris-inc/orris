package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type ListGroupNodesQuery struct {
	GroupID uint
}

type GroupNodeDTO struct {
	ID            uint   `json:"id"`
	Name          string `json:"name"`
	ServerAddress string `json:"server_address"`
	ServerPort    uint16 `json:"server_port"`
	Region        string `json:"region"`
	Status        string `json:"status"`
}

type ListGroupNodesResult struct {
	GroupID uint            `json:"group_id"`
	Nodes   []*GroupNodeDTO `json:"nodes"`
	Total   int             `json:"total"`
}

type ListGroupNodesUseCase struct {
	nodeRepo      node.NodeRepository
	nodeGroupRepo node.NodeGroupRepository
	logger        logger.Interface
}

func NewListGroupNodesUseCase(
	nodeRepo node.NodeRepository,
	nodeGroupRepo node.NodeGroupRepository,
	logger logger.Interface,
) *ListGroupNodesUseCase {
	return &ListGroupNodesUseCase{
		nodeRepo:      nodeRepo,
		nodeGroupRepo: nodeGroupRepo,
		logger:        logger,
	}
}

func (uc *ListGroupNodesUseCase) Execute(ctx context.Context, query ListGroupNodesQuery) (*ListGroupNodesResult, error) {
	uc.logger.Infow("executing list group nodes use case", "group_id", query.GroupID)

	if err := uc.validateQuery(query); err != nil {
		uc.logger.Errorw("invalid list group nodes query", "error", err)
		return nil, err
	}

	group, err := uc.nodeGroupRepo.GetByID(ctx, query.GroupID)
	if err != nil {
		uc.logger.Errorw("failed to get node group", "error", err, "group_id", query.GroupID)
		return nil, fmt.Errorf("failed to get node group: %w", err)
	}

	nodeIDs := group.NodeIDs()
	if len(nodeIDs) == 0 {
		uc.logger.Infow("no nodes found in group", "group_id", query.GroupID)
		return &ListGroupNodesResult{
			GroupID: query.GroupID,
			Nodes:   []*GroupNodeDTO{},
			Total:   0,
		}, nil
	}

	nodes := make([]*node.Node, 0, len(nodeIDs))
	for _, nodeID := range nodeIDs {
		n, err := uc.nodeRepo.GetByID(ctx, nodeID)
		if err != nil {
			uc.logger.Warnw("failed to get node from group", "node_id", nodeID, "error", err)
			continue
		}
		nodes = append(nodes, n)
	}

	nodeDTOs := make([]*GroupNodeDTO, 0, len(nodes))
	for _, n := range nodes {
		nodeDTOs = append(nodeDTOs, &GroupNodeDTO{
			ID:            n.ID(),
			Name:          n.Name(),
			ServerAddress: n.ServerAddress().Value(),
			ServerPort:    n.ServerPort(),
			Region:        n.Metadata().Region(),
			Status:        string(n.Status()),
		})
	}

	uc.logger.Infow("group nodes listed successfully",
		"group_id", query.GroupID,
		"count", len(nodeDTOs),
	)

	return &ListGroupNodesResult{
		GroupID: query.GroupID,
		Nodes:   nodeDTOs,
		Total:   len(nodeDTOs),
	}, nil
}

func (uc *ListGroupNodesUseCase) validateQuery(query ListGroupNodesQuery) error {
	if query.GroupID == 0 {
		return errors.NewValidationError("group ID is required")
	}

	return nil
}
