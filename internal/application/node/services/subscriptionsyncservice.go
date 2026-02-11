// Package services provides application-level services for the node domain.
package services

import (
	"context"
	"encoding/json"

	"github.com/orris-inc/orris/internal/application/node/dto"
	"github.com/orris-inc/orris/internal/domain/node"
	vo "github.com/orris-inc/orris/internal/domain/node/valueobjects"
	"github.com/orris-inc/orris/internal/domain/resource"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/infrastructure/config"
	"github.com/orris-inc/orris/internal/infrastructure/pubsub"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// SubscriptionSyncService handles subscription synchronization to node agents.
// It pushes subscription changes to relevant node agents via WebSocket.
type SubscriptionSyncService struct {
	nodeRepo          node.NodeRepository
	subscriptionRepo  subscription.SubscriptionRepository
	resourceGroupRepo resource.Repository
	hub               NodeSyncHub
	eventPublisher    pubsub.SubscriptionEventPublisher
	logger            logger.Interface
}

// NewSubscriptionSyncService creates a new SubscriptionSyncService.
func NewSubscriptionSyncService(
	nodeRepo node.NodeRepository,
	subscriptionRepo subscription.SubscriptionRepository,
	resourceGroupRepo resource.Repository,
	hub NodeSyncHub,
	log logger.Interface,
) *SubscriptionSyncService {
	return &SubscriptionSyncService{
		nodeRepo:          nodeRepo,
		subscriptionRepo:  subscriptionRepo,
		resourceGroupRepo: resourceGroupRepo,
		hub:               hub,
		logger:            log,
	}
}

// SetEventPublisher sets the event publisher for cross-instance synchronization.
// This should be called after creating the service to enable Redis Pub/Sub.
func (s *SubscriptionSyncService) SetEventPublisher(publisher pubsub.SubscriptionEventPublisher) {
	s.eventPublisher = publisher
}

// NotifySubscriptionChange notifies relevant nodes about subscription changes.
// It finds all nodes that can access the subscription's plan and sends them updates.
func (s *SubscriptionSyncService) NotifySubscriptionChange(
	ctx context.Context,
	sub *subscription.Subscription,
	changeType string,
) error {
	if sub == nil {
		return nil
	}

	s.logger.Debugw("notifying nodes of subscription change",
		"subscription_id", sub.ID(),
		"subscription_sid", sub.SID(),
		"plan_id", sub.PlanID(),
		"change_type", changeType,
	)

	// Get resource groups for this plan
	groups, err := s.resourceGroupRepo.GetByPlanID(ctx, sub.PlanID())
	if err != nil {
		s.logger.Errorw("failed to get resource groups for plan",
			"plan_id", sub.PlanID(),
			"error", err,
		)
		return err
	}

	if len(groups) == 0 {
		s.logger.Debugw("no resource groups found for plan, skipping notification",
			"plan_id", sub.PlanID(),
		)
		return nil
	}

	// Collect group IDs
	groupIDs := make([]uint, 0, len(groups))
	for _, g := range groups {
		if g.IsActive() {
			groupIDs = append(groupIDs, g.ID())
		}
	}

	if len(groupIDs) == 0 {
		s.logger.Debugw("no active resource groups found for plan",
			"plan_id", sub.PlanID(),
		)
		return nil
	}

	// Get nodes in these resource groups
	nodes, _, err := s.nodeRepo.List(ctx, node.NodeFilter{
		GroupIDs: groupIDs,
	})
	if err != nil {
		s.logger.Errorw("failed to get nodes for resource groups",
			"group_ids", groupIDs,
			"error", err,
		)
		return err
	}

	if len(nodes) == 0 {
		s.logger.Debugw("no nodes found in resource groups",
			"group_ids", groupIDs,
		)
		return nil
	}

	// Get HMAC secret from config for password generation
	hmacSecret := config.Get().Auth.JWT.Secret

	// Notify each online node
	notifiedCount := 0
	for _, n := range nodes {
		if !s.hub.IsNodeOnline(n.ID()) {
			continue
		}

		// Determine encryption method for password generation
		encryptionMethod := ""
		if n.Protocol().IsShadowsocks() {
			encryptionMethod = n.EncryptionConfig().Method()
		}

		// Build subscription info for this node
		subscriptionInfo := s.buildSubscriptionInfo(sub, hmacSecret, encryptionMethod)

		if err := s.sendSubscriptionSync(n, changeType, []dto.NodeSubscriptionInfo{subscriptionInfo}); err != nil {
			s.logger.Warnw("failed to send subscription sync to node",
				"node_id", n.ID(),
				"node_sid", n.SID(),
				"error", err,
			)
			continue
		}

		notifiedCount++
	}

	s.logger.Debugw("subscription change notification sent",
		"subscription_sid", sub.SID(),
		"change_type", changeType,
		"total_nodes", len(nodes),
		"notified_nodes", notifiedCount,
	)

	return nil
}

// NotifySubscriptionActivation notifies nodes when a subscription becomes active.
// It also publishes an event for cross-instance synchronization via Redis Pub/Sub.
func (s *SubscriptionSyncService) NotifySubscriptionActivation(
	ctx context.Context,
	sub *subscription.Subscription,
) error {
	if sub == nil {
		return nil
	}

	// Publish activation event to Redis for cross-instance synchronization
	// This must happen first so other instances receive the event promptly
	if s.eventPublisher != nil {
		if err := s.eventPublisher.PublishActivation(ctx, sub.ID(), sub.SID()); err != nil {
			// Log but don't fail the operation - local notification should still happen
			s.logger.Warnw("failed to publish activation event",
				"subscription_id", sub.ID(),
				"subscription_sid", sub.SID(),
				"error", err,
			)
		}
	}

	// Notify local connected nodes
	return s.NotifySubscriptionChange(ctx, sub, dto.SubscriptionChangeAdded)
}

// NotifySubscriptionDeactivation notifies nodes when a subscription is deactivated/expired.
// It also publishes an event for cross-instance synchronization via Redis Pub/Sub.
func (s *SubscriptionSyncService) NotifySubscriptionDeactivation(
	ctx context.Context,
	sub *subscription.Subscription,
) error {
	if sub == nil {
		return nil
	}

	// Publish deactivation event to Redis for cross-instance synchronization
	// This must happen first so other instances receive the event promptly
	if s.eventPublisher != nil {
		if err := s.eventPublisher.PublishDeactivation(ctx, sub.ID(), sub.SID()); err != nil {
			// Log but don't fail the operation - local notification should still happen
			s.logger.Warnw("failed to publish deactivation event",
				"subscription_id", sub.ID(),
				"subscription_sid", sub.SID(),
				"error", err,
			)
		}
	}

	// Notify local connected nodes
	return s.NotifySubscriptionChange(ctx, sub, dto.SubscriptionChangeRemoved)
}

// NotifySubscriptionUpdate notifies nodes when a subscription is updated.
// It also publishes an event for cross-instance synchronization via Redis Pub/Sub.
func (s *SubscriptionSyncService) NotifySubscriptionUpdate(
	ctx context.Context,
	sub *subscription.Subscription,
) error {
	if sub == nil {
		return nil
	}

	// Publish update event to Redis for cross-instance synchronization
	if s.eventPublisher != nil {
		if err := s.eventPublisher.PublishUpdate(ctx, sub.ID(), sub.SID()); err != nil {
			s.logger.Warnw("failed to publish update event",
				"subscription_id", sub.ID(),
				"subscription_sid", sub.SID(),
				"error", err,
			)
		}
	}

	// Notify local connected nodes
	return s.NotifySubscriptionChange(ctx, sub, dto.SubscriptionChangeUpdated)
}

// buildSubscriptionInfo builds a NodeSubscriptionInfo from a subscription entity.
func (s *SubscriptionSyncService) buildSubscriptionInfo(
	sub *subscription.Subscription,
	hmacSecret string,
	encryptionMethod string,
) dto.NodeSubscriptionInfo {
	// Use the same password generation logic as GetNodeSubscriptions
	password := s.generatePassword(sub, hmacSecret, encryptionMethod)

	return dto.NodeSubscriptionInfo{
		SubscriptionSID: sub.SID(),
		Password:        password,
		Name:            s.generateSubscriptionName(sub),
		SpeedLimit:      0, // Can be set from plan limits
		DeviceLimit:     0, // Can be set from plan limits
		ExpireTime:      sub.EndDate().Unix(),
	}
}

// generatePassword generates password based on encryption method type.
func (s *SubscriptionSyncService) generatePassword(
	sub *subscription.Subscription,
	secret string,
	method string,
) string {
	if sub == nil || sub.UUID() == "" {
		return ""
	}

	// Delegate to the shared password generation logic
	subs := []*subscription.Subscription{sub}
	response := dto.ToNodeSubscriptionsResponse(subs, secret, method)
	if len(response.Subscriptions) > 0 {
		return response.Subscriptions[0].Password
	}
	return ""
}

// generateSubscriptionName generates name identifier for subscription.
func (s *SubscriptionSyncService) generateSubscriptionName(sub *subscription.Subscription) string {
	if sub == nil {
		return ""
	}
	response := dto.ToNodeSubscriptionsResponse([]*subscription.Subscription{sub}, "", "")
	if len(response.Subscriptions) > 0 {
		return response.Subscriptions[0].Name
	}
	// Fallback: generate name directly if subscription is not active
	return sub.SID()
}

// sendSubscriptionSync sends subscription sync message to a node.
func (s *SubscriptionSyncService) sendSubscriptionSync(
	n *node.Node,
	changeType string,
	subscriptions []dto.NodeSubscriptionInfo,
) error {
	syncData := &dto.SubscriptionSyncData{
		ChangeType:    changeType,
		Subscriptions: subscriptions,
		Timestamp:     biztime.NowUTC().Unix(),
	}

	msg := &dto.NodeHubMessage{
		Type:      dto.NodeMsgTypeSubscriptionSync,
		NodeID:    n.SID(),
		Timestamp: biztime.NowUTC().Unix(),
		Data:      syncData,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		s.logger.Errorw("failed to marshal subscription sync message",
			"node_id", n.ID(),
			"error", err,
		)
		return err
	}

	if err := s.hub.SendMessageToNode(n.ID(), msgBytes); err != nil {
		return err
	}

	s.logger.Debugw("subscription sync sent to node",
		"node_id", n.ID(),
		"node_sid", n.SID(),
		"change_type", changeType,
		"subscription_count", len(subscriptions),
	)

	return nil
}

// FullSyncSubscriptionsToNode sends all active subscriptions to a specific node.
// This is useful when a node comes online and needs to sync all subscriptions.
func (s *SubscriptionSyncService) FullSyncSubscriptionsToNode(
	ctx context.Context,
	nodeID uint,
	subscriptions []*subscription.Subscription,
) error {
	if !s.hub.IsNodeOnline(nodeID) {
		s.logger.Debugw("node offline, skipping subscription full sync",
			"node_id", nodeID,
		)
		return nil
	}

	n, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		s.logger.Errorw("failed to get node for subscription sync",
			"node_id", nodeID,
			"error", err,
		)
		return err
	}

	if n == nil {
		s.logger.Warnw("node not found for subscription sync",
			"node_id", nodeID,
		)
		return nil
	}

	// Determine encryption method
	encryptionMethod := ""
	if n.Protocol().IsShadowsocks() {
		encryptionMethod = n.EncryptionConfig().Method()
	}

	// Get HMAC secret
	hmacSecret := config.Get().Auth.JWT.Secret

	// Convert subscriptions to NodeSubscriptionInfo
	subscriptionInfos := make([]dto.NodeSubscriptionInfo, 0, len(subscriptions))
	for _, sub := range subscriptions {
		if sub == nil || !sub.IsActive() {
			continue
		}
		info := s.buildSubscriptionInfo(sub, hmacSecret, encryptionMethod)
		subscriptionInfos = append(subscriptionInfos, info)
	}

	// Add node-to-node forwarding user
	var nodeForwardingPassword string
	if n.Protocol().IsTrojan() {
		nodeForwardingPassword = vo.GenerateTrojanServerPassword(n.TokenHash())
	} else if n.Protocol().IsShadowsocks() {
		nodeForwardingPassword = vo.GenerateShadowsocksServerPassword(n.TokenHash(), encryptionMethod)
	} else if n.Protocol().IsAnyTLS() {
		nodeForwardingPassword = vo.GenerateAnyTLSServerPassword(n.TokenHash())
	}

	if nodeForwardingPassword != "" {
		forwardingUser := dto.NodeSubscriptionInfo{
			SubscriptionSID: "node-forwarding",
			Password:        nodeForwardingPassword,
			Name:            "node-forward-" + n.SID(),
			SpeedLimit:      0,
			DeviceLimit:     0,
			ExpireTime:      0,
		}
		subscriptionInfos = append(subscriptionInfos, forwardingUser)
	}

	// Send full sync
	if err := s.sendSubscriptionSync(n, dto.SubscriptionChangeAdded, subscriptionInfos); err != nil {
		return err
	}

	s.logger.Infow("full subscription sync sent to node",
		"node_id", nodeID,
		"node_sid", n.SID(),
		"subscription_count", len(subscriptionInfos),
	)

	return nil
}

