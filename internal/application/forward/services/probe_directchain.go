package services

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	vo "github.com/orris-inc/orris/internal/domain/forward/valueobjects"
	"github.com/orris-inc/orris/internal/shared/id"
)

// probeDirectChainRule probes a direct_chain rule (entry → agent1 → agent2 → ... → target).
// Direct chain type uses direct TCP/UDP connections between agents.
func (s *ProbeService) probeDirectChainRule(ctx context.Context, rule *forward.ForwardRule, ipVersion vo.IPVersion, response *dto.RuleProbeResponse) (*dto.RuleProbeResponse, error) {
	// Build full chain: agentID (entry) -> chainAgentIDs[0] -> chainAgentIDs[1] -> ... -> target
	fullChain := append([]uint{rule.AgentID()}, rule.ChainAgentIDs()...)

	s.logger.Infow("probing direct_chain rule",
		"rule_id", rule.ID(),
		"chain_length", len(fullChain),
		"chain_agent_ids", fullChain,
	)

	ruleStripeID := id.FormatForwardRuleID(rule.ShortID())
	chainLatencies := make([]*dto.ChainHopLatency, 0, len(fullChain))
	var totalLatency int64
	allSuccess := true

	// Probe each hop in the chain
	for i := 0; i < len(fullChain); i++ {
		currentAgentID := fullChain[i]
		isLastAgent := i == len(fullChain)-1

		// Get current agent info
		currentAgent, err := s.agentRepo.GetByID(ctx, currentAgentID)
		if err != nil || currentAgent == nil {
			hopLatency := &dto.ChainHopLatency{
				From:    id.FormatForwardAgentID(fmt.Sprintf("unknown_%d", currentAgentID)),
				To:      "unknown",
				Success: false,
				Online:  false,
				Error:   "agent not found",
			}
			chainLatencies = append(chainLatencies, hopLatency)
			allSuccess = false
			continue
		}

		fromAgentStripeID := id.FormatForwardAgentID(currentAgent.ShortID())
		isOnline := s.hub.IsAgentOnline(currentAgentID)

		if isLastAgent {
			// Last agent probes the target
			var targetAddress string
			var targetPort uint16

			if rule.HasTargetNode() {
				targetNode, err := s.nodeRepo.GetByID(ctx, *rule.TargetNodeID())
				if err != nil || targetNode == nil {
					hopLatency := &dto.ChainHopLatency{
						From:    fromAgentStripeID,
						To:      "target",
						Success: false,
						Online:  isOnline,
						Error:   "target node not found",
					}
					chainLatencies = append(chainLatencies, hopLatency)
					allSuccess = false
					continue
				}
				targetAddress = s.resolveNodeAddress(targetNode, ipVersion)
				if targetAddress == "" {
					hopLatency := &dto.ChainHopLatency{
						From:    fromAgentStripeID,
						To:      "target",
						Success: false,
						Online:  isOnline,
						Error:   "target node has no available address for ip_version: " + ipVersion.String(),
					}
					chainLatencies = append(chainLatencies, hopLatency)
					allSuccess = false
					continue
				}
				if targetPort == 0 {
					targetPort = targetNode.AgentPort()
				}
			} else {
				targetAddress = rule.TargetAddress()
				targetPort = rule.TargetPort()
			}

			hopLatency := &dto.ChainHopLatency{
				From:   fromAgentStripeID,
				To:     "target",
				Online: isOnline,
			}

			if !isOnline {
				hopLatency.Success = false
				hopLatency.Error = "agent not connected"
				chainLatencies = append(chainLatencies, hopLatency)
				allSuccess = false
				continue
			}

			latency, err := s.sendProbeTask(ctx, currentAgentID, ruleStripeID, dto.ProbeTaskTypeTarget,
				targetAddress, targetPort, "tcp")
			if err != nil {
				hopLatency.Success = false
				hopLatency.Error = err.Error()
				chainLatencies = append(chainLatencies, hopLatency)
				allSuccess = false
				continue
			}

			hopLatency.Success = true
			hopLatency.LatencyMs = latency
			chainLatencies = append(chainLatencies, hopLatency)
			totalLatency += latency
			response.TargetLatencyMs = &latency
		} else {
			// Probe to next agent in chain
			nextAgentID := fullChain[i+1]
			nextAgent, err := s.agentRepo.GetByID(ctx, nextAgentID)
			if err != nil || nextAgent == nil {
				hopLatency := &dto.ChainHopLatency{
					From:    fromAgentStripeID,
					To:      "unknown",
					Success: false,
					Online:  isOnline,
					Error:   "next agent not found",
				}
				chainLatencies = append(chainLatencies, hopLatency)
				allSuccess = false
				continue
			}

			// Get next agent's listen port from chain_port_config
			nextPort := rule.GetAgentListenPort(nextAgentID)
			if nextPort == 0 {
				hopLatency := &dto.ChainHopLatency{
					From:    fromAgentStripeID,
					To:      id.FormatForwardAgentID(nextAgent.ShortID()),
					Success: false,
					Online:  isOnline,
					Error:   "listen port not configured for next agent",
				}
				chainLatencies = append(chainLatencies, hopLatency)
				allSuccess = false
				continue
			}

			hopLatency := &dto.ChainHopLatency{
				From:   fromAgentStripeID,
				To:     id.FormatForwardAgentID(nextAgent.ShortID()),
				Online: isOnline,
			}

			if !isOnline {
				hopLatency.Success = false
				hopLatency.Error = "agent not connected"
				chainLatencies = append(chainLatencies, hopLatency)
				allSuccess = false
				continue
			}

			// Use GetEffectiveTunnelAddress for direct chain connections
			targetAddr := nextAgent.GetEffectiveTunnelAddress()
			if targetAddr == "" {
				hopLatency.Success = false
				hopLatency.Error = "next agent has no tunnel address"
				chainLatencies = append(chainLatencies, hopLatency)
				allSuccess = false
				continue
			}

			latency, err := s.sendProbeTask(ctx, currentAgentID, ruleStripeID, dto.ProbeTaskTypeTarget,
				targetAddr, nextPort, "tcp")
			if err != nil {
				hopLatency.Success = false
				hopLatency.Error = err.Error()
				chainLatencies = append(chainLatencies, hopLatency)
				allSuccess = false
				continue
			}

			hopLatency.Success = true
			hopLatency.LatencyMs = latency
			chainLatencies = append(chainLatencies, hopLatency)
			totalLatency += latency
		}
	}

	response.ChainLatencies = chainLatencies
	response.Success = allSuccess
	if allSuccess {
		response.TotalLatencyMs = &totalLatency
	}
	return response, nil
}
