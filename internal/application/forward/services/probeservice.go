// Package services provides application services for the forward domain.
package services

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	vo "github.com/orris-inc/orris/internal/domain/forward/value_objects"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
)

const (
	probeTimeout        = 10 * time.Second
	probeSessionTimeout = 30 * time.Second
)

// ProbeService handles probe operations for forward rules.
// It implements agent.MessageHandler interface.
type ProbeService struct {
	repo      forward.Repository
	agentRepo forward.AgentRepository
	nodeRepo  node.NodeRepository

	// Hub interface for sending messages
	hub ProbeHub

	// Pending probes: map[taskID]chan *dto.ProbeTaskResult
	pendingProbes   map[string]chan *dto.ProbeTaskResult
	pendingProbesMu sync.RWMutex

	logger logger.Interface
}

// ProbeHub defines the interface for sending messages through the hub.
type ProbeHub interface {
	IsAgentOnline(agentID uint) bool
	SendMessageToAgent(agentID uint, msg *dto.HubMessage) error
}

// NewProbeService creates a new ProbeService.
func NewProbeService(
	repo forward.Repository,
	agentRepo forward.AgentRepository,
	nodeRepo node.NodeRepository,
	hub ProbeHub,
	log logger.Interface,
) *ProbeService {
	return &ProbeService{
		repo:          repo,
		agentRepo:     agentRepo,
		nodeRepo:      nodeRepo,
		hub:           hub,
		pendingProbes: make(map[string]chan *dto.ProbeTaskResult),
		logger:        log,
	}
}

// HandleMessage processes probe-related messages from agents.
// Implements agent.MessageHandler interface.
func (s *ProbeService) HandleMessage(agentID uint, msgType string, data any) bool {
	switch msgType {
	case dto.MsgTypeProbeResult:
		s.handleProbeResult(agentID, data)
		return true
	default:
		return false
	}
}

// ProbeRuleByShortID probes a single forward rule by short ID and returns the latency results.
// ipVersionOverride allows overriding the rule's IP version for this probe only.
func (s *ProbeService) ProbeRuleByShortID(ctx context.Context, shortID string, ipVersionOverride string) (*dto.RuleProbeResponse, error) {
	// Get the rule
	rule, err := s.repo.GetByShortID(ctx, shortID)
	if err != nil {
		return nil, err
	}
	if rule == nil {
		return nil, forward.ErrRuleNotFound
	}
	return s.probeRule(ctx, rule, ipVersionOverride)
}

// ProbeRule probes a single forward rule and returns the latency results.
// ipVersionOverride allows overriding the rule's IP version for this probe only.
// Deprecated: Use ProbeRuleByShortID instead for external API.
func (s *ProbeService) ProbeRule(ctx context.Context, ruleID uint, ipVersionOverride string) (*dto.RuleProbeResponse, error) {
	// Get the rule
	rule, err := s.repo.GetByID(ctx, ruleID)
	if err != nil {
		return nil, err
	}
	if rule == nil {
		return nil, forward.ErrRuleNotFound
	}
	return s.probeRule(ctx, rule, ipVersionOverride)
}

// probeRule is the internal implementation for probing a rule.
func (s *ProbeService) probeRule(ctx context.Context, rule *forward.ForwardRule, ipVersionOverride string) (*dto.RuleProbeResponse, error) {

	// Determine IP version to use (override or rule's default)
	ipVersion := rule.IPVersion()
	if ipVersionOverride != "" {
		ipVersion = vo.IPVersion(ipVersionOverride)
		if !ipVersion.IsValid() {
			return nil, forward.ErrInvalidIPVersion
		}
	}

	ruleType := rule.RuleType().String()
	response := &dto.RuleProbeResponse{
		RuleID:   id.FormatForwardRuleID(rule.ShortID()),
		RuleType: ruleType,
	}

	switch ruleType {
	case "direct":
		return s.probeDirectRule(ctx, rule, ipVersion, response)
	case "entry":
		return s.probeEntryRule(ctx, rule, response)
	case "exit":
		// Exit rules are probed through entry rules
		response.Error = "exit rules cannot be probed directly"
		return response, nil
	default:
		response.Error = "unknown rule type"
		return response, nil
	}
}

