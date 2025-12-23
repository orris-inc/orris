// Package forward provides HTTP handlers for forward rule management.
package forward

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/application/forward/usecases"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/infrastructure/adapters"
	"github.com/orris-inc/orris/internal/infrastructure/auth"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// AgentHandler handles RESTful agent API requests for forward client
type AgentHandler struct {
	repo               forward.Repository
	agentRepo          forward.AgentRepository
	nodeRepo           node.NodeRepository
	reportStatusUC     *usecases.ReportAgentStatusUseCase
	statusQuerier      usecases.AgentStatusQuerier
	tokenSigningSecret string
	agentTokenService  *auth.AgentTokenService
	trafficRecorder    adapters.ForwardTrafficRecorder
	logger             logger.Interface
}

// NewAgentHandler creates a new AgentHandler instance
func NewAgentHandler(
	repo forward.Repository,
	agentRepo forward.AgentRepository,
	nodeRepo node.NodeRepository,
	reportStatusUC *usecases.ReportAgentStatusUseCase,
	statusQuerier usecases.AgentStatusQuerier,
	tokenSigningSecret string,
	trafficRecorder adapters.ForwardTrafficRecorder,
	logger logger.Interface,
) *AgentHandler {
	return &AgentHandler{
		repo:               repo,
		agentRepo:          agentRepo,
		nodeRepo:           nodeRepo,
		reportStatusUC:     reportStatusUC,
		statusQuerier:      statusQuerier,
		tokenSigningSecret: tokenSigningSecret,
		agentTokenService:  auth.NewAgentTokenService(tokenSigningSecret),
		trafficRecorder:    trafficRecorder,
		logger:             logger,
	}
}

// ForwardRuleTrafficItem represents traffic data for a single forward rule
type ForwardRuleTrafficItem struct {
	RuleID        string `json:"rule_id" binding:"required"` // Stripe-style prefixed ID (e.g., "fr_xK9mP2vL3nQ")
	UploadBytes   int64  `json:"upload_bytes" binding:"min=0"`
	DownloadBytes int64  `json:"download_bytes" binding:"min=0"`
}

// ReportTrafficRequest represents traffic report request from forward client
type ReportTrafficRequest struct {
	Rules []ForwardRuleTrafficItem `json:"rules" binding:"required,dive"`
}

// getAuthenticatedAgentID extracts the authenticated forward agent ID from context.
// Returns the agent ID or an error if not found.
func (h *AgentHandler) getAuthenticatedAgentID(c *gin.Context) (uint, error) {
	agentID, exists := c.Get("forward_agent_id")
	if !exists {
		return 0, fmt.Errorf("forward_agent_id not found in context")
	}
	id, ok := agentID.(uint)
	if !ok {
		return 0, fmt.Errorf("invalid forward_agent_id type in context")
	}
	return id, nil
}

