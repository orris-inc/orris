package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/application/forward/usecases"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/infrastructure/auth"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// ConfigSyncService handles incremental configuration synchronization for agents.
// It implements agent.MessageHandler interface.
type ConfigSyncService struct {
	repo               forward.Repository
	agentRepo          forward.AgentRepository
	nodeRepo           node.NodeRepository
	statusQuerier      usecases.AgentStatusQuerier
	tokenSigningSecret string
	agentTokenService  *auth.AgentTokenService

	// Hub interface for sending messages
	hub SyncHub

	// Agent version tracking: map[agentID]version
	agentVersions sync.Map

	// Global version counter (incremented on each config change)
	globalVersion atomic.Uint64

	logger logger.Interface
}

// SyncHub defines the interface for sending messages through the hub.
type SyncHub interface {
	IsAgentOnline(agentID uint) bool
	SendMessageToAgent(agentID uint, msg *dto.HubMessage) error
}

// NewConfigSyncService creates a new ConfigSyncService.
func NewConfigSyncService(
	repo forward.Repository,
	agentRepo forward.AgentRepository,
	nodeRepo node.NodeRepository,
	statusQuerier usecases.AgentStatusQuerier,
	tokenSigningSecret string,
	hub SyncHub,
	log logger.Interface,
) *ConfigSyncService {
	svc := &ConfigSyncService{
		repo:               repo,
		agentRepo:          agentRepo,
		nodeRepo:           nodeRepo,
		statusQuerier:      statusQuerier,
		tokenSigningSecret: tokenSigningSecret,
		agentTokenService:  auth.NewAgentTokenService(tokenSigningSecret),
		hub:                hub,
		logger:             log,
	}
	// Initialize global version to 1
	svc.globalVersion.Store(1)
	return svc
}

// String implements fmt.Stringer for logging purposes.
func (s *ConfigSyncService) String() string {
	return "ConfigSyncService"
}

// HandleMessage processes config sync acknowledgment messages from agents.
// Implements agent.MessageHandler interface.
func (s *ConfigSyncService) HandleMessage(agentID uint, msgType string, data any) bool {
	switch msgType {
	case dto.MsgTypeConfigAck:
		s.handleConfigAck(agentID, data)
		return true
	default:
		return false
	}
}

