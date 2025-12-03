package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	vo "github.com/orris-inc/orris/internal/domain/forward/value_objects"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// CreateForwardChainNodeInput represents a node in the chain creation request.
type CreateForwardChainNodeInput struct {
	AgentID    uint   `json:"agent_id"`
	ListenPort uint16 `json:"listen_port"`
}

// CreateForwardChainCommand represents the input for creating a forward chain.
type CreateForwardChainCommand struct {
	Name          string                        `json:"name"`
	Protocol      string                        `json:"protocol"`
	Nodes         []CreateForwardChainNodeInput `json:"nodes"`
	TargetAddress string                        `json:"target_address"`
	TargetPort    uint16                        `json:"target_port"`
	Remark        string                        `json:"remark"`
}

// CreateForwardChainUseCase handles forward chain creation.
type CreateForwardChainUseCase struct {
	chainRepo forward.ChainRepository
	ruleRepo  forward.Repository
	agentRepo forward.AgentRepository
	logger    logger.Interface
}

// NewCreateForwardChainUseCase creates a new CreateForwardChainUseCase.
func NewCreateForwardChainUseCase(
	chainRepo forward.ChainRepository,
	ruleRepo forward.Repository,
	agentRepo forward.AgentRepository,
	logger logger.Interface,
) *CreateForwardChainUseCase {
	return &CreateForwardChainUseCase{
		chainRepo: chainRepo,
		ruleRepo:  ruleRepo,
		agentRepo: agentRepo,
		logger:    logger,
	}
}

// Execute creates a new forward chain and generates rules for each node.
func (uc *CreateForwardChainUseCase) Execute(ctx context.Context, cmd CreateForwardChainCommand) (*dto.ForwardChainDTO, error) {
	uc.logger.Infow("executing create forward chain use case", "name", cmd.Name, "node_count", len(cmd.Nodes))

	if err := uc.validateCommand(ctx, cmd); err != nil {
		uc.logger.Errorw("invalid create forward chain command", "error", err)
		return nil, err
	}

	// Convert input nodes to domain nodes
	nodes := make([]forward.ChainNode, len(cmd.Nodes))
	for i, n := range cmd.Nodes {
		nodes[i] = forward.ChainNode{
			AgentID:    n.AgentID,
			ListenPort: n.ListenPort,
		}
	}

	// Create domain entity
	protocol := vo.ForwardProtocol(cmd.Protocol)
	chain, err := forward.NewForwardChain(
		cmd.Name,
		protocol,
		nodes,
		cmd.TargetAddress,
		cmd.TargetPort,
		cmd.Remark,
	)
	if err != nil {
		uc.logger.Errorw("failed to create forward chain entity", "error", err)
		return nil, fmt.Errorf("failed to create forward chain: %w", err)
	}

	// Persist chain
	if err := uc.chainRepo.Create(ctx, chain); err != nil {
		uc.logger.Errorw("failed to persist forward chain", "error", err)
		return nil, fmt.Errorf("failed to save forward chain: %w", err)
	}

	// Generate and persist rules
	rules, err := chain.GenerateRules()
	if err != nil {
		uc.logger.Errorw("failed to generate rules for chain", "chain_id", chain.ID(), "error", err)
		return nil, fmt.Errorf("failed to generate rules: %w", err)
	}

	ruleIDs := make([]uint, len(rules))
	for i, rule := range rules {
		if err := uc.ruleRepo.Create(ctx, rule); err != nil {
			uc.logger.Errorw("failed to create rule", "chain_id", chain.ID(), "node_seq", i+1, "error", err)
			return nil, fmt.Errorf("failed to create rule for node %d: %w", i+1, err)
		}
		ruleIDs[i] = rule.ID()
	}

	// Associate rules with chain
	if err := uc.chainRepo.AssociateRules(ctx, chain.ID(), ruleIDs); err != nil {
		uc.logger.Errorw("failed to associate rules with chain", "chain_id", chain.ID(), "error", err)
		return nil, fmt.Errorf("failed to associate rules: %w", err)
	}

	uc.logger.Infow("forward chain created successfully", "id", chain.ID(), "name", cmd.Name, "rules_created", len(rules))
	return dto.ToForwardChainDTO(chain), nil
}

func (uc *CreateForwardChainUseCase) validateCommand(ctx context.Context, cmd CreateForwardChainCommand) error {
	if cmd.Name == "" {
		return errors.NewValidationError("name is required")
	}
	if cmd.Protocol == "" {
		return errors.NewValidationError("protocol is required")
	}
	protocol := vo.ForwardProtocol(cmd.Protocol)
	if !protocol.IsValid() {
		return errors.NewValidationError(fmt.Sprintf("invalid protocol: %s, must be tcp, udp or both", cmd.Protocol))
	}
	if len(cmd.Nodes) == 0 {
		return errors.NewValidationError("at least one node is required")
	}
	if cmd.TargetAddress == "" {
		return errors.NewValidationError("target_address is required")
	}
	if cmd.TargetPort == 0 {
		return errors.NewValidationError("target_port is required")
	}

	// Validate each node
	for i, node := range cmd.Nodes {
		if node.AgentID == 0 {
			return errors.NewValidationError(fmt.Sprintf("node %d: agent_id is required", i+1))
		}
		if node.ListenPort == 0 {
			return errors.NewValidationError(fmt.Sprintf("node %d: listen_port is required", i+1))
		}

		// Verify agent exists
		agent, err := uc.agentRepo.GetByID(ctx, node.AgentID)
		if err != nil {
			return fmt.Errorf("failed to verify agent %d: %w", node.AgentID, err)
		}
		if agent == nil {
			return errors.NewValidationError(fmt.Sprintf("node %d: agent %d not found", i+1, node.AgentID))
		}
	}

	return nil
}
