// Package externalforward provides HTTP handlers for external forward rule management.
package externalforward

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/externalforward/usecases"
	"github.com/orris-inc/orris/internal/shared/constants"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// Handler handles HTTP requests for external forward rules.
type Handler struct {
	createRuleUC  *usecases.CreateExternalForwardRuleUseCase
	updateRuleUC  *usecases.UpdateExternalForwardRuleUseCase
	deleteRuleUC  *usecases.DeleteExternalForwardRuleUseCase
	listRulesUC   *usecases.ListExternalForwardRulesUseCase
	getRuleUC     *usecases.GetExternalForwardRuleUseCase
	enableRuleUC  *usecases.EnableExternalForwardRuleUseCase
	disableRuleUC *usecases.DisableExternalForwardRuleUseCase
	logger        logger.Interface
}

// NewHandler creates a new external forward Handler.
func NewHandler(
	createRuleUC *usecases.CreateExternalForwardRuleUseCase,
	updateRuleUC *usecases.UpdateExternalForwardRuleUseCase,
	deleteRuleUC *usecases.DeleteExternalForwardRuleUseCase,
	listRulesUC *usecases.ListExternalForwardRulesUseCase,
	getRuleUC *usecases.GetExternalForwardRuleUseCase,
	enableRuleUC *usecases.EnableExternalForwardRuleUseCase,
	disableRuleUC *usecases.DisableExternalForwardRuleUseCase,
) *Handler {
	return &Handler{
		createRuleUC:  createRuleUC,
		updateRuleUC:  updateRuleUC,
		deleteRuleUC:  deleteRuleUC,
		listRulesUC:   listRulesUC,
		getRuleUC:     getRuleUC,
		enableRuleUC:  enableRuleUC,
		disableRuleUC: disableRuleUC,
		logger:        logger.NewLogger(),
	}
}

// CreateRuleRequest represents a request to create an external forward rule.
type CreateRuleRequest struct {
	Name           string `json:"name" binding:"required" example:"MySQL-Forward"`
	ServerAddress  string `json:"server_address" binding:"required" example:"192.168.1.100"`
	ListenPort     uint16 `json:"listen_port" binding:"required" example:"13306"`
	ExternalSource string `json:"external_source" binding:"required" example:"my-panel"`
	ExternalRuleID string `json:"external_rule_id,omitempty" example:"rule-123"`
	NodeID         string `json:"node_id,omitempty" example:"node_xxxxx"`
	Remark         string `json:"remark,omitempty" example:"Forward to internal server"`
	SortOrder      int    `json:"sort_order,omitempty" example:"100"`
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
	prefixedID := c.Param("rule_id")
	if prefixedID == "" {
		return "", errors.NewValidationError("external forward rule ID is required")
	}

	if err := id.ValidatePrefix(prefixedID, id.PrefixExternalForwardRule); err != nil {
		return "", errors.NewValidationError("invalid external forward rule ID format, expected efr_xxxxx")
	}

	return prefixedID, nil
}

// getSubscriptionIDFromContext retrieves subscription_id from context.
func getSubscriptionIDFromContext(c *gin.Context, log logger.Interface) (uint, error) {
	subscriptionIDInterface, exists := c.Get("subscription_id")
	if !exists {
		log.Warnw("subscription_id not found in context", "ip", c.ClientIP())
		return 0, errors.NewUnauthorizedError("subscription context not available")
	}

	subscriptionID, ok := subscriptionIDInterface.(uint)
	if !ok {
		log.Warnw("invalid subscription_id type in context", "subscription_id", subscriptionIDInterface, "ip", c.ClientIP())
		return 0, errors.NewInternalError("invalid subscription ID type")
	}

	return subscriptionID, nil
}

// getSubscriptionSIDFromContext retrieves subscription_sid from context.
func getSubscriptionSIDFromContext(c *gin.Context) string {
	if sid, exists := c.Get("subscription_sid"); exists {
		if sidStr, ok := sid.(string); ok {
			return sidStr
		}
	}
	// Fallback to URL parameter
	return c.Param("sid")
}

// getUserIDFromContext retrieves user_id from context.
func getUserIDFromContext(c *gin.Context, log logger.Interface) (uint, error) {
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		log.Warnw("user_id not found in context", "ip", c.ClientIP())
		return 0, errors.NewUnauthorizedError("user not authenticated")
	}

	userID, ok := userIDInterface.(uint)
	if !ok {
		log.Warnw("invalid user_id type in context", "user_id", userIDInterface, "ip", c.ClientIP())
		return 0, errors.NewInternalError("invalid user ID type")
	}

	return userID, nil
}

