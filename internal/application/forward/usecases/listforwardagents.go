package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/resource"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/version"
)

// LatestVersionQuerier provides access to the latest agent version.
type LatestVersionQuerier interface {
	// GetVersion returns the latest available agent version (e.g., "1.2.3").
	GetVersion(ctx context.Context) (string, error)
}

// ListForwardAgentsQuery represents the input for listing forward agents.
type ListForwardAgentsQuery struct {
	Page     int
	PageSize int
	Name     string
	Status   string
	OrderBy  string
	Order    string
}

// ListForwardAgentsResult represents the output of listing forward agents.
type ListForwardAgentsResult struct {
	Agents []*dto.ForwardAgentDTO `json:"agents"`
	Total  int64                  `json:"total"`
	Page   int                    `json:"page"`
	Pages  int                    `json:"pages"`
}

// ListForwardAgentsUseCase handles listing forward agents.
type ListForwardAgentsUseCase struct {
	repo              forward.AgentRepository
	resourceGroupRepo resource.Repository
	statusQuerier     AgentStatusQuerier
	versionQuerier    LatestVersionQuerier
	logger            logger.Interface
}

// NewListForwardAgentsUseCase creates a new ListForwardAgentsUseCase.
func NewListForwardAgentsUseCase(
	repo forward.AgentRepository,
	resourceGroupRepo resource.Repository,
	statusQuerier AgentStatusQuerier,
	versionQuerier LatestVersionQuerier,
	logger logger.Interface,
) *ListForwardAgentsUseCase {
	return &ListForwardAgentsUseCase{
		repo:              repo,
		resourceGroupRepo: resourceGroupRepo,
		statusQuerier:     statusQuerier,
		versionQuerier:    versionQuerier,
		logger:            logger,
	}
}

// Execute retrieves a list of forward agents.
func (uc *ListForwardAgentsUseCase) Execute(ctx context.Context, query ListForwardAgentsQuery) (*ListForwardAgentsResult, error) {
	uc.logger.Infow("executing list forward agents use case", "page", query.Page, "page_size", query.PageSize)

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

	filter := forward.AgentListFilter{
		Page:     query.Page,
		PageSize: query.PageSize,
		Name:     query.Name,
		Status:   query.Status,
		OrderBy:  query.OrderBy,
		Order:    query.Order,
	}

	agents, total, err := uc.repo.List(ctx, filter)
	if err != nil {
		uc.logger.Errorw("failed to list forward agents", "error", err)
		return nil, fmt.Errorf("failed to list forward agents: %w", err)
	}

	// Calculate total pages
	pages := int(total) / query.PageSize
	if int(total)%query.PageSize > 0 {
		pages++
	}

	// Convert to DTOs
	dtos := dto.ToForwardAgentDTOs(agents)

	// Collect unique group IDs from DTOs for batch query
	groupIDs := dto.CollectAgentGroupIDs(dtos)

	// Query resource groups and populate GroupSIDs
	if len(groupIDs) > 0 && uc.resourceGroupRepo != nil {
		groups, err := uc.resourceGroupRepo.GetByIDs(ctx, groupIDs)
		if err != nil {
			uc.logger.Warnw("failed to get resource groups, continuing without group info",
				"error", err,
			)
		} else {
			groupSIDMap := make(dto.GroupSIDMap, len(groups))
			for _, group := range groups {
				groupSIDMap[group.ID()] = group.SID()
			}
			for _, d := range dtos {
				d.PopulateGroupSIDs(groupSIDMap)
			}
		}
	}

	// Collect agent IDs for batch status query and create ID mapping
	agentIDs := make([]uint, 0, len(agents))
	idToIndexMap := make(map[uint]int, len(agents))
	for i, agent := range agents {
		agentIDs = append(agentIDs, agent.ID())
		idToIndexMap[agent.ID()] = i
	}

	// Query system status for all agents from Redis
	if len(agentIDs) > 0 && uc.statusQuerier != nil {
		statusMap, err := uc.statusQuerier.GetMultipleStatus(ctx, agentIDs)
		if err != nil {
			uc.logger.Warnw("failed to get agents system status, continuing without it",
				"error", err,
			)
		} else {
			// Attach system status to each agent DTO using the mapping
			for agentID, status := range statusMap {
				if idx, ok := idToIndexMap[agentID]; ok && status != nil {
					dtos[idx].SystemStatus = status
					// Extract agent version to top-level field for easy display
					// Only override if Redis has a non-empty version (prefer real-time over DB)
					if status.AgentVersion != "" {
						dtos[idx].AgentVersion = status.AgentVersion
					}
				}
			}
		}
	}

	// Query latest version and calculate HasUpdate for each agent
	if uc.versionQuerier != nil {
		latestVersion, err := uc.versionQuerier.GetVersion(ctx)
		if err != nil {
			uc.logger.Warnw("failed to get latest version, continuing without update check",
				"error", err,
			)
		} else {
			// Set HasUpdate for each agent by comparing versions
			for i := range dtos {
				dtos[i].HasUpdate = version.HasNewerVersion(dtos[i].AgentVersion, latestVersion)
			}
		}
	}

	return &ListForwardAgentsResult{
		Agents: dtos,
		Total:  total,
		Page:   query.Page,
		Pages:  pages,
	}, nil
}
