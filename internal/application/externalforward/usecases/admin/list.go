// Package admin provides admin use cases for external forward rules.
package admin

import (
	"context"

	"github.com/orris-inc/orris/internal/application/externalforward/dto"
	"github.com/orris-inc/orris/internal/domain/externalforward"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/domain/resource"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// AdminListExternalForwardRulesQuery represents the input for admin listing external forward rules.
type AdminListExternalForwardRulesQuery struct {
	Page           int
	PageSize       int
	SubscriptionID *uint
	UserID         *uint
	Status         string
	ExternalSource string
	OrderBy        string
	Order          string
}

// AdminListExternalForwardRulesResult represents the output of admin listing external forward rules.
type AdminListExternalForwardRulesResult struct {
	Rules []*dto.AdminExternalForwardRuleDTO `json:"rules"`
	Total int64                              `json:"total"`
}

// AdminListExternalForwardRulesUseCase handles admin listing external forward rules.
type AdminListExternalForwardRulesUseCase struct {
	repo              externalforward.Repository
	resourceGroupRepo resource.Repository
	nodeRepo          node.NodeRepository
	logger            logger.Interface
}

// NewAdminListExternalForwardRulesUseCase creates a new admin list use case.
func NewAdminListExternalForwardRulesUseCase(
	repo externalforward.Repository,
	resourceGroupRepo resource.Repository,
	nodeRepo node.NodeRepository,
	logger logger.Interface,
) *AdminListExternalForwardRulesUseCase {
	return &AdminListExternalForwardRulesUseCase{
		repo:              repo,
		resourceGroupRepo: resourceGroupRepo,
		nodeRepo:          nodeRepo,
		logger:            logger,
	}
}

// Execute lists external forward rules for admin with optional filters.
func (uc *AdminListExternalForwardRulesUseCase) Execute(ctx context.Context, query AdminListExternalForwardRulesQuery) (*AdminListExternalForwardRulesResult, error) {
	uc.logger.Infow("executing admin list external forward rules use case",
		"page", query.Page,
		"page_size", query.PageSize,
		"subscription_id", query.SubscriptionID,
		"user_id", query.UserID,
		"status", query.Status,
		"external_source", query.ExternalSource,
	)

	filter := externalforward.AdminListFilter{
		Page:           query.Page,
		PageSize:       query.PageSize,
		SubscriptionID: query.SubscriptionID,
		UserID:         query.UserID,
		Status:         query.Status,
		ExternalSource: query.ExternalSource,
		OrderBy:        query.OrderBy,
		Order:          query.Order,
	}

	rules, total, err := uc.repo.ListWithPagination(ctx, filter)
	if err != nil {
		uc.logger.Errorw("failed to list external forward rules", "error", err)
		return nil, err
	}

	// Collect all unique group IDs from the rules for batch lookup
	groupIDSet := make(map[uint]bool)
	for _, rule := range rules {
		for _, gid := range rule.GroupIDs() {
			groupIDSet[gid] = true
		}
	}

	// Batch fetch all resource groups to avoid N+1 queries
	groupIDToSID := make(map[uint]string)
	if len(groupIDSet) > 0 {
		groupIDs := make([]uint, 0, len(groupIDSet))
		for gid := range groupIDSet {
			groupIDs = append(groupIDs, gid)
		}
		groups, err := uc.resourceGroupRepo.GetByIDs(ctx, groupIDs)
		if err != nil {
			uc.logger.Warnw("failed to batch get resource groups", "error", err)
			// Continue without group SIDs rather than failing the whole request
		} else {
			for _, g := range groups {
				groupIDToSID[g.ID()] = g.SID()
			}
		}
	}

	// Collect node IDs and batch fetch node info
	nodeIDToInfo := make(map[uint]*dto.NodeInfo)
	var nodeIDs []uint
	for _, rule := range rules {
		if rule.NodeID() != nil {
			nodeIDs = append(nodeIDs, *rule.NodeID())
		}
	}
	if len(nodeIDs) > 0 {
		nodes, err := uc.nodeRepo.GetByIDs(ctx, nodeIDs)
		if err != nil {
			uc.logger.Warnw("failed to get nodes", "error", err)
			// Continue without node info rather than failing the request
		} else {
			for _, n := range nodes {
				info := &dto.NodeInfo{
					SID:           n.SID(),
					ServerAddress: n.ServerAddress().Value(),
				}
				if n.PublicIPv4() != nil {
					info.PublicIPv4 = *n.PublicIPv4()
				}
				if n.PublicIPv6() != nil {
					info.PublicIPv6 = *n.PublicIPv6()
				}
				nodeIDToInfo[n.ID()] = info
			}
		}
	}

	return &AdminListExternalForwardRulesResult{
		Rules: dto.FromDomainListToAdmin(rules, &dto.AdminDTOLookups{
			GroupIDToSID: groupIDToSID,
			NodeIDToInfo: nodeIDToInfo,
		}),
		Total: total,
	}, nil
}
