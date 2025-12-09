package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// GetForwardRuleQuery represents the input for getting a forward rule.
type GetForwardRuleQuery struct {
	ID uint
}

// GetForwardRuleUseCase handles getting a single forward rule.
type GetForwardRuleUseCase struct {
	repo      forward.Repository
	agentRepo forward.AgentRepository
	nodeRepo  node.NodeRepository
	logger    logger.Interface
}

// NewGetForwardRuleUseCase creates a new GetForwardRuleUseCase.
func NewGetForwardRuleUseCase(
	repo forward.Repository,
	agentRepo forward.AgentRepository,
	nodeRepo node.NodeRepository,
	logger logger.Interface,
) *GetForwardRuleUseCase {
	return &GetForwardRuleUseCase{
		repo:      repo,
		agentRepo: agentRepo,
		nodeRepo:  nodeRepo,
		logger:    logger,
	}
}

// Execute retrieves a forward rule by ID.
func (uc *GetForwardRuleUseCase) Execute(ctx context.Context, query GetForwardRuleQuery) (*dto.ForwardRuleDTO, error) {
	uc.logger.Infow("executing get forward rule use case", "id", query.ID)

	if query.ID == 0 {
		return nil, errors.NewValidationError("rule ID is required")
	}

	rule, err := uc.repo.GetByID(ctx, query.ID)
	if err != nil {
		uc.logger.Errorw("failed to get forward rule", "id", query.ID, "error", err)
		return nil, fmt.Errorf("failed to get forward rule: %w", err)
	}
	if rule == nil {
		return nil, errors.NewNotFoundError("forward rule", fmt.Sprintf("%d", query.ID))
	}

	ruleDTO := dto.ToForwardRuleDTO(rule)

	// Populate agent info (AgentID and ExitAgentID)
	agentIDs := dto.CollectAgentIDs([]*dto.ForwardRuleDTO{ruleDTO})
	if len(agentIDs) > 0 && uc.agentRepo != nil {
		agentShortIDs, err := uc.agentRepo.GetShortIDsByIDs(ctx, agentIDs)
		if err != nil {
			uc.logger.Warnw("failed to fetch agent short IDs", "error", err)
			// Continue without agent info
		} else {
			ruleDTO.PopulateAgentInfo(agentShortIDs)
		}
	}

	// Populate target node info if rule has target node
	if rule.HasTargetNode() && uc.nodeRepo != nil {
		nodes, err := uc.nodeRepo.GetByIDs(ctx, []uint{*rule.TargetNodeID()})
		if err != nil {
			uc.logger.Warnw("failed to fetch target node", "node_id", *rule.TargetNodeID(), "error", err)
			// Continue without node info
		} else if len(nodes) > 0 {
			n := nodes[0]
			ruleDTO.PopulateTargetNodeInfo(&dto.TargetNodeInfo{
				ServerAddress: n.ServerAddress().Value(),
				PublicIPv4:    n.PublicIPv4(),
				PublicIPv6:    n.PublicIPv6(),
			})
		}
	}

	return ruleDTO, nil
}
