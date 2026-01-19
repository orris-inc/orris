// Package admin provides HTTP handlers for administrative operations.
package admin

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	adminUsecases "github.com/orris-inc/orris/internal/application/externalforward/usecases/admin"
	"github.com/orris-inc/orris/internal/shared/constants"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// ExternalForwardRuleHandler handles admin external forward rule operations.
type ExternalForwardRuleHandler struct {
	createUC  *adminUsecases.AdminCreateExternalForwardRuleUseCase
	listUC    *adminUsecases.AdminListExternalForwardRulesUseCase
	getUC     *adminUsecases.AdminGetExternalForwardRuleUseCase
	updateUC  *adminUsecases.AdminUpdateExternalForwardRuleUseCase
	deleteUC  *adminUsecases.AdminDeleteExternalForwardRuleUseCase
	enableUC  *adminUsecases.AdminEnableExternalForwardRuleUseCase
	disableUC *adminUsecases.AdminDisableExternalForwardRuleUseCase
	logger    logger.Interface
}

// NewExternalForwardRuleHandler creates a new admin external forward rule handler.
func NewExternalForwardRuleHandler(
	createUC *adminUsecases.AdminCreateExternalForwardRuleUseCase,
	listUC *adminUsecases.AdminListExternalForwardRulesUseCase,
	getUC *adminUsecases.AdminGetExternalForwardRuleUseCase,
	updateUC *adminUsecases.AdminUpdateExternalForwardRuleUseCase,
	deleteUC *adminUsecases.AdminDeleteExternalForwardRuleUseCase,
	enableUC *adminUsecases.AdminEnableExternalForwardRuleUseCase,
	disableUC *adminUsecases.AdminDisableExternalForwardRuleUseCase,
	logger logger.Interface,
) *ExternalForwardRuleHandler {
	return &ExternalForwardRuleHandler{
		createUC:  createUC,
		listUC:    listUC,
		getUC:     getUC,
		updateUC:  updateUC,
		deleteUC:  deleteUC,
		enableUC:  enableUC,
		disableUC: disableUC,
		logger:    logger,
	}
}

// CreateRuleRequest represents a request to create an external forward rule.
type CreateRuleRequest struct {
	Name           string   `json:"name" binding:"required" example:"MySQL-Forward"`
	ServerAddress  string   `json:"server_address" binding:"required" example:"192.168.1.100"`
	ListenPort     uint16   `json:"listen_port" binding:"required" example:"13306"`
	ExternalSource string   `json:"external_source" binding:"required" example:"provider_abc"`
	ExternalRuleID string   `json:"external_rule_id,omitempty" example:"rule_123"`
	NodeID         string   `json:"node_id,omitempty" example:"node_xxxxx"`
	Remark         string   `json:"remark,omitempty" example:"MySQL database forward"`
	SortOrder      int      `json:"sort_order,omitempty" example:"0"`
	GroupSIDs      []string `json:"group_ids,omitempty" example:"rg_xxxxx"`
}

// UpdateRuleRequest represents a request to update an external forward rule.
type UpdateRuleRequest struct {
	Name          *string `json:"name,omitempty" example:"MySQL-Forward-Updated"`
	ServerAddress *string `json:"server_address,omitempty" example:"192.168.1.101"`
	ListenPort    *uint16 `json:"listen_port,omitempty" example:"13307"`
	NodeID        *string `json:"node_id,omitempty" example:"node_xxxxx"`
	Remark        *string `json:"remark,omitempty" example:"Updated remark"`
	SortOrder     *int    `json:"sort_order,omitempty" example:"100"`
}

// parseRuleSID validates a prefixed rule ID and returns the SID.
func parseRuleSID(c *gin.Context) (string, error) {
	prefixedID := c.Param("id")
	if prefixedID == "" {
		return "", errors.NewValidationError("external forward rule ID is required")
	}

	if err := id.ValidatePrefix(prefixedID, id.PrefixExternalForwardRule); err != nil {
		return "", errors.NewValidationError("invalid external forward rule ID format, expected efr_xxxxx")
	}

	return prefixedID, nil
}

// Create handles POST /admin/external-forward-rules
func (h *ExternalForwardRuleHandler) Create(c *gin.Context) {
	var req CreateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for admin create external forward rule", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Validate node_id format if provided
	if req.NodeID != "" {
		if err := id.ValidatePrefix(req.NodeID, id.PrefixNode); err != nil {
			utils.ErrorResponseWithError(c, errors.NewValidationError("invalid node_id format, expected node_xxxxx"))
			return
		}
	}

	cmd := adminUsecases.AdminCreateExternalForwardRuleCommand{
		Name:           req.Name,
		ServerAddress:  req.ServerAddress,
		ListenPort:     req.ListenPort,
		ExternalSource: req.ExternalSource,
		ExternalRuleID: req.ExternalRuleID,
		NodeSID:        req.NodeID,
		Remark:         req.Remark,
		SortOrder:      req.SortOrder,
		GroupSIDs:      req.GroupSIDs,
	}

	result, err := h.createUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusCreated, "External forward rule created successfully", result.Rule)
}

