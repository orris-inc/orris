// Package api provides HTTP handlers for forward agent REST API.
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// VerifyTunnelHandshakeRequest represents a request to verify a tunnel handshake.
// This is used by exit agents to verify incoming tunnel connections from entry agents.
type VerifyTunnelHandshakeRequest struct {
	AgentToken string `json:"agent_token" binding:"required"` // Entry agent's token (fwd_xxx_xxx format)
	RuleID     string `json:"rule_id" binding:"required"`     // Rule ID (Stripe-style, e.g., "fr_xK9mP2vL3nQ")
	IsProbe    bool   `json:"is_probe,omitempty"`             // True if this is a probe connection (tunnel ping)
}

// VerifyTunnelHandshakeResponse represents the result of tunnel handshake verification.
type VerifyTunnelHandshakeResponse struct {
	Success      bool   `json:"success"`
	Error        string `json:"error,omitempty"`
	EntryAgentID string `json:"entry_agent_id,omitempty"` // Verified entry agent ID (e.g., "fa_xK9mP2vL3nQ")
}

// GetExitEndpoint handles GET /forward-agent-api/exit-endpoint/:agent_id
// This endpoint allows an entry agent to get the exit endpoint information
// for establishing tunnel connections. Access is restricted to entry agents
// that have an entry rule pointing to the requested exit agent.
func (h *Handler) GetExitEndpoint(c *gin.Context) {
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
		h.logger.Warnw("failed to get exit agent",
			"agent_id", exitAgentIDStr,
			"entry_agent_id", entryAgentID,
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusNotFound, "exit agent not found")
		return
	}
	if exitAgent == nil {
		h.logger.Warnw("exit agent not found",
			"agent_id", exitAgentIDStr,
			"entry_agent_id", entryAgentID,
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

	if exitStatus == nil || (exitStatus.WsListenPort == 0 && exitStatus.TlsListenPort == 0) {
		h.logger.Debugw("exit agent has no tunnel port configured or is offline",
			"exit_agent_id", exitAgentID,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusNotFound, "exit agent is offline or has no tunnel port configured")
		return
	}

	// Use effective tunnel address (prefers tunnel_address over public_address)
	address := exitAgent.GetEffectiveTunnelAddress()

	h.logger.Infow("exit endpoint information retrieved successfully",
		"exit_agent_id", exitAgentID,
		"entry_agent_id", entryAgentID,
		"address", address,
		"ws_port", exitStatus.WsListenPort,
		"tls_port", exitStatus.TlsListenPort,
		"ip", c.ClientIP(),
	)

	// Return the connection information
	// Note: connection_token is no longer needed as agents use HMAC-based agent tokens for verification
	utils.SuccessResponse(c, http.StatusOK, "exit endpoint information retrieved successfully", map[string]any{
		"address":  address,
		"ws_port":  exitStatus.WsListenPort,
		"tls_port": exitStatus.TlsListenPort,
	})
}

// VerifyTunnelHandshake handles POST /forward-agent-api/verify-tunnel-handshake
// This endpoint allows exit agents to verify tunnel handshake requests from entry agents
// by validating the entry agent's token and checking rule access permissions.
func (h *Handler) VerifyTunnelHandshake(c *gin.Context) {
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

	// Validate rule ID format (Stripe-style ID like "fr_xK9mP2vL3nQ")
	if err := id.ValidatePrefix(req.RuleID, id.PrefixForwardRule); err != nil {
		h.logger.Warnw("invalid rule_id in handshake verification",
			"rule_id", req.RuleID,
			"exit_agent_id", exitAgentID,
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid rule_id format")
		return
	}

	// Look up the rule by SID (database stores full prefixed ID like "fr_xxx")
	rule, err := h.repo.GetBySID(ctx, req.RuleID)
	if err != nil {
		h.logger.Warnw("failed to get rule for handshake verification",
			"rule_id", req.RuleID,
			"exit_agent_id", exitAgentID,
			"error", err,
			"ip", c.ClientIP(),
		)
		utils.ErrorResponse(c, http.StatusNotFound, "rule not found")
		return
	}
	if rule == nil {
		h.logger.Warnw("rule not found for handshake verification",
			"rule_id", req.RuleID,
			"exit_agent_id", exitAgentID,
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
