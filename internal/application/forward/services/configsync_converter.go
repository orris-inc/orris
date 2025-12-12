package services

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/shared/id"
)

// convertRuleToSyncData converts a ForwardRule to RuleSyncData.
// This mirrors the logic in AgentHandler.GetEnabledRules for building rule DTOs.
func (s *ConfigSyncService) convertRuleToSyncData(ctx context.Context, rule *forward.ForwardRule, agentID uint) (*dto.RuleSyncData, error) {
	syncData := &dto.RuleSyncData{
		ID:         id.FormatForwardRuleID(rule.ShortID()),
		ShortID:    rule.ShortID(),
		RuleType:   rule.RuleType().String(),
		ListenPort: rule.ListenPort(),
		Protocol:   rule.Protocol().String(),
		BindIP:     rule.BindIP(),
	}

	// Resolve target address and port
	targetAddress := rule.TargetAddress()
	targetPort := rule.TargetPort()

	// If rule has target node, get address from node
	if rule.HasTargetNode() {
		targetNode, err := s.nodeRepo.GetByID(ctx, *rule.TargetNodeID())
		if err != nil {
			s.logger.Warnw("failed to get target node for rule",
				"rule_id", rule.ID(),
				"node_id", *rule.TargetNodeID(),
				"error", err,
			)
			// Use original values if node fetch fails
		} else if targetNode != nil {
			// Dynamically populate target address and port from node
			// Selection priority depends on rule's IP version setting
			nodeTargetAddress := s.resolveNodeAddress(targetNode, rule.IPVersion().String())
			targetAddress = nodeTargetAddress
			targetPort = targetNode.AgentPort()
		}
	}

	// Determine role based on rule type and requesting agent
	switch rule.RuleType().String() {
	case "direct":
		s.convertDirectRule(syncData, targetAddress, targetPort)

	case "entry":
		s.convertEntryRule(ctx, rule, agentID, syncData, targetAddress, targetPort)

	case "chain":
		if err := s.convertChainRule(ctx, rule, agentID, syncData, targetAddress, targetPort); err != nil {
			return nil, err
		}

	case "direct_chain":
		if err := s.convertDirectChainRule(ctx, rule, agentID, syncData, targetAddress, targetPort); err != nil {
			return nil, err
		}
	}

	return syncData, nil
}

// convertDirectRule handles direct rule type conversion
func (s *ConfigSyncService) convertDirectRule(syncData *dto.RuleSyncData, targetAddress string, targetPort uint16) {
	syncData.Role = "entry"
	syncData.TargetAddress = targetAddress
	syncData.TargetPort = targetPort
}

// convertEntryRule handles entry rule type conversion
func (s *ConfigSyncService) convertEntryRule(ctx context.Context, rule *forward.ForwardRule, agentID uint, syncData *dto.RuleSyncData, targetAddress string, targetPort uint16) {
	if rule.AgentID() == agentID {
		// This agent is the entry point
		syncData.Role = "entry"
		// Entry agent needs to know the exit agent info to establish tunnel
		exitAgentID := rule.ExitAgentID()
		if exitAgentID != 0 {
			exitAgent, err := s.agentRepo.GetByID(ctx, exitAgentID)
			if err != nil {
				s.logger.Warnw("failed to get exit agent for entry rule",
					"rule_id", rule.ID(),
					"exit_agent_id", exitAgentID,
					"error", err,
				)
			} else if exitAgent != nil {
				syncData.NextHopAgentID = id.FormatForwardAgentID(exitAgent.ShortID())
				syncData.NextHopAddress = exitAgent.GetEffectiveTunnelAddress()

				// Get ws_listen_port from cached agent status
				exitStatus, err := s.statusQuerier.GetStatus(ctx, exitAgentID)
				if err != nil {
					s.logger.Warnw("failed to get exit agent status",
						"rule_id", rule.ID(),
						"exit_agent_id", exitAgentID,
						"error", err,
					)
				} else if exitStatus != nil && exitStatus.WsListenPort > 0 {
					syncData.NextHopWsPort = exitStatus.WsListenPort
				} else {
					s.logger.Warnw("exit agent has no ws_listen_port configured or is offline",
						"rule_id", rule.ID(),
						"exit_agent_id", exitAgentID,
					)
				}
			}
		}
	} else if rule.ExitAgentID() == agentID {
		// This agent is the exit point
		syncData.Role = "exit"
		syncData.TargetAddress = targetAddress
		syncData.TargetPort = targetPort

		// Exit agent needs entry agent ID to verify tunnel handshake
		entryAgentID := rule.AgentID()
		if entryAgentID != 0 {
			entryAgent, err := s.agentRepo.GetByID(ctx, entryAgentID)
			if err != nil {
				s.logger.Warnw("failed to get entry agent for exit rule",
					"rule_id", rule.ID(),
					"entry_agent_id", entryAgentID,
					"error", err,
				)
			} else if entryAgent != nil {
				syncData.AgentID = id.FormatForwardAgentID(entryAgent.ShortID())
			}
		}
	}
}

