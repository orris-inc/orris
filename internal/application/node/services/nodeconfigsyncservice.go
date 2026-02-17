// Package services provides application-level services for the node domain.
package services

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"

	"github.com/orris-inc/orris/internal/application/node/dto"
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

// NodeConfigSyncService handles configuration synchronization for node agents.
// It pushes config changes to node agents when they come online or when their config changes.
type NodeConfigSyncService struct {
	nodeRepo node.NodeRepository
	hub      NodeSyncHub
	logger   logger.Interface

	// Version management
	globalVersion uint64
	nodeVersions  sync.Map // map[uint]uint64 - node ID to acknowledged version
}

// NewNodeConfigSyncService creates a new NodeConfigSyncService.
func NewNodeConfigSyncService(
	nodeRepo node.NodeRepository,
	hub NodeSyncHub,
	log logger.Interface,
) *NodeConfigSyncService {
	return &NodeConfigSyncService{
		nodeRepo: nodeRepo,
		hub:      hub,
		logger:   log,
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

	// Collect all referenced node SIDs from route and DNS configs
	var referencedNodes []*node.Node
	var allReferencedSIDs []string
	if n.RouteConfig() != nil && n.RouteConfig().HasNodeReferences() {
		allReferencedSIDs = append(allReferencedSIDs, n.RouteConfig().GetReferencedNodeSIDs()...)
	}
	if n.DnsConfig() != nil && n.DnsConfig().HasNodeReferences() {
		allReferencedSIDs = append(allReferencedSIDs, n.DnsConfig().GetReferencedNodeSIDs()...)
	}
	if len(allReferencedSIDs) > 0 {
		// Deduplicate SIDs
		allReferencedSIDs = uniqueStrings(allReferencedSIDs)
		referencedNodes, err = s.nodeRepo.GetBySIDs(ctx, allReferencedSIDs)
		if err != nil {
			s.logger.Warnw("failed to fetch referenced nodes for config sync",
				"node_id", nodeID,
				"referenced_sids", allReferencedSIDs,
				"error", err,
			)
			// Continue without referenced nodes rather than failing
		}
	}

	// Server key function for referenced nodes
	serverKeyFunc := func(refNode *node.Node) string {
		if refNode.Protocol().IsShadowsocks() {
			return vo.GenerateShadowsocksServerPassword(refNode.TokenHash(), refNode.EncryptionConfig().Method())
		}
		// For Trojan, generate password from token hash for node-to-node forwarding
		if refNode.Protocol().IsTrojan() {
			return vo.GenerateTrojanServerPassword(refNode.TokenHash())
		}
		// For AnyTLS, generate password from token hash for node-to-node forwarding
		if refNode.Protocol().IsAnyTLS() {
			return vo.GenerateAnyTLSServerPassword(refNode.TokenHash())
		}
		return ""
	}

	// Convert to NodeConfigData
	configData := dto.ToNodeConfigData(n, referencedNodes, serverKeyFunc)

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

	// Collect all referenced node SIDs from route and DNS configs
	var referencedNodes []*node.Node
	var allReferencedSIDs []string
	if n.RouteConfig() != nil && n.RouteConfig().HasNodeReferences() {
		allReferencedSIDs = append(allReferencedSIDs, n.RouteConfig().GetReferencedNodeSIDs()...)
	}
	if n.DnsConfig() != nil && n.DnsConfig().HasNodeReferences() {
		allReferencedSIDs = append(allReferencedSIDs, n.DnsConfig().GetReferencedNodeSIDs()...)
	}
	if len(allReferencedSIDs) > 0 {
		// Deduplicate SIDs
		allReferencedSIDs = uniqueStrings(allReferencedSIDs)
		referencedNodes, err = s.nodeRepo.GetBySIDs(ctx, allReferencedSIDs)
		if err != nil {
			s.logger.Warnw("failed to fetch referenced nodes for config change",
				"node_id", nodeID,
				"referenced_sids", allReferencedSIDs,
				"error", err,
			)
		}
	}

	// Server key function for referenced nodes
	serverKeyFunc := func(refNode *node.Node) string {
		if refNode.Protocol().IsShadowsocks() {
			return vo.GenerateShadowsocksServerPassword(refNode.TokenHash(), refNode.EncryptionConfig().Method())
		}
		// For Trojan, generate password from token hash for node-to-node forwarding
		if refNode.Protocol().IsTrojan() {
			return vo.GenerateTrojanServerPassword(refNode.TokenHash())
		}
		// For AnyTLS, generate password from token hash for node-to-node forwarding
		if refNode.Protocol().IsAnyTLS() {
			return vo.GenerateAnyTLSServerPassword(refNode.TokenHash())
		}
		return ""
	}

	// Convert to NodeConfigData
	configData := dto.ToNodeConfigData(n, referencedNodes, serverKeyFunc)

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
	)

	return nil
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
