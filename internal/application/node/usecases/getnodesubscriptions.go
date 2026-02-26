package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/node/dto"
	"github.com/orris-inc/orris/internal/domain/node"
	vo "github.com/orris-inc/orris/internal/domain/node/valueobjects"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/infrastructure/config"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// GetNodeSubscriptionsCommand represents the command to get subscriptions authorized for a node
type GetNodeSubscriptionsCommand struct {
	NodeID uint
}

// GetNodeSubscriptionsResult contains the list of subscriptions for the node
type GetNodeSubscriptionsResult struct {
	Subscriptions *dto.NodeSubscriptionsResponse
}

// GetNodeSubscriptionsUseCase handles fetching subscription list for node agents
type GetNodeSubscriptionsUseCase struct {
	subscriptionRepo subscription.SubscriptionRepository
	planRepo         subscription.PlanRepository
	nodeRepo         node.NodeRepository
	logger           logger.Interface
}

// NewGetNodeSubscriptionsUseCase creates a new instance of GetNodeSubscriptionsUseCase
func NewGetNodeSubscriptionsUseCase(
	subscriptionRepo subscription.SubscriptionRepository,
	nodeRepo node.NodeRepository,
	planRepo subscription.PlanRepository,
	logger logger.Interface,
) *GetNodeSubscriptionsUseCase {
	return &GetNodeSubscriptionsUseCase{
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
		nodeRepo:         nodeRepo,
		logger:           logger,
	}
}

// Execute retrieves the list of subscriptions authorized to use the node
func (uc *GetNodeSubscriptionsUseCase) Execute(ctx context.Context, cmd GetNodeSubscriptionsCommand) (*GetNodeSubscriptionsResult, error) {
	if cmd.NodeID == 0 {
		return nil, fmt.Errorf("node_id is required")
	}

	// Get node entity to determine encryption method
	nodeEntity, err := uc.nodeRepo.GetByID(ctx, cmd.NodeID)
	if err != nil {
		uc.logger.Errorw("failed to get node",
			"error", err,
			"node_id", cmd.NodeID,
		)
		return nil, fmt.Errorf("failed to retrieve node")
	}

	// Extract encryption method
	encryptionMethod := ""
	if nodeEntity.Protocol().IsShadowsocks() {
		encryptionMethod = nodeEntity.EncryptionConfig().Method()
	}

	// Get all active subscriptions that can access this node
	subscriptions, err := uc.subscriptionRepo.GetActiveSubscriptionsByNodeID(ctx, cmd.NodeID)
	if err != nil {
		uc.logger.Errorw("failed to get node subscriptions",
			"error", err,
			"node_id", cmd.NodeID,
		)
		return nil, fmt.Errorf("failed to retrieve subscriptions for node")
	}

	// Collect unique plan IDs and batch load device limits
	planDeviceLimits := uc.loadPlanDeviceLimits(ctx, subscriptions)

	// Get HMAC secret from config for password generation
	hmacSecret := config.Get().Auth.JWT.Secret

	// Convert subscriptions to agent subscriptions response
	subscriptionInfos := dto.ToNodeSubscriptionsResponse(subscriptions, hmacSecret, encryptionMethod, planDeviceLimits)

	// Add a special node-to-node forwarding user
	// This allows other nodes to forward traffic to this node using a derived password
	var nodeForwardingPassword string
	if nodeEntity.Protocol().IsTrojan() {
		nodeForwardingPassword = vo.GenerateTrojanServerPassword(nodeEntity.TokenHash())
	} else if nodeEntity.Protocol().IsShadowsocks() {
		nodeForwardingPassword = vo.GenerateShadowsocksServerPassword(nodeEntity.TokenHash(), encryptionMethod)
	} else if nodeEntity.Protocol().IsAnyTLS() {
		nodeForwardingPassword = vo.GenerateAnyTLSServerPassword(nodeEntity.TokenHash())
	}

	if nodeForwardingPassword != "" {
		forwardingUser := dto.NodeSubscriptionInfo{
			SubscriptionSID: "node-forwarding",
			Password:        nodeForwardingPassword,
			Name:            fmt.Sprintf("node-forward-%s", nodeEntity.SID()),
			SpeedLimit:      0, // unlimited
			DeviceLimit:     0, // unlimited
			ExpireTime:      0, // never expires
		}
		subscriptionInfos.Subscriptions = append(subscriptionInfos.Subscriptions, forwardingUser)
	}

	uc.logger.Debugw("node subscriptions retrieved",
		"node_id", cmd.NodeID,
		"subscription_count", len(subscriptionInfos.Subscriptions),
	)

	return &GetNodeSubscriptionsResult{
		Subscriptions: subscriptionInfos,
	}, nil
}

// loadPlanDeviceLimits collects unique plan IDs from subscriptions, batch loads plans,
// and returns a map of planID -> device limit count.
func (uc *GetNodeSubscriptionsUseCase) loadPlanDeviceLimits(ctx context.Context, subscriptions []*subscription.Subscription) map[uint]int {
	if uc.planRepo == nil {
		return nil
	}

	// Collect unique plan IDs
	planIDSet := make(map[uint]struct{})
	for _, sub := range subscriptions {
		if sub != nil && sub.IsActive() {
			planIDSet[sub.PlanID()] = struct{}{}
		}
	}
	if len(planIDSet) == 0 {
		return nil
	}

	planIDs := make([]uint, 0, len(planIDSet))
	for id := range planIDSet {
		planIDs = append(planIDs, id)
	}

	plans, err := uc.planRepo.GetByIDs(ctx, planIDs)
	if err != nil {
		uc.logger.Errorw("failed to load plans for device limits, all limits will be disabled", "error", err, "plan_ids", planIDs)
		return nil
	}

	return dto.BuildPlanDeviceLimits(plans)
}