// NotifyRuleChange notifies an agent about a rule change (add/update/delete).
// changeType should be "added", "updated", or "removed".
func (s *ConfigSyncService) NotifyRuleChange(ctx context.Context, agentID uint, ruleShortID string, changeType string) error {
	s.logger.Infow("notifying agent of rule change",
		"agent_id", agentID,
		"rule_short_id", ruleShortID,
		"change_type", changeType,
	)

	// Check if agent is online
	if !s.hub.IsAgentOnline(agentID) {
		s.logger.Debugw("agent offline, skipping incremental sync notification",
			"agent_id", agentID,
			"rule_short_id", ruleShortID,
		)
		return nil
	}

	// Increment global version
	version := s.globalVersion.Add(1)

	// Build sync data based on change type
	syncData := &dto.ConfigSyncData{
		Version:  version,
		FullSync: false,
	}

	switch changeType {
	case "added", "updated":
		// Fetch the rule to include in sync
		rule, err := s.repo.GetBySID(ctx, ruleShortID)
		if err != nil {
			s.logger.Errorw("failed to get rule for sync",
				"rule_short_id", ruleShortID,
				"error", err,
			)
			return err
		}
		if rule == nil {
			s.logger.Debugw("rule not found for sync",
				"rule_short_id", ruleShortID,
			)
			return forward.ErrRuleNotFound
		}

		// Convert to sync data
		ruleSyncData, err := s.convertRuleToSyncData(ctx, rule, agentID)
		if err != nil {
			s.logger.Errorw("failed to convert rule to sync data",
				"rule_short_id", ruleShortID,
				"error", err,
			)
			return err
		}

		if changeType == "added" {
			syncData.Added = []dto.RuleSyncData{*ruleSyncData}
		} else {
			syncData.Updated = []dto.RuleSyncData{*ruleSyncData}
		}

	case "removed":
		syncData.Removed = []string{ruleShortID}

	default:
		s.logger.Debugw("unknown change type for rule sync",
			"change_type", changeType,
		)
		return nil
	}

	// Get agent short ID for Stripe-style prefixed ID
	agent, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		s.logger.Errorw("failed to get agent for config sync",
			"agent_id", agentID,
			"error", err,
		)
		return err
	}
	if agent == nil {
		s.logger.Warnw("agent not found for config sync",
			"agent_id", agentID,
		)
		return forward.ErrAgentNotFound
	}

	// Send sync message
	msg := &dto.HubMessage{
		Type:      dto.MsgTypeConfigSync,
		AgentID:   agent.SID(),
		Timestamp: time.Now().Unix(),
		Data:      syncData,
	}

	if err := s.hub.SendMessageToAgent(agentID, msg); err != nil {
		s.logger.Errorw("failed to send config sync message",
			"agent_id", agentID,
			"version", version,
			"error", err,
		)
		return err
	}

	// Update agent version
	s.agentVersions.Store(agentID, version)

	// Aggregate log for sync details (debug level for individual rules)
	if len(syncData.Added) > 0 {
		s.logger.Debugw("config sync rules added",
			"count", len(syncData.Added),
			"first_rule_id", syncData.Added[0].ShortID,
			"first_rule_type", syncData.Added[0].RuleType,
		)
	}
	if len(syncData.Updated) > 0 {
		s.logger.Debugw("config sync rules updated",
			"count", len(syncData.Updated),
			"first_rule_id", syncData.Updated[0].ShortID,
			"first_rule_type", syncData.Updated[0].RuleType,
		)
	}

	s.logger.Infow("config sync notification sent",
		"agent_id", agentID,
		"version", version,
		"change_type", changeType,
		"rule_short_id", ruleShortID,
	)

	return nil
}

// FullSyncToAgent performs a full configuration sync to an agent (typically on reconnection).
func (s *ConfigSyncService) FullSyncToAgent(ctx context.Context, agentID uint) error {
	s.logger.Infow("performing full config sync to agent",
		"agent_id", agentID,
	)

	// Check if agent is online
	if !s.hub.IsAgentOnline(agentID) {
		s.logger.Debugw("agent offline, skipping full sync",
			"agent_id", agentID,
		)
		return nil
	}

	// Increment global version
	version := s.globalVersion.Add(1)

	// Retrieve all enabled rules for this agent
	rules, err := s.getEnabledRulesForAgent(ctx, agentID)
	if err != nil {
		s.logger.Errorw("failed to get enabled rules for full sync",
			"agent_id", agentID,
			"error", err,
		)
		return err
	}

	s.logger.Infow("fetched enabled rules for full sync",
		"agent_id", agentID,
		"rule_count", len(rules),
	)

	// Convert all rules to sync data
	ruleSyncDataList := make([]dto.RuleSyncData, 0, len(rules))
	for _, rule := range rules {
		ruleSyncData, err := s.convertRuleToSyncData(ctx, rule, agentID)
		if err != nil {
			s.logger.Warnw("failed to convert rule to sync data, skipping",
				"rule_id", rule.ID(),
				"error", err,
			)
			continue
		}
		ruleSyncDataList = append(ruleSyncDataList, *ruleSyncData)
	}

	// Get agent short ID for Stripe-style prefixed ID
	agent, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		s.logger.Errorw("failed to get agent for full config sync",
			"agent_id", agentID,
			"error", err,
		)
		return err
	}
	if agent == nil {
		s.logger.Warnw("agent not found for full config sync",
			"agent_id", agentID,
		)
		return forward.ErrAgentNotFound
	}

	// Generate client token for this agent
	clientToken, _ := s.agentTokenService.Generate(agent.SID())

	s.logger.Infow("generated client token for full sync",
		"agent_id", agentID,
		"short_id", agent.SID(),
		"client_token", clientToken,
	)

	// Build full sync data
	// Note: token_signing_secret is no longer included for security reasons.
	// Agents should use the server for token verification.
	syncData := &dto.ConfigSyncData{
		Version:     version,
		FullSync:    true,
		Added:       ruleSyncDataList,
		ClientToken: clientToken,
	}

	// Send sync message
	msg := &dto.HubMessage{
		Type:      dto.MsgTypeConfigSync,
		AgentID:   agent.SID(),
		Timestamp: time.Now().Unix(),
		Data:      syncData,
	}

	if err := s.hub.SendMessageToAgent(agentID, msg); err != nil {
		s.logger.Errorw("failed to send full config sync message",
			"agent_id", agentID,
			"version", version,
			"error", err,
		)
		return err
	}

	// Update agent version
	s.agentVersions.Store(agentID, version)

	s.logger.Infow("full config sync completed",
		"agent_id", agentID,
		"version", version,
		"rule_count", len(ruleSyncDataList),
	)

	return nil
}

