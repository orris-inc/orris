package services

import (
	"context"
	"encoding/json"

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
	converter *RuleSyncConverter
	notifier  *SyncNotifier
	finder    *AffectedAgentsFinder
	repo      forward.Repository
	agentRepo forward.AgentRepository
	hub       SyncHub
	logger    logger.Interface
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
	agentTokenService := auth.NewAgentTokenService(tokenSigningSecret)

	converter := NewRuleSyncConverter(
		agentRepo,
		nodeRepo,
		statusQuerier,
		agentTokenService,
		log,
	)

	notifier := NewSyncNotifier(
		hub,
		agentRepo,
		log,
	)

	finder := NewAffectedAgentsFinder(
		repo,
		agentRepo,
		log,
	)

	return &ConfigSyncService{
		converter: converter,
		notifier:  notifier,
		finder:    finder,
		repo:      repo,
		agentRepo: agentRepo,
		hub:       hub,
		logger:    log,
	}
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
	if !s.notifier.IsAgentOnline(agentID) {
		s.logger.Debugw("agent offline, skipping incremental sync notification",
			"agent_id", agentID,
			"rule_short_id", ruleShortID,
		)
		return nil
	}

	// Increment global version
	version := s.notifier.IncrementVersion()

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
		ruleSyncData, err := s.converter.Convert(ctx, rule, agentID)
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

	// Send sync message to agent
	if err := s.notifier.SendToAgent(ctx, agentID, syncData); err != nil {
		return err
	}

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
	s.logger.Debugw("performing full config sync to agent",
		"agent_id", agentID,
	)

	// Check if agent is online
	if !s.notifier.IsAgentOnline(agentID) {
		s.logger.Debugw("agent offline, skipping full sync",
			"agent_id", agentID,
		)
		return nil
	}

	// Increment global version
	version := s.notifier.IncrementVersion()

	// Retrieve all enabled rules for this agent
	rules, err := s.getEnabledRulesForAgent(ctx, agentID)
	if err != nil {
		s.logger.Errorw("failed to get enabled rules for full sync",
			"agent_id", agentID,
			"error", err,
		)
		return err
	}

	s.logger.Debugw("fetched enabled rules for full sync",
		"agent_id", agentID,
		"rule_count", len(rules),
	)

	// Convert all rules to sync data
	ruleSyncDataList := make([]dto.RuleSyncData, 0, len(rules))
	for _, rule := range rules {
		ruleSyncData, err := s.converter.Convert(ctx, rule, agentID)
		if err != nil {
			s.logger.Warnw("failed to convert rule to sync data, skipping",
				"rule_id", rule.ID(),
				"error", err,
			)
			continue
		}
		ruleSyncDataList = append(ruleSyncDataList, *ruleSyncData)
	}

	// Get agent short ID and generate client token
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
	clientToken := s.converter.GenerateClientToken(agent.SID())

	s.logger.Debugw("generated client token for full sync",
		"agent_id", agentID,
		"short_id", agent.SID(),
	)

	// Build full sync data
	// Note: token_signing_secret is no longer included for security reasons.
	// Agents should use the server for token verification.
	syncData := &dto.ConfigSyncData{
		Version:          version,
		FullSync:         true,
		Added:            ruleSyncDataList,
		ClientToken:      clientToken,
		BlockedProtocols: agent.BlockedProtocols().ToStringSlice(),
	}

	// Send sync message to agent
	if err := s.notifier.SendToAgent(ctx, agentID, syncData); err != nil {
		return err
	}

	s.logger.Infow("full config sync completed",
		"agent_id", agentID,
		"version", version,
		"rule_count", len(ruleSyncDataList),
		"blocked_protocols", agent.BlockedProtocols().ToStringSlice(),
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
	s.notifier.UpdateAgentVersion(agentID, ack.Version)
}

// GetAgentVersion returns the current version for an agent.
func (s *ConfigSyncService) GetAgentVersion(agentID uint) uint64 {
	return s.notifier.GetAgentVersion(agentID)
}

// GetGlobalVersion returns the current global version.
func (s *ConfigSyncService) GetGlobalVersion() uint64 {
	return s.notifier.GetGlobalVersion()
}

// NotifyExitPortChange notifies all entry agents that have rules pointing to this exit agent.
// This is called when an exit agent's ws_listen_port changes.
func (s *ConfigSyncService) NotifyExitPortChange(ctx context.Context, exitAgentID uint) error {
	s.logger.Infow("notifying entry agents of exit agent port change",
		"exit_agent_id", exitAgentID,
	)

	// Find all entry agents affected by this exit agent's port change
	entryAgentIDs, err := s.finder.FindByAgentPortChange(ctx, exitAgentID)
	if err != nil {
		return err
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
		if !s.notifier.IsAgentOnline(entryAgentID) {
			s.logger.Debugw("entry agent offline, skipping port change notification",
				"entry_agent_id", entryAgentID,
				"exit_agent_id", exitAgentID,
			)
			continue
		}

		// Get rules for this entry agent that point to the exit agent
		agentRules, err := s.finder.GetEntryRulesForExitAgent(ctx, entryAgentID, exitAgentID)
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
			syncData, err := s.converter.Convert(ctx, rule, entryAgentID)
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
		version := s.notifier.IncrementVersion()

		// Build sync data (as updated rules)
		syncData := &dto.ConfigSyncData{
			Version:  version,
			FullSync: false,
			Updated:  ruleSyncDataList,
		}

		// Send sync message to agent
		if err := s.notifier.SendToAgent(ctx, entryAgentID, syncData); err != nil {
			s.logger.Infow("port change notification skipped",
				"entry_agent_id", entryAgentID,
				"exit_agent_id", exitAgentID,
				"reason", err.Error(),
			)
			lastErr = err
			continue
		}

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

// NotifyNodeAddressChange notifies all forward agents that have rules targeting this node.
// This is called when a node's public_ipv4 or public_ipv6 changes.
func (s *ConfigSyncService) NotifyNodeAddressChange(ctx context.Context, nodeID uint) error {
	s.logger.Infow("notifying agents of node address change",
		"node_id", nodeID,
	)

	// Find all agents affected by this node address change
	agentRulesMap, err := s.finder.FindByNodeChange(ctx, nodeID)
	if err != nil {
		return err
	}

	if len(agentRulesMap) == 0 {
		s.logger.Debugw("no rules targeting node, skipping notification",
			"node_id", nodeID,
		)
		return nil
	}

	s.logger.Infow("found rules targeting node",
		"node_id", nodeID,
		"agent_count", len(agentRulesMap),
	)

	// Notify each agent with updated rules
	var lastErr error
	for agentID, agentRules := range agentRulesMap {
		if !s.notifier.IsAgentOnline(agentID) {
			s.logger.Debugw("agent offline, skipping node address change notification",
				"agent_id", agentID,
				"node_id", nodeID,
			)
			continue
		}

		// Deduplicate rules by ID
		ruleMap := make(map[uint]*forward.ForwardRule)
		for _, rule := range agentRules {
			ruleMap[rule.ID()] = rule
		}

		// Convert rules to sync data
		ruleSyncDataList := make([]dto.RuleSyncData, 0, len(ruleMap))
		for _, rule := range ruleMap {
			syncData, err := s.converter.Convert(ctx, rule, agentID)
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
		version := s.notifier.IncrementVersion()

		// Build sync data (as updated rules)
		syncData := &dto.ConfigSyncData{
			Version:  version,
			FullSync: false,
			Updated:  ruleSyncDataList,
		}

		// Send sync message to agent
		if err := s.notifier.SendToAgent(ctx, agentID, syncData); err != nil {
			s.logger.Infow("node address change notification skipped",
				"agent_id", agentID,
				"node_id", nodeID,
				"reason", err.Error(),
			)
			lastErr = err
			continue
		}

		s.logger.Infow("node address change notification sent to agent",
			"agent_id", agentID,
			"node_id", nodeID,
			"version", version,
			"rule_count", len(ruleSyncDataList),
		)
	}

	return lastErr
}

// NotifyAgentBlockedProtocolsChange notifies an agent when its blocked protocols configuration changes.
// This sends an incremental sync with only the updated blocked protocols list.
func (s *ConfigSyncService) NotifyAgentBlockedProtocolsChange(ctx context.Context, agentID uint) error {
	s.logger.Infow("notifying agent of blocked protocols change",
		"agent_id", agentID,
	)

	// Check if agent is online
	if !s.notifier.IsAgentOnline(agentID) {
		s.logger.Debugw("agent offline, skipping blocked protocols sync",
			"agent_id", agentID,
		)
		return nil
	}

	// Get agent to retrieve current blocked protocols
	agent, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		s.logger.Errorw("failed to get agent for blocked protocols sync",
			"agent_id", agentID,
			"error", err,
		)
		return err
	}
	if agent == nil {
		s.logger.Warnw("agent not found for blocked protocols sync",
			"agent_id", agentID,
		)
		return forward.ErrAgentNotFound
	}

	// Increment global version
	version := s.notifier.IncrementVersion()

	// Build incremental sync data with only blocked protocols
	syncData := &dto.ConfigSyncData{
		Version:          version,
		FullSync:         false,
		BlockedProtocols: agent.BlockedProtocols().ToStringSlice(),
	}

	// Send sync message to agent
	if err := s.notifier.SendToAgent(ctx, agentID, syncData); err != nil {
		return err
	}

	s.logger.Infow("blocked protocols sync notification sent",
		"agent_id", agentID,
		"version", version,
		"blocked_protocols", agent.BlockedProtocols().ToStringSlice(),
	)

	return nil
}
