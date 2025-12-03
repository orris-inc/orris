package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	subdto "github.com/orris-inc/orris/internal/application/subscription/dto"
	"github.com/orris-inc/orris/internal/application/subscription/usecases"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

var (
	_ = subdto.SubscriptionPlanDTO{}
	_ = subdto.PricingOptionDTO{}
)

type SubscriptionPlanHandler struct {
	createPlanUC      *usecases.CreateSubscriptionPlanUseCase
	updatePlanUC      *usecases.UpdateSubscriptionPlanUseCase
	getPlanUC         *usecases.GetSubscriptionPlanUseCase
	listPlansUC       *usecases.ListSubscriptionPlansUseCase
	getPublicPlansUC  *usecases.GetPublicPlansUseCase
	activatePlanUC    *usecases.ActivateSubscriptionPlanUseCase
	deactivatePlanUC  *usecases.DeactivateSubscriptionPlanUseCase
	getPlanPricingsUC *usecases.GetPlanPricingsUseCase
	logger            logger.Interface
}

func NewSubscriptionPlanHandler(
	createPlanUC *usecases.CreateSubscriptionPlanUseCase,
	updatePlanUC *usecases.UpdateSubscriptionPlanUseCase,
	getPlanUC *usecases.GetSubscriptionPlanUseCase,
	listPlansUC *usecases.ListSubscriptionPlansUseCase,
	getPublicPlansUC *usecases.GetPublicPlansUseCase,
	activatePlanUC *usecases.ActivateSubscriptionPlanUseCase,
	deactivatePlanUC *usecases.DeactivateSubscriptionPlanUseCase,
	getPlanPricingsUC *usecases.GetPlanPricingsUseCase,
) *SubscriptionPlanHandler {
	return &SubscriptionPlanHandler{
		createPlanUC:      createPlanUC,
		updatePlanUC:      updatePlanUC,
		getPlanUC:         getPlanUC,
		listPlansUC:       listPlansUC,
		getPublicPlansUC:  getPublicPlansUC,
		activatePlanUC:    activatePlanUC,
		deactivatePlanUC:  deactivatePlanUC,
		getPlanPricingsUC: getPlanPricingsUC,
		logger:            logger.NewLogger(),
	}
}

type CreatePlanRequest struct {
	Name         string                 `json:"name" binding:"required"`
	Slug         string                 `json:"slug" binding:"required"`
	Description  string                 `json:"description"`
	Price        uint64                 `json:"price" binding:"required"`
	Currency     string                 `json:"currency" binding:"required"`
	BillingCycle string                 `json:"billing_cycle" binding:"required"`
	TrialDays    int                    `json:"trial_days"`
	Features     []string               `json:"features"`
	Limits       map[string]interface{} `json:"limits"`
	APIRateLimit uint                   `json:"api_rate_limit"`
	MaxUsers     uint                   `json:"max_users"`
	MaxProjects  uint                   `json:"max_projects"`
	IsPublic     bool                   `json:"is_public"`
	SortOrder    int                    `json:"sort_order"`
}

type UpdatePlanRequest struct {
	Description  *string                 `json:"description"`
	Price        *uint64                 `json:"price"`
	Currency     *string                 `json:"currency"`
	Features     *[]string               `json:"features"`
	Limits       *map[string]interface{} `json:"limits"`
	APIRateLimit *uint                   `json:"api_rate_limit"`
	MaxUsers     *uint                   `json:"max_users"`
	MaxProjects  *uint                   `json:"max_projects"`
	IsPublic     *bool                   `json:"is_public"`
	SortOrder    *int                    `json:"sort_order"`
}

// UpdatePlanStatusRequest represents a unified request for plan status changes
type UpdatePlanStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=active inactive"`
}

func (h *SubscriptionPlanHandler) CreatePlan(c *gin.Context) {
	var req CreatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for create plan", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.CreateSubscriptionPlanCommand{
		Name:         req.Name,
		Slug:         req.Slug,
		Description:  req.Description,
		Price:        req.Price,
		Currency:     req.Currency,
		BillingCycle: req.BillingCycle,
		TrialDays:    req.TrialDays,
		Features:     req.Features,
		Limits:       req.Limits,
		APIRateLimit: req.APIRateLimit,
		MaxUsers:     req.MaxUsers,
		MaxProjects:  req.MaxProjects,
		IsPublic:     req.IsPublic,
		SortOrder:    req.SortOrder,
	}

	result, err := h.createPlanUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.CreatedResponse(c, result, "Subscription plan created successfully")
}

func (h *SubscriptionPlanHandler) UpdatePlan(c *gin.Context) {
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

	cmd := usecases.UpdateSubscriptionPlanCommand{
		PlanID:       planID,
		Description:  req.Description,
		Price:        req.Price,
		Currency:     req.Currency,
		Features:     req.Features,
		Limits:       req.Limits,
		APIRateLimit: req.APIRateLimit,
		MaxUsers:     req.MaxUsers,
		MaxProjects:  req.MaxProjects,
		IsPublic:     req.IsPublic,
		SortOrder:    req.SortOrder,
	}

	result, err := h.updatePlanUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Subscription plan updated successfully", result)
}

func (h *SubscriptionPlanHandler) GetPlan(c *gin.Context) {
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

func (h *SubscriptionPlanHandler) ListPlans(c *gin.Context) {
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

func (h *SubscriptionPlanHandler) GetPublicPlans(c *gin.Context) {
	result, err := h.getPublicPlansUC.Execute(c.Request.Context())
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}

func (h *SubscriptionPlanHandler) UpdatePlanStatus(c *gin.Context) {
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
	case "active":
		if err := h.activatePlanUC.Execute(c.Request.Context(), planID); err != nil {
			utils.ErrorResponseWithError(c, err)
			return
		}
		utils.SuccessResponse(c, http.StatusOK, "Subscription plan activated successfully", nil)

	case "inactive":
		if err := h.deactivatePlanUC.Execute(c.Request.Context(), planID); err != nil {
			utils.ErrorResponseWithError(c, err)
			return
		}
		utils.SuccessResponse(c, http.StatusOK, "Subscription plan deactivated successfully", nil)

	default:
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid status value")
	}
}

func (h *SubscriptionPlanHandler) GetPlanPricings(c *gin.Context) {
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

func parseListPlansQuery(c *gin.Context) (*usecases.ListSubscriptionPlansQuery, error) {
	query := &usecases.ListSubscriptionPlansQuery{
		Page:     1,
		PageSize: 20,
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
		if pageSize > 100 {
			pageSize = 100
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

	if billingCycle := c.Query("billing_cycle"); billingCycle != "" {
		query.BillingCycle = &billingCycle
	}

	return query, nil
}