// getEnabledRulesForAgent retrieves all enabled rules for a specific agent.
// This mirrors the logic in AgentHandler.GetEnabledRules.
func (s *ConfigSyncService) getEnabledRulesForAgent(ctx context.Context, agentID uint) ([]*forward.ForwardRule, error) {
	// Retrieve enabled forward rules for this agent (as entry agent)
	rules, err := s.repo.ListEnabledByAgentID(ctx, agentID)
	if err != nil {
		return nil, err
	}

	// Also retrieve entry rules where this agent is the exit agent
	exitRules, err := s.repo.ListEnabledByExitAgentID(ctx, agentID)
	if err != nil {
		return nil, err
	}

	// Also retrieve chain rules where this agent participates
	chainRules, err := s.repo.ListEnabledByChainAgentID(ctx, agentID)
	if err != nil {
		return nil, err
	}

	// Merge rules (avoid duplicates by using a map)
	ruleMap := make(map[uint]*forward.ForwardRule)
	for _, rule := range rules {
		ruleMap[rule.ID()] = rule
	}
	for _, rule := range exitRules {
		ruleMap[rule.ID()] = rule
	}
	for _, rule := range chainRules {
		ruleMap[rule.ID()] = rule
	}

	// Convert map back to slice
	allRules := make([]*forward.ForwardRule, 0, len(ruleMap))
	for _, rule := range ruleMap {
		allRules = append(allRules, rule)
	}

	return allRules, nil
}