// probeDirectRule probes a direct rule (agent → target).
func (s *ProbeService) probeDirectRule(ctx context.Context, rule *forward.ForwardRule, ipVersion vo.IPVersion, response *dto.RuleProbeResponse) (*dto.RuleProbeResponse, error) {
	agentID := rule.AgentID()

	// Resolve target address and port
	targetAddress := rule.TargetAddress()
	targetPort := rule.TargetPort()

	// If rule has target node, get address from node
	if rule.HasTargetNode() {
		targetNode, err := s.nodeRepo.GetByID(ctx, *rule.TargetNodeID())
		if err != nil {
			response.Error = "failed to get target node: " + err.Error()
			return response, nil
		}
		if targetNode == nil {
			response.Error = "target node not found"
			return response, nil
		}
		// Resolve target address based on IP version preference
		targetAddress = s.resolveNodeAddress(targetNode, ipVersion)
		if targetAddress == "" {
			response.Error = "target node has no available address for ip_version: " + ipVersion.String()
			return response, nil
		}
		// Use node's agent port if rule's target port is not set
		if targetPort == 0 {
			targetPort = targetNode.AgentPort()
		}
	}

	s.logger.Infow("probing direct rule",
		"rule_id", rule.ID(),
		"agent_id", agentID,
		"target", targetAddress,
		"port", targetPort,
		"has_target_node", rule.HasTargetNode(),
	)

	// Check if agent is online
	if !s.hub.IsAgentOnline(agentID) {
		if rule.HasTargetNode() {
			s.logger.Warnw("agent not connected for probe",
				"rule_id", rule.ID(),
				"agent_id", agentID,
				"target_node_id", *rule.TargetNodeID(),
				"target", targetAddress,
				"port", targetPort,
			)
		} else {
			s.logger.Warnw("agent not connected for probe",
				"rule_id", rule.ID(),
				"agent_id", agentID,
				"target", targetAddress,
				"port", targetPort,
			)
		}
		response.Error = "agent not connected"
		return response, nil
	}

	// Probe target using TCP for reliable connectivity check
	ruleStripeID := id.FormatForwardRuleID(rule.ShortID())
	targetLatency, err := s.sendProbeTask(ctx, agentID, ruleStripeID, dto.ProbeTaskTypeTarget,
		targetAddress, targetPort, "tcp")
	if err != nil {
		response.Error = err.Error()
		return response, nil
	}

	response.Success = true
	response.TargetLatencyMs = &targetLatency
	response.TotalLatencyMs = &targetLatency
	return response, nil
}

// probeEntryRule probes an entry rule (entry → exit → target).
func (s *ProbeService) probeEntryRule(ctx context.Context, rule *forward.ForwardRule, response *dto.RuleProbeResponse) (*dto.RuleProbeResponse, error) {
	entryAgentID := rule.AgentID()
	exitAgentID := rule.ExitAgentID()

	s.logger.Infow("probing entry rule",
		"rule_id", rule.ID(),
		"entry_agent_id", entryAgentID,
		"exit_agent_id", exitAgentID,
	)

	// Check if entry agent is online
	if !s.hub.IsAgentOnline(entryAgentID) {
		s.logger.Warnw("entry agent not connected for probe",
			"rule_id", rule.ID(),
			"entry_agent_id", entryAgentID,
		)
		response.Error = "entry agent not connected"
		return response, nil
	}

	// Get exit agent info
	exitAgent, err := s.agentRepo.GetByID(ctx, exitAgentID)
	if err != nil || exitAgent == nil {
		response.Error = "exit agent not found"
		return response, nil
	}

	// Get exit rule for WS port
	exitRule, err := s.repo.GetExitRuleByAgentID(ctx, exitAgentID)
	if err != nil || exitRule == nil {
		response.Error = "exit rule not found"
		return response, nil
	}

	// Step 1: Probe tunnel (entry → exit)
	ruleStripeID := id.FormatForwardRuleID(rule.ShortID())
	tunnelLatency, err := s.sendProbeTask(ctx, entryAgentID, ruleStripeID, dto.ProbeTaskTypeTunnel,
		exitAgent.PublicAddress(), exitRule.WsListenPort(), "tcp")
	if err != nil {
		response.Error = "tunnel probe failed: " + err.Error()
		return response, nil
	}
	response.TunnelLatencyMs = &tunnelLatency

	// Step 2: Probe target from exit agent (exit → target)
	if !s.hub.IsAgentOnline(exitAgentID) {
		response.Error = "exit agent not connected, cannot probe target"
		response.TunnelLatencyMs = &tunnelLatency
		return response, nil
	}

	// Probe target using TCP for reliable connectivity check
	targetLatency, err := s.sendProbeTask(ctx, exitAgentID, ruleStripeID, dto.ProbeTaskTypeTarget,
		exitRule.TargetAddress(), exitRule.TargetPort(), "tcp")
	if err != nil {
		response.Error = "target probe failed: " + err.Error()
		return response, nil
	}

	response.Success = true
	response.TargetLatencyMs = &targetLatency
	totalLatency := tunnelLatency + targetLatency
	response.TotalLatencyMs = &totalLatency
	return response, nil
}

