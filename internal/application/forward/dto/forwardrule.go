// Package dto provides data transfer objects for the forward domain.
package dto

import (
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/id"
)

// ForwardRuleDTO represents the data transfer object for forward rules.
type ForwardRuleDTO struct {
	ID            string `json:"id"`                       // Stripe-style prefixed ID (e.g., "fr_xK9mP2vL3nQ")
	AgentID       string `json:"agent_id"`                 // Stripe-style prefixed ID (e.g., "fa_xK9mP2vL3nQ")
	RuleType      string `json:"rule_type"`                // direct, entry, exit
	ExitAgentID   string `json:"exit_agent_id,omitempty"`  // for entry type (Stripe-style prefixed ID)
	WsListenPort  uint16 `json:"ws_listen_port,omitempty"` // for exit type
	Name          string `json:"name"`
	ListenPort    uint16 `json:"listen_port"`
	TargetAddress string `json:"target_address,omitempty"` // for direct and exit types
	TargetPort    uint16 `json:"target_port,omitempty"`    // for direct and exit types
	TargetNodeID  string `json:"target_node_id,omitempty"` // Stripe-style prefixed Node ID (e.g., "node_xK9mP2vL3nQ")
	IPVersion     string `json:"ip_version"`               // auto, ipv4, ipv6
	Protocol      string `json:"protocol"`
	Status        string `json:"status"`
	Remark        string `json:"remark"`
	UploadBytes   int64  `json:"upload_bytes"`
	DownloadBytes int64  `json:"download_bytes"`
	TotalBytes    int64  `json:"total_bytes"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`

	// Target node info (populated when targetNodeID is set)
	TargetNodeServerAddress string  `json:"target_node_server_address,omitempty"` // node's configured server address
	TargetNodePublicIPv4    *string `json:"target_node_public_ipv4,omitempty"`    // node's reported public IPv4
	TargetNodePublicIPv6    *string `json:"target_node_public_ipv6,omitempty"`    // node's reported public IPv6

	// Internal fields for mapping (not exposed in JSON)
	internalAgentID     uint   `json:"-"`
	internalExitAgentID uint   `json:"-"`
	internalTargetNode  *uint  `json:"-"` // internal node ID for lookup
	agentShortID        string `json:"-"`
	exitAgentShortID    string `json:"-"`
}

// ToForwardRuleDTO converts a domain forward rule to DTO.
// Note: TargetNode* fields are NOT populated by this function.
// Use PopulateTargetNodeInfo to fill them after getting node data.
// Note: AgentID and ExitAgentID will be empty strings. Use PopulateAgentInfo to fill them.
// Note: TargetNodeID requires PopulateTargetNodeShortID to be called for Stripe-style ID.
func ToForwardRuleDTO(rule *forward.ForwardRule) *ForwardRuleDTO {
	if rule == nil {
		return nil
	}

	return &ForwardRuleDTO{
		ID:                  id.FormatForwardRuleID(rule.ShortID()),
		AgentID:             "", // populated later via PopulateAgentInfo
		RuleType:            rule.RuleType().String(),
		ExitAgentID:         "", // populated later via PopulateAgentInfo
		WsListenPort:        rule.WsListenPort(),
		Name:                rule.Name(),
		ListenPort:          rule.ListenPort(),
		TargetAddress:       rule.TargetAddress(),
		TargetPort:          rule.TargetPort(),
		TargetNodeID:        "", // populated later via PopulateTargetNodeShortID
		IPVersion:           rule.IPVersion().String(),
		Protocol:            rule.Protocol().String(),
		Status:              rule.Status().String(),
		Remark:              rule.Remark(),
		UploadBytes:         rule.UploadBytes(),
		DownloadBytes:       rule.DownloadBytes(),
		TotalBytes:          rule.TotalBytes(),
		CreatedAt:           rule.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:           rule.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
		internalAgentID:     rule.AgentID(),
		internalExitAgentID: rule.ExitAgentID(),
		internalTargetNode:  rule.TargetNodeID(),
	}
}

