// Package services provides application services for the forward domain.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	vo "github.com/orris-inc/orris/internal/domain/forward/valueobjects"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/infrastructure/auth"
	"github.com/orris-inc/orris/internal/shared/biztime"
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
	repo              forward.Repository
	agentRepo         forward.AgentRepository
	nodeRepo          node.NodeRepository
	statusQuerier     ProbeStatusQuerier
	agentTokenService *auth.AgentTokenService

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

// ProbeStatusQuerier queries agent status for ws_listen_port.
type ProbeStatusQuerier interface {
	GetStatus(ctx context.Context, agentID uint) (*dto.AgentStatusDTO, error)
}

// NewProbeService creates a new ProbeService.
func NewProbeService(
	repo forward.Repository,
	agentRepo forward.AgentRepository,
	nodeRepo node.NodeRepository,
	statusQuerier ProbeStatusQuerier,
	hub ProbeHub,
	tokenSigningSecret string,
	log logger.Interface,
) *ProbeService {
	return &ProbeService{
		repo:              repo,
		agentRepo:         agentRepo,
		nodeRepo:          nodeRepo,
		statusQuerier:     statusQuerier,
		agentTokenService: auth.NewAgentTokenService(tokenSigningSecret),
		hub:               hub,
		pendingProbes:     make(map[string]chan *dto.ProbeTaskResult),
		logger:            log,
	}
}