// convertRuleToSyncData converts a ForwardRule to RuleSyncData.
// This mirrors the logic in AgentHandler.GetEnabledRules for building rule DTOs.
func (s *ConfigSyncService) convertRuleToSyncData(ctx context.Context, rule *forward.ForwardRule, agentID uint) (*dto.RuleSyncData, error) {
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
		syncData.Role = "entry"
		syncData.TargetAddress = targetAddress
		syncData.TargetPort = targetPort

	case "entry":
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
					syncData.NextHopAgentID = exitAgent.SID()
					syncData.NextHopAddress = exitAgent.GetEffectiveTunnelAddress()

					// Get tunnel ports from cached agent status
					exitStatus, err := s.statusQuerier.GetStatus(ctx, exitAgentID)
					if err != nil {
						s.logger.Warnw("failed to get exit agent status",
							"rule_id", rule.ID(),
							"exit_agent_id", exitAgentID,
							"error", err,
						)
					} else if exitStatus != nil {
						if exitStatus.WsListenPort > 0 {
							syncData.NextHopWsPort = exitStatus.WsListenPort
						}
						if exitStatus.TlsListenPort > 0 {
							syncData.NextHopTlsPort = exitStatus.TlsListenPort
						}
						if exitStatus.WsListenPort == 0 && exitStatus.TlsListenPort == 0 {
							s.logger.Debugw("exit agent has no tunnel port configured or is offline",
								"rule_id", rule.ID(),
								"exit_agent_id", exitAgentID,
							)
						}
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
					syncData.AgentID = entryAgent.SID()
				}
			}
		}

	case "chain":
		// Calculate chain position and last-in-chain flag for this agent
		chainPosition := rule.GetChainPosition(agentID)
		isLast := rule.IsLastInChain(agentID)

		syncData.ChainPosition = chainPosition
		syncData.IsLastInChain = isLast
		syncData.TunnelHops = rule.TunnelHops()

		// Determine hop mode for hybrid chain support
		hopMode := rule.GetHopMode(chainPosition)
		syncData.HopMode = hopMode

		// For boundary nodes, set inbound/outbound modes
		if hopMode == "boundary" {
			syncData.InboundMode = "tunnel"
			syncData.OutboundMode = "direct"
		}

		// Populate ChainAgentIDs (Stripe-style IDs)
		// Full chain: [entry_agent] + chain_agents (matches GetChainPosition calculation)
		fullChainIDs := append([]uint{rule.AgentID()}, rule.ChainAgentIDs()...)
		if len(fullChainIDs) > 0 {
			agentMap, err := s.agentRepo.GetSIDsByIDs(ctx, fullChainIDs)
			if err != nil {
				s.logger.Warnw("failed to get chain agent short IDs",
					"rule_id", rule.ID(),
					"error", err,
				)
			} else {
				syncData.ChainAgentIDs = make([]string, len(fullChainIDs))
				for i, chainAgentID := range fullChainIDs {
					if sid, ok := agentMap[chainAgentID]; ok {
						syncData.ChainAgentIDs[i] = sid
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
					syncData.NextHopAgentID = nextAgent.SID()
					syncData.NextHopAddress = nextAgent.GetEffectiveTunnelAddress()

					// Check if outbound uses tunnel or direct based on hop mode
					outboundNeedsTunnel := hopMode == "tunnel" || (hopMode == "boundary" && syncData.OutboundMode == "tunnel")
					if !outboundNeedsTunnel && (hopMode == "direct" || hopMode == "boundary") {
						// Direct connection mode: use chainPortConfig for next hop port
						nextHopPort := rule.GetAgentListenPort(nextHopAgentID)
						if nextHopPort > 0 {
							syncData.NextHopPort = nextHopPort
						}
						// Generate connection token for direct hop authentication
						if s.agentTokenService != nil {
							nextHopToken, _ := s.agentTokenService.Generate(nextAgent.SID())
							syncData.NextHopConnectionToken = nextHopToken
						}
					} else {
						// Tunnel mode: get tunnel ports from cached agent status
						nextStatus, err := s.statusQuerier.GetStatus(ctx, nextHopAgentID)
						if err != nil {
							s.logger.Warnw("failed to get next hop agent status",
								"rule_id", rule.ID(),
								"next_hop_agent_id", nextHopAgentID,
								"error", err,
							)
						} else if nextStatus != nil {
							if nextStatus.WsListenPort > 0 {
								syncData.NextHopWsPort = nextStatus.WsListenPort
							}
							if nextStatus.TlsListenPort > 0 {
								syncData.NextHopTlsPort = nextStatus.TlsListenPort
							}
							if nextStatus.WsListenPort == 0 && nextStatus.TlsListenPort == 0 {
								s.logger.Debugw("next hop agent has no tunnel port configured or is offline",
									"rule_id", rule.ID(),
									"next_hop_agent_id", nextHopAgentID,
								)
							}
						}
					}
				}
			}
		} else {
			// For exit agents, include target info
			syncData.TargetAddress = targetAddress
			syncData.TargetPort = targetPort
		}

		// For hybrid chain direct hops (boundary and pure direct), set listen port from chainPortConfig
		if hopMode == "boundary" || hopMode == "direct" {
			if chainPosition > 0 { // Not entry agent
				listenPort := rule.GetAgentListenPort(agentID)
				if listenPort > 0 {
					syncData.ListenPort = listenPort
				}
			}
		}

	case "direct_chain":
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
			return nil, fmt.Errorf("agent %d not found in direct_chain rule %d", agentID, rule.ID())
		}

		syncData.ChainPosition = chainPosition
		syncData.IsLastInChain = isLast

		// Populate ChainAgentIDs (Stripe-style IDs)
		// Include entry agent (rule.AgentID) + chain agents for complete chain
		entryAgentID := rule.AgentID()
		chainAgentIDs := rule.ChainAgentIDs()

		// Debug logging for chain position and role assignment
		s.logger.Debugw("direct_chain rule sync",
			"current_agent_id", agentID,
			"rule_entry_agent_id", entryAgentID,
			"chain_agent_ids", chainAgentIDs,
			"calculated_position", chainPosition,
			"is_last", isLast,
		)
		fullChainIDs := append([]uint{entryAgentID}, chainAgentIDs...)

		agentMap, err := s.agentRepo.GetSIDsByIDs(ctx, fullChainIDs)
		if err != nil {
			s.logger.Warnw("failed to get chain agent short IDs",
				"rule_id", rule.ID(),
				"error", err,
			)
		} else {
			syncData.ChainAgentIDs = make([]string, len(fullChainIDs))
			for i, chainAgentID := range fullChainIDs {
				if sid, ok := agentMap[chainAgentID]; ok {
					syncData.ChainAgentIDs[i] = sid
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
			nextHopAgentID, nextHopPort, err := rule.GetNextHopForDirectChainSafe(agentID)
			if err != nil {
				s.logger.Errorw("failed to get next hop for direct_chain rule in config sync",
					"rule_id", rule.ID(),
					"agent_id", agentID,
					"error", err,
				)
				return nil, fmt.Errorf("failed to get next hop for direct_chain rule: %w", err)
			}

			s.logger.Debugw("direct_chain next hop lookup",
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
					syncData.NextHopAgentID = nextAgent.SID()
					syncData.NextHopAddress = nextAgent.GetEffectiveTunnelAddress()
					syncData.NextHopPort = nextHopPort

					// Generate connection token for next hop authentication
					nextHopToken, _ := s.agentTokenService.Generate(nextAgent.SID())
					syncData.NextHopConnectionToken = nextHopToken

					s.logger.Debugw("direct_chain next hop token generated",
						"current_agent_id", agentID,
						"next_hop_short_id", nextAgent.SID(),
					)
				}
			}
		} else {
			// For exit agents, include target info
			syncData.TargetAddress = targetAddress
			syncData.TargetPort = targetPort
		}
	}

	return syncData, nil
}

