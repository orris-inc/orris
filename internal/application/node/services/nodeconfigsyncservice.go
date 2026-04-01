// Package services provides application-level services for the node domain.
package services

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"

	"github.com/orris-inc/orris/internal/application/node/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/node"
	vo "github.com/orris-inc/orris/internal/domain/node/valueobjects"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// NodeSyncHub defines the interface for sending messages to nodes through the hub.
type NodeSyncHub interface {
	IsNodeOnline(nodeID uint) bool
	SendMessageToNode(nodeID uint, msg []byte) error
}

// ForwardRuleQuerier defines a narrow interface for querying forward rules
// that target a specific node. Used to merge per-forward-rule routing into node config.
type ForwardRuleQuerier interface {
	ListEnabledByTargetNodeID(ctx context.Context, nodeID uint) ([]*forward.ForwardRule, error)
}

// NodeConfigSyncService handles configuration synchronization for node agents.
// It pushes config changes to node agents when they come online or when their config changes.
type NodeConfigSyncService struct {
	nodeRepo        node.NodeRepository
	forwardRuleRepo ForwardRuleQuerier
	hub             NodeSyncHub
	logger          logger.Interface

	// Version management
	globalVersion uint64
	nodeVersions  sync.Map // map[uint]uint64 - node ID to acknowledged version
}

// NewNodeConfigSyncService creates a new NodeConfigSyncService.
func NewNodeConfigSyncService(
	nodeRepo node.NodeRepository,
	forwardRuleRepo ForwardRuleQuerier,
	hub NodeSyncHub,
	log logger.Interface,
) *NodeConfigSyncService {
	return &NodeConfigSyncService{
		nodeRepo:        nodeRepo,
		forwardRuleRepo: forwardRuleRepo,
		hub:             hub,
		logger:          log,
	}
}

// IncrementVersion atomically increments and returns the new global version.
func (s *NodeConfigSyncService) IncrementVersion() uint64 {
	return atomic.AddUint64(&s.globalVersion, 1)
}

// GetGlobalVersion returns the current global version.
func (s *NodeConfigSyncService) GetGlobalVersion() uint64 {
	return atomic.LoadUint64(&s.globalVersion)
}

// UpdateNodeVersion updates the acknowledged version for a node.
func (s *NodeConfigSyncService) UpdateNodeVersion(nodeID uint, version uint64) {
	s.nodeVersions.Store(nodeID, version)
}

// GetNodeVersion returns the acknowledged version for a node.
func (s *NodeConfigSyncService) GetNodeVersion(nodeID uint) uint64 {
	if v, ok := s.nodeVersions.Load(nodeID); ok {
		return v.(uint64)
	}
	return 0
}

// FullSyncToNode performs a full configuration sync to a node (typically on connection).
func (s *NodeConfigSyncService) FullSyncToNode(ctx context.Context, nodeID uint) error {
	s.logger.Infow("performing full config sync to node",
		"node_id", nodeID,
	)

	// Check if node is online
	if !s.hub.IsNodeOnline(nodeID) {
		s.logger.Debugw("node offline, skipping full sync",
			"node_id", nodeID,
		)
		return nil
	}

	// Get node from repository
	n, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		s.logger.Errorw("failed to get node for config sync",
			"node_id", nodeID,
			"error", err,
		)
		return err
	}
	if n == nil {
		s.logger.Warnw("node not found for config sync",
			"node_id", nodeID,
		)
		return errors.NewNotFoundError("node not found")
	}

	// Query forward rules targeting this node (for per-rule routing)
	forwardRules := s.queryForwardRulesWithRouteConfig(ctx, nodeID)

	// Collect all referenced node SIDs from route, DNS, and forward rule route configs
	referencedNodes, serverKeyFunc := s.resolveReferencedNodes(ctx, n, forwardRules)

	// Convert to NodeConfigData
	configData := dto.ToNodeConfigData(n, referencedNodes, serverKeyFunc, forwardRules)

	// Build full sync data
	version := s.IncrementVersion()
	syncData := &dto.NodeConfigSyncData{
		Version:   version,
		FullSync:  true,
		Config:    configData,
		Timestamp: biztime.NowUTC().Unix(),
	}

	// Build hub message
	msg := &dto.NodeHubMessage{
		Type:      dto.NodeMsgTypeConfigSync,
		NodeID:    n.SID(),
		Timestamp: biztime.NowUTC().Unix(),
		Data:      syncData,
	}

	// Serialize message
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		s.logger.Errorw("failed to marshal config sync message",
			"node_id", nodeID,
			"error", err,
		)
		return err
	}

	// Send to node
	if err := s.hub.SendMessageToNode(nodeID, msgBytes); err != nil {
		s.logger.Warnw("failed to send config sync to node",
			"node_id", nodeID,
			"error", err,
		)
		return err
	}

	s.logger.Infow("full config sync sent to node",
		"node_id", nodeID,
		"node_sid", n.SID(),
		"version", version,
		"has_route", configData.Route != nil,
		"has_dns", configData.DNS != nil,
		"forward_rule_routes", len(configData.ForwardRuleRoutes),
	)

	return nil
}