// List handles GET /admin/external-forward-rules
func (h *ExternalForwardRuleHandler) List(c *gin.Context) {
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		p, err := strconv.Atoi(pageStr)
		if err != nil || p <= 0 {
			utils.ErrorResponseWithError(c, errors.NewValidationError("invalid page parameter"))
			return
		}
		page = p
	}

	pageSize := constants.DefaultPageSize
	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		ps, err := strconv.Atoi(pageSizeStr)
		if err != nil || ps <= 0 || ps > constants.MaxPageSize {
			utils.ErrorResponseWithError(c, errors.NewValidationError("invalid page_size parameter, must be 1-100"))
			return
		}
		pageSize = ps
	}

	var subscriptionID *uint
	if subIDStr := c.Query("subscription_id"); subIDStr != "" {
		sid, err := strconv.ParseUint(subIDStr, 10, 64)
		if err != nil {
			utils.ErrorResponseWithError(c, errors.NewValidationError("invalid subscription_id parameter"))
			return
		}
		sidVal := uint(sid)
		subscriptionID = &sidVal
	}

	var userID *uint
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		uid, err := strconv.ParseUint(userIDStr, 10, 64)
		if err != nil {
			utils.ErrorResponseWithError(c, errors.NewValidationError("invalid user_id parameter"))
			return
		}
		uidVal := uint(uid)
		userID = &uidVal
	}

	query := adminUsecases.AdminListExternalForwardRulesQuery{
		Page:           page,
		PageSize:       pageSize,
		SubscriptionID: subscriptionID,
		UserID:         userID,
		Status:         c.Query("status"),
		ExternalSource: c.Query("external_source"),
		OrderBy:        c.Query("order_by"),
		Order:          c.Query("order"),
	}

	result, err := h.listUC.Execute(c.Request.Context(), query)
	if err != nil {
		h.logger.Errorw("failed to list external forward rules", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.ListSuccessResponse(c, result.Rules, result.Total, page, pageSize)
}

// Get handles GET /admin/external-forward-rules/:id
func (h *ExternalForwardRuleHandler) Get(c *gin.Context) {
	sid, err := parseRuleSID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	query := adminUsecases.AdminGetExternalForwardRuleQuery{
		SID: sid,
	}

	result, err := h.getUC.Execute(c.Request.Context(), query)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result.Rule)
}

// Update handles PUT /admin/external-forward-rules/:id
func (h *ExternalForwardRuleHandler) Update(c *gin.Context) {
	sid, err := parseRuleSID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	var req UpdateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for admin update external forward rule", "sid", sid, "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Validate node_id format if provided and not empty (empty string clears the node)
	if req.NodeID != nil && *req.NodeID != "" {
		if err := id.ValidatePrefix(*req.NodeID, id.PrefixNode); err != nil {
			utils.ErrorResponseWithError(c, errors.NewValidationError("invalid node_id format, expected node_xxxxx"))
			return
		}
	}

	cmd := adminUsecases.AdminUpdateExternalForwardRuleCommand{
		SID:           sid,
		NodeSID:       req.NodeID,
		Name:          req.Name,
		ServerAddress: req.ServerAddress,
		ListenPort:    req.ListenPort,
		Remark:        req.Remark,
		SortOrder:     req.SortOrder,
	}

	if err := h.updateUC.Execute(c.Request.Context(), cmd); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "External forward rule updated successfully", nil)
}

// Delete handles DELETE /admin/external-forward-rules/:id
func (h *ExternalForwardRuleHandler) Delete(c *gin.Context) {
	sid, err := parseRuleSID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := adminUsecases.AdminDeleteExternalForwardRuleCommand{
		SID: sid,
	}

	if err := h.deleteUC.Execute(c.Request.Context(), cmd); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.NoContentResponse(c)
}

// Enable handles POST /admin/external-forward-rules/:id/enable
func (h *ExternalForwardRuleHandler) Enable(c *gin.Context) {
	sid, err := parseRuleSID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := adminUsecases.AdminEnableExternalForwardRuleCommand{
		SID: sid,
	}

	if err := h.enableUC.Execute(c.Request.Context(), cmd); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "External forward rule enabled successfully", nil)
}

// Disable handles POST /admin/external-forward-rules/:id/disable
func (h *ExternalForwardRuleHandler) Disable(c *gin.Context) {
	sid, err := parseRuleSID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := adminUsecases.AdminDisableExternalForwardRuleCommand{
		SID: sid,
	}

	if err := h.disableUC.Execute(c.Request.Context(), cmd); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "External forward rule disabled successfully", nil)
}