func (h *AgentHandler) GetEnabledRules(c *gin.Context) {
	ctx := c.Request.Context()

	// Get authenticated agent ID from context
	agentID, err := h.getAuthenticatedAgentID(c)
	if err != nil {
		h.logger.Warnw("failed to get authenticated agent ID",
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	h.logger.Infow("forward client requesting enabled rules",
		"agent_id", agentID,
		"ip", c.ClientIP(),
	)

	// Retrieve enabled forward rules for this agent (as entry agent)
	rules, err := h.repo.ListEnabledByAgentID(ctx, agentID)
	if err != nil {
		h.logger.Errorw("failed to retrieve enabled forward rules",
			"error", err,
			"agent_id", agentID,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to retrieve enabled forward rules")
		return
	}

	// Also retrieve entry rules where this agent is the exit agent
	exitRules, err := h.repo.ListEnabledByExitAgentID(ctx, agentID)
	if err != nil {
		h.logger.Errorw("failed to retrieve enabled exit rules",
			"error", err,
			"agent_id", agentID,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to retrieve enabled forward rules")
		return
	}

	// Also retrieve chain rules where this agent participates
	chainRules, err := h.repo.ListEnabledByChainAgentID(ctx, agentID)
	if err != nil {
		h.logger.Errorw("failed to retrieve enabled chain rules",
			"error", err,
			"agent_id", agentID,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to retrieve enabled forward rules")
		return
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

	h.logger.Infow("enabled forward rules retrieved successfully",
		"rule_count", len(allRules),
		"entry_rules", len(rules),
		"exit_rules", len(exitRules),
		"chain_rules", len(chainRules),
		"agent_id", agentID,
		"ip", c.ClientIP(),
	)

	rules = allRules

	// Convert to DTOs and resolve dynamic node addresses
	ruleDTOs := dto.ToForwardRuleDTOs(rules)

	// Collect all agent IDs that need short ID lookup (for AgentID and ExitAgentID)
	agentIDs := dto.CollectAgentIDs(ruleDTOs)
	if len(agentIDs) > 0 {
		// Batch fetch agent short IDs
		agentMap, err := h.agentRepo.GetSIDsByIDs(ctx, agentIDs)
		if err != nil {
			h.logger.Warnw("failed to get agent short IDs",
				"agent_ids", agentIDs,
				"error", err,
			)
			// Continue with empty map
			agentMap = make(dto.AgentSIDMap)
		}
		// Populate agent info (AgentID and ExitAgentID) for all DTOs
		for _, ruleDTO := range ruleDTOs {
			ruleDTO.PopulateAgentInfo(agentMap)
		}
	}

	// Resolve node addresses for rules with targetNodeID
	for _, ruleDTO := range ruleDTOs {
		targetNodeID := ruleDTO.InternalTargetNodeID()
		if targetNodeID != nil && *targetNodeID != 0 {
			// Fetch node information
			node, err := h.nodeRepo.GetByID(ctx, *targetNodeID)
			if err != nil {
				h.logger.Warnw("failed to get target node for rule",
					"rule_id", ruleDTO.ID,
					"node_id", *targetNodeID,
					"error", err,
				)
				// Keep original values if node fetch fails
				continue
			}
			if node == nil {
				h.logger.Warnw("target node not found for rule",
					"rule_id", ruleDTO.ID,
					"node_id", *targetNodeID,
				)
				// Keep original values if node not found
				continue
			}

			// Dynamically populate target address and port from node
			// Selection priority depends on rule's IP version setting
			targetAddress := h.resolveNodeAddress(node, ruleDTO.IPVersion)

			ruleDTO.TargetAddress = targetAddress
			ruleDTO.TargetPort = node.AgentPort()

			h.logger.Debugw("resolved target node address for rule",
				"rule_id", ruleDTO.ID,
				"node_id", *targetNodeID,
				"target_address", ruleDTO.TargetAddress,
				"target_port", ruleDTO.TargetPort,
			)
		}
	}

	// Process entry rules to populate next hop information for entry agents
	for i, rule := range rules {
		if rule.RuleType().String() != "entry" {
			continue
		}

		ruleDTO := ruleDTOs[i]

		// Only populate next hop info for entry agent (not exit agent)
		if rule.AgentID() == agentID {
			exitAgentID := rule.ExitAgentID()
			if exitAgentID != 0 {
				exitAgent, err := h.agentRepo.GetByID(ctx, exitAgentID)
				if err != nil {
					h.logger.Warnw("failed to get exit agent for entry rule",
						"rule_id", ruleDTO.ID,
						"exit_agent_id", exitAgentID,
						"error", err,
					)
				} else if exitAgent != nil {
					ruleDTO.NextHopAgentID = exitAgent.SID()
					// Use effective tunnel address (prefers tunnel_address over public_address)
					ruleDTO.NextHopAddress = exitAgent.GetEffectiveTunnelAddress()

					// Get ws_listen_port from cached agent status
					exitStatus, err := h.statusQuerier.GetStatus(ctx, exitAgentID)
					if err != nil {
						h.logger.Warnw("failed to get exit agent status for entry rule",
							"rule_id", ruleDTO.ID,
							"exit_agent_id", exitAgentID,
							"error", err,
						)
					} else if exitStatus != nil && exitStatus.WsListenPort > 0 {
						ruleDTO.NextHopWsPort = exitStatus.WsListenPort

					} else {
						h.logger.Debugw("exit agent has no ws_listen_port configured or is offline",
							"rule_id", ruleDTO.ID,
							"exit_agent_id", exitAgentID,
						)
					}
				}
			}
		}
	}

	// Process chain rules to populate role-specific information
	for i, rule := range rules {
		if rule.RuleType().String() != "chain" {
			continue
		}

		ruleDTO := ruleDTOs[i]

		// Calculate chain position and last-in-chain flag for this agent
		chainPosition := rule.GetChainPosition(agentID)
		isLast := rule.IsLastInChain(agentID)

		ruleDTO.ChainPosition = chainPosition
		ruleDTO.IsLastInChain = isLast

		// For non-exit agents in chain, populate next hop information
		if !isLast {
			nextHopAgentID := rule.GetNextHopAgentID(agentID)
			if nextHopAgentID != 0 {
				// Get next hop agent details
				nextAgent, err := h.agentRepo.GetByID(ctx, nextHopAgentID)
				if err != nil {
					h.logger.Warnw("failed to get next hop agent for chain rule",
						"rule_id", ruleDTO.ID,
						"next_hop_agent_id", nextHopAgentID,
						"error", err,
					)
				} else if nextAgent != nil {
					ruleDTO.NextHopAgentID = nextAgent.SID()
					// Use effective tunnel address (prefers tunnel_address over public_address)
					ruleDTO.NextHopAddress = nextAgent.GetEffectiveTunnelAddress()

					// Get ws_listen_port from cached agent status (same as GetExitEndpoint)
					nextStatus, err := h.statusQuerier.GetStatus(ctx, nextHopAgentID)
					if err != nil {
						h.logger.Warnw("failed to get next hop agent status",
							"rule_id", ruleDTO.ID,
							"next_hop_agent_id", nextHopAgentID,
							"error", err,
						)
					} else if nextStatus != nil && nextStatus.WsListenPort > 0 {
						ruleDTO.NextHopWsPort = nextStatus.WsListenPort

						// Note: connection token is no longer needed as agents use HMAC-based agent tokens

					} else {
						h.logger.Debugw("next hop agent has no ws_listen_port configured or is offline",
							"rule_id", ruleDTO.ID,
							"next_hop_agent_id", nextHopAgentID,
						)
					}
				}
			}

			// Clear target info for non-exit agents (minimum info principle)
			ruleDTO.TargetAddress = ""
			ruleDTO.TargetPort = 0

		} else {
			// For exit agents, clear next hop info (minimum info principle)
			ruleDTO.NextHopAgentID = ""
			ruleDTO.NextHopAddress = ""
			ruleDTO.NextHopWsPort = 0

		}
	}

	// Process direct_chain rules to populate role-specific information
	for i, rule := range rules {
		if rule.RuleType().String() != "direct_chain" {
			continue
		}

		ruleDTO := ruleDTOs[i]

		// Calculate chain position and last-in-chain flag for this agent
		chainPosition := rule.GetChainPosition(agentID)
		isLast := rule.IsLastInChain(agentID)

		// Defensive check: agent must be in chain
		if chainPosition < 0 {
			h.logger.Errorw("agent not found in direct_chain rule",
				"agent_id", agentID,
				"rule_id", ruleDTO.ID,
				"entry_agent_id", rule.AgentID(),
				"chain_agent_ids", rule.ChainAgentIDs(),
			)
			// Skip this rule instead of failing the entire request
			continue
		}

		ruleDTO.ChainPosition = chainPosition
		ruleDTO.IsLastInChain = isLast

		// Set ListenPort based on role
		// Entry agent uses rule.ListenPort(), other agents use chainPortConfig
		if chainPosition == 0 {
			// Entry agent uses the rule's listen_port field
			ruleDTO.ListenPort = rule.ListenPort()
		} else {
			// Relay/exit agents use chainPortConfig
			ruleDTO.ListenPort = rule.GetAgentListenPort(agentID)
		}

		// For non-exit agents in chain, populate next hop information
		if !isLast {
			nextHopAgentID, nextHopPort, err := rule.GetNextHopForDirectChainSafe(agentID)
			if err != nil {
				h.logger.Errorw("failed to get next hop for direct_chain rule",
					"rule_id", ruleDTO.ID,
					"agent_id", agentID,
					"error", err,
				)
				// Skip this rule instead of failing the entire request
				continue
			}

			if nextHopAgentID != 0 {
				// Get next hop agent details
				nextAgent, err := h.agentRepo.GetByID(ctx, nextHopAgentID)
				if err != nil {
					h.logger.Warnw("failed to get next hop agent for direct_chain rule",
						"rule_id", ruleDTO.ID,
						"next_hop_agent_id", nextHopAgentID,
						"error", err,
					)
				} else if nextAgent != nil {
					ruleDTO.NextHopAgentID = nextAgent.SID()
					// Use effective tunnel address (prefers tunnel_address over public_address)
					ruleDTO.NextHopAddress = nextAgent.GetEffectiveTunnelAddress()
					ruleDTO.NextHopPort = nextHopPort

					// Generate connection token for next hop authentication
					nextHopToken, _ := h.agentTokenService.Generate(nextAgent.SID())
					ruleDTO.NextHopConnectionToken = nextHopToken

				}
			}

			// Clear target info for non-exit agents (minimum info principle)
			ruleDTO.TargetAddress = ""
			ruleDTO.TargetPort = 0

		} else {
			// For exit agents, clear next hop info (minimum info principle)
			ruleDTO.NextHopAgentID = ""
			ruleDTO.NextHopAddress = ""
			ruleDTO.NextHopPort = 0

		}
	}

	// Set role field for all rules based on requesting agent's position
	for i, rule := range rules {
		ruleDTO := ruleDTOs[i]

		switch rule.RuleType().String() {
		case "direct":
			// Direct rules: agent is always the forwarder
			ruleDTO.Role = "entry"

		case "entry":
			if rule.AgentID() == agentID {
				// This agent is the entry point
				ruleDTO.Role = "entry"
			} else if rule.ExitAgentID() == agentID {
				// This agent is the exit point - clear exit_agent_id (minimum info principle)
				ruleDTO.Role = "exit"
				ruleDTO.ExitAgentID = ""

			}

		case "chain":
			// Chain role is already set based on position
			if ruleDTO.ChainPosition == 0 {
				ruleDTO.Role = "entry"
			} else if ruleDTO.IsLastInChain {
				ruleDTO.Role = "exit"
			} else {
				ruleDTO.Role = "relay"
			}

		case "direct_chain":
			// Direct chain role is set based on position
			if ruleDTO.ChainPosition == 0 {
				ruleDTO.Role = "entry"
			} else if ruleDTO.IsLastInChain {
				ruleDTO.Role = "exit"
			} else {
				ruleDTO.Role = "relay"
			}
		}
	}

	// Get the requesting agent's token for tunnel handshake
	var clientToken string
	requestingAgent, err := h.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		h.logger.Warnw("failed to get requesting agent for token",
			"agent_id", agentID,
			"error", err,
		)
	} else if requestingAgent != nil {
		// Generate token using agentTokenService to ensure correct format (fwd_xxx_xxx)
		clientToken, _ = h.agentTokenService.Generate(requestingAgent.SID())
		h.logger.Infow("generated client token for agent",
			"agent_id", agentID,
			"short_id", requestingAgent.SID(),
			"client_token", clientToken,
		)
	}

	// Return success response
	// Note: token_signing_secret is no longer returned for security reasons.
	// Agents should use the server for token verification, not local HMAC verification.
	utils.SuccessResponse(c, http.StatusOK, "enabled forward rules retrieved successfully", map[string]any{
		"rules":        ruleDTOs,
		"client_token": clientToken,
	})
}

// RefreshRule handles GET /forward-agent-api/rules/:rule_id
// This endpoint allows an agent to refresh the configuration for a specific rule.
// It returns the latest next_hop_ws_port and other dynamic configuration.
// This is useful when the agent detects a connection failure to the next hop.
func (h *AgentHandler) RefreshRule(c *gin.Context) {
	ctx := c.Request.Context()

	// Get authenticated agent ID from context
	agentID, err := h.getAuthenticatedAgentID(c)
	if err != nil {
		h.logger.Warnw("failed to get authenticated agent ID",
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Parse rule ID from path (Stripe-style ID like "fr_xK9mP2vL3nQ")
	ruleIDStr := c.Param("rule_id")
	shortID, err := id.ParseForwardRuleID(ruleIDStr)
	if err != nil {
		h.logger.Warnw("invalid rule_id parameter",
			"rule_id", ruleIDStr,
			"agent_id", agentID,
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid rule_id parameter: must be in format fr_xxx")
		return
	}

	// Look up the rule by short ID
	rule, err := h.repo.GetBySID(ctx, shortID)
	if err != nil {
		h.logger.Warnw("rule not found",
			"rule_id", ruleIDStr,
			"short_id", shortID,
			"agent_id", agentID,
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusNotFound, "rule not found")
		return
	}

	// Verify that this agent has access to the rule
	hasAccess := false
	if rule.AgentID() == agentID {
		hasAccess = true
	} else if rule.RuleType().String() == "entry" && rule.ExitAgentID() == agentID {
		hasAccess = true
	} else if rule.RuleType().String() == "chain" || rule.RuleType().String() == "direct_chain" {
		// Check if agent is in the chain
		if rule.GetChainPosition(agentID) >= 0 {
			hasAccess = true
		}
	}

	if !hasAccess {
		h.logger.Warnw("agent does not have access to rule",
			"rule_id", ruleIDStr,
			"agent_id", agentID,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusForbidden, "access denied")
		return
	}

	h.logger.Infow("forward client requesting rule refresh",
		"rule_id", ruleIDStr,
		"agent_id", agentID,
		"ip", c.ClientIP(),
	)

	// Convert to DTO
	ruleDTO := dto.ToForwardRuleDTO(rule)

	// Populate agent info
	agentIDs := dto.CollectAgentIDs([]*dto.ForwardRuleDTO{ruleDTO})
	if len(agentIDs) > 0 {
		agentMap, err := h.agentRepo.GetSIDsByIDs(ctx, agentIDs)
		if err != nil {
			h.logger.Warnw("failed to get agent short IDs",
				"agent_ids", agentIDs,
				"error", err,
			)
			agentMap = make(dto.AgentSIDMap)
		}
		ruleDTO.PopulateAgentInfo(agentMap)
	}

	// Resolve node address if applicable
	targetNodeID := ruleDTO.InternalTargetNodeID()
	if targetNodeID != nil && *targetNodeID != 0 {
		node, err := h.nodeRepo.GetByID(ctx, *targetNodeID)
		if err == nil && node != nil {
			targetAddress := h.resolveNodeAddress(node, ruleDTO.IPVersion)
			ruleDTO.TargetAddress = targetAddress
			ruleDTO.TargetPort = node.AgentPort()
		}
	}

	// Populate role-specific information based on rule type
	switch rule.RuleType().String() {
	case "entry":
		// Populate next hop info for entry agent
		if rule.AgentID() == agentID {
			exitAgentID := rule.ExitAgentID()
			if exitAgentID != 0 {
				exitAgent, err := h.agentRepo.GetByID(ctx, exitAgentID)
				if err == nil && exitAgent != nil {
					ruleDTO.NextHopAgentID = exitAgent.SID()
					ruleDTO.NextHopAddress = exitAgent.GetEffectiveTunnelAddress()

					exitStatus, err := h.statusQuerier.GetStatus(ctx, exitAgentID)
					if err == nil && exitStatus != nil && exitStatus.WsListenPort > 0 {
						ruleDTO.NextHopWsPort = exitStatus.WsListenPort
					}
				}
			}
		}

	case "chain":
		chainPosition := rule.GetChainPosition(agentID)
		isLast := rule.IsLastInChain(agentID)

		ruleDTO.ChainPosition = chainPosition
		ruleDTO.IsLastInChain = isLast

		// Populate ChainAgentIDs (full chain: entry + chain_agents)
		fullChainIDs := append([]uint{rule.AgentID()}, rule.ChainAgentIDs()...)
		if len(fullChainIDs) > 0 {
			agentMap, err := h.agentRepo.GetSIDsByIDs(ctx, fullChainIDs)
			if err == nil {
				ruleDTO.ChainAgentIDs = make([]string, len(fullChainIDs))
				for i, chainAgentID := range fullChainIDs {
					if sid, ok := agentMap[chainAgentID]; ok {
						ruleDTO.ChainAgentIDs[i] = sid
					}
				}
			}
		}

		// Determine role
		if chainPosition == 0 {
			ruleDTO.Role = "entry"
		} else if isLast {
			ruleDTO.Role = "exit"
		} else {
			ruleDTO.Role = "relay"
		}

		// For non-exit agents, populate next hop info
		if !isLast {
			nextHopAgentID := rule.GetNextHopAgentID(agentID)
			if nextHopAgentID != 0 {
				nextAgent, err := h.agentRepo.GetByID(ctx, nextHopAgentID)
				if err == nil && nextAgent != nil {
					ruleDTO.NextHopAgentID = nextAgent.SID()
					ruleDTO.NextHopAddress = nextAgent.GetEffectiveTunnelAddress()

					nextStatus, err := h.statusQuerier.GetStatus(ctx, nextHopAgentID)
					if err == nil && nextStatus != nil && nextStatus.WsListenPort > 0 {
						ruleDTO.NextHopWsPort = nextStatus.WsListenPort
					}
				}
			}
			// Clear target info for non-exit agents (minimum info principle)
			ruleDTO.TargetAddress = ""
			ruleDTO.TargetPort = 0
		}

	case "direct_chain":
		chainPosition := rule.GetChainPosition(agentID)
		isLast := rule.IsLastInChain(agentID)

		ruleDTO.ChainPosition = chainPosition
		ruleDTO.IsLastInChain = isLast

		// Populate ChainAgentIDs (full chain: entry + chain_agents)
		fullChainIDs := append([]uint{rule.AgentID()}, rule.ChainAgentIDs()...)
		if len(fullChainIDs) > 0 {
			agentMap, err := h.agentRepo.GetSIDsByIDs(ctx, fullChainIDs)
			if err == nil {
				ruleDTO.ChainAgentIDs = make([]string, len(fullChainIDs))
				for i, chainAgentID := range fullChainIDs {
					if sid, ok := agentMap[chainAgentID]; ok {
						ruleDTO.ChainAgentIDs[i] = sid
					}
				}
			}
		}

		// Determine role and set ListenPort
		if chainPosition == 0 {
			ruleDTO.Role = "entry"
			ruleDTO.ListenPort = rule.ListenPort()
		} else if isLast {
			ruleDTO.Role = "exit"
			ruleDTO.ListenPort = rule.GetAgentListenPort(agentID)
		} else {
			ruleDTO.Role = "relay"
			ruleDTO.ListenPort = rule.GetAgentListenPort(agentID)
		}

		// For non-exit agents, populate next hop info
		if !isLast {
			nextHopAgentID, nextHopPort, err := rule.GetNextHopForDirectChainSafe(agentID)
			if err != nil {
				h.logger.Errorw("failed to get next hop for direct_chain rule refresh",
					"rule_id", ruleIDStr,
					"agent_id", agentID,
					"error", err,
				)
				utils.ErrorResponse(c, http.StatusInternalServerError, "failed to get next hop configuration")
				return
			}

			if nextHopAgentID != 0 {
				nextAgent, err := h.agentRepo.GetByID(ctx, nextHopAgentID)
				if err == nil && nextAgent != nil {
					ruleDTO.NextHopAgentID = nextAgent.SID()
					ruleDTO.NextHopAddress = nextAgent.GetEffectiveTunnelAddress()
					ruleDTO.NextHopPort = nextHopPort

					// Generate connection token for next hop authentication
					nextHopToken, _ := h.agentTokenService.Generate(nextAgent.SID())
					ruleDTO.NextHopConnectionToken = nextHopToken
				}
			}
			// Clear target info for non-exit agents (minimum info principle)
			ruleDTO.TargetAddress = ""
			ruleDTO.TargetPort = 0
		}
	}

	h.logger.Infow("rule refresh successful",
		"rule_id", ruleIDStr,
		"agent_id", agentID,
		"rule_type", rule.RuleType().String(),
		"next_hop_ws_port", ruleDTO.NextHopWsPort,
		"ip", c.ClientIP(),
	)

	utils.SuccessResponse(c, http.StatusOK, "rule refreshed successfully", ruleDTO)
}

func (h *AgentHandler) ReportTraffic(c *gin.Context) {
	ctx := c.Request.Context()

	// Get authenticated agent ID from context
	agentID, err := h.getAuthenticatedAgentID(c)
	if err != nil {
		h.logger.Warnw("failed to get authenticated agent ID",
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Parse request body
	var req ReportTrafficRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid traffic report request body",
			"error", err,
			"agent_id", agentID,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	h.logger.Infow("forward client traffic report received",
		"rule_count", len(req.Rules),
		"agent_id", agentID,
		"ip", c.ClientIP(),
	)

	// Build a map of valid rule IDs for this agent (Stripe-style ID -> internal uint ID)
	// Include rules where this agent is the entry agent
	agentRules, err := h.repo.ListByAgentID(ctx, agentID)
	if err != nil {
		h.logger.Errorw("failed to get agent rules for validation",
			"agent_id", agentID,
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to validate rules")
		return
	}

	// Also include rules where this agent is the exit agent
	exitRules, err := h.repo.ListByExitAgentID(ctx, agentID)
	if err != nil {
		h.logger.Errorw("failed to get exit rules for validation",
			"agent_id", agentID,
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to validate rules")
		return
	}

	// Also include chain rules where this agent participates
	chainRules, err := h.repo.ListEnabledByChainAgentID(ctx, agentID)
	if err != nil {
		h.logger.Errorw("failed to get chain rules for validation",
			"agent_id", agentID,
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to validate rules")
		return
	}

	// ruleInfo holds rule ID and user ID for traffic recording
	type ruleInfo struct {
		id     uint
		userID *uint
	}

	// Merge all rules into validRuleIDs map (use rule.ID() to deduplicate)
	validRuleIDs := make(map[string]ruleInfo) // Stripe-style ID -> ruleInfo
	for _, rule := range agentRules {
		stripeID := rule.SID()
		validRuleIDs[stripeID] = ruleInfo{id: rule.ID(), userID: rule.UserID()}
	}
	for _, rule := range exitRules {
		stripeID := rule.SID()
		validRuleIDs[stripeID] = ruleInfo{id: rule.ID(), userID: rule.UserID()}
	}
	for _, rule := range chainRules {
		stripeID := rule.SID()
		validRuleIDs[stripeID] = ruleInfo{id: rule.ID(), userID: rule.UserID()}
	}

	h.logger.Debugw("validated rule sources for traffic report",
		"agent_id", agentID,
		"entry_rules", len(agentRules),
		"exit_rules", len(exitRules),
		"chain_rules", len(chainRules),
		"total_valid_rules", len(validRuleIDs),
	)

	// Update traffic for each rule
	successCount := 0
	errorCount := 0
	deniedCount := 0

	// Maximum reasonable traffic per report: 10TB
	// This prevents overflow and detects potential malicious clients
	const maxReasonableTraffic = 10 * 1024 * 1024 * 1024 * 1024 // 10TB

	for _, item := range req.Rules {
		// Validate rule belongs to this agent and get internal ID
		info, valid := validRuleIDs[item.RuleID]
		if !valid {
			h.logger.Warnw("traffic report for unauthorized rule",
				"rule_id", item.RuleID,
				"agent_id", agentID,
				"ip", c.ClientIP(),
			)
			deniedCount++
			continue
		}

		// Validate traffic data range
		if item.UploadBytes < 0 || item.DownloadBytes < 0 {
			h.logger.Warnw("negative traffic data rejected",
				"rule_id", item.RuleID,
				"agent_id", agentID,
				"upload", item.UploadBytes,
				"download", item.DownloadBytes,
				"ip", c.ClientIP(),
			)
			errorCount++
			continue
		}

		// Check for suspiciously large values
		if item.UploadBytes > maxReasonableTraffic || item.DownloadBytes > maxReasonableTraffic {
			h.logger.Warnw("suspiciously large traffic data rejected",
				"rule_id", item.RuleID,
				"agent_id", agentID,
				"upload", item.UploadBytes,
				"download", item.DownloadBytes,
				"max_allowed", maxReasonableTraffic,
				"ip", c.ClientIP(),
			)
			errorCount++
			continue
		}

		// Skip if no traffic to report
		if item.UploadBytes == 0 && item.DownloadBytes == 0 {
			continue
		}

		// Update traffic in forward_rules table
		err := h.repo.UpdateTraffic(ctx, info.id, item.UploadBytes, item.DownloadBytes)
		if err != nil {
			h.logger.Errorw("failed to update rule traffic",
				"rule_id", item.RuleID,
				"internal_id", info.id,
				"agent_id", agentID,
				"upload", item.UploadBytes,
				"download", item.DownloadBytes,
				"error", err,
			)
			errorCount++
			continue
		}

		// Also record traffic to subscription_usages table (for unified traffic tracking)
		if h.trafficRecorder != nil && info.userID != nil {
			if err := h.trafficRecorder.RecordForwardTraffic(ctx, info.id, *info.userID, item.UploadBytes, item.DownloadBytes); err != nil {
				// Log warning but don't fail the request - forward_rules update already succeeded
				h.logger.Warnw("failed to record forward traffic to subscription_usages",
					"rule_id", item.RuleID,
					"internal_id", info.id,
					"user_id", *info.userID,
					"error", err,
				)
			}
		}

		successCount++
	}

	h.logger.Infow("forward traffic report processed",
		"success_count", successCount,
		"error_count", errorCount,
		"denied_count", deniedCount,
		"agent_id", agentID,
		"ip", c.ClientIP(),
	)

	// Return success response with statistics
	utils.SuccessResponse(c, http.StatusOK, "traffic reported successfully", map[string]any{
		"rules_updated": successCount,
		"rules_failed":  errorCount,
		"rules_denied":  deniedCount,
	})
}

// GetExitEndpoint handles GET /forward-agent-api/exit-endpoint/:agent_id
// This endpoint allows an entry agent to get the exit endpoint information
// for establishing tunnel connections. Access is restricted to entry agents
// that have an entry rule pointing to the requested exit agent.
func (h *AgentHandler) GetExitEndpoint(c *gin.Context) {
	ctx := c.Request.Context()

	// Get authenticated agent ID (the entry agent making the request)
	entryAgentID, err := h.getAuthenticatedAgentID(c)
	if err != nil {
		h.logger.Warnw("failed to get authenticated agent ID",
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Parse requested exit agent ID from path (supports Stripe-style ID like "fa_xK9mP2vL3nQ")
	exitAgentIDStr := c.Param("agent_id")

	// Validate Stripe-style prefixed ID (database stores full SID with prefix)
	if err := id.ValidatePrefix(exitAgentIDStr, id.PrefixForwardAgent); err != nil {
		h.logger.Warnw("invalid agent_id parameter",
			"agent_id", exitAgentIDStr,
			"entry_agent_id", entryAgentID,
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid agent_id parameter: must be in format fa_xxx")
		return
	}

	// Look up the internal agent ID by SID
	exitAgent, err := h.agentRepo.GetBySID(ctx, exitAgentIDStr)
	if err != nil {
		h.logger.Warnw("exit agent not found",
			"agent_id", exitAgentIDStr,
			"entry_agent_id", entryAgentID,
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusNotFound, "exit agent not found")
		return
	}
	exitAgentID := exitAgent.ID()

	h.logger.Infow("forward client requesting exit endpoint information",
		"exit_agent_id", exitAgentID,
		"entry_agent_id", entryAgentID,
		"ip", c.ClientIP(),
	)

	// Verify that the entry agent has an entry rule pointing to this exit agent
	entryRules, err := h.repo.ListByAgentID(ctx, entryAgentID)
	if err != nil {
		h.logger.Errorw("failed to get entry agent rules",
			"entry_agent_id", entryAgentID,
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to validate access")
		return
	}

	// Check if any entry or chain rule points to the requested exit agent
	hasAccess := false
	for _, rule := range entryRules {
		// Entry rule: entry agent connects to exit agent
		if rule.RuleType().String() == "entry" && rule.ExitAgentID() == exitAgentID {
			hasAccess = true
			break
		}
		// Chain rule: agent connects to its next hop in the chain
		if rule.RuleType().String() == "chain" {
			nextHopID := rule.GetNextHopAgentID(entryAgentID)
			if nextHopID == exitAgentID {
				hasAccess = true
				break
			}
		}
	}

	if !hasAccess {
		h.logger.Warnw("entry agent not authorized to access exit endpoint",
			"entry_agent_id", entryAgentID,
			"exit_agent_id", exitAgentID,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusForbidden, "access denied")
		return
	}

	// exitAgent was already retrieved by GetBySID above
	if exitAgent == nil {
		h.logger.Warnw("forward agent not found",
			"exit_agent_id", exitAgentID,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusNotFound, "forward agent not found")
		return
	}

	// Check if exit agent has an address (tunnel_address or public_address)
	if exitAgent.GetEffectiveTunnelAddress() == "" {
		h.logger.Warnw("exit agent has no address configured",
			"exit_agent_id", exitAgentID,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusNotFound, "agent has no address configured")
		return
	}

	// Get exit agent status from cache to retrieve ws_listen_port
	exitStatus, err := h.statusQuerier.GetStatus(ctx, exitAgentID)
	if err != nil {
		h.logger.Errorw("failed to get exit agent status",
			"exit_agent_id", exitAgentID,
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to retrieve exit agent status")
		return
	}

	if exitStatus == nil || exitStatus.WsListenPort == 0 {
		h.logger.Debugw("exit agent has no ws_listen_port configured or is offline",
			"exit_agent_id", exitAgentID,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusNotFound, "exit agent is offline or has no ws_listen_port configured")
		return
	}

	// Use effective tunnel address (prefers tunnel_address over public_address)
	address := exitAgent.GetEffectiveTunnelAddress()

	h.logger.Infow("exit endpoint information retrieved successfully",
		"exit_agent_id", exitAgentID,
		"entry_agent_id", entryAgentID,
		"address", address,
		"ws_port", exitStatus.WsListenPort,
		"ip", c.ClientIP(),
	)

	// Return the connection information
	// Note: connection_token is no longer needed as agents use HMAC-based agent tokens for verification
	utils.SuccessResponse(c, http.StatusOK, "exit endpoint information retrieved successfully", map[string]any{
		"address": address,
		"ws_port": exitStatus.WsListenPort,
	})
}

// resolveNodeAddress selects the appropriate node address based on IP version preference.
// ipVersion: "auto", "ipv4", or "ipv6"
func (h *AgentHandler) resolveNodeAddress(targetNode *node.Node, ipVersion string) string {
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

// VerifyTunnelHandshakeRequest represents a request to verify a tunnel handshake.
// This is used by exit agents to verify incoming tunnel connections from entry agents.
type VerifyTunnelHandshakeRequest struct {
	AgentToken string `json:"agent_token" binding:"required"` // Entry agent's token (fwd_xxx_xxx format)
	RuleID     string `json:"rule_id" binding:"required"`     // Rule ID (Stripe-style, e.g., "fr_xK9mP2vL3nQ")
}

// VerifyTunnelHandshakeResponse represents the result of tunnel handshake verification.
type VerifyTunnelHandshakeResponse struct {
	Success      bool   `json:"success"`
	Error        string `json:"error,omitempty"`
	EntryAgentID string `json:"entry_agent_id,omitempty"` // Verified entry agent ID (e.g., "fa_xK9mP2vL3nQ")
}

// ReportStatusRequest represents status report request from forward client
type ReportStatusRequest struct {
	CPUPercent        float64           `json:"cpu_percent"`
	MemoryPercent     float64           `json:"memory_percent"`
	MemoryUsed        uint64            `json:"memory_used"`
	MemoryTotal       uint64            `json:"memory_total"`
	DiskPercent       float64           `json:"disk_percent"`
	DiskUsed          uint64            `json:"disk_used"`
	DiskTotal         uint64            `json:"disk_total"`
	UptimeSeconds     int64             `json:"uptime_seconds"`
	TCPConnections    int               `json:"tcp_connections"`
	UDPConnections    int               `json:"udp_connections"`
	ActiveRules       int               `json:"active_rules"`
	ActiveConnections int               `json:"active_connections"`
	TunnelStatus      map[string]string `json:"tunnel_status,omitempty"`  // Key is Stripe-style rule ID (e.g., "fr_xK9mP2vL3nQ")
	WsListenPort      uint16            `json:"ws_listen_port,omitempty"` // WebSocket listen port for exit agent tunnel connections
}

// ReportStatus handles POST /forward-agent-api/status
func (h *AgentHandler) ReportStatus(c *gin.Context) {
	ctx := c.Request.Context()

	// Get agent ID from context (set by auth middleware)
	agentID, exists := c.Get("forward_agent_id")
	if !exists {
		h.logger.Warnw("forward_agent_id not found in context",
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Parse request body
	var req ReportStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid status report request body",
			"error", err,
			"agent_id", agentID,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	h.logger.Debugw("forward agent status report received",
		"agent_id", agentID,
		"cpu", req.CPUPercent,
		"memory", req.MemoryPercent,
		"active_rules", req.ActiveRules,
		"ip", c.ClientIP(),
	)

	// Convert request to DTO
	statusDTO := &dto.AgentStatusDTO{
		CPUPercent:        req.CPUPercent,
		MemoryPercent:     req.MemoryPercent,
		MemoryUsed:        req.MemoryUsed,
		MemoryTotal:       req.MemoryTotal,
		DiskPercent:       req.DiskPercent,
		DiskUsed:          req.DiskUsed,
		DiskTotal:         req.DiskTotal,
		UptimeSeconds:     req.UptimeSeconds,
		TCPConnections:    req.TCPConnections,
		UDPConnections:    req.UDPConnections,
		ActiveRules:       req.ActiveRules,
		ActiveConnections: req.ActiveConnections,
		TunnelStatus:      req.TunnelStatus,
		WsListenPort:      req.WsListenPort,
	}

	// Execute use case
	input := &dto.ReportAgentStatusInput{
		AgentID: agentID.(uint),
		Status:  statusDTO,
	}

	if err := h.reportStatusUC.Execute(ctx, input); err != nil {
		h.logger.Errorw("failed to report agent status",
			"error", err,
			"agent_id", agentID,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusInternalServerError, "failed to report status")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "status reported successfully", nil)
}

// VerifyTunnelHandshake handles POST /forward-agent-api/verify-tunnel-handshake
// This endpoint allows exit agents to verify tunnel handshake requests from entry agents
// by validating the entry agent's token and checking rule access permissions.
func (h *AgentHandler) VerifyTunnelHandshake(c *gin.Context) {
	ctx := c.Request.Context()

	// Get authenticated agent ID (exit agent making the request)
	exitAgentID, err := h.getAuthenticatedAgentID(c)
	if err != nil {
		h.logger.Warnw("failed to get authenticated agent ID",
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Parse request body
	var req VerifyTunnelHandshakeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid tunnel handshake verification request",
			"error", err,
			"exit_agent_id", exitAgentID,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid request body")
		return
	}

	h.logger.Infow("verifying tunnel handshake request",
		"exit_agent_id", exitAgentID,
		"rule_id", req.RuleID,
		"ip", c.ClientIP(),
	)

	// Parse rule ID (Stripe-style ID like "fr_xK9mP2vL3nQ")
	ruleShortID, err := id.ParseForwardRuleID(req.RuleID)
	if err != nil {
		h.logger.Warnw("invalid rule_id in handshake verification",
			"rule_id", req.RuleID,
			"exit_agent_id", exitAgentID,
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid rule_id format")
		return
	}

	// Look up the rule by short ID
	rule, err := h.repo.GetBySID(ctx, ruleShortID)
	if err != nil {
		h.logger.Warnw("rule not found for handshake verification",
			"rule_id", req.RuleID,
			"short_id", ruleShortID,
			"exit_agent_id", exitAgentID,
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusNotFound, "rule not found")
		return
	}

	// Verify that the requesting agent (exit agent) has access to this rule
	hasExitAccess := false
	ruleType := rule.RuleType().String()

	switch ruleType {
	case "entry":
		// For entry rules, exit agent must be the exit_agent_id
		if rule.ExitAgentID() == exitAgentID {
			hasExitAccess = true
		}
	case "chain", "direct_chain":
		// For chain rules, exit agent must be in the chain
		chainPosition := rule.GetChainPosition(exitAgentID)
		if chainPosition > 0 {
			hasExitAccess = true
		}
	}

	if !hasExitAccess {
		h.logger.Warnw("exit agent not authorized for rule",
			"rule_id", req.RuleID,
			"exit_agent_id", exitAgentID,
			"rule_type", ruleType,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusForbidden, "access denied")
		return
	}

	// Verify the entry agent's token using agentTokenService
	entryAgentShortID, err := h.agentTokenService.Verify(req.AgentToken)
	if err != nil {
		h.logger.Warnw("invalid entry agent token in handshake",
			"rule_id", req.RuleID,
			"exit_agent_id", exitAgentID,
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.SuccessResponse(c, http.StatusOK, "handshake verification completed", VerifyTunnelHandshakeResponse{
			Success: false,
			Error:   "invalid token",
		})
		return
	}

	// Use the entry agent SID directly (already contains prefix)
	entryAgentIDStr := entryAgentShortID

	// Verify that the entry agent has permission to access this rule
	hasEntryAccess := false

	switch ruleType {
	case "entry":
		// For entry rules, entry agent must be the owner (agent_id)
		entryAgent, err := h.agentRepo.GetBySID(ctx, entryAgentShortID)
		if err == nil && entryAgent != nil && entryAgent.ID() == rule.AgentID() {
			hasEntryAccess = true
		}

	case "chain", "direct_chain":
		// For chain rules, entry agent must be the previous hop in the chain
		// Full chain: [entry_agent] + chain_agents
		fullChainIDs := append([]uint{rule.AgentID()}, rule.ChainAgentIDs()...)

		// Find exit agent's position
		exitPosition := rule.GetChainPosition(exitAgentID)
		if exitPosition > 0 && exitPosition <= len(fullChainIDs) {
			// Get the previous agent ID (should match entry agent)
			prevAgentID := fullChainIDs[exitPosition-1]

			// Lookup entry agent by short ID to get internal ID
			entryAgent, err := h.agentRepo.GetBySID(ctx, entryAgentShortID)
			if err == nil && entryAgent != nil && entryAgent.ID() == prevAgentID {
				hasEntryAccess = true
			}
		}
	}

	if !hasEntryAccess {
		h.logger.Warnw("entry agent not authorized for rule",
			"rule_id", req.RuleID,
			"entry_agent_id", entryAgentIDStr,
			"exit_agent_id", exitAgentID,
			"rule_type", ruleType,
			"ip", c.ClientIP(),
		)
		utils.SuccessResponse(c, http.StatusOK, "handshake verification completed", VerifyTunnelHandshakeResponse{
			Success: false,
			Error:   "access denied",
		})
		return
	}

	// Handshake verification successful
	h.logger.Infow("tunnel handshake verified successfully",
		"rule_id", req.RuleID,
		"entry_agent_id", entryAgentIDStr,
		"exit_agent_id", exitAgentID,
		"rule_type", ruleType,
		"ip", c.ClientIP(),
	)

	utils.SuccessResponse(c, http.StatusOK, "handshake verification completed", VerifyTunnelHandshakeResponse{
		Success:      true,
		EntryAgentID: entryAgentIDStr,
	})
}
