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

// ListUserForwardRulesQuery represents the input for listing user's forward rules.
type ListUserForwardRulesQuery struct {
	UserID   uint
	Page     int
	PageSize int
	Name     string
	Protocol string
	Status   string
	OrderBy  string
	Order    string
}

// ListUserForwardRulesResult represents the output of listing user's forward rules.
type ListUserForwardRulesResult struct {
	Rules []*dto.ForwardRuleDTO `json:"rules"`
	Total int64                 `json:"total"`
	Page  int                   `json:"page"`
	Pages int                   `json:"pages"`
}

// ListUserForwardRulesUseCase handles listing forward rules for a specific user.
type ListUserForwardRulesUseCase struct {
	repo      forward.Repository
	agentRepo forward.AgentRepository
	nodeRepo  node.NodeRepository
	logger    logger.Interface
}

// NewListUserForwardRulesUseCase creates a new ListUserForwardRulesUseCase.
func NewListUserForwardRulesUseCase(
	repo forward.Repository,
	agentRepo forward.AgentRepository,
	nodeRepo node.NodeRepository,
	logger logger.Interface,
) *ListUserForwardRulesUseCase {
	return &ListUserForwardRulesUseCase{
		repo:      repo,
		agentRepo: agentRepo,
		nodeRepo:  nodeRepo,
		logger:    logger,
	}
}

// Execute retrieves a list of forward rules for a specific user.
func (uc *ListUserForwardRulesUseCase) Execute(ctx context.Context, query ListUserForwardRulesQuery) (*ListUserForwardRulesResult, error) {
	uc.logger.Infow("executing list user forward rules use case", "user_id", query.UserID, "page", query.Page, "page_size", query.PageSize)

	// Validate user ID
	if query.UserID == 0 {
		return nil, errors.NewValidationError("user_id is required")
	}

	// Set defaults
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		query.PageSize = 100
	}

	filter := forward.ListFilter{
		Page:     query.Page,
		PageSize: query.PageSize,
		UserID:   &query.UserID,
		Name:     query.Name,
		Protocol: query.Protocol,
		Status:   query.Status,
		OrderBy:  query.OrderBy,
		Order:    query.Order,
	}

	rules, total, err := uc.repo.ListByUserID(ctx, query.UserID, filter)
	if err != nil {
		uc.logger.Errorw("failed to list user forward rules", "user_id", query.UserID, "error", err)
		return nil, fmt.Errorf("failed to list user forward rules: %w", err)
	}

	// Calculate total pages
	pages := int(total) / query.PageSize
	if int(total)%query.PageSize > 0 {
		pages++
	}

	// Convert to DTOs
	dtos := dto.ToForwardRuleDTOs(rules)

	// Populate agent info (AgentID and ExitAgentID)
	agentIDs := dto.CollectAgentIDs(dtos)
	if len(agentIDs) > 0 && uc.agentRepo != nil {
		agentShortIDs, err := uc.agentRepo.GetShortIDsByIDs(ctx, agentIDs)
		if err != nil {
			uc.logger.Warnw("failed to fetch agent short IDs", "user_id", query.UserID, "error", err)
			// Continue without agent info
		} else {
			for _, ruleDTO := range dtos {
				ruleDTO.PopulateAgentInfo(agentShortIDs)
			}
		}
	}

	// Collect target node IDs from DTOs
	nodeIDs := dto.CollectTargetNodeIDs(dtos)

	// Fetch target nodes and populate info
	if len(nodeIDs) > 0 && uc.nodeRepo != nil {
		nodes, err := uc.nodeRepo.GetByIDs(ctx, nodeIDs)
		if err != nil {
			uc.logger.Warnw("failed to fetch target nodes", "user_id", query.UserID, "error", err)
			// Continue without node info
		} else {
			// Build node map for info and SID map
			nodeMap := make(map[uint]*node.Node)
			nodeSIDMap := make(dto.NodeSIDMap)
			for _, n := range nodes {
				nodeMap[n.ID()] = n
				nodeSIDMap[n.ID()] = n.SID()
			}
			// Populate target node SID and info
			for _, ruleDTO := range dtos {
				ruleDTO.PopulateTargetNodeSID(nodeSIDMap)
				if targetNodeID := ruleDTO.InternalTargetNodeID(); targetNodeID != nil {
					if n, ok := nodeMap[*targetNodeID]; ok {
						ruleDTO.PopulateTargetNodeInfo(&dto.TargetNodeInfo{
							ServerAddress: n.ServerAddress().Value(),
							PublicIPv4:    n.PublicIPv4(),
							PublicIPv6:    n.PublicIPv6(),
						})
					}
				}
			}
		}
	}

	uc.logger.Infow("user forward rules listed successfully", "user_id", query.UserID, "total", total)

	return &ListUserForwardRulesResult{
		Rules: dtos,
		Total: total,
		Page:  query.Page,
		Pages: pages,
	}, nil
}
