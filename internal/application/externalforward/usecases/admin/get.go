package admin

import (
	"context"

	"github.com/orris-inc/orris/internal/application/externalforward/dto"
	"github.com/orris-inc/orris/internal/domain/externalforward"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/domain/resource"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// AdminGetExternalForwardRuleQuery represents the input for getting a single external forward rule.
type AdminGetExternalForwardRuleQuery struct {
	SID string
}

// AdminGetExternalForwardRuleResult represents the output of getting a single external forward rule.
type AdminGetExternalForwardRuleResult struct {
	Rule *dto.AdminExternalForwardRuleDTO `json:"rule"`
}

// AdminGetExternalForwardRuleUseCase handles getting a single external forward rule.
type AdminGetExternalForwardRuleUseCase struct {
	repo              externalforward.Repository
	resourceGroupRepo resource.Repository
	nodeRepo          node.NodeRepository
	logger            logger.Interface
}

// NewAdminGetExternalForwardRuleUseCase creates a new admin get use case.
func NewAdminGetExternalForwardRuleUseCase(
	repo externalforward.Repository,
	resourceGroupRepo resource.Repository,
	nodeRepo node.NodeRepository,
	logger logger.Interface,
) *AdminGetExternalForwardRuleUseCase {
	return &AdminGetExternalForwardRuleUseCase{
		repo:              repo,
		resourceGroupRepo: resourceGroupRepo,
		nodeRepo:          nodeRepo,
		logger:            logger,
	}
}

// Execute gets a single external forward rule by SID.
func (uc *AdminGetExternalForwardRuleUseCase) Execute(ctx context.Context, query AdminGetExternalForwardRuleQuery) (*AdminGetExternalForwardRuleResult, error) {
	uc.logger.Infow("executing admin get external forward rule use case", "sid", query.SID)

	rule, err := uc.repo.GetBySID(ctx, query.SID)
	if err != nil {
		return nil, err
	}

	// Fetch group SIDs
	groupIDToSID := make(map[uint]string)
	if len(rule.GroupIDs()) > 0 {
		groups, err := uc.resourceGroupRepo.GetByIDs(ctx, rule.GroupIDs())
		if err != nil {
			uc.logger.Warnw("failed to get resource groups", "error", err)
			// Continue without group SIDs
		} else {
			for _, g := range groups {
				groupIDToSID[g.ID()] = g.SID()
			}
		}
	}

	// Fetch node info
	nodeIDToInfo := make(map[uint]*dto.NodeInfo)
	if rule.NodeID() != nil {
		nodes, err := uc.nodeRepo.GetByIDs(ctx, []uint{*rule.NodeID()})
		if err != nil || len(nodes) == 0 {
			uc.logger.Warnw("failed to get node", "node_id", *rule.NodeID(), "error", err)
			// Continue without node info
		} else {
			n := nodes[0]
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

	return &AdminGetExternalForwardRuleResult{
		Rule: dto.FromDomainToAdmin(rule, &dto.AdminDTOLookups{
			GroupIDToSID: groupIDToSID,
			NodeIDToInfo: nodeIDToInfo,
		}),
	}, nil
}
