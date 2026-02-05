package rule

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/forward/usecases"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// EnableRule handles POST /forward-rules/:id/enable
func (h *Handler) EnableRule(c *gin.Context) {
	shortID, err := utils.ParseSIDParam(c, "id", id.PrefixForwardRule, "forward rule")
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.EnableForwardRuleCommand{ShortID: shortID}
	if err := h.enableRuleUC.Execute(c.Request.Context(), cmd); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Forward rule enabled successfully", nil)
}

// DisableRule handles POST /forward-rules/:id/disable
func (h *Handler) DisableRule(c *gin.Context) {
	shortID, err := utils.ParseSIDParam(c, "id", id.PrefixForwardRule, "forward rule")
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.DisableForwardRuleCommand{ShortID: shortID}
	if err := h.disableRuleUC.Execute(c.Request.Context(), cmd); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Forward rule disabled successfully", nil)
}

// UpdateStatus handles PATCH /forward-rules/:id/status
func (h *Handler) UpdateStatus(c *gin.Context) {
	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for update status", "error", err, "ip", c.ClientIP())
		utils.ErrorResponseWithError(c, err)
		return
	}

	if req.Status == "enabled" {
		h.EnableRule(c)
	} else {
		h.DisableRule(c)
	}
}

// ResetTraffic handles POST /forward-rules/:id/reset-traffic
func (h *Handler) ResetTraffic(c *gin.Context) {
	shortID, err := utils.ParseSIDParam(c, "id", id.PrefixForwardRule, "forward rule")
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.ResetForwardRuleTrafficCommand{ShortID: shortID}
	if err := h.resetTrafficUC.Execute(c.Request.Context(), cmd); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Traffic counters reset successfully", nil)
}

// ProbeRule handles POST /forward-rules/:id/probe
func (h *Handler) ProbeRule(c *gin.Context) {
	shortID, err := utils.ParseSIDParam(c, "id", id.PrefixForwardRule, "forward rule")
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	if h.probeService == nil {
		utils.ErrorResponse(c, http.StatusServiceUnavailable, "Probe service not available")
		return
	}

	// Parse optional request body
	var req ProbeRuleRequest
	// Ignore binding errors for optional body
	_ = c.ShouldBindJSON(&req)

	result, err := h.probeService.ProbeRuleByShortID(c.Request.Context(), shortID, req.IPVersion)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Probe completed", result)
}

// ReorderRules handles PATCH /forward-rules/reorder
func (h *Handler) ReorderRules(c *gin.Context) {
	var req ReorderForwardRulesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for reorder rules", "error", err, "ip", c.ClientIP())
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Validate rule IDs
	ruleOrders := make([]usecases.RuleOrder, len(req.RuleOrders))
	for i, order := range req.RuleOrders {
		if err := id.ValidatePrefix(order.RuleID, id.PrefixForwardRule); err != nil {
			h.logger.Warnw("invalid rule_id format", "rule_id", order.RuleID, "error", err, "ip", c.ClientIP())
			utils.ErrorResponseWithError(c, errors.NewValidationError("invalid rule_id format, expected fr_xxxxx"))
			return
		}
		ruleOrders[i] = usecases.RuleOrder{
			RuleSID:   order.RuleID,
			SortOrder: order.SortOrder,
		}
	}

	cmd := usecases.ReorderForwardRulesCommand{
		RuleOrders: ruleOrders,
	}

	if err := h.reorderRulesUC.Execute(c.Request.Context(), cmd); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.NoContentResponse(c)
}
