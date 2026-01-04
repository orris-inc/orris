package usecases

import (
	"context"
	"strings"

	"github.com/orris-inc/orris/internal/application/node/dto"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/domain/resource"
	"github.com/orris-inc/orris/internal/domain/user"
	"github.com/orris-inc/orris/internal/shared/logger"
	sharedquery "github.com/orris-inc/orris/internal/shared/query"
	"github.com/orris-inc/orris/internal/shared/version"
)

type ListNodesQuery struct {
	Status           *string
	GroupID          *uint
	Search           *string
	Limit            int
	Offset           int
	SortBy           string
	SortOrder        string
	IncludeUserNodes bool // If false (default), only return admin-created nodes
}

// NodeListItem is deprecated - use dto.NodeDTO instead
// type NodeListItem struct {
// 	ID            uint
// 	Name          string
// 	ServerAddress string
// 	ServerPort    uint16
// 	Region        string
// 	Status        string
// 	SortOrder     int
// 	CreatedAt     string
// 	UpdatedAt     string
// }

type ListNodesResult struct {
	Nodes      []*dto.NodeDTO
	TotalCount int
	Limit      int
	Offset     int
}

// MultipleNodeSystemStatusQuerier defines the interface for querying multiple nodes' system status
type MultipleNodeSystemStatusQuerier interface {
	GetMultipleNodeSystemStatus(ctx context.Context, nodeIDs []uint) (map[uint]*NodeSystemStatus, error)
}

// LatestVersionQuerier provides access to the latest agent version.
type LatestVersionQuerier interface {
	// GetVersion returns the latest available agent version (e.g., "1.2.3").
	GetVersion(ctx context.Context) (string, error)
}

type ListNodesUseCase struct {
	nodeRepo          node.NodeRepository
	resourceGroupRepo resource.Repository
	userRepo          user.Repository
	statusQuerier     MultipleNodeSystemStatusQuerier
	versionQuerier    LatestVersionQuerier
	logger            logger.Interface
}

func NewListNodesUseCase(
	nodeRepo node.NodeRepository,
	resourceGroupRepo resource.Repository,
	userRepo user.Repository,
	statusQuerier MultipleNodeSystemStatusQuerier,
	versionQuerier LatestVersionQuerier,
	logger logger.Interface,
) *ListNodesUseCase {
	return &ListNodesUseCase{
		nodeRepo:          nodeRepo,
		resourceGroupRepo: resourceGroupRepo,
		userRepo:          userRepo,
		statusQuerier:     statusQuerier,
		versionQuerier:    versionQuerier,
		logger:            logger,
	}
}