// CreateRule handles POST /subscriptions/:sid/external-forward-rules
func (h *Handler) CreateRule(c *gin.Context) {
	userID, err := getUserIDFromContext(c, h.logger)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	subscriptionID, err := getSubscriptionIDFromContext(c, h.logger)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	subscriptionSID := getSubscriptionSIDFromContext(c)

	var req CreateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for create external forward rule", "subscription_id", subscriptionID, "error", err)
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

	cmd := usecases.CreateExternalForwardRuleCommand{
		SubscriptionID:  subscriptionID,
		SubscriptionSID: subscriptionSID,
		UserID:          userID,
		NodeSID:         req.NodeID,
		Name:            req.Name,
		ServerAddress:   req.ServerAddress,
		ListenPort:      req.ListenPort,
		ExternalSource:  req.ExternalSource,
		ExternalRuleID:  req.ExternalRuleID,
		Remark:          req.Remark,
		SortOrder:       req.SortOrder,
	}

	result, err := h.createRuleUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.CreatedResponse(c, result.Rule, "External forward rule created successfully")
}

// ListRules handles GET /subscriptions/:sid/external-forward-rules
func (h *Handler) ListRules(c *gin.Context) {
	subscriptionID, err := getSubscriptionIDFromContext(c, h.logger)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	subscriptionSID := getSubscriptionSIDFromContext(c)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	if page < 1 {
		page = 1
	}

	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", strconv.Itoa(constants.DefaultPageSize)))
	if pageSize < 1 || pageSize > constants.MaxPageSize {
		pageSize = constants.DefaultPageSize
	}

	query := usecases.ListExternalForwardRulesQuery{
		SubscriptionID:  subscriptionID,
		SubscriptionSID: subscriptionSID,
		Page:            page,
		PageSize:        pageSize,
		Status:          c.Query("status"),
		OrderBy:         c.Query("order_by"),
		Order:           c.Query("order"),
	}

	result, err := h.listRulesUC.Execute(c.Request.Context(), query)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.ListSuccessResponse(c, result.Rules, result.Total, page, pageSize)
}

// GetRule handles GET /subscriptions/:sid/external-forward-rules/:rule_id
func (h *Handler) GetRule(c *gin.Context) {
	sid, err := parseRuleSID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	subscriptionID, err := getSubscriptionIDFromContext(c, h.logger)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	subscriptionSID := getSubscriptionSIDFromContext(c)

	query := usecases.GetExternalForwardRuleQuery{
		SID:             sid,
		SubscriptionID:  subscriptionID,
		SubscriptionSID: subscriptionSID,
	}

	result, err := h.getRuleUC.Execute(c.Request.Context(), query)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result.Rule)
}

// UpdateRule handles PUT /subscriptions/:sid/external-forward-rules/:rule_id
func (h *Handler) UpdateRule(c *gin.Context) {
	sid, err := parseRuleSID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	subscriptionID, err := getSubscriptionIDFromContext(c, h.logger)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	subscriptionSID := getSubscriptionSIDFromContext(c)

	var req UpdateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for update external forward rule", "sid", sid, "error", err)
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

	cmd := usecases.UpdateExternalForwardRuleCommand{
		SID:             sid,
		SubscriptionID:  subscriptionID,
		SubscriptionSID: subscriptionSID,
		NodeSID:         req.NodeID,
		Name:            req.Name,
		ServerAddress:   req.ServerAddress,
		ListenPort:      req.ListenPort,
		Remark:          req.Remark,
		SortOrder:       req.SortOrder,
	}

	if err := h.updateRuleUC.Execute(c.Request.Context(), cmd); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "External forward rule updated successfully", nil)
}

// DeleteRule handles DELETE /subscriptions/:sid/external-forward-rules/:rule_id
func (h *Handler) DeleteRule(c *gin.Context) {
	sid, err := parseRuleSID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	subscriptionID, err := getSubscriptionIDFromContext(c, h.logger)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.DeleteExternalForwardRuleCommand{
		SID:            sid,
		SubscriptionID: subscriptionID,
	}
	if err := h.deleteRuleUC.Execute(c.Request.Context(), cmd); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.NoContentResponse(c)
}

// EnableRule handles POST /subscriptions/:sid/external-forward-rules/:rule_id/enable
func (h *Handler) EnableRule(c *gin.Context) {
	sid, err := parseRuleSID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	subscriptionID, err := getSubscriptionIDFromContext(c, h.logger)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.EnableExternalForwardRuleCommand{
		SID:            sid,
		SubscriptionID: subscriptionID,
	}
	if err := h.enableRuleUC.Execute(c.Request.Context(), cmd); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "External forward rule enabled successfully", nil)
}

// DisableRule handles POST /subscriptions/:sid/external-forward-rules/:rule_id/disable
func (h *Handler) DisableRule(c *gin.Context) {
	sid, err := parseRuleSID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	subscriptionID, err := getSubscriptionIDFromContext(c, h.logger)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.DisableExternalForwardRuleCommand{
		SID:            sid,
		SubscriptionID: subscriptionID,
	}
	if err := h.disableRuleUC.Execute(c.Request.Context(), cmd); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "External forward rule disabled successfully", nil)
}
