// Package dto provides data transfer objects for the forward domain.
package dto

import (
	"context"

	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// AgentStatusInfo contains agent status information for rule conversion.
type AgentStatusInfo struct {
	WsListenPort  uint16
	TlsListenPort uint16
}

// AgentInfo contains agent information for rule conversion.
type AgentInfo struct {
	ID                       uint
	SID                      string
	EffectiveTunnelAddress   string
}

// AgentStatusProvider defines the interface for querying agent status.
type AgentStatusProvider interface {
	GetStatus(ctx context.Context, agentID uint) (*AgentStatusDTO, error)
}

// AgentInfoProvider defines the interface for querying agent information.
type AgentInfoProvider interface {
	GetByID(ctx context.Context, id uint) (*forward.ForwardAgent, error)
	GetSIDsByIDs(ctx context.Context, ids []uint) (map[uint]string, error)
}

// NodeInfoProvider defines the interface for querying node information.
type NodeInfoProvider interface {
	GetByID(ctx context.Context, id uint) (*node.Node, error)
}

// TokenGenerator defines the interface for generating agent tokens.
type TokenGenerator interface {
	Generate(shortID string) (plainToken string, tokenHash string)
}

// AgentRuleConverter converts forward rules to DTOs for agent API responses.
// It handles role-specific information population including:
// - Node address resolution based on IP version preference
// - Next hop information for chain rules
// - Agent status (WS/TLS ports) for tunnel connections
type AgentRuleConverter struct {
	agentRepo     AgentInfoProvider
	nodeRepo      NodeInfoProvider
	statusQuerier AgentStatusProvider
	tokenService  TokenGenerator
	logger        logger.Interface
}

// NewAgentRuleConverter creates a new AgentRuleConverter.
func NewAgentRuleConverter(
	agentRepo AgentInfoProvider,
	nodeRepo NodeInfoProvider,
	statusQuerier AgentStatusProvider,
	tokenService TokenGenerator,
	logger logger.Interface,
) *AgentRuleConverter {
	return &AgentRuleConverter{
		agentRepo:     agentRepo,
		nodeRepo:      nodeRepo,
		statusQuerier: statusQuerier,
		tokenService:  tokenService,
		logger:        logger,
	}
}

// ConvertBatch converts multiple rules for the same agent.
// It optimizes by batching agent SID lookups where possible.
func (c *AgentRuleConverter) ConvertBatch(ctx context.Context, rules []*forward.ForwardRule, agentID uint) ([]*ForwardRuleDTO, error) {
	if len(rules) == 0 {
		return []*ForwardRuleDTO{}, nil
	}

	// Convert to DTOs
	ruleDTOs := ToForwardRuleDTOs(rules)

	// Collect all agent IDs that need short ID lookup
	agentIDs := CollectAgentIDs(ruleDTOs)
	var agentMap map[uint]string
	if len(agentIDs) > 0 {
		var err error
		agentMap, err = c.agentRepo.GetSIDsByIDs(ctx, agentIDs)
		if err != nil {
			c.logger.Warnw("failed to get agent short IDs", "agent_ids", agentIDs, "error", err)
			agentMap = make(map[uint]string)
		}
		// Populate agent info for all DTOs
		for _, ruleDTO := range ruleDTOs {
			ruleDTO.PopulateAgentInfo(agentMap)
		}
	}

	// Resolve node addresses and populate role-specific information
	for i, rule := range rules {
		ruleDTO := ruleDTOs[i]
		c.resolveTargetNodeAddress(ctx, ruleDTO)
		c.populateRoleSpecificInfo(ctx, rule, ruleDTO, agentID)
	}

	return ruleDTOs, nil
}

// ConvertForAgent converts a single rule to DTO with role-specific information.
func (c *AgentRuleConverter) ConvertForAgent(ctx context.Context, rule *forward.ForwardRule, agentID uint) (*ForwardRuleDTO, error) {
	dtos, err := c.ConvertBatch(ctx, []*forward.ForwardRule{rule}, agentID)
	if err != nil {
		return nil, err
	}
	if len(dtos) == 0 {
		return nil, nil
	}
	return dtos[0], nil
}

// GenerateClientToken generates a client token for the given agent SID.
func (c *AgentRuleConverter) GenerateClientToken(agentSID string) string {
	token, _ := c.tokenService.Generate(agentSID)
	return token
}

// resolveTargetNodeAddress resolves the target node address for a rule DTO.
func (c *AgentRuleConverter) resolveTargetNodeAddress(ctx context.Context, ruleDTO *ForwardRuleDTO) {
	targetNodeID := ruleDTO.InternalTargetNodeID()
	if targetNodeID == nil || *targetNodeID == 0 {
		return
	}

	targetNode, err := c.nodeRepo.GetByID(ctx, *targetNodeID)
	if err != nil {
		c.logger.Warnw("failed to get target node for rule",
			"rule_id", ruleDTO.ID,
			"node_id", *targetNodeID,
			"error", err,
		)
		return
	}
	if targetNode == nil {
		c.logger.Warnw("target node not found for rule",
			"rule_id", ruleDTO.ID,
			"node_id", *targetNodeID,
		)
		return
	}

	// Dynamically populate target address and port from node
	targetAddress := c.resolveNodeAddress(targetNode, ruleDTO.IPVersion)
	ruleDTO.TargetAddress = targetAddress
	ruleDTO.TargetPort = targetNode.AgentPort()

	c.logger.Debugw("resolved target node address for rule",
		"rule_id", ruleDTO.ID,
		"node_id", *targetNodeID,
		"target_address", ruleDTO.TargetAddress,
		"target_port", ruleDTO.TargetPort,
	)
}

// resolveNodeAddress selects the appropriate node address based on IP version preference.
// ipVersion: "auto", "ipv4", or "ipv6"
func (c *AgentRuleConverter) resolveNodeAddress(targetNode *node.Node, ipVersion string) string {
	serverAddr := targetNode.ServerAddress().Value()
	ipv4 := ""
	ipv6 := ""

	if targetNode.PublicIPv4() != nil {
		ipv4 = *targetNode.PublicIPv4()
	}
	if targetNode.PublicIPv6() != nil {
		ipv6 = *targetNode.PublicIPv6()
	}

	// Check if server_address is a valid usable address
	isValidServerAddr := serverAddr != "" && serverAddr != "0.0.0.0" && serverAddr != "::"

	switch ipVersion {
	case "ipv6":
		// Prefer IPv6: ipv6 > server_address > ipv4
		if ipv6 != "" {
			return ipv6
		}
		if isValidServerAddr {
			return serverAddr
		}
		if ipv4 != "" {
			return ipv4
		}

	case "ipv4":
		// Prefer IPv4: ipv4 > server_address > ipv6
		if ipv4 != "" {
			return ipv4
		}
		if isValidServerAddr {
			return serverAddr
		}
		if ipv6 != "" {
			return ipv6
		}

	default: // "auto" or unknown
		// Default priority: server_address > ipv4 > ipv6
		if isValidServerAddr {
			return serverAddr
		}
		if ipv4 != "" {
			return ipv4
		}
		if ipv6 != "" {
			return ipv6
		}
	}

	return serverAddr
}

// populateRoleSpecificInfo populates role-specific information based on rule type.
func (c *AgentRuleConverter) populateRoleSpecificInfo(ctx context.Context, rule *forward.ForwardRule, ruleDTO *ForwardRuleDTO, agentID uint) {
	ruleType := rule.RuleType().String()

	switch ruleType {
	case "direct":
		c.populateDirectRuleInfo(ruleDTO)
	case "entry":
		c.populateEntryRuleInfo(ctx, rule, ruleDTO, agentID)
	case "chain":
		c.populateChainRuleInfo(ctx, rule, ruleDTO, agentID)
	case "direct_chain":
		c.populateDirectChainRuleInfo(ctx, rule, ruleDTO, agentID)
	}
}

// populateDirectRuleInfo populates info for direct rules.
func (c *AgentRuleConverter) populateDirectRuleInfo(ruleDTO *ForwardRuleDTO) {
	ruleDTO.Role = "entry"
}

// populateEntryRuleInfo populates info for entry rules.
func (c *AgentRuleConverter) populateEntryRuleInfo(ctx context.Context, rule *forward.ForwardRule, ruleDTO *ForwardRuleDTO, agentID uint) {
	if rule.AgentID() == agentID {
		// This agent is the entry point
		ruleDTO.Role = "entry"
		c.populateEntryNextHopInfo(ctx, rule, ruleDTO)
	} else if rule.ExitAgentID() == agentID {
		// This agent is the exit point - clear exit_agent_id (minimum info principle)
		ruleDTO.Role = "exit"
		ruleDTO.ExitAgentID = ""
	}
}

// populateEntryNextHopInfo populates next hop info for entry agent.
func (c *AgentRuleConverter) populateEntryNextHopInfo(ctx context.Context, rule *forward.ForwardRule, ruleDTO *ForwardRuleDTO) {
	exitAgentID := rule.ExitAgentID()
	if exitAgentID == 0 {
		return
	}

	exitAgent, err := c.agentRepo.GetByID(ctx, exitAgentID)
	if err != nil {
		c.logger.Warnw("failed to get exit agent for entry rule",
			"rule_id", ruleDTO.ID,
			"exit_agent_id", exitAgentID,
			"error", err,
		)
		return
	}
	if exitAgent == nil {
		return
	}

	ruleDTO.NextHopAgentID = exitAgent.SID()
	ruleDTO.NextHopAddress = exitAgent.GetEffectiveTunnelAddress()

	// Get tunnel ports from cached agent status
	exitStatus, err := c.statusQuerier.GetStatus(ctx, exitAgentID)
	if err != nil {
		c.logger.Warnw("failed to get exit agent status for entry rule",
			"rule_id", ruleDTO.ID,
			"exit_agent_id", exitAgentID,
			"error", err,
		)
		return
	}
	if exitStatus != nil {
		if exitStatus.WsListenPort > 0 {
			ruleDTO.NextHopWsPort = exitStatus.WsListenPort
		}
		if exitStatus.TlsListenPort > 0 {
			ruleDTO.NextHopTlsPort = exitStatus.TlsListenPort
		}
		if exitStatus.WsListenPort == 0 && exitStatus.TlsListenPort == 0 {
			c.logger.Debugw("exit agent has no tunnel port configured or is offline",
				"rule_id", ruleDTO.ID,
				"exit_agent_id", exitAgentID,
			)
		}
	}
}

// populateChainRuleInfo populates info for chain rules.
func (c *AgentRuleConverter) populateChainRuleInfo(ctx context.Context, rule *forward.ForwardRule, ruleDTO *ForwardRuleDTO, agentID uint) {
	chainPosition := rule.GetChainPosition(agentID)
	isLast := rule.IsLastInChain(agentID)

	ruleDTO.ChainPosition = chainPosition
	ruleDTO.IsLastInChain = isLast
	ruleDTO.TunnelHops = rule.TunnelHops()

	// Determine hop mode for hybrid chain support
	hopMode := rule.GetHopMode(chainPosition)
	ruleDTO.HopMode = hopMode

	// For boundary nodes, set inbound/outbound modes
	if hopMode == "boundary" {
		ruleDTO.InboundMode = "tunnel"
		ruleDTO.OutboundMode = "direct"
	}

	// Set listen port based on hop mode
	if chainPosition > 0 { // Not entry agent
		if hopMode == "direct" {
			ruleDTO.ListenPort = rule.GetAgentListenPort(agentID)
		} else {
			// boundary and tunnel nodes don't need ListenPort (they use WS/TLS tunnel)
			ruleDTO.ListenPort = 0
		}
	}

	// Determine role
	if chainPosition == 0 {
		ruleDTO.Role = "entry"
	} else if isLast {
		ruleDTO.Role = "exit"
	} else {
		ruleDTO.Role = "relay"
	}

	// For non-exit agents, populate next hop info
	if !isLast {
		c.populateChainNextHopInfo(ctx, rule, ruleDTO, agentID, hopMode)
		// Clear target info for non-exit agents (minimum info principle)
		ruleDTO.TargetAddress = ""
		ruleDTO.TargetPort = 0
	} else {
		// For exit agents, clear next hop info (minimum info principle)
		ruleDTO.NextHopAgentID = ""
		ruleDTO.NextHopAddress = ""
		ruleDTO.NextHopWsPort = 0
		ruleDTO.NextHopTlsPort = 0
	}
}

// populateChainNextHopInfo populates next hop info for chain agents.
func (c *AgentRuleConverter) populateChainNextHopInfo(ctx context.Context, rule *forward.ForwardRule, ruleDTO *ForwardRuleDTO, agentID uint, hopMode string) {
	nextHopAgentID := rule.GetNextHopAgentID(agentID)
	if nextHopAgentID == 0 {
		return
	}

	nextAgent, err := c.agentRepo.GetByID(ctx, nextHopAgentID)
	if err != nil {
		c.logger.Warnw("failed to get next hop agent for chain rule",
			"rule_id", ruleDTO.ID,
			"next_hop_agent_id", nextHopAgentID,
			"error", err,
		)
		return
	}
	if nextAgent == nil {
		return
	}

	ruleDTO.NextHopAgentID = nextAgent.SID()
	ruleDTO.NextHopAddress = nextAgent.GetEffectiveTunnelAddress()

	// Check if outbound uses tunnel or direct based on hop mode
	outboundNeedsTunnel := hopMode == "tunnel" || (hopMode == "boundary" && ruleDTO.OutboundMode == "tunnel")
	if !outboundNeedsTunnel && (hopMode == "direct" || hopMode == "boundary") {
		// Direct connection mode: use chainPortConfig for next hop port
		nextHopPort := rule.GetAgentListenPort(nextHopAgentID)
		if nextHopPort > 0 {
			ruleDTO.NextHopPort = nextHopPort
		}
		// Generate connection token for direct hop authentication
		nextHopToken, _ := c.tokenService.Generate(nextAgent.SID())
		ruleDTO.NextHopConnectionToken = nextHopToken
	} else {
		// Tunnel mode: get tunnel ports from cached agent status
		nextStatus, err := c.statusQuerier.GetStatus(ctx, nextHopAgentID)
		if err != nil {
			c.logger.Warnw("failed to get next hop agent status",
				"rule_id", ruleDTO.ID,
				"next_hop_agent_id", nextHopAgentID,
				"error", err,
			)
			return
		}
		if nextStatus != nil {
			if nextStatus.WsListenPort > 0 {
				ruleDTO.NextHopWsPort = nextStatus.WsListenPort
			}
			if nextStatus.TlsListenPort > 0 {
				ruleDTO.NextHopTlsPort = nextStatus.TlsListenPort
			}
			if nextStatus.WsListenPort == 0 && nextStatus.TlsListenPort == 0 {
				c.logger.Debugw("next hop agent has no tunnel port configured or is offline",
					"rule_id", ruleDTO.ID,
					"next_hop_agent_id", nextHopAgentID,
				)
			}
		}
	}
}

// populateDirectChainRuleInfo populates info for direct_chain rules.
func (c *AgentRuleConverter) populateDirectChainRuleInfo(ctx context.Context, rule *forward.ForwardRule, ruleDTO *ForwardRuleDTO, agentID uint) {
	chainPosition := rule.GetChainPosition(agentID)
	isLast := rule.IsLastInChain(agentID)

	// Defensive check: agent must be in chain
	if chainPosition < 0 {
		c.logger.Errorw("agent not found in direct_chain rule",
			"agent_id", agentID,
			"rule_id", ruleDTO.ID,
			"entry_agent_id", rule.AgentID(),
			"chain_agent_ids", rule.ChainAgentIDs(),
		)
		return
	}

	ruleDTO.ChainPosition = chainPosition
	ruleDTO.IsLastInChain = isLast

	// Determine role and set ListenPort
	if chainPosition == 0 {
		ruleDTO.Role = "entry"
		ruleDTO.ListenPort = rule.ListenPort()
	} else if isLast {
		ruleDTO.Role = "exit"
		ruleDTO.ListenPort = rule.GetAgentListenPort(agentID)
	} else {
		ruleDTO.Role = "relay"
		ruleDTO.ListenPort = rule.GetAgentListenPort(agentID)
	}

	// For non-exit agents, populate next hop info
	if !isLast {
		c.populateDirectChainNextHopInfo(ctx, rule, ruleDTO, agentID)
		// Clear target info for non-exit agents (minimum info principle)
		ruleDTO.TargetAddress = ""
		ruleDTO.TargetPort = 0
	} else {
		// For exit agents, clear next hop info (minimum info principle)
		ruleDTO.NextHopAgentID = ""
		ruleDTO.NextHopAddress = ""
		ruleDTO.NextHopPort = 0
	}
}

// populateDirectChainNextHopInfo populates next hop info for direct_chain agents.
func (c *AgentRuleConverter) populateDirectChainNextHopInfo(ctx context.Context, rule *forward.ForwardRule, ruleDTO *ForwardRuleDTO, agentID uint) {
	nextHopAgentID, nextHopPort, err := rule.GetNextHopForDirectChainSafe(agentID)
	if err != nil {
		c.logger.Errorw("failed to get next hop for direct_chain rule",
			"rule_id", ruleDTO.ID,
			"agent_id", agentID,
			"error", err,
		)
		return
	}

	if nextHopAgentID == 0 {
		return
	}

	nextAgent, err := c.agentRepo.GetByID(ctx, nextHopAgentID)
	if err != nil {
		c.logger.Warnw("failed to get next hop agent for direct_chain rule",
			"rule_id", ruleDTO.ID,
			"next_hop_agent_id", nextHopAgentID,
			"error", err,
		)
		return
	}
	if nextAgent == nil {
		return
	}

	ruleDTO.NextHopAgentID = nextAgent.SID()
	ruleDTO.NextHopAddress = nextAgent.GetEffectiveTunnelAddress()
	ruleDTO.NextHopPort = nextHopPort

	// Generate connection token for next hop authentication
	nextHopToken, _ := c.tokenService.Generate(nextAgent.SID())
	ruleDTO.NextHopConnectionToken = nextHopToken
}
