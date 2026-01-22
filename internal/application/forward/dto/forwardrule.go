// Package dto provides data transfer objects for the forward domain.
package dto

import (
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/mapper"
)

// ForwardRuleDTO represents the data transfer object for forward rules.
// Note: ws_listen_port field has been removed (exit type deprecated).
// Database column is kept for backward compatibility but not exposed in API.
type ForwardRuleDTO struct {
	ID              string            `json:"id"`                          // Stripe-style prefixed ID (e.g., "fr_xK9mP2vL3nQ")
	AgentID         string            `json:"agent_id"`                    // Stripe-style prefixed ID (e.g., "fa_xK9mP2vL3nQ")
	UserID          *uint             `json:"user_id,omitempty"`           // user ID for user-owned rules (nil for admin-created rules)
	RuleType        string            `json:"rule_type"`                   // direct, entry, chain, direct_chain
	ExitAgentID     string            `json:"exit_agent_id,omitempty"`     // for entry type (Stripe-style prefixed ID)
	ChainAgentIDs   []string          `json:"chain_agent_ids,omitempty"`   // for chain and direct_chain types (ordered Stripe-style prefixed IDs)
	ChainPortConfig map[string]uint16 `json:"chain_port_config,omitempty"` // for direct_chain type (Stripe-style agent ID -> listen port)
	Name            string            `json:"name"`
	ListenPort      uint16            `json:"listen_port"`
	TargetAddress   string            `json:"target_address,omitempty"` // for all types (exit role only for chain/direct_chain)
	TargetPort      uint16            `json:"target_port,omitempty"`    // for all types (exit role only for chain/direct_chain)
	TargetNodeID    string            `json:"target_node_id,omitempty"` // Stripe-style prefixed Node ID (e.g., "node_xK9mP2vL3nQ")
	BindIP          string            `json:"bind_ip,omitempty"`        // Bind IP address for outbound connections
	IPVersion       string            `json:"ip_version"`               // auto, ipv4, ipv6
	Protocol        string            `json:"protocol"`
	Status          string            `json:"status"`
	Remark          string            `json:"remark"`
	UploadBytes     int64             `json:"upload_bytes"`   // traffic with multiplier already applied
	DownloadBytes   int64             `json:"download_bytes"` // traffic with multiplier already applied
	TotalBytes      int64             `json:"total_bytes"`    // traffic with multiplier already applied
	CreatedAt       string            `json:"created_at"`
	UpdatedAt       string            `json:"updated_at"`

	// Traffic multiplier fields
	TrafficMultiplier          *float64 `json:"traffic_multiplier,omitempty"`
	EffectiveTrafficMultiplier float64  `json:"effective_traffic_multiplier"`
	NodeCount                  int      `json:"node_count"`
	IsAutoMultiplier           bool     `json:"is_auto_multiplier"`

	// Sort order for custom ordering
	SortOrder int `json:"sort_order"`

	// Target node info (populated when targetNodeID is set)
	TargetNodeServerAddress string  `json:"target_node_server_address,omitempty"` // node's configured server address
	TargetNodePublicIPv4    *string `json:"target_node_public_ipv4,omitempty"`    // node's reported public IPv4
	TargetNodePublicIPv6    *string `json:"target_node_public_ipv6,omitempty"`    // node's reported public IPv6

	// Role indicates the requesting agent's role in this rule
	// Values: "entry" (needs to establish tunnel), "exit" (accepts tunnel connections), "relay" (chain middle node)
	Role string `json:"role,omitempty"`

	// Tunnel configuration
	TunnelType string `json:"tunnel_type,omitempty"` // tunnel type: "ws" or "tls" (default: "ws")
	TunnelHops *int   `json:"tunnel_hops,omitempty"` // number of hops using tunnel (nil=full tunnel, N=first N hops use tunnel)

	// Hop mode for hybrid chain (populated based on agent's position)
	HopMode      string `json:"hop_mode,omitempty"`      // "tunnel", "direct", or "boundary"
	InboundMode  string `json:"inbound_mode,omitempty"`  // for boundary nodes: "tunnel" or "direct"
	OutboundMode string `json:"outbound_mode,omitempty"` // for boundary nodes: "tunnel" or "direct"

	// Runtime sync status (populated from agent status cache)
	SyncStatus      string `json:"sync_status,omitempty"`       // aggregated: synced, pending, failed
	RunStatus       string `json:"run_status,omitempty"`        // aggregated: running, stopped, error, starting, unknown
	TotalAgents     int    `json:"total_agents,omitempty"`      // total agents in forwarding chain
	HealthyAgents   int    `json:"healthy_agents,omitempty"`    // number of healthy agents
	StatusUpdatedAt int64  `json:"status_updated_at,omitempty"` // last status update timestamp (Unix seconds)

	// Chain-specific fields (populated for chain rules based on requesting agent's role)
	ChainPosition          int    `json:"chain_position,omitempty"`            // agent's position in chain (0-indexed)
	IsLastInChain          bool   `json:"is_last_in_chain,omitempty"`          // true if agent is last in chain
	NextHopAgentID         string `json:"next_hop_agent_id,omitempty"`         // next agent in chain (Stripe-style ID)
	NextHopAddress         string `json:"next_hop_address,omitempty"`          // next agent's public address
	NextHopWsPort          uint16 `json:"next_hop_ws_port,omitempty"`          // next agent's WS port (from status cache)
	NextHopTlsPort         uint16 `json:"next_hop_tls_port,omitempty"`         // next agent's TLS port (from status cache)
	NextHopPort            uint16 `json:"next_hop_port,omitempty"`             // next agent's listen port (for direct_chain type)
	NextHopConnectionToken string `json:"next_hop_connection_token,omitempty"` // short-term JWT for next hop authentication

	// Resource group IDs (admin only)
	GroupSIDs []string `json:"group_sids,omitempty"` // resource group SIDs

	// External rule fields (for rule_type = "external")
	ServerAddress  string `json:"server_address,omitempty"`   // server address for external rules
	ExternalSource string `json:"external_source,omitempty"`  // external source identifier
	ExternalRuleID string `json:"external_rule_id,omitempty"` // external rule reference ID

	// Internal fields for mapping (not exposed in JSON)
	internalAgentID         uint            `json:"-"`
	internalExitAgentID     uint            `json:"-"`
	internalChainAgents     []uint          `json:"-"` // internal chain agent IDs for lookup
	internalChainPortConfig map[uint]uint16 `json:"-"` // internal chain port config for lookup
	internalTargetNode      *uint           `json:"-"` // internal node ID for lookup
	internalGroupIDs        []uint          `json:"-"` // internal resource group IDs for lookup
}

