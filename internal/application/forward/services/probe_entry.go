package services

import (
	"context"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	vo "github.com/orris-inc/orris/internal/domain/forward/valueobjects"
	"github.com/orris-inc/orris/internal/shared/id"
)

// probeEntryRule probes an entry rule (entry → exit → target).
func (s *ProbeService) probeEntryRule(ctx context.Context, rule *forward.ForwardRule, ipVersion vo.IPVersion, response *dto.RuleProbeResponse) (*dto.RuleProbeResponse, error) {
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

	// Get WS port from exit agent status cache
	exitStatus, err := s.statusQuerier.GetStatus(ctx, exitAgentID)
	if err != nil || exitStatus == nil || exitStatus.WsListenPort == 0 {
		response.Error = "exit agent status not found or ws_listen_port not configured"
		return response, nil
	}

	// Step 1: Probe tunnel (entry → exit)
	// Use GetEffectiveTunnelAddress for tunnel connections (prefers tunnel_address over public_address)
	tunnelAddr := exitAgent.GetEffectiveTunnelAddress()
	if tunnelAddr == "" {
		response.Error = "exit agent has no tunnel address"
		return response, nil
	}

	ruleStripeID := id.FormatForwardRuleID(rule.ShortID())
	tunnelLatency, err := s.sendProbeTask(ctx, entryAgentID, ruleStripeID, dto.ProbeTaskTypeTunnel,
		tunnelAddr, exitStatus.WsListenPort, "tcp")
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

	// Resolve target address and port
	targetAddress := rule.TargetAddress()
	targetPort := rule.TargetPort()

	// If rule has target node, get address from node
	if rule.HasTargetNode() {
		targetNode, err := s.nodeRepo.GetByID(ctx, *rule.TargetNodeID())
		if err != nil {
			response.Error = "failed to get target node: " + err.Error()
			response.TunnelLatencyMs = &tunnelLatency
			return response, nil
		}
		if targetNode == nil {
			response.Error = "target node not found"
			response.TunnelLatencyMs = &tunnelLatency
			return response, nil
		}
		// Resolve target address based on IP version preference
		targetAddress = s.resolveNodeAddress(targetNode, ipVersion)
		if targetAddress == "" {
			response.Error = "target node has no available address for ip_version: " + ipVersion.String()
			response.TunnelLatencyMs = &tunnelLatency
			return response, nil
		}
		// Use node's agent port if rule's target port is not set
		if targetPort == 0 {
			targetPort = targetNode.AgentPort()
		}
	}

	// Probe target using TCP for reliable connectivity check
	targetLatency, err := s.sendProbeTask(ctx, exitAgentID, ruleStripeID, dto.ProbeTaskTypeTarget,
		targetAddress, targetPort, "tcp")
	if err != nil {
		response.Error = "target probe failed: " + err.Error()
		response.TunnelLatencyMs = &tunnelLatency
		return response, nil
	}

	response.Success = true
	response.TargetLatencyMs = &targetLatency
	totalLatency := tunnelLatency + targetLatency
	response.TotalLatencyMs = &totalLatency
	return response, nil
}
