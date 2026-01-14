package services

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/application/forward/usecases"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/infrastructure/auth"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// RuleSyncConverter converts ForwardRule to RuleSyncData for config sync.
// It handles different rule types (direct, entry, chain, direct_chain)
// and populates appropriate fields based on agent's role in the rule.
type RuleSyncConverter struct {
	agentRepo         forward.AgentRepository
	nodeRepo          node.NodeRepository
	statusQuerier     usecases.AgentStatusQuerier
	agentTokenService *auth.AgentTokenService
	logger            logger.Interface
}

// NewRuleSyncConverter creates a new RuleSyncConverter.
func NewRuleSyncConverter(
	agentRepo forward.AgentRepository,
	nodeRepo node.NodeRepository,
	statusQuerier usecases.AgentStatusQuerier,
	agentTokenService *auth.AgentTokenService,
	log logger.Interface,
) *RuleSyncConverter {
	return &RuleSyncConverter{
		agentRepo:         agentRepo,
		nodeRepo:          nodeRepo,
		statusQuerier:     statusQuerier,
		agentTokenService: agentTokenService,
		logger:            log,
	}
}

// SetNodeRepo sets the node repository for circular dependency handling.
func (c *RuleSyncConverter) SetNodeRepo(nodeRepo node.NodeRepository) {
	c.nodeRepo = nodeRepo
}

// Convert converts a ForwardRule to RuleSyncData for a specific agent.
// This mirrors the logic in AgentHandler.GetEnabledRules for building rule DTOs.
func (c *RuleSyncConverter) Convert(ctx context.Context, rule *forward.ForwardRule, agentID uint) (*dto.RuleSyncData, error) {
	syncData := &dto.RuleSyncData{
		ID:         rule.SID(),
		ShortID:    rule.SID(),
		RuleType:   rule.RuleType().String(),
		ListenPort: rule.ListenPort(),
		Protocol:   rule.Protocol().String(),
		BindIP:     rule.BindIP(),
		TunnelType: rule.TunnelType().String(),
	}

	// Resolve target address and port
	targetAddress := rule.TargetAddress()
	targetPort := rule.TargetPort()

	// If rule has target node, get address from node
	if rule.HasTargetNode() {
		targetNode, err := c.nodeRepo.GetByID(ctx, *rule.TargetNodeID())
		if err != nil {
			c.logger.Warnw("failed to get target node for rule",
				"rule_id", rule.ID(),
				"node_id", *rule.TargetNodeID(),
				"error", err,
			)
			// Use original values if node fetch fails
		} else if targetNode != nil {
			// Dynamically populate target address and port from node
			// Selection priority depends on rule's IP version setting
			nodeTargetAddress := c.resolveNodeAddress(targetNode, rule.IPVersion().String())
			targetAddress = nodeTargetAddress
			targetPort = targetNode.AgentPort()
		}
	}

	// Determine role and populate fields based on rule type
	switch rule.RuleType().String() {
	case "direct":
		c.convertDirectRule(syncData, targetAddress, targetPort)

	case "entry":
		if err := c.convertEntryRule(ctx, rule, agentID, syncData, targetAddress, targetPort); err != nil {
			return nil, err
		}

	case "chain":
		if err := c.convertChainRule(ctx, rule, agentID, syncData, targetAddress, targetPort); err != nil {
			return nil, err
		}

	case "direct_chain":
		if err := c.convertDirectChainRule(ctx, rule, agentID, syncData, targetAddress, targetPort); err != nil {
			return nil, err
		}
	}

	return syncData, nil
}

// convertDirectRule handles direct rule type conversion.
func (c *RuleSyncConverter) convertDirectRule(data *dto.RuleSyncData, targetAddress string, targetPort uint16) {
	data.Role = "entry"
	data.TargetAddress = targetAddress
	data.TargetPort = targetPort
}

