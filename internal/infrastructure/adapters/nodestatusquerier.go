// Package adapters provides infrastructure adapters.
package adapters

import (
	"context"
	"time"

	nodeUsecases "github.com/orris-inc/orris/internal/application/node/usecases"
	"github.com/orris-inc/orris/internal/domain/node"
	nodevo "github.com/orris-inc/orris/internal/domain/node/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/services"
	"github.com/orris-inc/orris/internal/shared/logger"
)

const (
	// batchNodeStatusQueryTimeout is the maximum time allowed for batch node status queries.
	batchNodeStatusQueryTimeout = 10 * time.Second
)

// NodeStatusQuerierAdapter implements services.NodeStatusQuerier.
// It fetches node status from Redis and resolves node metadata from database.
type NodeStatusQuerierAdapter struct {
	nodeRepo      node.NodeRepository
	statusQuerier *NodeSystemStatusQuerierAdapter
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
		logger:        log,
	}
}

// GetBatchStatus returns status for multiple nodes by their SIDs.
// If nodeSIDs is nil, returns status for all active nodes.
// Returns a map of nodeSID -> (name, status).
func (a *NodeStatusQuerierAdapter) GetBatchStatus(nodeSIDs []string) (map[string]*services.AgentStatusData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), batchNodeStatusQueryTimeout)
	defer cancel()

	result := make(map[string]*services.AgentStatusData)

	var nodes []*node.Node
	var err error

	if nodeSIDs == nil {
		// Get all active nodes
		activeStatus := string(nodevo.NodeStatusActive)
		nodes, _, err = a.nodeRepo.List(ctx, node.NodeFilter{
			Status: &activeStatus,
		})
		if err != nil {
			a.logger.Errorw("failed to list nodes",
				"error", err,
			)
			return nil, err
		}
	} else {
		// Get nodes by SIDs
		nodes = make([]*node.Node, 0, len(nodeSIDs))
		for _, sid := range nodeSIDs {
			n, err := a.nodeRepo.GetBySID(ctx, sid)
			if err != nil {
				a.logger.Warnw("failed to get node by SID",
					"sid", sid,
					"error", err,
				)
				continue
			}
			if n != nil {
				nodes = append(nodes, n)
			}
		}
	}

	if len(nodes) == 0 {
		return result, nil
	}

	// Build ID to node mapping
	nodeIDs := make([]uint, 0, len(nodes))
	idToNode := make(map[uint]*node.Node, len(nodes))
	for _, n := range nodes {
		nodeIDs = append(nodeIDs, n.ID())
		idToNode[n.ID()] = n
	}

	// Batch get status from Redis
	statusMap, err := a.statusQuerier.GetMultipleNodeSystemStatus(ctx, nodeIDs)
	if err != nil {
		a.logger.Errorw("failed to get batch node status from redis",
			"error", err,
			"node_count", len(nodeIDs),
		)
		return nil, err
	}

	// Build result map
	for nodeID, status := range statusMap {
		n, ok := idToNode[nodeID]
		if !ok {
			continue
		}

		result[n.SID()] = &services.AgentStatusData{
			Name:   n.Name(),
			Status: a.toStatusResponse(status),
		}
	}

	return result, nil
}

// toStatusResponse converts internal status to response format.
func (a *NodeStatusQuerierAdapter) toStatusResponse(status *nodeUsecases.NodeSystemStatus) *nodeUsecases.NodeSystemStatus {
	if status == nil {
		return nil
	}
	return status
}

// Ensure NodeStatusQuerierAdapter implements NodeStatusQuerier interface.
var _ services.NodeStatusQuerier = (*NodeStatusQuerierAdapter)(nil)
