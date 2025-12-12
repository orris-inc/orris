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
	logger logger.Interface,
) *AgentHandler {
	return &AgentHandler{
		repo:               repo,
		agentRepo:          agentRepo,
		nodeRepo:           nodeRepo,
		reportStatusUC:     reportStatusUC,
		statusQuerier:      statusQuerier,
		tokenSigningSecret: tokenSigningSecret,
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
		agentMap, err := h.agentRepo.GetShortIDsByIDs(ctx, agentIDs)
		if err != nil {
			h.logger.Warnw("failed to get agent short IDs",
				"agent_ids", agentIDs,
				"error", err,
			)
			// Continue with empty map
			agentMap = make(dto.AgentShortIDMap)
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
					ruleDTO.NextHopAgentID = id.FormatForwardAgentID(exitAgent.ShortID())
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

						h.logger.Debugw("populated next hop info for entry rule",
							"rule_id", ruleDTO.ID,
							"next_hop_agent_id", ruleDTO.NextHopAgentID,
							"next_hop_address", ruleDTO.NextHopAddress,
							"next_hop_ws_port", ruleDTO.NextHopWsPort,
						)
					} else {
						h.logger.Warnw("exit agent has no ws_listen_port configured or is offline",
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

		h.logger.Debugw("processing chain rule for agent",
			"rule_id", ruleDTO.ID,
			"agent_id", agentID,
			"chain_position", chainPosition,
			"is_last", isLast,
		)

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
					ruleDTO.NextHopAgentID = id.FormatForwardAgentID(nextAgent.ShortID())
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

						h.logger.Debugw("populated next hop info for chain rule",
							"rule_id", ruleDTO.ID,
							"next_hop_agent_id", ruleDTO.NextHopAgentID,
							"next_hop_address", ruleDTO.NextHopAddress,
							"next_hop_ws_port", ruleDTO.NextHopWsPort,
						)
					} else {
						h.logger.Warnw("next hop agent has no ws_listen_port configured or is offline",
							"rule_id", ruleDTO.ID,
							"next_hop_agent_id", nextHopAgentID,
						)
					}
				}
			}

			// Clear target info for non-exit agents (minimum info principle)
			ruleDTO.TargetAddress = ""
			ruleDTO.TargetPort = 0

			h.logger.Debugw("cleared target info for non-exit chain agent",
				"rule_id", ruleDTO.ID,
				"agent_id", agentID,
			)
		} else {
			// For exit agents, clear next hop info (minimum info principle)
			ruleDTO.NextHopAgentID = ""
			ruleDTO.NextHopAddress = ""
			ruleDTO.NextHopWsPort = 0

			h.logger.Debugw("cleared next hop info for exit chain agent",
				"rule_id", ruleDTO.ID,
				"agent_id", agentID,
			)
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

		ruleDTO.ChainPosition = chainPosition
		ruleDTO.IsLastInChain = isLast

		h.logger.Debugw("processing direct_chain rule for agent",
			"rule_id", ruleDTO.ID,
			"agent_id", agentID,
			"chain_position", chainPosition,
			"is_last", isLast,
		)

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
			nextHopAgentID, nextHopPort := rule.GetNextHopForDirectChain(agentID)
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
					ruleDTO.NextHopAgentID = id.FormatForwardAgentID(nextAgent.ShortID())
					// Use effective tunnel address (prefers tunnel_address over public_address)
					ruleDTO.NextHopAddress = nextAgent.GetEffectiveTunnelAddress()
					ruleDTO.NextHopPort = nextHopPort

					h.logger.Debugw("populated next hop info for direct_chain rule",
						"rule_id", ruleDTO.ID,
						"next_hop_agent_id", ruleDTO.NextHopAgentID,
						"next_hop_address", ruleDTO.NextHopAddress,
						"next_hop_port", ruleDTO.NextHopPort,
					)
				}
			}

			// Clear target info for non-exit agents (minimum info principle)
			ruleDTO.TargetAddress = ""
			ruleDTO.TargetPort = 0

			h.logger.Debugw("cleared target info for non-exit direct_chain agent",
				"rule_id", ruleDTO.ID,
				"agent_id", agentID,
			)
		} else {
			// For exit agents, clear next hop info (minimum info principle)
			ruleDTO.NextHopAgentID = ""
			ruleDTO.NextHopAddress = ""
			ruleDTO.NextHopPort = 0

			h.logger.Debugw("cleared next hop info for exit direct_chain agent",
				"rule_id", ruleDTO.ID,
				"agent_id", agentID,
			)
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

				h.logger.Debugw("set role=exit for exit agent",
					"rule_id", ruleDTO.ID,
					"agent_id", agentID,
				)
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
		clientToken = requestingAgent.GetAPIToken()
	}

	// Return success response with token signing secret for local verification
	utils.SuccessResponse(c, http.StatusOK, "enabled forward rules retrieved successfully", map[string]any{
		"rules":                ruleDTOs,
		"token_signing_secret": h.tokenSigningSecret,
		"client_token":         clientToken,
	})
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

	// Merge all rules into validRuleIDs map (use rule.ID() to deduplicate)
	validRuleIDs := make(map[string]uint) // Stripe-style ID -> internal uint ID
	for _, rule := range agentRules {
		stripeID := id.FormatForwardRuleID(rule.ShortID())
		validRuleIDs[stripeID] = rule.ID()
	}
	for _, rule := range exitRules {
		stripeID := id.FormatForwardRuleID(rule.ShortID())
		validRuleIDs[stripeID] = rule.ID()
	}
	for _, rule := range chainRules {
		stripeID := id.FormatForwardRuleID(rule.ShortID())
		validRuleIDs[stripeID] = rule.ID()
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

	for _, item := range req.Rules {
		// Validate rule belongs to this agent and get internal ID
		internalID, valid := validRuleIDs[item.RuleID]
		if !valid {
			h.logger.Warnw("traffic report for unauthorized rule",
				"rule_id", item.RuleID,
				"agent_id", agentID,
				"ip", c.ClientIP(),
			)
			deniedCount++
			continue
		}

		// Skip invalid traffic data
		if item.UploadBytes < 0 || item.DownloadBytes < 0 {
			h.logger.Warnw("invalid traffic data for rule",
				"rule_id", item.RuleID,
				"agent_id", agentID,
				"upload", item.UploadBytes,
				"download", item.DownloadBytes,
			)
			errorCount++
			continue
		}

		// Skip if no traffic to report
		if item.UploadBytes == 0 && item.DownloadBytes == 0 {
			continue
		}

		err := h.repo.UpdateTraffic(ctx, internalID, item.UploadBytes, item.DownloadBytes)
		if err != nil {
			h.logger.Errorw("failed to update rule traffic",
				"rule_id", item.RuleID,
				"internal_id", internalID,
				"agent_id", agentID,
				"upload", item.UploadBytes,
				"download", item.DownloadBytes,
				"error", err,
			)
			errorCount++
			continue
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

	// Parse Stripe-style prefixed ID
	shortID, err := id.ParseForwardAgentID(exitAgentIDStr)
	if err != nil {
		h.logger.Warnw("invalid agent_id parameter",
			"agent_id", exitAgentIDStr,
			"entry_agent_id", entryAgentID,
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid agent_id parameter: must be in format fa_xxx")
		return
	}

	// Look up the internal agent ID by short ID
	exitAgent, err := h.agentRepo.GetByShortID(ctx, shortID)
	if err != nil {
		h.logger.Warnw("exit agent not found",
			"agent_id", exitAgentIDStr,
			"short_id", shortID,
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

	// exitAgent was already retrieved by GetByShortID above
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
		h.logger.Warnw("exit agent has no ws_listen_port configured or is offline",
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
			h.logger.Debugw("using IPv6 address per ip_version setting",
				"ipv6", ipv6,
			)
			return ipv6
		}
		if isValidServerAddr {
			return serverAddr
		}
		if ipv4 != "" {
			h.logger.Debugw("falling back to IPv4 (IPv6 not available)",
				"ipv4", ipv4,
			)
			return ipv4
		}

	case "ipv4":
		// Prefer IPv4: ipv4 > server_address > ipv6
		if ipv4 != "" {
			h.logger.Debugw("using IPv4 address per ip_version setting",
				"ipv4", ipv4,
			)
			return ipv4
		}
		if isValidServerAddr {
			return serverAddr
		}
		if ipv6 != "" {
			h.logger.Debugw("falling back to IPv6 (IPv4 not available)",
				"ipv6", ipv6,
			)
			return ipv6
		}

	default: // "auto" or unknown
		// Default priority: server_address > ipv4 > ipv6
		if isValidServerAddr {
			return serverAddr
		}
		if ipv4 != "" {
			h.logger.Debugw("using public IPv4 as fallback",
				"ipv4", ipv4,
			)
			return ipv4
		}
		if ipv6 != "" {
			h.logger.Debugw("using public IPv6 as fallback",
				"ipv6", ipv6,
			)
			return ipv6
		}
	}

	return serverAddr
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