// convertEntryRule handles entry rule type conversion.
func (c *RuleSyncConverter) convertEntryRule(
	ctx context.Context,
	rule *forward.ForwardRule,
	agentID uint,
	data *dto.RuleSyncData,
	targetAddress string,
	targetPort uint16,
) error {
	if rule.AgentID() == agentID {
		// This agent is the entry point
		data.Role = "entry"
		// Entry agent needs to know the exit agent info to establish tunnel
		exitAgentID := rule.ExitAgentID()
		if exitAgentID != 0 {
			if err := c.populateNextHopInfo(ctx, data, exitAgentID, "tunnel", rule); err != nil {
				c.logger.Warnw("failed to populate next hop info for entry rule",
					"rule_id", rule.ID(),
					"exit_agent_id", exitAgentID,
					"error", err,
				)
			}
		}
	} else if rule.ExitAgentID() == agentID {
		// This agent is the exit point
		data.Role = "exit"
		data.TargetAddress = targetAddress
		data.TargetPort = targetPort

		// Exit agent needs entry agent ID to verify tunnel handshake
		entryAgentID := rule.AgentID()
		if entryAgentID != 0 {
			entryAgent, err := c.agentRepo.GetByID(ctx, entryAgentID)
			if err != nil {
				c.logger.Warnw("failed to get entry agent for exit rule",
					"rule_id", rule.ID(),
					"entry_agent_id", entryAgentID,
					"error", err,
				)
			} else if entryAgent != nil {
				data.AgentID = entryAgent.SID()
			}
		}
	}

	return nil
}

// convertChainRule handles chain rule type conversion.
func (c *RuleSyncConverter) convertChainRule(
	ctx context.Context,
	rule *forward.ForwardRule,
	agentID uint,
	data *dto.RuleSyncData,
	targetAddress string,
	targetPort uint16,
) error {
	// Calculate chain position and last-in-chain flag for this agent
	chainPosition := rule.GetChainPosition(agentID)
	isLast := rule.IsLastInChain(agentID)

	data.ChainPosition = chainPosition
	data.IsLastInChain = isLast
	data.TunnelHops = rule.TunnelHops()

	// Determine hop mode for hybrid chain support
	hopMode := rule.GetHopMode(chainPosition)
	data.HopMode = hopMode

	// For boundary nodes, set inbound/outbound modes
	if hopMode == "boundary" {
		data.InboundMode = "tunnel"
		data.OutboundMode = "direct"
	}

	// Populate ChainAgentIDs (Stripe-style IDs)
	if err := c.populateChainAgentIDs(ctx, rule, data); err != nil {
		c.logger.Warnw("failed to populate chain agent IDs",
			"rule_id", rule.ID(),
			"error", err,
		)
	}

	// Determine role
	if chainPosition == 0 {
		data.Role = "entry"
	} else if isLast {
		data.Role = "exit"
	} else {
		data.Role = "relay"
	}

	// For non-exit agents in chain, populate next hop information
	if !isLast {
		c.populateChainNextHop(ctx, rule, agentID, data, hopMode)
	} else {
		// For exit agents, include target info
		data.TargetAddress = targetAddress
		data.TargetPort = targetPort
	}

	// Set listen port based on hop mode
	c.setChainListenPort(rule, agentID, data, chainPosition, hopMode)

	return nil
}

// convertDirectChainRule handles direct_chain rule type conversion.
func (c *RuleSyncConverter) convertDirectChainRule(
	ctx context.Context,
	rule *forward.ForwardRule,
	agentID uint,
	data *dto.RuleSyncData,
	targetAddress string,
	targetPort uint16,
) error {
	// Calculate chain position and last-in-chain flag for this agent
	chainPosition := rule.GetChainPosition(agentID)
	isLast := rule.IsLastInChain(agentID)

	// Defensive check: agent must be in chain
	if chainPosition < 0 {
		c.logger.Errorw("agent not found in direct_chain rule",
			"agent_id", agentID,
			"rule_id", rule.ID(),
			"entry_agent_id", rule.AgentID(),
			"chain_agent_ids", rule.ChainAgentIDs(),
		)
		return fmt.Errorf("agent %d not found in direct_chain rule %d", agentID, rule.ID())
	}

	data.ChainPosition = chainPosition
	data.IsLastInChain = isLast

	// Debug logging for chain position and role assignment
	c.logger.Debugw("direct_chain rule sync",
		"current_agent_id", agentID,
		"rule_entry_agent_id", rule.AgentID(),
		"chain_agent_ids", rule.ChainAgentIDs(),
		"calculated_position", chainPosition,
		"is_last", isLast,
	)

	// Populate ChainAgentIDs (Stripe-style IDs)
	if err := c.populateChainAgentIDs(ctx, rule, data); err != nil {
		c.logger.Warnw("failed to populate chain agent IDs",
			"rule_id", rule.ID(),
			"error", err,
		)
	}

	// Determine role and set ListenPort
	if chainPosition == 0 {
		data.Role = "entry"
		data.ListenPort = rule.ListenPort()
	} else if isLast {
		data.Role = "exit"
		data.ListenPort = rule.GetAgentListenPort(agentID)
	} else {
		data.Role = "relay"
		data.ListenPort = rule.GetAgentListenPort(agentID)
	}

	// For non-exit agents in chain, populate next hop information
	if !isLast {
		if err := c.populateDirectChainNextHop(ctx, rule, agentID, data); err != nil {
			return err
		}
	} else {
		// For exit agents, include target info
		data.TargetAddress = targetAddress
		data.TargetPort = targetPort
	}

	return nil
}