// String implements fmt.Stringer for logging purposes.
func (s *ProbeService) String() string {
	return "ProbeService"
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
	rule, err := s.repo.GetBySID(ctx, shortID)
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
		RuleID:   rule.SID(),
		RuleType: ruleType,
	}

	switch ruleType {
	case "direct":
		return s.probeDirectRule(ctx, rule, ipVersion, response)
	case "entry":
		return s.probeEntryRule(ctx, rule, ipVersion, response)
	case "chain":
		return s.probeChainRule(ctx, rule, ipVersion, response)
	case "direct_chain":
		return s.probeDirectChainRule(ctx, rule, ipVersion, response)
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
	ruleStripeID := rule.SID()
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
// Uses tunnel_ping to measure actual tunnel RTT instead of simple TCP connection test.
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

	// Get entry agent info (for generating tunnel token)
	entryAgent, err := s.agentRepo.GetByID(ctx, entryAgentID)
	if err != nil || entryAgent == nil {
		response.Error = "entry agent not found"
		return response, nil
	}

	// Get exit agent info
	exitAgent, err := s.agentRepo.GetByID(ctx, exitAgentID)
	if err != nil || exitAgent == nil {
		response.Error = "exit agent not found"
		return response, nil
	}

	// Get tunnel port from exit agent status cache based on rule's tunnel type
	exitStatus, err := s.statusQuerier.GetStatus(ctx, exitAgentID)
	if err != nil || exitStatus == nil {
		response.Error = "exit agent status not found"
		return response, nil
	}

	// Select port based on tunnel type
	var tunnelPort uint16
	tunnelType := rule.TunnelType().String()
	if rule.TunnelType().IsTLS() {
		tunnelPort = exitStatus.TlsListenPort
		if tunnelPort == 0 {
			response.Error = "exit agent has no tls_listen_port configured"
			return response, nil
		}
	} else {
		tunnelPort = exitStatus.WsListenPort
		if tunnelPort == 0 {
			response.Error = "exit agent has no ws_listen_port configured"
			return response, nil
		}
	}

	// Use GetEffectiveTunnelAddress for tunnel connections (prefers tunnel_address over public_address)
	tunnelAddr := exitAgent.GetEffectiveTunnelAddress()
	if tunnelAddr == "" {
		response.Error = "exit agent has no tunnel address"
		return response, nil
	}

	// Generate tunnel token for entry agent
	tunnelToken, _ := s.agentTokenService.Generate(entryAgent.SID())

	ruleStripeID := rule.SID()

	// Step 1: Probe tunnel using tunnel_ping (measures actual tunnel RTT)
	tunnelPingResult, err := s.sendTunnelPingTask(ctx, entryAgentID, ruleStripeID,
		tunnelAddr, tunnelPort, tunnelType, tunnelToken, 3)
	if err != nil {
		response.Error = "tunnel ping failed: " + err.Error()
		return response, nil
	}

	// Set tunnel latency results
	response.TunnelLatencyMs = &tunnelPingResult.AvgLatencyMs
	response.TunnelMinLatencyMs = &tunnelPingResult.MinLatencyMs
	response.TunnelMaxLatencyMs = &tunnelPingResult.MaxLatencyMs
	response.TunnelPacketLoss = &tunnelPingResult.PacketLoss

	// Step 2: Probe target from exit agent (exit → target)
	if !s.hub.IsAgentOnline(exitAgentID) {
		response.Error = "exit agent not connected, cannot probe target"
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

	// Probe target using TCP for reliable connectivity check
	targetLatency, err := s.sendProbeTask(ctx, exitAgentID, ruleStripeID, dto.ProbeTaskTypeTarget,
		targetAddress, targetPort, "tcp")
	if err != nil {
		response.Error = "target probe failed: " + err.Error()
		return response, nil
	}

	response.Success = true
	response.TargetLatencyMs = &targetLatency
	totalLatency := tunnelPingResult.AvgLatencyMs + targetLatency
	response.TotalLatencyMs = &totalLatency
	return response, nil
}

// probeChainRule probes a chain rule (entry → relay1 → relay2 → ... → lastAgent → target).
// Chain type uses WS tunnel connections between agents.
func (s *ProbeService) probeChainRule(ctx context.Context, rule *forward.ForwardRule, ipVersion vo.IPVersion, response *dto.RuleProbeResponse) (*dto.RuleProbeResponse, error) {
	// Build full chain: agentID (entry) -> chainAgentIDs[0] -> chainAgentIDs[1] -> ... -> target
	fullChain := append([]uint{rule.AgentID()}, rule.ChainAgentIDs()...)

	s.logger.Infow("probing chain rule",
		"rule_id", rule.ID(),
		"chain_length", len(fullChain),
		"chain_agent_ids", fullChain,
	)

	ruleStripeID := rule.SID()
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
				From:    id.FormatWithPrefix(id.PrefixForwardAgent, fmt.Sprintf("unknown_%d", currentAgentID)),
				To:      "unknown",
				Success: false,
				Online:  false,
				Error:   "agent not found",
			}
			chainLatencies = append(chainLatencies, hopLatency)
			allSuccess = false
			continue
		}

		fromAgentStripeID := currentAgent.SID()
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

			hopLatency := &dto.ChainHopLatency{
				From:   fromAgentStripeID,
				To:     nextAgent.SID(),
				Online: isOnline,
			}

			if !isOnline {
				hopLatency.Success = false
				hopLatency.Error = "agent not connected"
				chainLatencies = append(chainLatencies, hopLatency)
				allSuccess = false
				continue
			}

			// Determine if this hop uses tunnel or direct connection based on hop mode
			hopMode := rule.GetHopMode(i)
			outboundNeedsTunnel := hopMode == "tunnel" || (hopMode == "boundary" && false) // boundary outbound is always direct

			var probeAddr string
			var probePort uint16
			var probeType dto.ProbeTaskType

			if !outboundNeedsTunnel && (hopMode == "direct" || hopMode == "boundary") {
				// Direct connection mode: use chainPortConfig for next hop port
				probePort = rule.GetAgentListenPort(nextAgentID)
				if probePort == 0 {
					hopLatency.Success = false
					hopLatency.Error = "next agent has no direct port configured in chain_port_config"
					chainLatencies = append(chainLatencies, hopLatency)
					allSuccess = false
					continue
				}
				probeAddr = nextAgent.GetEffectiveTunnelAddress()
				if probeAddr == "" {
					hopLatency.Success = false
					hopLatency.Error = "next agent has no address"
					chainLatencies = append(chainLatencies, hopLatency)
					allSuccess = false
					continue
				}
				probeType = dto.ProbeTaskTypeTarget
			} else {
				// Tunnel mode: get tunnel ports from status cache
				nextStatus, err := s.statusQuerier.GetStatus(ctx, nextAgentID)
				if err != nil || nextStatus == nil {
					hopLatency.Success = false
					hopLatency.Error = "next agent status not found"
					chainLatencies = append(chainLatencies, hopLatency)
					allSuccess = false
					continue
				}

				// Select port based on tunnel type
				if rule.TunnelType().IsTLS() {
					probePort = nextStatus.TlsListenPort
					if probePort == 0 {
						hopLatency.Success = false
						hopLatency.Error = "next agent has no tls_listen_port configured"
						chainLatencies = append(chainLatencies, hopLatency)
						allSuccess = false
						continue
					}
				} else {
					probePort = nextStatus.WsListenPort
					if probePort == 0 {
						hopLatency.Success = false
						hopLatency.Error = "next agent has no ws_listen_port configured"
						chainLatencies = append(chainLatencies, hopLatency)
						allSuccess = false
						continue
					}
				}

				probeAddr = nextAgent.GetEffectiveTunnelAddress()
				if probeAddr == "" {
					hopLatency.Success = false
					hopLatency.Error = "next agent has no tunnel address"
					chainLatencies = append(chainLatencies, hopLatency)
					allSuccess = false
					continue
				}
				probeType = dto.ProbeTaskTypeTunnel
			}

			latency, err := s.sendProbeTask(ctx, currentAgentID, ruleStripeID, probeType,
				probeAddr, probePort, "tcp")
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

	ruleStripeID := rule.SID()
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
				From:    id.FormatWithPrefix(id.PrefixForwardAgent, fmt.Sprintf("unknown_%d", currentAgentID)),
				To:      "unknown",
				Success: false,
				Online:  false,
				Error:   "agent not found",
			}
			chainLatencies = append(chainLatencies, hopLatency)
			allSuccess = false
			continue
		}

		fromAgentStripeID := currentAgent.SID()
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
					To:      nextAgent.SID(),
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
				To:     nextAgent.SID(),
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
		AgentID:   agent.SID(),
		Timestamp: biztime.NowUTC().Unix(),
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