// convertChainRule handles chain rule type conversion
func (s *ConfigSyncService) convertChainRule(ctx context.Context, rule *forward.ForwardRule, agentID uint, syncData *dto.RuleSyncData, targetAddress string, targetPort uint16) error {
	// Calculate chain position and last-in-chain flag for this agent
	chainPosition := rule.GetChainPosition(agentID)
	isLast := rule.IsLastInChain(agentID)

	syncData.ChainPosition = chainPosition
	syncData.IsLastInChain = isLast

	// Populate ChainAgentIDs (Stripe-style IDs)
	// Full chain: [entry_agent] + chain_agents (matches GetChainPosition calculation)
	fullChainIDs := append([]uint{rule.AgentID()}, rule.ChainAgentIDs()...)
	if len(fullChainIDs) > 0 {
		agentMap, err := s.agentRepo.GetShortIDsByIDs(ctx, fullChainIDs)
		if err != nil {
			s.logger.Warnw("failed to get chain agent short IDs",
				"rule_id", rule.ID(),
				"error", err,
			)
		} else {
			syncData.ChainAgentIDs = make([]string, len(fullChainIDs))
			for i, chainAgentID := range fullChainIDs {
				if shortID, ok := agentMap[chainAgentID]; ok {
					syncData.ChainAgentIDs[i] = id.FormatForwardAgentID(shortID)
				}
			}
		}
	}

	// Determine role
	if chainPosition == 0 {
		syncData.Role = "entry"
	} else if isLast {
		syncData.Role = "exit"
	} else {
		syncData.Role = "relay"
	}

	// For non-exit agents in chain, populate next hop information
	if !isLast {
		nextHopAgentID := rule.GetNextHopAgentID(agentID)
		if nextHopAgentID != 0 {
			// Get next hop agent details
			nextAgent, err := s.agentRepo.GetByID(ctx, nextHopAgentID)
			if err != nil {
				s.logger.Warnw("failed to get next hop agent for chain rule",
					"rule_id", rule.ID(),
					"next_hop_agent_id", nextHopAgentID,
					"error", err,
				)
			} else if nextAgent != nil {
				syncData.NextHopAgentID = id.FormatForwardAgentID(nextAgent.ShortID())
				syncData.NextHopAddress = nextAgent.GetEffectiveTunnelAddress()

				// Get ws_listen_port from cached agent status
				nextStatus, err := s.statusQuerier.GetStatus(ctx, nextHopAgentID)
				if err != nil {
					s.logger.Warnw("failed to get next hop agent status",
						"rule_id", rule.ID(),
						"next_hop_agent_id", nextHopAgentID,
						"error", err,
					)
				} else if nextStatus != nil && nextStatus.WsListenPort > 0 {
					syncData.NextHopWsPort = nextStatus.WsListenPort
				} else {
					s.logger.Warnw("next hop agent has no ws_listen_port configured or is offline",
						"rule_id", rule.ID(),
						"next_hop_agent_id", nextHopAgentID,
					)
				}
			}
		}
	} else {
		// For exit agents, include target info
		syncData.TargetAddress = targetAddress
		syncData.TargetPort = targetPort
	}

	return nil
}