// handleConfigAck handles config acknowledgment from agent.
func (s *ConfigSyncService) handleConfigAck(agentID uint, data any) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		s.logger.Warnw("failed to marshal config ack data",
			"agent_id", agentID,
			"error", err,
		)
		return
	}

	var ack dto.ConfigAckData
	if err := json.Unmarshal(dataBytes, &ack); err != nil {
		s.logger.Warnw("failed to parse config ack",
			"error", err,
			"agent_id", agentID,
		)
		return
	}

	if ack.Success {
		s.logger.Infow("agent acknowledged config sync",
			"agent_id", agentID,
			"version", ack.Version,
		)
	} else {
		s.logger.Warnw("agent reported config sync failure",
			"agent_id", agentID,
			"version", ack.Version,
			"error", ack.Error,
		)
	}

	// Update agent's acknowledged version
	s.agentVersions.Store(agentID, ack.Version)
}

// GetAgentVersion returns the current version for an agent.
func (s *ConfigSyncService) GetAgentVersion(agentID uint) uint64 {
	if version, ok := s.agentVersions.Load(agentID); ok {
		return version.(uint64)
	}
	return 0
}

// GetGlobalVersion returns the current global version.
func (s *ConfigSyncService) GetGlobalVersion() uint64 {
	return s.globalVersion.Load()
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

// NotifyExitPortChange notifies all entry agents that have rules pointing to this exit agent.
// This is called when an exit agent's ws_listen_port changes.
func (s *ConfigSyncService) NotifyExitPortChange(ctx context.Context, exitAgentID uint) error {
	s.logger.Infow("notifying entry agents of exit agent port change",
		"exit_agent_id", exitAgentID,
	)

	// Find all entry rules where this agent is the exit agent
	entryRules, err := s.repo.ListEnabledByExitAgentID(ctx, exitAgentID)
	if err != nil {
		s.logger.Errorw("failed to list entry rules for exit agent",
			"exit_agent_id", exitAgentID,
			"error", err,
		)
		return err
	}

	// Collect unique entry agent IDs from entry rules
	entryAgentIDs := make(map[uint]bool)
	for _, rule := range entryRules {
		if rule.RuleType().String() == "entry" {
			entryAgentIDs[rule.AgentID()] = true
		}
	}

	// Also check chain rules where this agent participates
	chainRules, err := s.repo.ListEnabledByChainAgentID(ctx, exitAgentID)
	if err != nil {
		s.logger.Warnw("failed to list chain rules for exit agent",
			"exit_agent_id", exitAgentID,
			"error", err,
		)
		// Continue with entry rules
	} else {
		for _, rule := range chainRules {
			ruleType := rule.RuleType().String()
			if ruleType == "chain" || ruleType == "direct_chain" {
				// Find agents that have this exit agent as their next hop
				// Full chain: [entry_agent] + chainAgentIDs
				fullChain := append([]uint{rule.AgentID()}, rule.ChainAgentIDs()...)
				for i, agentID := range fullChain {
					// Check if exitAgentID is the next hop for this agent
					if i+1 < len(fullChain) && fullChain[i+1] == exitAgentID {
						entryAgentIDs[agentID] = true
					}
				}
			}
		}
	}

	if len(entryAgentIDs) == 0 {
		s.logger.Debugw("no agents to notify for exit agent address/port change",
			"exit_agent_id", exitAgentID,
		)
		return nil
	}

	s.logger.Infow("found entry agents to notify",
		"exit_agent_id", exitAgentID,
		"entry_agent_count", len(entryAgentIDs),
	)

	// Notify each entry agent with updated rules
	var lastErr error
	for entryAgentID := range entryAgentIDs {
		if !s.hub.IsAgentOnline(entryAgentID) {
			s.logger.Debugw("entry agent offline, skipping port change notification",
				"entry_agent_id", entryAgentID,
				"exit_agent_id", exitAgentID,
			)
			continue
		}

		// Get rules for this entry agent that point to the exit agent
		agentRules, err := s.getEntryRulesForExitAgent(ctx, entryAgentID, exitAgentID)
		if err != nil {
			s.logger.Warnw("failed to get rules for entry agent",
				"entry_agent_id", entryAgentID,
				"exit_agent_id", exitAgentID,
				"error", err,
			)
			lastErr = err
			continue
		}

		if len(agentRules) == 0 {
			continue
		}

		// Convert rules to sync data
		ruleSyncDataList := make([]dto.RuleSyncData, 0, len(agentRules))
		for _, rule := range agentRules {
			syncData, err := s.convertRuleToSyncData(ctx, rule, entryAgentID)
			if err != nil {
				s.logger.Warnw("failed to convert rule to sync data",
					"rule_id", rule.ID(),
					"error", err,
				)
				continue
			}
			ruleSyncDataList = append(ruleSyncDataList, *syncData)
		}

		if len(ruleSyncDataList) == 0 {
			continue
		}

		// Increment global version
		version := s.globalVersion.Add(1)

		// Build sync data (as updated rules)
		syncData := &dto.ConfigSyncData{
			Version:  version,
			FullSync: false,
			Updated:  ruleSyncDataList,
		}

		// Get entry agent short ID
		entryAgent, err := s.agentRepo.GetByID(ctx, entryAgentID)
		if err != nil || entryAgent == nil {
			s.logger.Warnw("failed to get entry agent",
				"entry_agent_id", entryAgentID,
				"error", err,
			)
			continue
		}

		// Send sync message
		msg := &dto.HubMessage{
			Type:      dto.MsgTypeConfigSync,
			AgentID:   entryAgent.SID(),
			Timestamp: time.Now().Unix(),
			Data:      syncData,
		}

		if err := s.hub.SendMessageToAgent(entryAgentID, msg); err != nil {
			s.logger.Infow("port change notification skipped",
				"entry_agent_id", entryAgentID,
				"exit_agent_id", exitAgentID,
				"reason", err.Error(),
			)
			lastErr = err
			continue
		}

		// Update agent version
		s.agentVersions.Store(entryAgentID, version)

		s.logger.Infow("port change notification sent to entry agent",
			"entry_agent_id", entryAgentID,
			"exit_agent_id", exitAgentID,
			"version", version,
			"rule_count", len(ruleSyncDataList),
		)
	}

	return lastErr
}

// NotifyAgentAddressChange notifies all entry agents that have rules using this agent.
// This is called when an agent's public_address or tunnel_address changes.
// The logic is similar to NotifyExitPortChange as both require re-syncing rules
// to update the tunnel/next-hop address information.
func (s *ConfigSyncService) NotifyAgentAddressChange(ctx context.Context, agentID uint) error {
	s.logger.Infow("notifying entry agents of agent address change",
		"agent_id", agentID,
	)

	// Reuse the same logic as NotifyExitPortChange since address changes
	// also require updating next-hop information in entry/relay agents
	return s.NotifyExitPortChange(ctx, agentID)
}

// getEntryRulesForExitAgent retrieves enabled rules for an agent that has exitAgentID as its next hop.
// The agentID parameter is the agent that needs to be notified (not necessarily the entry agent).
func (s *ConfigSyncService) getEntryRulesForExitAgent(ctx context.Context, agentID, exitAgentID uint) ([]*forward.ForwardRule, error) {
	// Get all enabled rules where this agent is the entry agent
	rules, err := s.repo.ListEnabledByAgentID(ctx, agentID)
	if err != nil {
		return nil, err
	}

	// Also get chain rules where this agent participates (for chain middle nodes)
	chainRules, err := s.repo.ListEnabledByChainAgentID(ctx, agentID)
	if err != nil {
		s.logger.Warnw("failed to list chain rules for agent",
			"agent_id", agentID,
			"error", err,
		)
		// Continue with entry rules
	} else {
		rules = append(rules, chainRules...)
	}

	// Deduplicate rules by ID
	ruleMap := make(map[uint]*forward.ForwardRule)
	for _, rule := range rules {
		ruleMap[rule.ID()] = rule
	}

	// Filter rules that have exitAgentID as next hop for this agent
	result := make([]*forward.ForwardRule, 0)
	for _, rule := range ruleMap {
		switch rule.RuleType().String() {
		case "entry":
			if rule.ExitAgentID() == exitAgentID {
				result = append(result, rule)
			}
		case "chain":
			// Check if this agent has exitAgentID as its next hop
			nextHop := rule.GetNextHopAgentID(agentID)
			if nextHop == exitAgentID {
				result = append(result, rule)
			}
		case "direct_chain":
			// Check if this agent has exitAgentID as its next hop in direct_chain
			nextHop, _, err := rule.GetNextHopForDirectChainSafe(agentID)
			if err != nil {
				s.logger.Warnw("failed to get next hop for direct_chain in exit endpoint change notification",
					"rule_id", rule.ID(),
					"agent_id", agentID,
					"error", err,
				)
				continue
			}
			if nextHop == exitAgentID {
				result = append(result, rule)
			}
		}
	}

	return result, nil
}
