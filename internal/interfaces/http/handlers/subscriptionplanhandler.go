package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"orris/internal/application/subscription/usecases"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
	"orris/internal/shared/utils"
)

type SubscriptionPlanHandler struct {
	createPlanUC     *usecases.CreateSubscriptionPlanUseCase
	updatePlanUC     *usecases.UpdateSubscriptionPlanUseCase
	getPlanUC        *usecases.GetSubscriptionPlanUseCase
	listPlansUC      *usecases.ListSubscriptionPlansUseCase
	getPublicPlansUC *usecases.GetPublicPlansUseCase
	activatePlanUC   *usecases.ActivateSubscriptionPlanUseCase
	deactivatePlanUC *usecases.DeactivateSubscriptionPlanUseCase
	logger           logger.Interface
}

func NewSubscriptionPlanHandler(
	createPlanUC *usecases.CreateSubscriptionPlanUseCase,
	updatePlanUC *usecases.UpdateSubscriptionPlanUseCase,
	getPlanUC *usecases.GetSubscriptionPlanUseCase,
	listPlansUC *usecases.ListSubscriptionPlansUseCase,
	getPublicPlansUC *usecases.GetPublicPlansUseCase,
	activatePlanUC *usecases.ActivateSubscriptionPlanUseCase,
	deactivatePlanUC *usecases.DeactivateSubscriptionPlanUseCase,
) *SubscriptionPlanHandler {
	return &SubscriptionPlanHandler{
		createPlanUC:     createPlanUC,
		updatePlanUC:     updatePlanUC,
		getPlanUC:        getPlanUC,
		listPlansUC:      listPlansUC,
		getPublicPlansUC: getPublicPlansUC,
		activatePlanUC:   activatePlanUC,
		deactivatePlanUC: deactivatePlanUC,
		logger:           logger.NewLogger(),
	}
}

type CreatePlanRequest struct {
	Name           string                 `json:"name" binding:"required"`
	Slug           string                 `json:"slug" binding:"required"`
	Description    string                 `json:"description"`
	Price          uint64                 `json:"price" binding:"required"`
	Currency       string                 `json:"currency" binding:"required"`
	BillingCycle   string                 `json:"billing_cycle" binding:"required"`
	TrialDays      int                    `json:"trial_days"`
	Features       []string               `json:"features"`
	Limits         map[string]interface{} `json:"limits"`
	CustomEndpoint string                 `json:"custom_endpoint"`
	APIRateLimit   uint                   `json:"api_rate_limit"`
	MaxUsers       uint                   `json:"max_users"`
	MaxProjects    uint                   `json:"max_projects"`
	StorageLimit   uint64                 `json:"storage_limit"`
	IsPublic       bool                   `json:"is_public"`
	SortOrder      int                    `json:"sort_order"`
}

type UpdatePlanRequest struct {
	Description    *string                `json:"description"`
	Price          *uint64                `json:"price"`
	Currency       *string                `json:"currency"`
	Features       []string               `json:"features"`
	Limits         map[string]interface{} `json:"limits"`
	CustomEndpoint *string                `json:"custom_endpoint"`
	APIRateLimit   *uint                  `json:"api_rate_limit"`
	MaxUsers       *uint                  `json:"max_users"`
	MaxProjects    *uint                  `json:"max_projects"`
	StorageLimit   *uint64                `json:"storage_limit"`
	IsPublic       *bool                  `json:"is_public"`
	SortOrder      *int                   `json:"sort_order"`
}

// @Summary Create subscription plan
// @Description Create a new subscription plan with pricing and features
// @Tags subscription-plans
// @Accept json
// @Produce json
// @Security Bearer
// @Param plan body CreatePlanRequest true "Subscription plan data"
// @Success 201 {object} utils.APIResponse "Subscription plan created successfully"
// @Failure 400 {object} utils.APIResponse "Bad request"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 409 {object} utils.APIResponse "Plan slug already exists"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /subscription-plans [post]
func (h *SubscriptionPlanHandler) CreatePlan(c *gin.Context) {
	var req CreatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for create plan", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.CreateSubscriptionPlanCommand{
		Name:           req.Name,
		Slug:           req.Slug,
		Description:    req.Description,
		Price:          req.Price,
		Currency:       req.Currency,
		BillingCycle:   req.BillingCycle,
		TrialDays:      req.TrialDays,
		Features:       req.Features,
		Limits:         req.Limits,
		CustomEndpoint: req.CustomEndpoint,
		APIRateLimit:   req.APIRateLimit,
		MaxUsers:       req.MaxUsers,
		MaxProjects:    req.MaxProjects,
		StorageLimit:   req.StorageLimit,
		IsPublic:       req.IsPublic,
		SortOrder:      req.SortOrder,
	}

	result, err := h.createPlanUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.CreatedResponse(c, result, "Subscription plan created successfully")
}