func (uc *ListNodesUseCase) Execute(ctx context.Context, query ListNodesQuery) (*ListNodesResult, error) {
	// Validate and normalize pagination parameters
	if query.Limit <= 0 {
		query.Limit = 20
	}

	if query.Limit > 100 {
		query.Limit = 100
	}

	if query.Offset < 0 {
		query.Offset = 0
	}

	// Validate and normalize sort parameters
	if query.SortBy == "" {
		query.SortBy = "sort_order"
	}

	if query.SortOrder == "" {
		query.SortOrder = "asc"
	}

	// Calculate page from offset and limit
	page := 1
	if query.Limit > 0 && query.Offset > 0 {
		page = (query.Offset / query.Limit) + 1
	}

	// Build domain filter from query parameters
	adminOnly := !query.IncludeUserNodes
	filter := node.NodeFilter{
		BaseFilter: sharedquery.NewBaseFilter(
			sharedquery.WithPage(page, query.Limit),
			sharedquery.WithSort(query.SortBy, query.SortOrder),
		),
		Name:      query.Search,
		Status:    query.Status,
		AdminOnly: &adminOnly,
	}

	// Query nodes from repository
	nodes, totalCount, err := uc.nodeRepo.List(ctx, filter)
	if err != nil {
		uc.logger.Errorw("failed to list nodes from repository", "error", err)
		return nil, err
	}

	// Convert domain entities to DTOs
	nodeDTOs := dto.ToNodeDTOList(nodes)

	// Collect node IDs for batch status query and create ID mapping
	nodeIDs := make([]uint, 0, len(nodes))
	idToIndexMap := make(map[uint]int, len(nodes))
	// Collect unique group IDs for batch query
	groupIDSet := make(map[uint]bool)
	// Collect unique user IDs for batch query (for user-created nodes)
	userIDSet := make(map[uint]bool)
	userIDToNodeIndices := make(map[uint][]int) // Map user ID to node indices
	for i, n := range nodes {
		nodeIDs = append(nodeIDs, n.ID())
		idToIndexMap[n.ID()] = i
		for _, gid := range n.GroupIDs() {
			groupIDSet[gid] = true
		}
		// Collect user IDs for user-created nodes
		if n.UserID() != nil {
			uid := *n.UserID()
			userIDSet[uid] = true
			userIDToNodeIndices[uid] = append(userIDToNodeIndices[uid], i)
		}
	}

	// Batch query resource groups to resolve GroupID -> GroupSID
	groupIDToSID := make(map[uint]string)
	if len(groupIDSet) > 0 && uc.resourceGroupRepo != nil {
		// Convert set to slice for batch query
		groupIDs := make([]uint, 0, len(groupIDSet))
		for groupID := range groupIDSet {
			groupIDs = append(groupIDs, groupID)
		}

		groups, err := uc.resourceGroupRepo.GetByIDs(ctx, groupIDs)
		if err != nil {
			uc.logger.Warnw("failed to batch get resource groups, skipping",
				"group_ids", groupIDs,
				"error", err,
			)
		} else {
			for _, group := range groups {
				groupIDToSID[group.ID()] = group.SID()
			}
		}

		// Set GroupSIDs for each node DTO
		for i, n := range nodes {
			groupSIDs := make([]string, 0, len(n.GroupIDs()))
			for _, gid := range n.GroupIDs() {
				if sid, ok := groupIDToSID[gid]; ok {
					groupSIDs = append(groupSIDs, sid)
				}
			}
			if len(groupSIDs) > 0 {
				nodeDTOs[i].GroupSIDs = groupSIDs
			}
		}
	}

	// Batch query users to resolve UserID -> Owner info
	if len(userIDSet) > 0 && uc.userRepo != nil {
		// Convert set to slice for batch query
		userIDs := make([]uint, 0, len(userIDSet))
		for userID := range userIDSet {
			userIDs = append(userIDs, userID)
		}

		users, err := uc.userRepo.GetByIDs(ctx, userIDs)
		if err != nil {
			uc.logger.Warnw("failed to batch get users, skipping owner info",
				"user_ids", userIDs,
				"error", err,
			)
		} else {
			// Build userID -> user map
			userMap := make(map[uint]*user.User, len(users))
			for _, u := range users {
				userMap[u.ID()] = u
			}

			// Set Owner for each node DTO
			for userID, nodeIndices := range userIDToNodeIndices {
				if u, ok := userMap[userID]; ok {
					ownerDTO := &dto.NodeOwnerDTO{
						ID: u.SID(),
					}
					if u.Email() != nil {
						ownerDTO.Email = u.Email().String()
					}
					if u.Name() != nil {
						ownerDTO.Name = u.Name().String()
					}
					for _, idx := range nodeIndices {
						nodeDTOs[idx].Owner = ownerDTO
					}
				}
			}
		}
	}

	// Query system status for all nodes from Redis
	if len(nodeIDs) > 0 && uc.statusQuerier != nil {
		statusMap, err := uc.statusQuerier.GetMultipleNodeSystemStatus(ctx, nodeIDs)
		if err != nil {
			uc.logger.Warnw("failed to get nodes system status, continuing without it",
				"error", err,
			)
		} else {
			// Attach system status to each node DTO using the mapping
			for nodeID, status := range statusMap {
				if idx, ok := idToIndexMap[nodeID]; ok && status != nil {
					nodeDTOs[idx].SystemStatus = &dto.NodeSystemStatusDTO{
						CPUPercent:     status.CPUPercent,
						MemoryPercent:  status.MemoryPercent,
						MemoryUsed:     status.MemoryUsed,
						MemoryTotal:    status.MemoryTotal,
						MemoryAvail:    status.MemoryAvail,
						DiskPercent:    status.DiskPercent,
						DiskUsed:       status.DiskUsed,
						DiskTotal:      status.DiskTotal,
						UptimeSeconds:  status.UptimeSeconds,
						LoadAvg1:       status.LoadAvg1,
						LoadAvg5:       status.LoadAvg5,
						LoadAvg15:      status.LoadAvg15,
						NetworkRxBytes: status.NetworkRxBytes,
						NetworkTxBytes: status.NetworkTxBytes,
						NetworkRxRate:  status.NetworkRxRate,
						NetworkTxRate:  status.NetworkTxRate,
						TCPConnections: status.TCPConnections,
						UDPConnections: status.UDPConnections,
						PublicIPv4:     status.PublicIPv4,
						PublicIPv6:     status.PublicIPv6,
						AgentVersion:   status.AgentVersion,
						Platform:       status.Platform,
						Arch:           status.Arch,
						CPUCores:       status.CPUCores,
						CPUModelName:   status.CPUModelName,
						CPUMHz:         status.CPUMHz,

						// Swap memory
						SwapTotal:   status.SwapTotal,
						SwapUsed:    status.SwapUsed,
						SwapPercent: status.SwapPercent,

						// Disk I/O
						DiskReadBytes:  status.DiskReadBytes,
						DiskWriteBytes: status.DiskWriteBytes,
						DiskReadRate:   status.DiskReadRate,
						DiskWriteRate:  status.DiskWriteRate,
						DiskIOPS:       status.DiskIOPS,

						// Pressure Stall Information (PSI)
						PSICPUSome:    status.PSICPUSome,
						PSICPUFull:    status.PSICPUFull,
						PSIMemorySome: status.PSIMemorySome,
						PSIMemoryFull: status.PSIMemoryFull,
						PSIIOSome:     status.PSIIOSome,
						PSIIOFull:     status.PSIIOFull,

						// Network extended stats
						NetworkRxPackets: status.NetworkRxPackets,
						NetworkTxPackets: status.NetworkTxPackets,
						NetworkRxErrors:  status.NetworkRxErrors,
						NetworkTxErrors:  status.NetworkTxErrors,
						NetworkRxDropped: status.NetworkRxDropped,
						NetworkTxDropped: status.NetworkTxDropped,

						// Socket statistics
						SocketsUsed:      status.SocketsUsed,
						SocketsTCPInUse:  status.SocketsTCPInUse,
						SocketsUDPInUse:  status.SocketsUDPInUse,
						SocketsTCPOrphan: status.SocketsTCPOrphan,
						SocketsTCPTW:     status.SocketsTCPTW,

						// Process statistics
						ProcessesTotal:   status.ProcessesTotal,
						ProcessesRunning: status.ProcessesRunning,
						ProcessesBlocked: status.ProcessesBlocked,

						// File descriptors
						FileNrAllocated: status.FileNrAllocated,
						FileNrMax:       status.FileNrMax,

						// Context switches and interrupts
						ContextSwitches: status.ContextSwitches,
						Interrupts:      status.Interrupts,

						// Kernel info
						KernelVersion: status.KernelVersion,
						Hostname:      status.Hostname,

						// Virtual memory statistics
						VMPageIn:  status.VMPageIn,
						VMPageOut: status.VMPageOut,
						VMSwapIn:  status.VMSwapIn,
						VMSwapOut: status.VMSwapOut,
						VMOOMKill: status.VMOOMKill,

						// Entropy pool
						EntropyAvailable: status.EntropyAvailable,

						// Metadata
						UpdatedAt: status.UpdatedAt,
					}
					// Extract agent info to top-level fields for easy display
					// Normalize version format by removing "v" prefix for consistency
					nodeDTOs[idx].AgentVersion = strings.TrimPrefix(status.AgentVersion, "v")
					nodeDTOs[idx].Platform = status.Platform
					nodeDTOs[idx].Arch = status.Arch
				}
			}
		}
	}

	// Query latest version and calculate HasUpdate for each node
	if uc.versionQuerier != nil {
		latestVersion, err := uc.versionQuerier.GetVersion(ctx)
		if err != nil {
			uc.logger.Warnw("failed to get latest version, continuing without update check",
				"error", err,
			)
		} else {
			// Set HasUpdate for each node by comparing versions
			for i := range nodeDTOs {
				nodeDTOs[i].HasUpdate = version.HasNewerVersion(nodeDTOs[i].AgentVersion, latestVersion)
			}
		}
	}

	uc.logger.Debugw("nodes listed",
		"count", len(nodeDTOs),
		"total", totalCount,
	)

	return &ListNodesResult{
		Nodes:      nodeDTOs,
		TotalCount: int(totalCount),
		Limit:      query.Limit,
		Offset:     query.Offset,
	}, nil
}