// NotifyConfigChange notifies a node about a configuration change.
// This is called when node config is updated (including route config changes).
func (s *NodeConfigSyncService) NotifyConfigChange(ctx context.Context, nodeID uint) error {
	s.logger.Infow("notifying node of config change",
		"node_id", nodeID,
	)

	// Check if node is online
	if !s.hub.IsNodeOnline(nodeID) {
		s.logger.Debugw("node offline, skipping config change notification",
			"node_id", nodeID,
		)
		return nil
	}

	// Get updated node from repository
	n, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		s.logger.Errorw("failed to get node for config change notification",
			"node_id", nodeID,
			"error", err,
		)
		return err
	}
	if n == nil {
		s.logger.Warnw("node not found for config change notification",
			"node_id", nodeID,
		)
		return errors.NewNotFoundError("node not found")
	}

	// Query forward rules targeting this node (for per-rule routing)
	forwardRules := s.queryForwardRulesWithRouteConfig(ctx, nodeID)

	// Collect all referenced node SIDs from route, DNS, and forward rule route configs
	referencedNodes, serverKeyFunc := s.resolveReferencedNodes(ctx, n, forwardRules)

	// Convert to NodeConfigData
	configData := dto.ToNodeConfigData(n, referencedNodes, serverKeyFunc, forwardRules)

	// Build sync data (incremental update)
	version := s.IncrementVersion()
	syncData := &dto.NodeConfigSyncData{
		Version:   version,
		FullSync:  false, // Incremental update
		Config:    configData,
		Timestamp: biztime.NowUTC().Unix(),
	}

	// Build hub message
	msg := &dto.NodeHubMessage{
		Type:      dto.NodeMsgTypeConfigSync,
		NodeID:    n.SID(),
		Timestamp: biztime.NowUTC().Unix(),
		Data:      syncData,
	}

	// Serialize message
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		s.logger.Errorw("failed to marshal config change message",
			"node_id", nodeID,
			"error", err,
		)
		return err
	}

	// Send to node
	if err := s.hub.SendMessageToNode(nodeID, msgBytes); err != nil {
		s.logger.Warnw("failed to send config change to node",
			"node_id", nodeID,
			"error", err,
		)
		return err
	}

	s.logger.Infow("config change notification sent to node",
		"node_id", nodeID,
		"node_sid", n.SID(),
		"version", version,
		"has_route", configData.Route != nil,
		"has_dns", configData.DNS != nil,
		"forward_rule_routes", len(configData.ForwardRuleRoutes),
	)

	return nil
}

// queryForwardRulesWithRouteConfig queries enabled forward rules targeting the node
// and filters to only those with RouteConfig configured.
func (s *NodeConfigSyncService) queryForwardRulesWithRouteConfig(ctx context.Context, nodeID uint) []*forward.ForwardRule {
	if s.forwardRuleRepo == nil {
		return nil
	}

	frs, err := s.forwardRuleRepo.ListEnabledByTargetNodeID(ctx, nodeID)
	if err != nil {
		s.logger.Warnw("failed to fetch forward rules for node config sync",
			"node_id", nodeID,
			"error", err,
		)
		return nil
	}

	var result []*forward.ForwardRule
	for _, fr := range frs {
		if fr.RouteConfig() != nil {
			result = append(result, fr)
		}
	}
	return result
}

// resolveReferencedNodes collects all referenced node SIDs from route, DNS, and
// per-forward-rule route configs, then fetches the corresponding nodes.
func (s *NodeConfigSyncService) resolveReferencedNodes(
	ctx context.Context,
	n *node.Node,
	forwardRules []*forward.ForwardRule,
) ([]*node.Node, func(*node.Node) string) {
	var allReferencedSIDs []string
	if n.RouteConfig() != nil && n.RouteConfig().HasNodeReferences() {
		allReferencedSIDs = append(allReferencedSIDs, n.RouteConfig().GetReferencedNodeSIDs()...)
	}
	if n.DnsConfig() != nil && n.DnsConfig().HasNodeReferences() {
		allReferencedSIDs = append(allReferencedSIDs, n.DnsConfig().GetReferencedNodeSIDs()...)
	}
	for _, fr := range forwardRules {
		if fr.RouteConfig().HasNodeReferences() {
			allReferencedSIDs = append(allReferencedSIDs, fr.RouteConfig().GetReferencedNodeSIDs()...)
		}
	}

	var referencedNodes []*node.Node
	if len(allReferencedSIDs) > 0 {
		allReferencedSIDs = uniqueStrings(allReferencedSIDs)
		var err error
		referencedNodes, err = s.nodeRepo.GetBySIDs(ctx, allReferencedSIDs)
		if err != nil {
			s.logger.Warnw("failed to fetch referenced nodes for config sync",
				"node_id", n.ID(),
				"referenced_sids", allReferencedSIDs,
				"error", err,
			)
		}
	}

	serverKeyFunc := func(refNode *node.Node) string {
		if refNode.Protocol().IsShadowsocks() {
			return vo.GenerateShadowsocksServerPassword(refNode.TokenHash(), refNode.EncryptionConfig().Method())
		}
		if refNode.Protocol().IsTrojan() {
			return vo.GenerateTrojanServerPassword(refNode.TokenHash())
		}
		if refNode.Protocol().IsAnyTLS() {
			return vo.GenerateAnyTLSServerPassword(refNode.TokenHash())
		}
		return ""
	}

	return referencedNodes, serverKeyFunc
}

// uniqueStrings returns a deduplicated copy of the input slice, preserving order.
func uniqueStrings(input []string) []string {
	seen := make(map[string]bool, len(input))
	result := make([]string, 0, len(input))
	for _, s := range input {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
