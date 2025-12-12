package services

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/application/forward/usecases"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/infrastructure/auth"
	"github.com/orris-inc/orris/internal/shared/id"
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
		rule, err := s.repo.GetByShortID(ctx, ruleShortID)
		if err != nil {
			s.logger.Errorw("failed to get rule for sync",
				"rule_short_id", ruleShortID,
				"error", err,
			)
			return err
		}
		if rule == nil {
			s.logger.Warnw("rule not found for sync",
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
		syncData.Removed = []string{id.FormatForwardRuleID(ruleShortID)}

	default:
		s.logger.Warnw("unknown change type for rule sync",
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
		AgentID:   id.FormatForwardAgentID(agent.ShortID()),
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

	// Debug log: print sync data details
	if len(syncData.Added) > 0 {
		for _, rule := range syncData.Added {
			s.logger.Infow("config sync rule details (added)",
				"short_id", rule.ShortID,
				"rule_type", rule.RuleType,
				"role", rule.Role,
				"next_hop_agent_id", rule.NextHopAgentID,
				"next_hop_address", rule.NextHopAddress,
				"next_hop_ws_port", rule.NextHopWsPort,
			)
		}
	}
	if len(syncData.Updated) > 0 {
		for _, rule := range syncData.Updated {
			s.logger.Infow("config sync rule details (updated)",
				"short_id", rule.ShortID,
				"rule_type", rule.RuleType,
				"role", rule.Role,
				"next_hop_agent_id", rule.NextHopAgentID,
				"next_hop_address", rule.NextHopAddress,
				"next_hop_ws_port", rule.NextHopWsPort,
			)
		}
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
	clientToken, _ := s.agentTokenService.Generate(agent.ShortID())

	s.logger.Infow("generated client token for full sync",
		"agent_id", agentID,
		"short_id", agent.ShortID(),
		"client_token", clientToken,
	)

	// Build full sync data
	syncData := &dto.ConfigSyncData{
		Version:            version,
		FullSync:           true,
		Added:              ruleSyncDataList,
		ClientToken:        clientToken,
		TokenSigningSecret: s.tokenSigningSecret,
	}

	// Send sync message
	msg := &dto.HubMessage{
		Type:      dto.MsgTypeConfigSync,
		AgentID:   id.FormatForwardAgentID(agent.ShortID()),
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

	if len(entryRules) == 0 {
		s.logger.Debugw("no entry rules found for exit agent",
			"exit_agent_id", exitAgentID,
		)
		return nil
	}

	// Collect unique entry agent IDs
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
			AgentID:   id.FormatForwardAgentID(entryAgent.ShortID()),
			Timestamp: time.Now().Unix(),
			Data:      syncData,
		}

		if err := s.hub.SendMessageToAgent(entryAgentID, msg); err != nil {
			s.logger.Warnw("failed to send port change notification to entry agent",
				"entry_agent_id", entryAgentID,
				"exit_agent_id", exitAgentID,
				"error", err,
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

// getEntryRulesForExitAgent retrieves enabled rules for an entry agent that point to a specific exit agent.
func (s *ConfigSyncService) getEntryRulesForExitAgent(ctx context.Context, entryAgentID, exitAgentID uint) ([]*forward.ForwardRule, error) {
	// Get all enabled rules for this entry agent
	rules, err := s.repo.ListEnabledByAgentID(ctx, entryAgentID)
	if err != nil {
		return nil, err
	}

	// Filter rules that point to the exit agent
	result := make([]*forward.ForwardRule, 0)
	for _, rule := range rules {
		switch rule.RuleType().String() {
		case "entry":
			if rule.ExitAgentID() == exitAgentID {
				result = append(result, rule)
			}
		case "chain":
			// Check if this entry agent has exitAgentID as its next hop
			nextHop := rule.GetNextHopAgentID(entryAgentID)
			if nextHop == exitAgentID {
				result = append(result, rule)
			}
		case "direct_chain":
			// Check if this agent has exitAgentID as its next hop in direct_chain
			nextHop, _ := rule.GetNextHopForDirectChain(entryAgentID)
			if nextHop == exitAgentID {
				result = append(result, rule)
			}
		}
	}

	return result, nil
}
