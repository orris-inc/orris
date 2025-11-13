package usecases

import (
	"context"
	"fmt"

	"orris/internal/application/node/dto"
	"orris/internal/domain/node"
	"orris/internal/shared/logger"
)

// GetNodeConfigCommand represents the command to get node configuration for XrayR
type GetNodeConfigCommand struct {
	NodeID   uint
	NodeType string // shadowsocks or trojan
}

// GetNodeConfigResult contains the node configuration response
type GetNodeConfigResult struct {
	Config *dto.NodeConfigResponse
}

// NodeConfigRepository defines the interface for node configuration retrieval
type NodeConfigRepository interface {
	GetByID(ctx context.Context, id uint) (*node.Node, error)
}

// GetNodeConfigUseCase handles fetching node configuration for XrayR clients
type GetNodeConfigUseCase struct {
	nodeRepo node.NodeRepository
	logger   logger.Interface
}

// NewGetNodeConfigUseCase creates a new instance of GetNodeConfigUseCase
func NewGetNodeConfigUseCase(
	nodeRepo node.NodeRepository,
	logger logger.Interface,
) *GetNodeConfigUseCase {
	return &GetNodeConfigUseCase{
		nodeRepo: nodeRepo,
		logger:   logger,
	}
}

// Execute retrieves the node configuration for XrayR backend
func (uc *GetNodeConfigUseCase) Execute(ctx context.Context, cmd GetNodeConfigCommand) (*GetNodeConfigResult, error) {
	if cmd.NodeID == 0 {
		return nil, fmt.Errorf("node_id is required")
	}

	// Retrieve node from repository
	n, err := uc.nodeRepo.GetByID(ctx, cmd.NodeID)
	if err != nil {
		uc.logger.Errorw("failed to get node configuration",
			"error", err,
			"node_id", cmd.NodeID,
		)
		return nil, fmt.Errorf("node not found")
	}

	// Check if node is active
	if !n.IsAvailable() {
		uc.logger.Warnw("attempt to get configuration for inactive node",
			"node_id", cmd.NodeID,
			"status", n.Status(),
		)
		return nil, fmt.Errorf("node is not active")
	}

	// Convert domain node to XrayR config response
	config := dto.ToNodeConfigResponse(n)
	if config == nil {
		uc.logger.Errorw("failed to convert node to config response",
			"node_id", cmd.NodeID,
		)
		return nil, fmt.Errorf("failed to generate node configuration")
	}

	// Override protocol type if provided
	if cmd.NodeType != "" {
		config.Protocol = cmd.NodeType
	}

	uc.logger.Infow("node configuration retrieved successfully",
		"node_id", cmd.NodeID,
		"protocol", config.Protocol,
	)

	return &GetNodeConfigResult{
		Config: config,
	}, nil
}