// SyncSubscriptionsToNode syncs all active subscriptions to a specific node.
// This is an adapter that delegates to SyncSubscriptionsOnNodeConnect, used by
// resource group membership use cases to push subscription updates after group changes.
func (s *SubscriptionSyncService) SyncSubscriptionsToNode(ctx context.Context, nodeID uint) error {
	return s.SyncSubscriptionsOnNodeConnect(ctx, nodeID)
}

// SyncSubscriptionsOnNodeConnect syncs all active subscriptions to a node when it connects.
// This method queries subscriptions internally and pushes them to the node.
func (s *SubscriptionSyncService) SyncSubscriptionsOnNodeConnect(ctx context.Context, nodeID uint) error {
	if s.subscriptionRepo == nil {
		s.logger.Warnw("subscription repo not set, skipping sync on connect",
			"node_id", nodeID,
		)
		return nil
	}

	// Get all active subscriptions for this node
	subscriptions, err := s.subscriptionRepo.GetActiveSubscriptionsByNodeID(ctx, nodeID)
	if err != nil {
		s.logger.Errorw("failed to get subscriptions for node on connect",
			"node_id", nodeID,
			"error", err,
		)
		return err
	}

	return s.FullSyncSubscriptionsToNode(ctx, nodeID, subscriptions)
}
