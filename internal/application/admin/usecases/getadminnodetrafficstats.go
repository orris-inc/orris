package usecases

import (
	"context"
	"sort"
	"time"

	dto "github.com/orris-inc/orris/internal/application/admin/dto"
	"github.com/orris-inc/orris/internal/application/admin/usecases/trafficstatsutil"
	nodeUsecases "github.com/orris-inc/orris/internal/application/node/usecases"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/infrastructure/cache"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

const (
	// maxAggregationLimit is the maximum number of records to fetch from MySQL
	// when aggregating data with Redis. This is a safety limit to prevent OOM.
	// If data exceeds this limit, results may be incomplete and a warning is logged.
	maxAggregationLimit = 10000
)

// GetAdminNodeTrafficStatsQuery represents the query parameters for node traffic statistics
type GetAdminNodeTrafficStatsQuery struct {
	From     time.Time
	To       time.Time
	Page     int
	PageSize int
}

// GetAdminNodeTrafficStatsUseCase handles retrieving traffic statistics grouped by node
type GetAdminNodeTrafficStatsUseCase struct {
	usageStatsRepo     subscription.SubscriptionUsageStatsRepository
	hourlyTrafficCache cache.HourlyTrafficCache
	nodeRepo           node.NodeRepository
	onlineSubCounter   nodeUsecases.NodeOnlineSubscriptionCounter
	logger             logger.Interface
}

// SetOnlineSubscriptionCounter injects an optional NodeOnlineSubscriptionCounter.
func (uc *GetAdminNodeTrafficStatsUseCase) SetOnlineSubscriptionCounter(c nodeUsecases.NodeOnlineSubscriptionCounter) {
	uc.onlineSubCounter = c
}

// NewGetAdminNodeTrafficStatsUseCase creates a new GetAdminNodeTrafficStatsUseCase
func NewGetAdminNodeTrafficStatsUseCase(
	usageStatsRepo subscription.SubscriptionUsageStatsRepository,
	hourlyTrafficCache cache.HourlyTrafficCache,
	nodeRepo node.NodeRepository,
	logger logger.Interface,
) *GetAdminNodeTrafficStatsUseCase {
	return &GetAdminNodeTrafficStatsUseCase{
		usageStatsRepo:     usageStatsRepo,
		hourlyTrafficCache: hourlyTrafficCache,
		nodeRepo:           nodeRepo,
		logger:             logger,
	}
}

// Execute retrieves node traffic statistics
func (uc *GetAdminNodeTrafficStatsUseCase) Execute(
	ctx context.Context,
	query GetAdminNodeTrafficStatsQuery,
) (*dto.NodeTrafficStatsResponse, error) {
	uc.logger.Debugw("fetching node traffic stats",
		"from", query.From,
		"to", query.To,
		"page", query.Page,
		"page_size", query.PageSize,
	)

	if err := uc.validateQuery(query); err != nil {
		uc.logger.Warnw("invalid node traffic stats query", "error", err)
		return nil, err
	}

	pagination := utils.ValidatePagination(query.Page, query.PageSize)
	timeWindow := trafficstatsutil.CalculateTimeWindow(query.From, query.To)

	// Resource type for nodes
	resourceType := subscription.ResourceTypeNode.String()

	// Prepare to merge data from MySQL and Redis
	nodeUsageMap := make(map[uint]*subscription.ResourceUsageSummary)

	// If query overlaps with Redis data window, get Redis data first
	if timeWindow.IncludesRedisWindow {
		redisFrom, redisTo := timeWindow.GetRedisQueryRange(query.From)

		redisTraffic, err := uc.hourlyTrafficCache.GetTrafficGroupedByResourceID(ctx, resourceType, redisFrom, redisTo)
		if err != nil {
			uc.logger.Warnw("failed to get node traffic from Redis",
				"error", err,
			)
		} else {
			for nodeID, traffic := range redisTraffic {
				nodeUsageMap[nodeID] = &subscription.ResourceUsageSummary{
					ResourceType: resourceType,
					ResourceID:   nodeID,
					Upload:       traffic.Upload,
					Download:     traffic.Download,
					Total:        traffic.Total,
				}
			}
			uc.logger.Debugw("got node traffic from Redis",
				"nodes_count", len(redisTraffic),
			)
		}
	}

	// If query includes historical data (before Redis window), get MySQL data
	if timeWindow.IncludesHistory {
		_, mysqlTo := timeWindow.GetMySQLQueryRange(query.From)

		resourceUsages, mysqlTotal, err := uc.usageStatsRepo.GetUsageGroupedByResourceID(
			ctx,
			resourceType,
			query.From,
			mysqlTo,
			1,                   // Get all data without pagination for merging
			maxAggregationLimit, // Safety limit to prevent OOM
		)
		if err != nil {
			uc.logger.Errorw("failed to fetch node usage", "error", err)
			return nil, errors.NewInternalError("failed to fetch node usage")
		}

		// Warn if data may be truncated
		if mysqlTotal > int64(maxAggregationLimit) {
			uc.logger.Warnw("node traffic data may be incomplete due to aggregation limit",
				"total_records", mysqlTotal,
				"limit", maxAggregationLimit,
				"from", query.From,
				"to", mysqlTo,
			)
		}

		// Merge MySQL data with Redis data
		for _, usage := range resourceUsages {
			if existing, ok := nodeUsageMap[usage.ResourceID]; ok {
				existing.Upload += usage.Upload
				existing.Download += usage.Download
				existing.Total += usage.Total
			} else {
				nodeUsageMap[usage.ResourceID] = &subscription.ResourceUsageSummary{
					ResourceType: resourceType,
					ResourceID:   usage.ResourceID,
					Upload:       usage.Upload,
					Download:     usage.Download,
					Total:        usage.Total,
				}
			}
		}
	}

	// If no data found
	if len(nodeUsageMap) == 0 {
		return &dto.NodeTrafficStatsResponse{
			Items:    []dto.NodeTrafficStatsItem{},
			Total:    0,
			Page:     pagination.Page,
			PageSize: pagination.PageSize,
		}, nil
	}

	// Convert map to slice and sort by total descending
	resourceUsages := make([]subscription.ResourceUsageSummary, 0, len(nodeUsageMap))
	for _, usage := range nodeUsageMap {
		resourceUsages = append(resourceUsages, *usage)
	}
	sort.Slice(resourceUsages, func(i, j int) bool {
		return resourceUsages[i].Total > resourceUsages[j].Total
	})

	// Apply pagination
	total := int64(len(resourceUsages))
	start, end := utils.ApplyPagination(len(resourceUsages), pagination.Page, pagination.PageSize)
	pagedUsages := resourceUsages[start:end]

	// Extract node IDs
	nodeIDs := make([]uint, len(pagedUsages))
	for i, usage := range pagedUsages {
		nodeIDs[i] = usage.ResourceID
	}

	// Fetch nodes
	nodes, err := uc.nodeRepo.GetByIDs(ctx, nodeIDs)
	if err != nil {
		uc.logger.Errorw("failed to fetch nodes", "error", err)
		return nil, errors.NewInternalError("failed to fetch node information")
	}

	// Create node map for quick lookup
	nodesMap := make(map[uint]*node.Node)
	for _, n := range nodes {
		nodesMap[n.ID()] = n
	}

	// Build response items
	items := make([]dto.NodeTrafficStatsItem, 0, len(pagedUsages))
	for _, usage := range pagedUsages {
		n, ok := nodesMap[usage.ResourceID]
		if !ok {
			// Node might have been deleted, skip
			uc.logger.Warnw("node not found for usage record", "node_id", usage.ResourceID)
			continue
		}

		items = append(items, dto.NodeTrafficStatsItem{
			NodeSID:  n.SID(),
			NodeName: n.Name(),
			Status:   n.Status().String(),
			Upload:   usage.Upload,
			Download: usage.Download,
			Total:    usage.Total,
		})
	}

	// Batch query online subscription counts for paged nodes
	if len(nodeIDs) > 0 && uc.onlineSubCounter != nil {
		countMap, err := uc.onlineSubCounter.GetNodeOnlineSubscriptionCounts(ctx, nodeIDs)
		if err != nil {
			uc.logger.Warnw("failed to get node online subscription counts, continuing without it",
				"error", err,
			)
		} else {
			// Build nodeID -> items index map
			sidToIndex := make(map[string]int, len(items))
			for i, item := range items {
				sidToIndex[item.NodeSID] = i
			}
			for nodeID, count := range countMap {
				if n, ok := nodesMap[nodeID]; ok {
					if idx, ok := sidToIndex[n.SID()]; ok {
						items[idx].OnlineSubscriptionCount = count
					}
				}
			}
		}
	}

	response := &dto.NodeTrafficStatsResponse{
		Items:    items,
		Total:    total,
		Page:     pagination.Page,
		PageSize: pagination.PageSize,
	}

	uc.logger.Debugw("node traffic stats fetched successfully",
		"count", len(items),
		"total", total,
	)

	return response, nil
}

func (uc *GetAdminNodeTrafficStatsUseCase) validateQuery(query GetAdminNodeTrafficStatsQuery) error {
	return trafficstatsutil.ValidateTimeRange(query.From, query.To)
}
