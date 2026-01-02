// Package adapters provides infrastructure adapters.
package adapters

import (
	"context"
	"time"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/infrastructure/services"
	"github.com/orris-inc/orris/internal/shared/logger"
)

const (
	// batchStatusQueryTimeout is the maximum time allowed for batch status queries.
	batchStatusQueryTimeout = 10 * time.Second
)

// AgentStatusQuerierAdapter implements services.AgentStatusQuerier.
// It fetches agent status from Redis and resolves agent metadata from database.
type AgentStatusQuerierAdapter struct {
	agentRepo     forward.AgentRepository
	statusAdapter *ForwardAgentStatusAdapter
	logger        logger.Interface
}

// NewAgentStatusQuerierAdapter creates a new AgentStatusQuerierAdapter.
func NewAgentStatusQuerierAdapter(
	agentRepo forward.AgentRepository,
	statusAdapter *ForwardAgentStatusAdapter,
	log logger.Interface,
) *AgentStatusQuerierAdapter {
	return &AgentStatusQuerierAdapter{
		agentRepo:     agentRepo,
		statusAdapter: statusAdapter,
		logger:        log,
	}
}

// GetBatchStatus returns status for multiple agents by their SIDs.
// If agentSIDs is nil, returns status for all enabled agents.
// Returns a map of agentSID -> (name, status).
func (a *AgentStatusQuerierAdapter) GetBatchStatus(agentSIDs []string) (map[string]*services.AgentStatusData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), batchStatusQueryTimeout)
	defer cancel()

	result := make(map[string]*services.AgentStatusData)

	var agents []*forward.ForwardAgent
	var err error

	if agentSIDs == nil {
		// Get all enabled agents
		agents, err = a.agentRepo.ListEnabled(ctx)
		if err != nil {
			a.logger.Errorw("failed to list enabled agents",
				"error", err,
			)
			return nil, err
		}
	} else {
		// Get agents by SIDs
		agents = make([]*forward.ForwardAgent, 0, len(agentSIDs))
		for _, sid := range agentSIDs {
			agent, err := a.agentRepo.GetBySID(ctx, sid)
			if err != nil {
				a.logger.Warnw("failed to get agent by SID",
					"sid", sid,
					"error", err,
				)
				continue
			}
			if agent != nil {
				agents = append(agents, agent)
			}
		}
	}

	if len(agents) == 0 {
		return result, nil
	}

	// Build ID to agent mapping
	agentIDs := make([]uint, 0, len(agents))
	idToAgent := make(map[uint]*forward.ForwardAgent, len(agents))
	for _, agent := range agents {
		agentIDs = append(agentIDs, agent.ID())
		idToAgent[agent.ID()] = agent
	}

	// Batch get status from Redis
	statusMap, err := a.statusAdapter.GetMultipleStatus(ctx, agentIDs)
	if err != nil {
		a.logger.Errorw("failed to get batch agent status from redis",
			"error", err,
			"agent_count", len(agentIDs),
		)
		return nil, err
	}

	// Build result map
	for agentID, status := range statusMap {
		agent, ok := idToAgent[agentID]
		if !ok {
			continue
		}

		result[agent.SID()] = &services.AgentStatusData{
			Name:   agent.Name(),
			Status: a.toStatusResponse(status),
		}
	}

	return result, nil
}

// toStatusResponse converts internal DTO to response format.
func (a *AgentStatusQuerierAdapter) toStatusResponse(status *dto.AgentStatusDTO) *dto.AgentStatusDTO {
	if status == nil {
		return nil
	}
	return status
}

// Ensure AgentStatusQuerierAdapter implements AgentStatusQuerier interface.
var _ services.AgentStatusQuerier = (*AgentStatusQuerierAdapter)(nil)
