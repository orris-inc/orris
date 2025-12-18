package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	subdto "github.com/orris-inc/orris/internal/application/subscription/dto"
	"github.com/orris-inc/orris/internal/application/subscription/usecases"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/constants"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

var (
	_ = subdto.PlanDTO{}
	_ = subdto.PricingOptionDTO{}
)

type PlanHandler struct {
	createPlanUC      *usecases.CreatePlanUseCase
	updatePlanUC      *usecases.UpdatePlanUseCase
	getPlanUC         *usecases.GetPlanUseCase
	listPlansUC       *usecases.ListPlansUseCase
	getPublicPlansUC  *usecases.GetPublicPlansUseCase
	activatePlanUC    *usecases.ActivatePlanUseCase
	deactivatePlanUC  *usecases.DeactivatePlanUseCase
	getPlanPricingsUC *usecases.GetPlanPricingsUseCase
	managePlanNodesUC *usecases.ManagePlanNodesUseCase
	logger            logger.Interface
}

func NewPlanHandler(
	createPlanUC *usecases.CreatePlanUseCase,
	updatePlanUC *usecases.UpdatePlanUseCase,
	getPlanUC *usecases.GetPlanUseCase,
	listPlansUC *usecases.ListPlansUseCase,
	getPublicPlansUC *usecases.GetPublicPlansUseCase,
	activatePlanUC *usecases.ActivatePlanUseCase,
	deactivatePlanUC *usecases.DeactivatePlanUseCase,
	getPlanPricingsUC *usecases.GetPlanPricingsUseCase,
	managePlanNodesUC *usecases.ManagePlanNodesUseCase,
) *PlanHandler {
	return &PlanHandler{
		createPlanUC:      createPlanUC,
		updatePlanUC:      updatePlanUC,
		getPlanUC:         getPlanUC,
		listPlansUC:       listPlansUC,
		getPublicPlansUC:  getPublicPlansUC,
		activatePlanUC:    activatePlanUC,
		deactivatePlanUC:  deactivatePlanUC,
		getPlanPricingsUC: getPlanPricingsUC,
		managePlanNodesUC: managePlanNodesUC,
		logger:            logger.NewLogger(),
	}
}

type CreatePlanRequest struct {
	Name         string                      `json:"name" binding:"required"`
	Slug         string                      `json:"slug" binding:"required"`
	Description  string                      `json:"description"`
	PlanType     string                      `json:"plan_type" binding:"required,oneof=node forward"`
	TrialDays    int                         `json:"trial_days"`
	Limits       map[string]interface{}      `json:"limits"`
	APIRateLimit uint                        `json:"api_rate_limit"`
	MaxUsers     uint                        `json:"max_users"`
	MaxProjects  uint                        `json:"max_projects"`
	IsPublic     bool                        `json:"is_public"`
	SortOrder    int                         `json:"sort_order"`
	Pricings     []subdto.PricingOptionInput `json:"pricings" binding:"required,min=1"`
}

type UpdatePlanRequest struct {
	Description  *string                      `json:"description"`
	Limits       *map[string]interface{}      `json:"limits"`
	APIRateLimit *uint                        `json:"api_rate_limit"`
	MaxUsers     *uint                        `json:"max_users"`
	MaxProjects  *uint                        `json:"max_projects"`
	IsPublic     *bool                        `json:"is_public"`
	SortOrder    *int                         `json:"sort_order"`
	Pricings     *[]subdto.PricingOptionInput `json:"pricings"` // Optional: update pricing options
}

// UpdatePlanStatusRequest represents a unified request for plan status changes
type UpdatePlanStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=active inactive"`
}

func (h *PlanHandler) CreatePlan(c *gin.Context) {
	var req CreatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for create plan", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.CreatePlanCommand{
		Name:         req.Name,
		Slug:         req.Slug,
		Description:  req.Description,
		PlanType:     req.PlanType,
		TrialDays:    req.TrialDays,
		Limits:       req.Limits,
		APIRateLimit: req.APIRateLimit,
		MaxUsers:     req.MaxUsers,
		MaxProjects:  req.MaxProjects,
		IsPublic:     req.IsPublic,
		SortOrder:    req.SortOrder,
		Pricings:     req.Pricings,
	}

	result, err := h.createPlanUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.CreatedResponse(c, result, "Plan created successfully")
}

func (h *PlanHandler) UpdatePlan(c *gin.Context) {
	planID, err := parsePlanID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	var req UpdatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for update plan",
			"plan_id", planID,
			"error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.UpdatePlanCommand{
		PlanID:       planID,
		Description:  req.Description,
		Limits:       req.Limits,
		APIRateLimit: req.APIRateLimit,
		MaxUsers:     req.MaxUsers,
		MaxProjects:  req.MaxProjects,
		IsPublic:     req.IsPublic,
		SortOrder:    req.SortOrder,
		Pricings:     req.Pricings,
	}

	result, err := h.updatePlanUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Plan updated successfully", result)
}

func (h *PlanHandler) GetPlan(c *gin.Context) {
	planID, err := parsePlanID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	result, err := h.getPlanUC.ExecuteByID(c.Request.Context(), planID)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}

