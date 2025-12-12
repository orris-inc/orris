package agent

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/utils"
)

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
