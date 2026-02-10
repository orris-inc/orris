// Package adapters provides infrastructure adapters.
package adapters

import (
	"context"
	"time"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/interfaces/adapters/cacheutil"
	"github.com/orris-inc/orris/internal/infrastructure/services"
	"github.com/orris-inc/orris/internal/shared/logger"
)

const (
	// batchStatusQueryTimeout is the maximum time allowed for batch status queries.
	batchStatusQueryTimeout = 10 * time.Second

	// agentMetadataCacheTTL is the TTL for agent metadata cache.
	agentMetadataCacheTTL = 1 * time.Minute
)

// AgentStatusQuerierAdapter implements services.AgentStatusQuerier.
// It fetches agent status from Redis and resolves agent metadata from database.
// Metadata is cached in memory to reduce database queries.
type AgentStatusQuerierAdapter struct {
	agentRepo     forward.AgentRepository
	statusAdapter *ForwardAgentStatusAdapter
	cache         *cacheutil.MetadataCache[forward.AgentMetadata]
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
		cache:         cacheutil.NewMetadataCache[forward.AgentMetadata](agentMetadataCacheTTL),
		logger:        log,
	}
}

// refreshCacheIfNeeded refreshes the metadata cache if it's expired.
func (a *AgentStatusQuerierAdapter) refreshCacheIfNeeded(ctx context.Context) error {
	if !a.cache.TryRefresh() {
		return nil
	}

	// Refresh cache from database using lightweight query
	metadata, err := a.agentRepo.GetAllEnabledMetadata(ctx)
	if err != nil {
		a.cache.AbortRefresh()
		return err
	}

	// Update cache with new data
	a.cache.FinishRefresh(metadata,
		func(m *forward.AgentMetadata) uint { return m.ID },
		func(m *forward.AgentMetadata) string { return m.SID },
	)

	a.logger.Debugw("agent metadata cache refreshed", "agent_count", len(metadata))
	return nil
}

// GetBatchStatus returns status for multiple agents by their SIDs.
// If agentSIDs is nil, returns status for all enabled agents.
// Returns a map of agentSID -> (name, status).
func (a *AgentStatusQuerierAdapter) GetBatchStatus(agentSIDs []string) (map[string]*services.AgentStatusData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), batchStatusQueryTimeout)
	defer cancel()

	result := make(map[string]*services.AgentStatusData)

	// Refresh cache if needed
	if err := a.refreshCacheIfNeeded(ctx); err != nil {
		a.logger.Errorw("failed to refresh agent metadata cache", "error", err)
		return nil, err
	}

	// Get metadata from cache
	cacheResult := a.cache.GetBySIDs(agentSIDs)

	if len(cacheResult.IDs) == 0 {
		return result, nil
	}

	// Build ID to metadata mapping
	idToMetadata := cacheutil.BuildIDMap(cacheResult.Items,
		func(m *forward.AgentMetadata) uint { return m.ID },
	)

	// Batch get status from Redis
	statusMap, err := a.statusAdapter.GetMultipleStatus(ctx, cacheResult.IDs)
	if err != nil {
		a.logger.Errorw("failed to get batch agent status from redis",
			"error", err,
			"agent_count", len(cacheResult.IDs),
		)
		return nil, err
	}

	// Build result map
	for agentID, status := range statusMap {
		m, ok := idToMetadata[agentID]
		if !ok {
			continue
		}

		result[m.SID] = &services.AgentStatusData{
			Name:   m.Name,
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
