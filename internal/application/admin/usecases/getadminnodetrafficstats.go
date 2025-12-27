package usecases

import (
	"context"
	"time"

	dto "github.com/orris-inc/orris/internal/application/admin/dto"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/constants"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
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
	usageRepo subscription.SubscriptionUsageRepository
	nodeRepo  node.NodeRepository
	logger    logger.Interface
}

// NewGetAdminNodeTrafficStatsUseCase creates a new GetAdminNodeTrafficStatsUseCase
func NewGetAdminNodeTrafficStatsUseCase(
	usageRepo subscription.SubscriptionUsageRepository,
	nodeRepo node.NodeRepository,
	logger logger.Interface,
) *GetAdminNodeTrafficStatsUseCase {
	return &GetAdminNodeTrafficStatsUseCase{
		usageRepo: usageRepo,
		nodeRepo:  nodeRepo,
		logger:    logger,
	}
}

// Execute retrieves node traffic statistics
func (uc *GetAdminNodeTrafficStatsUseCase) Execute(
	ctx context.Context,
	query GetAdminNodeTrafficStatsQuery,
) (*dto.NodeTrafficStatsResponse, error) {
	uc.logger.Infow("fetching node traffic stats",
		"from", query.From,
		"to", query.To,
		"page", query.Page,
		"page_size", query.PageSize,
	)

	if err := uc.validateQuery(query); err != nil {
		uc.logger.Errorw("invalid node traffic stats query", "error", err)
		return nil, err
	}

	page, pageSize := uc.getPaginationParams(query)

	// Adjust 'to' time to end of day to include all records from that day
	adjustedTo := biztime.EndOfDayUTC(query.To)

	// Get usage data grouped by node (resource_type = "node")
	resourceType := subscription.ResourceTypeNode.String()
	resourceUsages, total, err := uc.usageRepo.GetUsageGroupedByResourceID(
		ctx,
		resourceType,
		query.From,
		adjustedTo,
		page,
		pageSize,
	)
	if err != nil {
		uc.logger.Errorw("failed to fetch node usage", "error", err)
		return nil, errors.NewInternalError("failed to fetch node usage")
	}

	if len(resourceUsages) == 0 {
		return &dto.NodeTrafficStatsResponse{
			Items:    []dto.NodeTrafficStatsItem{},
			Total:    0,
			Page:     page,
			PageSize: pageSize,
		}, nil
	}

	// Extract node IDs
	nodeIDs := make([]uint, len(resourceUsages))
	for i, usage := range resourceUsages {
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
	items := make([]dto.NodeTrafficStatsItem, 0, len(resourceUsages))
	for _, usage := range resourceUsages {
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

	response := &dto.NodeTrafficStatsResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}

	uc.logger.Infow("node traffic stats fetched successfully",
		"count", len(items),
		"total", total,
	)

	return response, nil
}

func (uc *GetAdminNodeTrafficStatsUseCase) validateQuery(query GetAdminNodeTrafficStatsQuery) error {
	if query.From.IsZero() {
		return errors.NewValidationError("from time is required")
	}

	if query.To.IsZero() {
		return errors.NewValidationError("to time is required")
	}

	if query.To.Before(query.From) {
		return errors.NewValidationError("to time must be after from time")
	}

	if query.Page < 0 {
		return errors.NewValidationError("page must be non-negative")
	}

	if query.PageSize < 0 {
		return errors.NewValidationError("page_size must be non-negative")
	}

	return nil
}

func (uc *GetAdminNodeTrafficStatsUseCase) getPaginationParams(query GetAdminNodeTrafficStatsQuery) (int, int) {
	page := query.Page
	if page == 0 {
		page = constants.DefaultPage
	}

	pageSize := query.PageSize
	if pageSize == 0 {
		pageSize = constants.DefaultPageSize
	}
	if pageSize > constants.MaxPageSize {
		pageSize = constants.MaxPageSize
	}

	return page, pageSize
}