// TunnelPingResult contains detailed tunnel ping results.
type TunnelPingResult struct {
	AvgLatencyMs int64
	MinLatencyMs int64
	MaxLatencyMs int64
	PacketLoss   float64
	PingsSent    int
	PingsRecv    int
}

// sendTunnelPingTask sends a tunnel_ping probe task to an agent and waits for the result.
func (s *ProbeService) sendTunnelPingTask(
	ctx context.Context,
	agentID uint,
	ruleID string,
	target string,
	port uint16,
	tunnelType string,
	tunnelToken string,
	pingCount int,
) (*TunnelPingResult, error) {
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

	// Get agent
	agent, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		return nil, err
	}
	if agent == nil {
		return nil, forward.ErrAgentNotFound
	}

	// Set defaults
	if pingCount <= 0 {
		pingCount = 3
	}

	// Send tunnel ping task
	task := &dto.ProbeTask{
		ID:                taskID,
		Type:              dto.ProbeTaskTypeTunnelPing,
		RuleID:            ruleID,
		Target:            target,
		Port:              port,
		Timeout:           int(probeTimeout.Milliseconds()),
		TunnelType:        tunnelType,
		TunnelToken:       tunnelToken,
		PingCount:         pingCount,
		PingIntervalMs:    200,
		TunnelConnTimeout: int(probeTimeout.Milliseconds()),
	}

	msg := &dto.HubMessage{
		Type:      dto.MsgTypeProbeTask,
		AgentID:   agent.SID(),
		Timestamp: biztime.NowUTC().Unix(),
		Data:      task,
	}

	if err := s.hub.SendMessageToAgent(agentID, msg); err != nil {
		return nil, err
	}

	// Wait for result with extended timeout for tunnel ping
	tunnelPingTimeout := probeTimeout + time.Duration(pingCount)*time.Second
	select {
	case result := <-resultChan:
		if !result.Success {
			return nil, &probeError{message: result.Error}
		}
		return &TunnelPingResult{
			AvgLatencyMs: result.AvgLatencyMs,
			MinLatencyMs: result.MinLatencyMs,
			MaxLatencyMs: result.MaxLatencyMs,
			PacketLoss:   result.PacketLoss,
			PingsSent:    result.PingsSent,
			PingsRecv:    result.PingsRecv,
		}, nil
	case <-time.After(tunnelPingTimeout):
		return nil, &probeError{message: "tunnel ping timeout"}
	case <-ctx.Done():
		return nil, ctx.Err()
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

// resolveNodeAddress selects the appropriate node address based on IP version preference.
// ipVersion: "auto", "ipv4", or "ipv6"
func (s *ProbeService) resolveNodeAddress(n *node.Node, ipVersion vo.IPVersion) string {
	serverAddr := n.ServerAddress().Value()
	ipv4 := ""
	ipv6 := ""

	if n.PublicIPv4() != nil {
		ipv4 = *n.PublicIPv4()
	}
	if n.PublicIPv6() != nil {
		ipv6 = *n.PublicIPv6()
	}

	// Check if server_address is a valid usable address
	isValidServerAddr := serverAddr != "" && serverAddr != "0.0.0.0" && serverAddr != "::"

	switch ipVersion {
	case vo.IPVersionIPv6:
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

	case vo.IPVersionIPv4:
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