// convertDirectChainRule handles direct_chain rule type conversion
func (s *ConfigSyncService) convertDirectChainRule(ctx context.Context, rule *forward.ForwardRule, agentID uint, syncData *dto.RuleSyncData, targetAddress string, targetPort uint16) error {
	// Calculate chain position and last-in-chain flag for this agent
	chainPosition := rule.GetChainPosition(agentID)
	isLast := rule.IsLastInChain(agentID)

	// Defensive check: agent must be in chain
	if chainPosition < 0 {
		s.logger.Errorw("agent not found in direct_chain rule",
			"agent_id", agentID,
			"rule_id", rule.ID(),
			"entry_agent_id", rule.AgentID(),
			"chain_agent_ids", rule.ChainAgentIDs(),
		)
		return fmt.Errorf("agent %d not found in direct_chain rule %d", agentID, rule.ID())
	}

	syncData.ChainPosition = chainPosition
	syncData.IsLastInChain = isLast

	// Populate ChainAgentIDs (Stripe-style IDs)
	// Include entry agent (rule.AgentID) + chain agents for complete chain
	entryAgentID := rule.AgentID()
	chainAgentIDs := rule.ChainAgentIDs()

	// Debug logging for chain position and role assignment
	s.logger.Infow("direct_chain rule sync debug",
		"current_agent_id", agentID,
		"rule_entry_agent_id", entryAgentID,
		"chain_agent_ids", chainAgentIDs,
		"calculated_position", chainPosition,
		"is_last", isLast,
	)
	fullChainIDs := append([]uint{entryAgentID}, chainAgentIDs...)

	agentMap, err := s.agentRepo.GetShortIDsByIDs(ctx, fullChainIDs)
	if err != nil {
		s.logger.Warnw("failed to get chain agent short IDs",
			"rule_id", rule.ID(),
			"error", err,
		)
	} else {
		syncData.ChainAgentIDs = make([]string, len(fullChainIDs))
		for i, chainAgentID := range fullChainIDs {
			if shortID, ok := agentMap[chainAgentID]; ok {
				syncData.ChainAgentIDs[i] = id.FormatForwardAgentID(shortID)
			} else {
				s.logger.Warnw("chain agent ID not found in agent map",
					"rule_id", rule.ID(),
					"chain_agent_id", chainAgentID,
					"position", i,
				)
			}
		}
	}

	// Determine role and set ListenPort
	// Entry agent uses rule.ListenPort(), other agents use chainPortConfig
	if chainPosition == 0 {
		syncData.Role = "entry"
		// Entry agent uses the rule's listen_port field
		syncData.ListenPort = rule.ListenPort()
	} else if isLast {
		syncData.Role = "exit"
		syncData.ListenPort = rule.GetAgentListenPort(agentID)
	} else {
		syncData.Role = "relay"
		syncData.ListenPort = rule.GetAgentListenPort(agentID)
	}

	// For non-exit agents in chain, populate next hop information
	if !isLast {
		nextHopAgentID, nextHopPort := rule.GetNextHopForDirectChain(agentID)
		s.logger.Infow("direct_chain next hop lookup",
			"current_agent_id", agentID,
			"next_hop_agent_id", nextHopAgentID,
			"next_hop_port", nextHopPort,
		)
		if nextHopAgentID != 0 {
			// Get next hop agent details
			nextAgent, err := s.agentRepo.GetByID(ctx, nextHopAgentID)
			if err != nil {
				s.logger.Warnw("failed to get next hop agent for direct_chain rule",
					"rule_id", rule.ID(),
					"next_hop_agent_id", nextHopAgentID,
					"error", err,
				)
			} else if nextAgent != nil {
				syncData.NextHopAgentID = id.FormatForwardAgentID(nextAgent.ShortID())
				syncData.NextHopAddress = nextAgent.GetEffectiveTunnelAddress()
				syncData.NextHopPort = nextHopPort

				// Generate connection token for next hop authentication
				nextHopToken, _ := s.agentTokenService.Generate(nextAgent.ShortID())
				syncData.NextHopConnectionToken = nextHopToken

				s.logger.Infow("direct_chain next hop token generated",
					"current_agent_id", agentID,
					"next_hop_short_id", nextAgent.ShortID(),
					"next_hop_token", nextHopToken,
				)
			}
		}
	} else {
		// For exit agents, include target info
		syncData.TargetAddress = targetAddress
		syncData.TargetPort = targetPort
	}

	return nil
}

// resolveNodeAddress selects the appropriate node address based on IP version preference.
// ipVersion: "auto", "ipv4", or "ipv6"
func (s *ConfigSyncService) resolveNodeAddress(targetNode *node.Node, ipVersion string) string {
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
