package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/node/dto"
	"github.com/orris-inc/orris/internal/domain/node"
	vo "github.com/orris-inc/orris/internal/domain/node/valueobjects"
	apperrors "github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// GetNodeConfigCommand represents the command to get node configuration for node agent
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

// GetNodeConfigUseCase handles fetching node configuration for node agents
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

// Execute retrieves the node configuration for node agent
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
		return nil, fmt.Errorf("failed to get node: %w", err)
	}
	if n == nil {
		return nil, apperrors.NewNotFoundError("node not found")
	}

	// Node can be connected regardless of activation status.
	// Status is only used for business logic (e.g., subscription routing).

	// Collect all referenced node SIDs from route and DNS configs
	var referencedNodes []*node.Node
	var allReferencedSIDs []string
	if n.RouteConfig() != nil && n.RouteConfig().HasNodeReferences() {
		allReferencedSIDs = append(allReferencedSIDs, n.RouteConfig().GetReferencedNodeSIDs()...)
	}
	if n.DnsConfig() != nil && n.DnsConfig().HasNodeReferences() {
		allReferencedSIDs = append(allReferencedSIDs, n.DnsConfig().GetReferencedNodeSIDs()...)
	}
	if len(allReferencedSIDs) > 0 {
		// Deduplicate SIDs
		seen := make(map[string]bool, len(allReferencedSIDs))
		unique := make([]string, 0, len(allReferencedSIDs))
		for _, sid := range allReferencedSIDs {
			if !seen[sid] {
				seen[sid] = true
				unique = append(unique, sid)
			}
		}
		referencedNodes, err = uc.nodeRepo.GetBySIDs(ctx, unique)
		if err != nil {
			uc.logger.Warnw("failed to fetch referenced nodes",
				"node_id", cmd.NodeID,
				"referenced_sids", unique,
				"error", err,
			)
			// Continue without referenced nodes rather than failing
		}
	}

	// Server key function for referenced nodes
	serverKeyFunc := func(refNode *node.Node) string {
		if refNode.Protocol().IsShadowsocks() {
			return vo.GenerateShadowsocksServerPassword(refNode.TokenHash(), refNode.EncryptionConfig().Method())
		}
		// For Trojan, generate password from token hash for node-to-node forwarding
		if refNode.Protocol().IsTrojan() {
			return vo.GenerateTrojanServerPassword(refNode.TokenHash())
		}
		// For AnyTLS, generate password from token hash for node-to-node forwarding
		if refNode.Protocol().IsAnyTLS() {
			return vo.GenerateAnyTLSServerPassword(refNode.TokenHash())
		}
		return ""
	}

	// Convert domain node to agent config response
	config := dto.ToNodeConfigResponse(n, referencedNodes, serverKeyFunc)
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

	uc.logger.Debugw("node configuration retrieved",
		"node_id", cmd.NodeID,
		"protocol", config.Protocol,
	)

	if len(config.Outbounds) > 0 {
		for i, ob := range config.Outbounds {
			uc.logger.Debugw("outbound configuration",
				"index", i,
				"tag", ob.Tag,
				"type", ob.Type,
				"server", ob.Server,
				"port", ob.Port,
				"password_set", ob.Password != "",
				"password_len", len(ob.Password),
			)
		}
	}

	return &GetNodeConfigResult{
		Config: config,
	}, nil
}
