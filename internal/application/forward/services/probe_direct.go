package services

import (
	"context"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	vo "github.com/orris-inc/orris/internal/domain/forward/valueobjects"
	"github.com/orris-inc/orris/internal/shared/id"
)

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
