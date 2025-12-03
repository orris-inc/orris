// Package forward provides HTTP handlers for forward rule management.
package forward

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/forward/usecases"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// ForwardHandler handles HTTP requests for forward rules.
type ForwardHandler struct {
	createRuleUC   *usecases.CreateForwardRuleUseCase
	getRuleUC      *usecases.GetForwardRuleUseCase
	updateRuleUC   *usecases.UpdateForwardRuleUseCase
	deleteRuleUC   *usecases.DeleteForwardRuleUseCase
	listRulesUC    *usecases.ListForwardRulesUseCase
	enableRuleUC   *usecases.EnableForwardRuleUseCase
	disableRuleUC  *usecases.DisableForwardRuleUseCase
	resetTrafficUC *usecases.ResetForwardRuleTrafficUseCase
	logger         logger.Interface
}

// NewForwardHandler creates a new ForwardHandler.
func NewForwardHandler(
	createRuleUC *usecases.CreateForwardRuleUseCase,
	getRuleUC *usecases.GetForwardRuleUseCase,
	updateRuleUC *usecases.UpdateForwardRuleUseCase,
	deleteRuleUC *usecases.DeleteForwardRuleUseCase,
	listRulesUC *usecases.ListForwardRulesUseCase,
	enableRuleUC *usecases.EnableForwardRuleUseCase,
	disableRuleUC *usecases.DisableForwardRuleUseCase,
	resetTrafficUC *usecases.ResetForwardRuleTrafficUseCase,
) *ForwardHandler {
	return &ForwardHandler{
		createRuleUC:   createRuleUC,
		getRuleUC:      getRuleUC,
		updateRuleUC:   updateRuleUC,
		deleteRuleUC:   deleteRuleUC,
		listRulesUC:    listRulesUC,
		enableRuleUC:   enableRuleUC,
		disableRuleUC:  disableRuleUC,
		resetTrafficUC: resetTrafficUC,
		logger:         logger.NewLogger(),
	}
}

// CreateForwardRuleRequest represents a request to create a forward rule.
type CreateForwardRuleRequest struct {
	Name          string `json:"name" binding:"required" example:"MySQL-Forward"`
	ListenPort    uint16 `json:"listen_port" binding:"required" example:"13306"`
	TargetAddress string `json:"target_address" binding:"required" example:"192.168.1.100"`
	TargetPort    uint16 `json:"target_port" binding:"required" example:"3306"`
	Protocol      string `json:"protocol" binding:"required,oneof=tcp udp both" example:"tcp"`
	Remark        string `json:"remark,omitempty" example:"Forward to internal MySQL server"`
}

// UpdateForwardRuleRequest represents a request to update a forward rule.
type UpdateForwardRuleRequest struct {
	Name          *string `json:"name,omitempty" example:"MySQL-Forward-Updated"`
	ListenPort    *uint16 `json:"listen_port,omitempty" example:"13307"`
	TargetAddress *string `json:"target_address,omitempty" example:"192.168.1.101"`
	TargetPort    *uint16 `json:"target_port,omitempty" example:"3307"`
	Protocol      *string `json:"protocol,omitempty" binding:"omitempty,oneof=tcp udp both" example:"tcp"`
	Remark        *string `json:"remark,omitempty" example:"Updated remark"`
}

// CreateRule handles POST /forward-rules
func (h *ForwardHandler) CreateRule(c *gin.Context) {
	var req CreateForwardRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for create forward rule", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.CreateForwardRuleCommand{
		Name:          req.Name,
		ListenPort:    req.ListenPort,
		TargetAddress: req.TargetAddress,
		TargetPort:    req.TargetPort,
		Protocol:      req.Protocol,
		Remark:        req.Remark,
	}

	result, err := h.createRuleUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.CreatedResponse(c, result, "Forward rule created successfully")
}

// GetRule handles GET /forward-rules/:id
func (h *ForwardHandler) GetRule(c *gin.Context) {
	ruleID, err := parseRuleID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	query := usecases.GetForwardRuleQuery{ID: ruleID}
	result, err := h.getRuleUC.Execute(c.Request.Context(), query)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}

// UpdateRule handles PUT /forward-rules/:id
func (h *ForwardHandler) UpdateRule(c *gin.Context) {
	ruleID, err := parseRuleID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	var req UpdateForwardRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for update forward rule", "rule_id", ruleID, "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.UpdateForwardRuleCommand{
		ID:            ruleID,
		Name:          req.Name,
		ListenPort:    req.ListenPort,
		TargetAddress: req.TargetAddress,
		TargetPort:    req.TargetPort,
		Protocol:      req.Protocol,
		Remark:        req.Remark,
	}

	if err := h.updateRuleUC.Execute(c.Request.Context(), cmd); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Forward rule updated successfully", nil)
}

// DeleteRule handles DELETE /forward-rules/:id
func (h *ForwardHandler) DeleteRule(c *gin.Context) {
	ruleID, err := parseRuleID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.DeleteForwardRuleCommand{ID: ruleID}
	if err := h.deleteRuleUC.Execute(c.Request.Context(), cmd); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.NoContentResponse(c)
}

// ListRules handles GET /forward-rules
func (h *ForwardHandler) ListRules(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := usecases.ListForwardRulesQuery{
		Page:     page,
		PageSize: pageSize,
		Name:     c.Query("name"),
		Protocol: c.Query("protocol"),
		Status:   c.Query("status"),
		OrderBy:  c.Query("order_by"),
		Order:    c.Query("order"),
	}

	result, err := h.listRulesUC.Execute(c.Request.Context(), query)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.ListSuccessResponse(c, result.Rules, result.Total, page, pageSize)
}

// EnableRule handles POST /forward-rules/:id/enable
func (h *ForwardHandler) EnableRule(c *gin.Context) {
	ruleID, err := parseRuleID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.EnableForwardRuleCommand{ID: ruleID}
	if err := h.enableRuleUC.Execute(c.Request.Context(), cmd); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Forward rule enabled successfully", nil)
}

// DisableRule handles POST /forward-rules/:id/disable
func (h *ForwardHandler) DisableRule(c *gin.Context) {
	ruleID, err := parseRuleID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.DisableForwardRuleCommand{ID: ruleID}
	if err := h.disableRuleUC.Execute(c.Request.Context(), cmd); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Forward rule disabled successfully", nil)
}

// UpdateStatusRequest represents a request to update forward rule status.
type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=enabled disabled" example:"enabled"`
}

// UpdateStatus handles PATCH /forward-rules/:id/status
func (h *ForwardHandler) UpdateStatus(c *gin.Context) {
	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for update status", "error", err)
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
func (h *ForwardHandler) ResetTraffic(c *gin.Context) {
	ruleID, err := parseRuleID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.ResetForwardRuleTrafficCommand{ID: ruleID}
	if err := h.resetTrafficUC.Execute(c.Request.Context(), cmd); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Traffic counters reset successfully", nil)
}

func parseRuleID(c *gin.Context) (uint, error) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, errors.NewValidationError("Invalid forward rule ID")
	}
	if id == 0 {
		return 0, errors.NewValidationError("Forward rule ID must be greater than 0")
	}
	return uint(id), nil
}