func (h *PlanHandler) ListPlans(c *gin.Context) {
	query, err := parseListPlansQuery(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	result, err := h.listPlansUC.Execute(c.Request.Context(), *query)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.ListSuccessResponse(c, result.Plans, result.Total, query.Page, query.PageSize)
}

func (h *PlanHandler) GetPublicPlans(c *gin.Context) {
	result, err := h.getPublicPlansUC.Execute(c.Request.Context())
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}

func (h *PlanHandler) UpdatePlanStatus(c *gin.Context) {
	planID, err := parsePlanID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	var req UpdatePlanStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for update plan status", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	switch req.Status {
	case string(subscription.PlanStatusActive):
		if err := h.activatePlanUC.Execute(c.Request.Context(), planID); err != nil {
			utils.ErrorResponseWithError(c, err)
			return
		}
		utils.SuccessResponse(c, http.StatusOK, "Plan activated successfully", nil)

	case string(subscription.PlanStatusInactive):
		if err := h.deactivatePlanUC.Execute(c.Request.Context(), planID); err != nil {
			utils.ErrorResponseWithError(c, err)
			return
		}
		utils.SuccessResponse(c, http.StatusOK, "Plan deactivated successfully", nil)

	default:
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid status value")
	}
}

func (h *PlanHandler) GetPlanPricings(c *gin.Context) {
	planID, err := parsePlanID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	query := usecases.GetPlanPricingsQuery{
		PlanID: planID,
	}

	result, err := h.getPlanPricingsUC.Execute(c.Request.Context(), query)
	if err != nil {
		h.logger.Errorw("failed to get plan pricings", "error", err, "plan_id", planID)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}

func parsePlanID(c *gin.Context) (uint, error) {
	idStr := c.Param("id")
	if idStr == "" {
		return 0, errors.NewValidationError("Plan ID is required")
	}

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, errors.NewValidationError("Invalid plan ID format")
	}

	if id == 0 {
		return 0, errors.NewValidationError("Plan ID cannot be zero")
	}

	return uint(id), nil
}

func parseListPlansQuery(c *gin.Context) (*usecases.ListPlansQuery, error) {
	query := &usecases.ListPlansQuery{
		Page:     constants.DefaultPage,
		PageSize: constants.DefaultPageSize,
	}

	if pageStr := c.Query("page"); pageStr != "" {
		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			return nil, errors.NewValidationError("Invalid page parameter")
		}
		query.Page = page
	}

	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		pageSize, err := strconv.Atoi(pageSizeStr)
		if err != nil || pageSize < 1 {
			return nil, errors.NewValidationError("Invalid page_size parameter")
		}
		if pageSize > constants.MaxPageSize {
			pageSize = constants.MaxPageSize
		}
		query.PageSize = pageSize
	}

	if status := c.Query("status"); status != "" {
		query.Status = &status
	}

	if isPublicStr := c.Query("is_public"); isPublicStr != "" {
		isPublic, err := strconv.ParseBool(isPublicStr)
		if err != nil {
			return nil, errors.NewValidationError("Invalid is_public parameter")
		}
		query.IsPublic = &isPublic
	}

	if planType := c.Query("plan_type"); planType != "" {
		query.PlanType = &planType
	}

	return query, nil
}

// BindNodesRequest represents the request to bind nodes to a plan
type BindNodesRequest struct {
	NodeIDs []uint `json:"node_ids" binding:"required,min=1"`
}

// UnbindNodesRequest represents the request to unbind nodes from a plan
type UnbindNodesRequest struct {
	NodeIDs []uint `json:"node_ids" binding:"required,min=1"`
}

// GetPlanNodes returns all nodes associated with a plan
// @Summary Get plan nodes
// @Description Get all nodes associated with a subscription plan
// @Tags Plans
// @Accept json
// @Produce json
// @Param id path int true "Plan ID"
// @Success 200 {object} utils.Response{data=usecases.GetPlanNodesResult}
// @Failure 400 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /admin/plans/{id}/nodes [get]
func (h *PlanHandler) GetPlanNodes(c *gin.Context) {
	planID, err := parsePlanID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	result, err := h.managePlanNodesUC.GetPlanNodes(c.Request.Context(), planID)
	if err != nil {
		h.logger.Errorw("failed to get plan nodes", "plan_id", planID, "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}

// BindNodes binds nodes to a plan
// @Summary Bind nodes to plan
// @Description Bind multiple nodes to a subscription plan (node type only)
// @Tags Plans
// @Accept json
// @Produce json
// @Param id path int true "Plan ID"
// @Param request body BindNodesRequest true "Node IDs to bind"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /admin/plans/{id}/nodes [post]
func (h *PlanHandler) BindNodes(c *gin.Context) {
	planID, err := parsePlanID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	var req BindNodesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for bind nodes", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.BindNodesCommand{
		PlanID:  planID,
		NodeIDs: req.NodeIDs,
	}

	if err := h.managePlanNodesUC.BindNodes(c.Request.Context(), cmd); err != nil {
		h.logger.Errorw("failed to bind nodes to plan", "plan_id", planID, "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Nodes bound to plan successfully", nil)
}

// UnbindNodes unbinds nodes from a plan
// @Summary Unbind nodes from plan
// @Description Unbind multiple nodes from a subscription plan
// @Tags Plans
// @Accept json
// @Produce json
// @Param id path int true "Plan ID"
// @Param request body UnbindNodesRequest true "Node IDs to unbind"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Router /admin/plans/{id}/nodes [delete]
func (h *PlanHandler) UnbindNodes(c *gin.Context) {
	planID, err := parsePlanID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	var req UnbindNodesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for unbind nodes", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.UnbindNodesCommand{
		PlanID:  planID,
		NodeIDs: req.NodeIDs,
	}

	if err := h.managePlanNodesUC.UnbindNodes(c.Request.Context(), cmd); err != nil {
		h.logger.Errorw("failed to unbind nodes from plan", "plan_id", planID, "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Nodes unbound from plan successfully", nil)
}