// TargetNodeInfo contains node information for target address resolution.
type TargetNodeInfo struct {
	ServerAddress string
	PublicIPv4    *string
	PublicIPv6    *string
}

// PopulateTargetNodeInfo fills in the target node info fields.
func (d *ForwardRuleDTO) PopulateTargetNodeInfo(info *TargetNodeInfo) {
	if info == nil {
		return
	}
	d.TargetNodeServerAddress = info.ServerAddress
	d.TargetNodePublicIPv4 = info.PublicIPv4
	d.TargetNodePublicIPv6 = info.PublicIPv6
}

// AgentShortIDMap maps internal agent ID to short ID.
type AgentShortIDMap map[uint]string

// PopulateAgentInfo fills in the agent ID fields using the short ID map.
func (d *ForwardRuleDTO) PopulateAgentInfo(agentMap AgentShortIDMap) {
	if shortID, ok := agentMap[d.internalAgentID]; ok {
		d.AgentID = id.FormatForwardAgentID(shortID)
	}
	if d.internalExitAgentID != 0 {
		if shortID, ok := agentMap[d.internalExitAgentID]; ok {
			d.ExitAgentID = id.FormatForwardAgentID(shortID)
		}
	}
}

// InternalAgentID returns the internal agent ID for repository lookups.
func (d *ForwardRuleDTO) InternalAgentID() uint {
	return d.internalAgentID
}

// InternalExitAgentID returns the internal exit agent ID for repository lookups.
func (d *ForwardRuleDTO) InternalExitAgentID() uint {
	return d.internalExitAgentID
}

// InternalTargetNodeID returns the internal target node ID for repository lookups.
func (d *ForwardRuleDTO) InternalTargetNodeID() *uint {
	return d.internalTargetNode
}

// NodeShortIDMap maps internal node ID to short ID.
type NodeShortIDMap map[uint]string

// PopulateTargetNodeShortID fills in the target node ID field using the short ID map.
func (d *ForwardRuleDTO) PopulateTargetNodeShortID(nodeMap NodeShortIDMap) {
	if d.internalTargetNode == nil || *d.internalTargetNode == 0 {
		return
	}
	if shortID, ok := nodeMap[*d.internalTargetNode]; ok {
		d.TargetNodeID = id.FormatNodeID(shortID)
	}
}

// CollectTargetNodeIDs collects unique target node IDs from DTOs for batch lookup.
func CollectTargetNodeIDs(dtos []*ForwardRuleDTO) []uint {
	idSet := make(map[uint]struct{})
	for _, dto := range dtos {
		if dto.internalTargetNode != nil && *dto.internalTargetNode != 0 {
			idSet[*dto.internalTargetNode] = struct{}{}
		}
	}

	ids := make([]uint, 0, len(idSet))
	for nodeID := range idSet {
		ids = append(ids, nodeID)
	}
	return ids
}

// ToForwardRuleDTOs converts a slice of domain forward rules to DTOs.
func ToForwardRuleDTOs(rules []*forward.ForwardRule) []*ForwardRuleDTO {
	dtos := make([]*ForwardRuleDTO, len(rules))
	for i, rule := range rules {
		dtos[i] = ToForwardRuleDTO(rule)
	}
	return dtos
}

// CollectAgentIDs collects unique agent IDs from DTOs for batch lookup.
func CollectAgentIDs(dtos []*ForwardRuleDTO) []uint {
	idSet := make(map[uint]struct{})
	for _, dto := range dtos {
		if dto.internalAgentID != 0 {
			idSet[dto.internalAgentID] = struct{}{}
		}
		if dto.internalExitAgentID != 0 {
			idSet[dto.internalExitAgentID] = struct{}{}
		}
	}

	ids := make([]uint, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	return ids
}