// populateNextHopInfo populates next hop agent information into sync data.
// hopMode: "tunnel" or "direct"
func (c *RuleSyncConverter) populateNextHopInfo(
	ctx context.Context,
	data *dto.RuleSyncData,
	nextAgentID uint,
	hopMode string,
	rule *forward.ForwardRule,
) error {
	nextAgent, err := c.agentRepo.GetByID(ctx, nextAgentID)
	if err != nil {
		return fmt.Errorf("get next agent: %w", err)
	}
	if nextAgent == nil {
		return fmt.Errorf("next agent not found: %d", nextAgentID)
	}

	data.NextHopAgentID = nextAgent.SID()
	data.NextHopAddress = nextAgent.GetEffectiveTunnelAddress()

	// Check if outbound uses tunnel or direct based on hop mode
	outboundNeedsTunnel := hopMode == "tunnel" || (hopMode == "boundary" && data.OutboundMode == "tunnel")

	if !outboundNeedsTunnel && (hopMode == "direct" || hopMode == "boundary") {
		// Direct connection mode: use chainPortConfig for next hop port
		if rule != nil {
			nextHopPort := rule.GetAgentListenPort(nextAgentID)
			if nextHopPort > 0 {
				data.NextHopPort = nextHopPort
			}
		}
		// Generate connection token for direct hop authentication
		if c.agentTokenService != nil {
			nextHopToken, _ := c.agentTokenService.Generate(nextAgent.SID())
			data.NextHopConnectionToken = nextHopToken
		}
	} else {
		// Tunnel mode: get tunnel ports from cached agent status
		nextStatus, err := c.statusQuerier.GetStatus(ctx, nextAgentID)
		if err != nil {
			c.logger.Warnw("failed to get next hop agent status",
				"next_hop_agent_id", nextAgentID,
				"error", err,
			)
		} else if nextStatus != nil {
			if nextStatus.WsListenPort > 0 {
				data.NextHopWsPort = nextStatus.WsListenPort
			}
			if nextStatus.TlsListenPort > 0 {
				data.NextHopTlsPort = nextStatus.TlsListenPort
			}
			if nextStatus.WsListenPort == 0 && nextStatus.TlsListenPort == 0 {
				c.logger.Debugw("next hop agent has no tunnel port configured or is offline",
					"next_hop_agent_id", nextAgentID,
				)
			}
		}
	}

	return nil
}

// populateChainAgentIDs populates ChainAgentIDs with Stripe-style IDs.
func (c *RuleSyncConverter) populateChainAgentIDs(ctx context.Context, rule *forward.ForwardRule, data *dto.RuleSyncData) error {
	// Full chain: [entry_agent] + chain_agents (matches GetChainPosition calculation)
	fullChainIDs := append([]uint{rule.AgentID()}, rule.ChainAgentIDs()...)
	if len(fullChainIDs) == 0 {
		return nil
	}

	agentMap, err := c.agentRepo.GetSIDsByIDs(ctx, fullChainIDs)
	if err != nil {
		return fmt.Errorf("get chain agent SIDs: %w", err)
	}

	data.ChainAgentIDs = make([]string, len(fullChainIDs))
	for i, chainAgentID := range fullChainIDs {
		if sid, ok := agentMap[chainAgentID]; ok {
			data.ChainAgentIDs[i] = sid
		} else {
			c.logger.Warnw("chain agent ID not found in agent map",
				"rule_id", rule.ID(),
				"chain_agent_id", chainAgentID,
				"position", i,
			)
		}
	}

	return nil
}