// ToForwardRuleDTO converts a domain forward rule to DTO.
// Note: TargetNode* fields are NOT populated by this function.
// Use PopulateTargetNodeInfo to fill them after getting node data.
// Note: AgentID and ExitAgentID will be empty strings. Use PopulateAgentInfo to fill them.
// Note: TargetNodeID requires PopulateTargetNodeSID to be called for Stripe-style ID.
func ToForwardRuleDTO(rule *forward.ForwardRule) *ForwardRuleDTO {
	if rule == nil {
		return nil
	}

	effectiveMultiplier := rule.GetEffectiveMultiplier()
	nodeCount := rule.CalculateNodeCount()
	isAuto := rule.GetTrafficMultiplier() == nil

	// direct, direct_chain, and external types do not use tunnel, so tunnel_type should be empty
	tunnelType := ""
	if !rule.RuleType().IsDirect() && !rule.RuleType().IsDirectChain() && !rule.RuleType().IsExternal() {
		tunnelType = rule.TunnelType().String()
	}

	return &ForwardRuleDTO{
		ID:                         rule.SID(),
		AgentID:                    "", // populated later via PopulateAgentInfo
		UserID:                     rule.UserID(),
		RuleType:                   rule.RuleType().String(),
		ExitAgentID:                "",  // populated later via PopulateAgentInfo
		ChainAgentIDs:              nil, // populated later via PopulateAgentInfo
		Name:                       rule.Name(),
		ListenPort:                 rule.ListenPort(),
		TargetAddress:              rule.TargetAddress(),
		TargetPort:                 rule.TargetPort(),
		TargetNodeID:               "", // populated later via PopulateTargetNodeSID
		BindIP:                     rule.BindIP(),
		IPVersion:                  rule.IPVersion().String(),
		Protocol:                   rule.Protocol().String(),
		Status:                     rule.Status().String(),
		Remark:                     rule.Remark(),
		UploadBytes:                rule.UploadBytes(),   // traffic with multiplier already applied
		DownloadBytes:              rule.DownloadBytes(), // traffic with multiplier already applied
		TotalBytes:                 rule.TotalBytes(),    // traffic with multiplier already applied
		CreatedAt:                  rule.CreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:                  rule.UpdatedAt().Format("2006-01-02T15:04:05Z07:00"),
		TrafficMultiplier:          rule.GetTrafficMultiplier(),
		EffectiveTrafficMultiplier: effectiveMultiplier,
		NodeCount:                  nodeCount,
		IsAutoMultiplier:           isAuto,
		SortOrder:                  rule.SortOrder(),
		TunnelType:                 tunnelType,
		TunnelHops:                 rule.TunnelHops(),
		ServerAddress:              rule.ServerAddress(),
		ExternalSource:             rule.ExternalSource(),
		ExternalRuleID:             rule.ExternalRuleID(),
		internalAgentID:            rule.AgentID(),
		internalExitAgentID:        rule.ExitAgentID(),
		internalChainAgents:        rule.ChainAgentIDs(),
		internalChainPortConfig:    rule.ChainPortConfig(),
		internalTargetNode:         rule.TargetNodeID(),
		internalGroupIDs:           rule.GroupIDs(),
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

// AgentSIDMap maps internal agent ID to SID.
type AgentSIDMap map[uint]string

// PopulateAgentInfo fills in the agent ID fields using the SID map.
func (d *ForwardRuleDTO) PopulateAgentInfo(agentMap AgentSIDMap) {
	if sid, ok := agentMap[d.internalAgentID]; ok {
		d.AgentID = sid
	}
	if d.internalExitAgentID != 0 {
		if sid, ok := agentMap[d.internalExitAgentID]; ok {
			d.ExitAgentID = sid
		}
	}
	// Populate chain agent IDs
	// For chain and direct_chain types, include entry agent (d.internalAgentID) as first element
	if len(d.internalChainAgents) > 0 && (d.RuleType == "chain" || d.RuleType == "direct_chain") {
		// Full chain: [entry_agent] + chain_agents
		fullChain := append([]uint{d.internalAgentID}, d.internalChainAgents...)
		d.ChainAgentIDs = make([]string, len(fullChain))
		for i, agentID := range fullChain {
			if sid, ok := agentMap[agentID]; ok {
				d.ChainAgentIDs[i] = sid
			}
		}
	} else if len(d.internalChainAgents) > 0 {
		// Fallback for other types (shouldn't happen)
		d.ChainAgentIDs = make([]string, len(d.internalChainAgents))
		for i, agentID := range d.internalChainAgents {
			if sid, ok := agentMap[agentID]; ok {
				d.ChainAgentIDs[i] = sid
			}
		}
	}
	// Populate chain port config (for direct_chain type)
	if len(d.internalChainPortConfig) > 0 {
		d.ChainPortConfig = make(map[string]uint16, len(d.internalChainPortConfig))
		for agentID, port := range d.internalChainPortConfig {
			if sid, ok := agentMap[agentID]; ok {
				d.ChainPortConfig[sid] = port
			}
		}
	}
}

// InternalChainAgentIDs returns the internal chain agent IDs for repository lookups.
func (d *ForwardRuleDTO) InternalChainAgentIDs() []uint {
	return d.internalChainAgents
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

// NodeSIDMap maps internal node ID to SID.
type NodeSIDMap map[uint]string

// PopulateTargetNodeSID fills in the target node ID field using the SID map.
func (d *ForwardRuleDTO) PopulateTargetNodeSID(nodeMap NodeSIDMap) {
	if d.internalTargetNode == nil || *d.internalTargetNode == 0 {
		return
	}
	if sid, ok := nodeMap[*d.internalTargetNode]; ok {
		d.TargetNodeID = sid
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
	return mapper.MapSlice(rules, ToForwardRuleDTO)
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
		// Collect chain agent IDs
		for _, chainAgentID := range dto.internalChainAgents {
			if chainAgentID != 0 {
				idSet[chainAgentID] = struct{}{}
			}
		}
	}

	ids := make([]uint, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	return ids
}

// RuleSyncStatusInfo contains aggregated sync status for a rule.
type RuleSyncStatusInfo struct {
	SyncStatus    string // aggregated: synced, pending, failed
	RunStatus     string // aggregated: running, stopped, error, starting, unknown
	TotalAgents   int    // total agents in chain
	HealthyAgents int    // number of healthy agents
	UpdatedAt     int64  // last update timestamp (Unix seconds)
}

// PopulateSyncStatus fills in the runtime sync status fields.
func (d *ForwardRuleDTO) PopulateSyncStatus(info *RuleSyncStatusInfo) {
	if info == nil {
		return
	}
	d.SyncStatus = info.SyncStatus
	d.RunStatus = info.RunStatus
	d.TotalAgents = info.TotalAgents
	d.HealthyAgents = info.HealthyAgents
	d.StatusUpdatedAt = info.UpdatedAt
}

// CollectAllAgentIDsForRules collects all agent IDs involved in forwarding for each rule.
// Returns a map from rule SID to list of agent IDs (including entry, exit, and chain agents).
func CollectAllAgentIDsForRules(dtos []*ForwardRuleDTO) map[string][]uint {
	result := make(map[string][]uint, len(dtos))
	for _, dto := range dtos {
		var agentIDs []uint
		if dto.internalAgentID != 0 {
			agentIDs = append(agentIDs, dto.internalAgentID)
		}
		switch dto.RuleType {
		case "entry":
			if dto.internalExitAgentID != 0 {
				agentIDs = append(agentIDs, dto.internalExitAgentID)
			}
		case "chain", "direct_chain":
			agentIDs = append(agentIDs, dto.internalChainAgents...)
		}
		result[dto.ID] = agentIDs
	}
	return result
}

// GroupSIDMap maps internal resource group ID to SID.
type GroupSIDMap map[uint]string

// PopulateGroupSIDs fills in the group SIDs field using the SID map.
func (d *ForwardRuleDTO) PopulateGroupSIDs(groupMap GroupSIDMap) {
	if len(d.internalGroupIDs) == 0 {
		return
	}
	d.GroupSIDs = make([]string, 0, len(d.internalGroupIDs))
	for _, groupID := range d.internalGroupIDs {
		if sid, ok := groupMap[groupID]; ok && sid != "" {
			d.GroupSIDs = append(d.GroupSIDs, sid)
		}
	}
}

// InternalGroupIDs returns the internal resource group IDs for repository lookups.
func (d *ForwardRuleDTO) InternalGroupIDs() []uint {
	return d.internalGroupIDs
}

// CollectGroupIDs collects unique resource group IDs from DTOs for batch lookup.
func CollectGroupIDs(dtos []*ForwardRuleDTO) []uint {
	idSet := make(map[uint]struct{})
	for _, dto := range dtos {
		for _, groupID := range dto.internalGroupIDs {
			if groupID != 0 {
				idSet[groupID] = struct{}{}
			}
		}
	}

	ids := make([]uint, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	return ids
}
