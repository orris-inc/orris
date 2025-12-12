package agent

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// GetEnabledRules handles GET /forward-agent-api/rules
func (h *Handler) GetEnabledRules(c *gin.Context) {
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

					// Generate connection token for next hop authentication
					nextHopToken, _ := h.agentTokenService.Generate(nextAgent.ShortID())
					ruleDTO.NextHopConnectionToken = nextHopToken

					h.logger.Debugw("populated next hop info for direct_chain rule",
						"rule_id", ruleDTO.ID,
						"next_hop_agent_id", ruleDTO.NextHopAgentID,
						"next_hop_address", ruleDTO.NextHopAddress,
						"next_hop_port", ruleDTO.NextHopPort,
						"next_hop_connection_token", nextHopToken,
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
		// Generate token using agentTokenService to ensure correct format (fwd_xxx_xxx)
		clientToken, _ = h.agentTokenService.Generate(requestingAgent.ShortID())
		h.logger.Infow("generated client token for agent",
			"agent_id", agentID,
			"short_id", requestingAgent.ShortID(),
			"client_token", clientToken,
		)
	}

	// Return success response with token signing secret for local verification
	utils.SuccessResponse(c, http.StatusOK, "enabled forward rules retrieved successfully", map[string]any{
		"rules":                ruleDTOs,
		"token_signing_secret": h.tokenSigningSecret,
		"client_token":         clientToken,
	})
}

// RefreshRule handles GET /forward-agent-api/rules/:rule_id
// This endpoint allows an agent to refresh the configuration for a specific rule.
// It returns the latest next_hop_ws_port and other dynamic configuration.
// This is useful when the agent detects a connection failure to the next hop.
func (h *Handler) RefreshRule(c *gin.Context) {
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
	rule, err := h.repo.GetByShortID(ctx, shortID)
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
		agentMap, err := h.agentRepo.GetShortIDsByIDs(ctx, agentIDs)
		if err != nil {
			h.logger.Warnw("failed to get agent short IDs",
				"agent_ids", agentIDs,
				"error", err,
			)
			agentMap = make(dto.AgentShortIDMap)
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
					ruleDTO.NextHopAgentID = id.FormatForwardAgentID(exitAgent.ShortID())
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
			agentMap, err := h.agentRepo.GetShortIDsByIDs(ctx, fullChainIDs)
			if err == nil {
				ruleDTO.ChainAgentIDs = make([]string, len(fullChainIDs))
				for i, chainAgentID := range fullChainIDs {
					if shortID, ok := agentMap[chainAgentID]; ok {
						ruleDTO.ChainAgentIDs[i] = id.FormatForwardAgentID(shortID)
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
					ruleDTO.NextHopAgentID = id.FormatForwardAgentID(nextAgent.ShortID())
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
			agentMap, err := h.agentRepo.GetShortIDsByIDs(ctx, fullChainIDs)
			if err == nil {
				ruleDTO.ChainAgentIDs = make([]string, len(fullChainIDs))
				for i, chainAgentID := range fullChainIDs {
					if shortID, ok := agentMap[chainAgentID]; ok {
						ruleDTO.ChainAgentIDs[i] = id.FormatForwardAgentID(shortID)
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
			nextHopAgentID, nextHopPort := rule.GetNextHopForDirectChain(agentID)
			if nextHopAgentID != 0 {
				nextAgent, err := h.agentRepo.GetByID(ctx, nextHopAgentID)
				if err == nil && nextAgent != nil {
					ruleDTO.NextHopAgentID = id.FormatForwardAgentID(nextAgent.ShortID())
					ruleDTO.NextHopAddress = nextAgent.GetEffectiveTunnelAddress()
					ruleDTO.NextHopPort = nextHopPort

					// Generate connection token for next hop authentication
					nextHopToken, _ := h.agentTokenService.Generate(nextAgent.ShortID())
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