// populateChainNextHop populates next hop information for chain rules.
func (c *RuleSyncConverter) populateChainNextHop(
	ctx context.Context,
	rule *forward.ForwardRule,
	agentID uint,
	data *dto.RuleSyncData,
	hopMode string,
) {
	nextHopAgentID := rule.GetNextHopAgentID(agentID)
	if nextHopAgentID == 0 {
		return
	}

	if err := c.populateNextHopInfo(ctx, data, nextHopAgentID, hopMode, rule); err != nil {
		c.logger.Warnw("failed to populate next hop info for chain rule",
			"rule_id", rule.ID(),
			"next_hop_agent_id", nextHopAgentID,
			"error", err,
		)
	}
}

// populateDirectChainNextHop populates next hop information for direct_chain rules.
func (c *RuleSyncConverter) populateDirectChainNextHop(
	ctx context.Context,
	rule *forward.ForwardRule,
	agentID uint,
	data *dto.RuleSyncData,
) error {
	nextHopAgentID, nextHopPort, err := rule.GetNextHopForDirectChainSafe(agentID)
	if err != nil {
		c.logger.Errorw("failed to get next hop for direct_chain rule in config sync",
			"rule_id", rule.ID(),
			"agent_id", agentID,
			"error", err,
		)
		return fmt.Errorf("failed to get next hop for direct_chain rule: %w", err)
	}

	c.logger.Debugw("direct_chain next hop lookup",
		"current_agent_id", agentID,
		"next_hop_agent_id", nextHopAgentID,
		"next_hop_port", nextHopPort,
	)

	if nextHopAgentID == 0 {
		return nil
	}

	nextAgent, err := c.agentRepo.GetByID(ctx, nextHopAgentID)
	if err != nil {
		c.logger.Warnw("failed to get next hop agent for direct_chain rule",
			"rule_id", rule.ID(),
			"next_hop_agent_id", nextHopAgentID,
			"error", err,
		)
		return nil
	}

	if nextAgent == nil {
		return nil
	}

	data.NextHopAgentID = nextAgent.SID()
	data.NextHopAddress = nextAgent.GetEffectiveTunnelAddress()
	data.NextHopPort = nextHopPort

	// Generate connection token for next hop authentication
	nextHopToken, _ := c.agentTokenService.Generate(nextAgent.SID())
	data.NextHopConnectionToken = nextHopToken

	c.logger.Debugw("direct_chain next hop token generated",
		"current_agent_id", agentID,
		"next_hop_short_id", nextAgent.SID(),
	)

	return nil
}

// setChainListenPort sets listen port based on hop mode for chain rules.
// - entry (pos 0): uses rule.ListenPort() (already set in initialization)
// - boundary: receives via tunnel, so clear ListenPort (use WS/TLS port instead)
// - direct: receives via direct connection, use chainPortConfig port
func (c *RuleSyncConverter) setChainListenPort(
	rule *forward.ForwardRule,
	agentID uint,
	data *dto.RuleSyncData,
	chainPosition int,
	hopMode string,
) {
	if chainPosition == 0 {
		// Entry agent: uses rule.ListenPort() (already set in initialization)
		return
	}

	if hopMode == "direct" {
		listenPort := rule.GetAgentListenPort(agentID)
		if listenPort > 0 {
			data.ListenPort = listenPort
		} else {
			data.ListenPort = 0
		}
	} else {
		// boundary and tunnel nodes don't need ListenPort (they use WS/TLS tunnel)
		data.ListenPort = 0
	}
}

// GenerateClientToken generates a client token for the given agent SID.
// Returns the plain token (not the hash).
func (c *RuleSyncConverter) GenerateClientToken(agentSID string) string {
	plainToken, _ := c.agentTokenService.Generate(agentSID)
	return plainToken
}

// resolveNodeAddress selects the appropriate node address based on IP version preference.
// ipVersion: "auto", "ipv4", or "ipv6"
func (c *RuleSyncConverter) resolveNodeAddress(targetNode *node.Node, ipVersion string) string {
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