// @Summary Update subscription plan
// @Description Update an existing subscription plan's details
// @Tags subscription-plans
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "Plan ID"
// @Param plan body UpdatePlanRequest true "Plan update data"
// @Success 200 {object} utils.APIResponse "Subscription plan updated successfully"
// @Failure 400 {object} utils.APIResponse "Bad request"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 404 {object} utils.APIResponse "Plan not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /subscription-plans/{id} [put]
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
		PlanID:         planID,
		Description:    req.Description,
		Price:          req.Price,
		Currency:       req.Currency,
		Features:       req.Features,
		Limits:         req.Limits,
		CustomEndpoint: req.CustomEndpoint,
		APIRateLimit:   req.APIRateLimit,
		MaxUsers:       req.MaxUsers,
		MaxProjects:    req.MaxProjects,
		StorageLimit:   req.StorageLimit,
		IsPublic:       req.IsPublic,
		SortOrder:      req.SortOrder,
	}

	result, err := h.updatePlanUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Subscription plan updated successfully", result)
}

// @Summary Get subscription plan by ID
// @Description Get details of a specific subscription plan by its ID
// @Tags subscription-plans
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "Plan ID"
// @Success 200 {object} utils.APIResponse "Subscription plan details"
// @Failure 400 {object} utils.APIResponse "Invalid plan ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 404 {object} utils.APIResponse "Plan not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /subscription-plans/{id} [get]
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

// @Summary List subscription plans
// @Description Get a paginated list of subscription plans with optional filters
// @Tags subscription-plans
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Param status query string false "Plan status filter" Enums(active,inactive,archived)
// @Param is_public query bool false "Filter by public/private plans"
// @Param billing_cycle query string false "Filter by billing cycle" Enums(monthly,quarterly,semi_annual,annual,lifetime)
// @Success 200 {object} utils.APIResponse "Subscription plans list"
// @Failure 400 {object} utils.APIResponse "Invalid query parameters"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /subscription-plans [get]
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

// @Summary Get public subscription plans
// @Description Get all publicly available subscription plans for display to potential customers
// @Tags subscription-plans
// @Accept json
// @Produce json
// @Success 200 {object} utils.APIResponse "Public subscription plans"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /subscription-plans/public [get]
func (h *SubscriptionPlanHandler) GetPublicPlans(c *gin.Context) {
	result, err := h.getPublicPlansUC.Execute(c.Request.Context())
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}

// @Summary Activate subscription plan
// @Description Activate a subscription plan to make it available for new subscriptions
// @Tags subscription-plans
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "Plan ID"
// @Success 200 {object} utils.APIResponse "Subscription plan activated successfully"
// @Failure 400 {object} utils.APIResponse "Invalid plan ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 404 {object} utils.APIResponse "Plan not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /subscription-plans/{id}/activate [post]
func (h *SubscriptionPlanHandler) ActivatePlan(c *gin.Context) {
	planID, err := parsePlanID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	if err := h.activatePlanUC.Execute(c.Request.Context(), planID); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Subscription plan activated successfully", nil)
}

// @Summary Deactivate subscription plan
// @Description Deactivate a subscription plan to prevent new subscriptions
// @Tags subscription-plans
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path int true "Plan ID"
// @Success 200 {object} utils.APIResponse "Subscription plan deactivated successfully"
// @Failure 400 {object} utils.APIResponse "Invalid plan ID"
// @Failure 401 {object} utils.APIResponse "Unauthorized"
// @Failure 404 {object} utils.APIResponse "Plan not found"
// @Failure 500 {object} utils.APIResponse "Internal server error"
// @Router /subscription-plans/{id}/deactivate [post]
func (h *SubscriptionPlanHandler) DeactivatePlan(c *gin.Context) {
	planID, err := parsePlanID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	if err := h.deactivatePlanUC.Execute(c.Request.Context(), planID); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Subscription plan deactivated successfully", nil)
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
