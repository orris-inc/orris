// Package adapters provides infrastructure adapters.
package adapters

import (
	"context"
	"time"

	commondto "github.com/orris-inc/orris/internal/application/common/dto"
	nodeUsecases "github.com/orris-inc/orris/internal/application/node/usecases"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/interfaces/adapters/cacheutil"
	"github.com/orris-inc/orris/internal/infrastructure/services"
	"github.com/orris-inc/orris/internal/shared/logger"
)

const (
	// batchNodeStatusQueryTimeout is the maximum time allowed for batch node status queries.
	batchNodeStatusQueryTimeout = 10 * time.Second

	// nodeMetadataCacheTTL is the TTL for node metadata cache.
	nodeMetadataCacheTTL = 1 * time.Minute
)

// NodeStatusQuerierAdapter implements services.NodeStatusQuerier.
// It fetches node status from Redis and resolves node metadata from database.
// Metadata is cached in memory to reduce database queries.
type NodeStatusQuerierAdapter struct {
	nodeRepo      node.NodeRepository
	statusQuerier *NodeSystemStatusQuerierAdapter
	cache         *cacheutil.MetadataCache[node.NodeMetadata]
	logger        logger.Interface
}

// NewNodeStatusQuerierAdapter creates a new NodeStatusQuerierAdapter.
func NewNodeStatusQuerierAdapter(
	nodeRepo node.NodeRepository,
	statusQuerier *NodeSystemStatusQuerierAdapter,
	log logger.Interface,
) *NodeStatusQuerierAdapter {
	return &NodeStatusQuerierAdapter{
		nodeRepo:      nodeRepo,
		statusQuerier: statusQuerier,
		cache:         cacheutil.NewMetadataCache[node.NodeMetadata](nodeMetadataCacheTTL),
		logger:        log,
	}
}

// refreshCacheIfNeeded refreshes the metadata cache if it's expired.
func (a *NodeStatusQuerierAdapter) refreshCacheIfNeeded(ctx context.Context) error {
	if !a.cache.TryRefresh() {
		return nil
	}

	// Refresh cache from database using lightweight query
	metadata, err := a.nodeRepo.GetAllMetadata(ctx)
	if err != nil {
		a.cache.AbortRefresh()
		return err
	}

	// Update cache with new data
	a.cache.FinishRefresh(metadata,
		func(m *node.NodeMetadata) uint { return m.ID },
		func(m *node.NodeMetadata) string { return m.SID },
	)

	a.logger.Debugw("node metadata cache refreshed", "node_count", len(metadata))
	return nil
}

// GetBatchStatus returns status for multiple nodes by their SIDs.
// If nodeSIDs is nil, returns status for all active nodes.
// Returns a map of nodeSID -> (name, status).
func (a *NodeStatusQuerierAdapter) GetBatchStatus(nodeSIDs []string) (map[string]*services.AgentStatusData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), batchNodeStatusQueryTimeout)
	defer cancel()

	result := make(map[string]*services.AgentStatusData)

	// Refresh cache if needed
	if err := a.refreshCacheIfNeeded(ctx); err != nil {
		a.logger.Errorw("failed to refresh node metadata cache", "error", err)
		return nil, err
	}

	// Get metadata from cache
	cacheResult := a.cache.GetBySIDs(nodeSIDs)

	if len(cacheResult.IDs) == 0 {
		return result, nil
	}

	// Build ID to metadata mapping
	idToMetadata := cacheutil.BuildIDMap(cacheResult.Items,
		func(m *node.NodeMetadata) uint { return m.ID },
	)

	// Batch get status from Redis
	statusMap, err := a.statusQuerier.GetMultipleNodeSystemStatus(ctx, cacheResult.IDs)
	if err != nil {
		a.logger.Errorw("failed to get batch node status from redis",
			"error", err,
			"node_count", len(cacheResult.IDs),
		)
		return nil, err
	}

	// Build result map
	for nodeID, status := range statusMap {
		m, ok := idToMetadata[nodeID]
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

// toStatusResponse converts internal NodeSystemStatus to commondto.SystemStatus for consistent JSON output.
// NodeSystemStatus embeds commondto.SystemStatus, so direct access works.
func (a *NodeStatusQuerierAdapter) toStatusResponse(status *nodeUsecases.NodeSystemStatus) *commondto.SystemStatus {
	if status == nil {
		return nil
	}
	return &status.SystemStatus
}

// Ensure NodeStatusQuerierAdapter implements NodeStatusQuerier interface.
var _ services.NodeStatusQuerier = (*NodeStatusQuerierAdapter)(nil)
