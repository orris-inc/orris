// Package adapters provides infrastructure adapters.
package adapters

import (
	"context"
	"sync"
	"time"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/infrastructure/services"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

const (
	// batchStatusQueryTimeout is the maximum time allowed for batch status queries.
	batchStatusQueryTimeout = 10 * time.Second

	// agentMetadataCacheTTL is the TTL for agent metadata cache.
	agentMetadataCacheTTL = 1 * time.Minute
)

// agentMetadataCache holds cached agent metadata.
type agentMetadataCache struct {
	// allAgents maps agentID -> metadata for all enabled agents
	allAgents map[uint]*forward.AgentMetadata
	// sidToID maps SID -> agentID for quick lookup
	sidToID map[string]uint
	// lastUpdated is the time when cache was last refreshed
	lastUpdated time.Time
	mu          sync.RWMutex
}

// AgentStatusQuerierAdapter implements services.AgentStatusQuerier.
// It fetches agent status from Redis and resolves agent metadata from database.
// Metadata is cached in memory to reduce database queries.
type AgentStatusQuerierAdapter struct {
	agentRepo     forward.AgentRepository
	statusAdapter *ForwardAgentStatusAdapter
	cache         *agentMetadataCache
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
		cache: &agentMetadataCache{
			allAgents: make(map[uint]*forward.AgentMetadata),
			sidToID:   make(map[string]uint),
		},
		logger: log,
	}
}

// refreshCacheIfNeeded refreshes the metadata cache if it's expired.
func (a *AgentStatusQuerierAdapter) refreshCacheIfNeeded(ctx context.Context) error {
	a.cache.mu.RLock()
	needsRefresh := biztime.NowUTC().Sub(a.cache.lastUpdated) > agentMetadataCacheTTL
	a.cache.mu.RUnlock()

	if !needsRefresh {
		return nil
	}

	// Acquire write lock and check again
	a.cache.mu.Lock()
	defer a.cache.mu.Unlock()

	// Double-check after acquiring write lock
	if biztime.NowUTC().Sub(a.cache.lastUpdated) <= agentMetadataCacheTTL {
		return nil
	}

	// Refresh cache from database using lightweight query
	metadata, err := a.agentRepo.GetAllEnabledMetadata(ctx)
	if err != nil {
		return err
	}

	// Rebuild cache
	a.cache.allAgents = make(map[uint]*forward.AgentMetadata, len(metadata))
	a.cache.sidToID = make(map[string]uint, len(metadata))
	for _, m := range metadata {
		a.cache.allAgents[m.ID] = m
		a.cache.sidToID[m.SID] = m.ID
	}
	a.cache.lastUpdated = biztime.NowUTC()

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
	a.cache.mu.RLock()
	var agentIDs []uint
	var agentMetadata []*forward.AgentMetadata

	if agentSIDs == nil {
		// Get all agents from cache
		agentIDs = make([]uint, 0, len(a.cache.allAgents))
		agentMetadata = make([]*forward.AgentMetadata, 0, len(a.cache.allAgents))
		for id, m := range a.cache.allAgents {
			agentIDs = append(agentIDs, id)
			agentMetadata = append(agentMetadata, m)
		}
	} else {
		// Get specific agents from cache
		agentIDs = make([]uint, 0, len(agentSIDs))
		agentMetadata = make([]*forward.AgentMetadata, 0, len(agentSIDs))
		for _, sid := range agentSIDs {
			if id, ok := a.cache.sidToID[sid]; ok {
				agentIDs = append(agentIDs, id)
				if m, ok := a.cache.allAgents[id]; ok {
					agentMetadata = append(agentMetadata, m)
				}
			}
		}
	}
	a.cache.mu.RUnlock()

	if len(agentIDs) == 0 {
		return result, nil
	}

	// Build ID to metadata mapping
	idToMetadata := make(map[uint]*forward.AgentMetadata, len(agentMetadata))
	for _, m := range agentMetadata {
		idToMetadata[m.ID] = m
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