// sendProbeTask sends a probe task to an agent and waits for the result.
func (s *ProbeService) sendProbeTask(
	ctx context.Context,
	agentID uint,
	ruleID string, // Stripe-style prefixed ID
	taskType dto.ProbeTaskType,
	target string,
	port uint16,
	protocol string,
) (int64, error) {
	taskID := uuid.New().String()

	// Create result channel
	resultChan := make(chan *dto.ProbeTaskResult, 1)
	s.pendingProbesMu.Lock()
	s.pendingProbes[taskID] = resultChan
	s.pendingProbesMu.Unlock()

	defer func() {
		s.pendingProbesMu.Lock()
		delete(s.pendingProbes, taskID)
		s.pendingProbesMu.Unlock()
	}()

	// Get agent short ID for Stripe-style prefixed ID
	agent, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		return 0, err
	}
	if agent == nil {
		return 0, forward.ErrAgentNotFound
	}

	// Send probe task
	task := &dto.ProbeTask{
		ID:       taskID,
		Type:     taskType,
		RuleID:   ruleID,
		Target:   target,
		Port:     port,
		Protocol: protocol,
		Timeout:  int(probeTimeout.Milliseconds()),
	}

	msg := &dto.HubMessage{
		Type:      dto.MsgTypeProbeTask,
		AgentID:   id.FormatForwardAgentID(agent.ShortID()),
		Timestamp: time.Now().Unix(),
		Data:      task,
	}

	if err := s.hub.SendMessageToAgent(agentID, msg); err != nil {
		return 0, err
	}

	// Wait for result with timeout
	select {
	case result := <-resultChan:
		if !result.Success {
			return 0, &probeError{message: result.Error}
		}
		return result.LatencyMs, nil
	case <-time.After(probeTimeout):
		return 0, &probeError{message: "probe timeout"}
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

// handleProbeResult handles probe result from agent.
func (s *ProbeService) handleProbeResult(agentID uint, data any) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return
	}

	var result dto.ProbeTaskResult
	if err := json.Unmarshal(dataBytes, &result); err != nil {
		s.logger.Warnw("failed to parse probe result",
			"error", err,
			"agent_id", agentID,
		)
		return
	}

	s.pendingProbesMu.RLock()
	resultChan, ok := s.pendingProbes[result.TaskID]
	s.pendingProbesMu.RUnlock()

	if ok {
		select {
		case resultChan <- &result:
		default:
			// Channel full or closed, ignore
		}
	} else {
		s.logger.Warnw("received probe result for unknown task",
			"task_id", result.TaskID,
			"agent_id", agentID,
		)
	}
}

// probeError represents a probe error.
type probeError struct {
	message string
}

func (e *probeError) Error() string {
	return e.message
}

// resolveNodeAddress resolves the node address based on IP version preference.
// Priority: ServerAddress > PublicIP (based on ipVersion)
func (s *ProbeService) resolveNodeAddress(n *node.Node, ipVersion vo.IPVersion) string {
	// If server address is set, use it directly
	serverAddr := n.ServerAddress().Value()
	if serverAddr != "" {
		return serverAddr
	}

	// Fallback to public IP based on IP version preference
	switch ipVersion {
	case vo.IPVersionIPv4:
		if n.PublicIPv4() != nil && *n.PublicIPv4() != "" {
			return *n.PublicIPv4()
		}
	case vo.IPVersionIPv6:
		if n.PublicIPv6() != nil && *n.PublicIPv6() != "" {
			return *n.PublicIPv6()
		}
	default: // auto: prefer IPv4, fallback to IPv6
		if n.PublicIPv4() != nil && *n.PublicIPv4() != "" {
			return *n.PublicIPv4()
		}
		if n.PublicIPv6() != nil && *n.PublicIPv6() != "" {
			return *n.PublicIPv6()
		}
	}

	return ""
}
