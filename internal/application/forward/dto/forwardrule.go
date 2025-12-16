// Package dto provides data transfer objects for the forward domain.
package dto

import (
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/id"
)

// ForwardRuleDTO represents the data transfer object for forward rules.
// Note: ws_listen_port field has been removed (exit type deprecated).
// Database column is kept for backward compatibility but not exposed in API.
type ForwardRuleDTO struct {
	ID              string            `json:"id"`                          // Stripe-style prefixed ID (e.g., "fr_xK9mP2vL3nQ")
	AgentID         string            `json:"agent_id"`                    // Stripe-style prefixed ID (e.g., "fa_xK9mP2vL3nQ")
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

	// Target node info (populated when targetNodeID is set)
	TargetNodeServerAddress string  `json:"target_node_server_address,omitempty"` // node's configured server address
	TargetNodePublicIPv4    *string `json:"target_node_public_ipv4,omitempty"`    // node's reported public IPv4
	TargetNodePublicIPv6    *string `json:"target_node_public_ipv6,omitempty"`    // node's reported public IPv6

	// Role indicates the requesting agent's role in this rule
	// Values: "entry" (needs to establish tunnel), "exit" (accepts tunnel connections), "relay" (chain middle node)
	Role string `json:"role,omitempty"`

	// Chain-specific fields (populated for chain rules based on requesting agent's role)
	ChainPosition          int    `json:"chain_position,omitempty"`            // agent's position in chain (0-indexed)
	IsLastInChain          bool   `json:"is_last_in_chain,omitempty"`          // true if agent is last in chain
	NextHopAgentID         string `json:"next_hop_agent_id,omitempty"`         // next agent in chain (Stripe-style ID)
	NextHopAddress         string `json:"next_hop_address,omitempty"`          // next agent's public address
	NextHopWsPort          uint16 `json:"next_hop_ws_port,omitempty"`          // next agent's WS port (from status cache)
	NextHopPort            uint16 `json:"next_hop_port,omitempty"`             // next agent's listen port (for direct_chain type)
	NextHopConnectionToken string `json:"next_hop_connection_token,omitempty"` // short-term JWT for next hop authentication

	// Internal fields for mapping (not exposed in JSON)
	internalAgentID         uint            `json:"-"`
	internalExitAgentID     uint            `json:"-"`
	internalChainAgents     []uint          `json:"-"` // internal chain agent IDs for lookup
	internalChainPortConfig map[uint]uint16 `json:"-"` // internal chain port config for lookup
	internalTargetNode      *uint           `json:"-"` // internal node ID for lookup
	agentShortID            string          `json:"-"`
	exitAgentShortID        string          `json:"-"`
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

	effectiveMultiplier := rule.GetEffectiveMultiplier()
	nodeCount := rule.CalculateNodeCount()
	isAuto := rule.GetTrafficMultiplier() == nil

	return &ForwardRuleDTO{
		ID:                         id.FormatForwardRuleID(rule.ShortID()),
		AgentID:                    "", // populated later via PopulateAgentInfo
		RuleType:                   rule.RuleType().String(),
		ExitAgentID:                "",  // populated later via PopulateAgentInfo
		ChainAgentIDs:              nil, // populated later via PopulateAgentInfo
		Name:                       rule.Name(),
		ListenPort:                 rule.ListenPort(),
		TargetAddress:              rule.TargetAddress(),
		TargetPort:                 rule.TargetPort(),
		TargetNodeID:               "", // populated later via PopulateTargetNodeShortID
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
		internalAgentID:            rule.AgentID(),
		internalExitAgentID:        rule.ExitAgentID(),
		internalChainAgents:        rule.ChainAgentIDs(),
		internalChainPortConfig:    rule.ChainPortConfig(),
		internalTargetNode:         rule.TargetNodeID(),
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
	// Populate chain agent IDs
	// For chain and direct_chain types, include entry agent (d.internalAgentID) as first element
	if len(d.internalChainAgents) > 0 && (d.RuleType == "chain" || d.RuleType == "direct_chain") {
		// Full chain: [entry_agent] + chain_agents
		fullChain := append([]uint{d.internalAgentID}, d.internalChainAgents...)
		d.ChainAgentIDs = make([]string, len(fullChain))
		for i, agentID := range fullChain {
			if shortID, ok := agentMap[agentID]; ok {
				d.ChainAgentIDs[i] = id.FormatForwardAgentID(shortID)
			}
		}
	} else if len(d.internalChainAgents) > 0 {
		// Fallback for other types (shouldn't happen)
		d.ChainAgentIDs = make([]string, len(d.internalChainAgents))
		for i, agentID := range d.internalChainAgents {
			if shortID, ok := agentMap[agentID]; ok {
				d.ChainAgentIDs[i] = id.FormatForwardAgentID(shortID)
			}
		}
	}
	// Populate chain port config (for direct_chain type)
	if len(d.internalChainPortConfig) > 0 {
		d.ChainPortConfig = make(map[string]uint16, len(d.internalChainPortConfig))
		for agentID, port := range d.internalChainPortConfig {
			if shortID, ok := agentMap[agentID]; ok {
				d.ChainPortConfig[id.FormatForwardAgentID(shortID)] = port
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
