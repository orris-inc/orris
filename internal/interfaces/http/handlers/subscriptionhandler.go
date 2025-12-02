package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	subdto "orris/internal/application/subscription/dto"
	"orris/internal/application/subscription/usecases"
	"orris/internal/shared/logger"
	"orris/internal/shared/utils"
)

// SubscriptionHandler handles user subscription operations
type SubscriptionHandler struct {
	createUseCase     *usecases.CreateSubscriptionUseCase
	getUseCase        *usecases.GetSubscriptionUseCase
	listUserUseCase   *usecases.ListUserSubscriptionsUseCase
	cancelUseCase     *usecases.CancelSubscriptionUseCase
	changePlanUseCase *usecases.ChangePlanUseCase
	logger            logger.Interface
}

// NewSubscriptionHandler creates a new user subscription handler
func NewSubscriptionHandler(
	createUC *usecases.CreateSubscriptionUseCase,
	getUC *usecases.GetSubscriptionUseCase,
	listUserUC *usecases.ListUserSubscriptionsUseCase,
	cancelUC *usecases.CancelSubscriptionUseCase,
	changePlanUC *usecases.ChangePlanUseCase,
	logger logger.Interface,
) *SubscriptionHandler {
	return &SubscriptionHandler{
		createUseCase:     createUC,
		getUseCase:        getUC,
		listUserUseCase:   listUserUC,
		cancelUseCase:     cancelUC,
		changePlanUseCase: changePlanUC,
		logger:            logger,
	}
}

// CreateSubscriptionRequest represents the request to create a subscription for self
type CreateSubscriptionRequest struct {
	PlanID       uint                   `json:"plan_id" binding:"required"`
	BillingCycle string                 `json:"billing_cycle" binding:"required,oneof=weekly monthly quarterly semi_annual yearly lifetime"`
	StartDate    *time.Time             `json:"start_date"`
	AutoRenew    *bool                  `json:"auto_renew"`
	PaymentInfo  map[string]interface{} `json:"payment_info"`
}

// CancelSubscriptionRequest represents the request to cancel a subscription
type CancelSubscriptionRequest struct {
	Reason    string `json:"reason" binding:"required"`
	Immediate *bool  `json:"immediate"`
}

// ChangePlanRequest represents the request to change subscription plan
type ChangePlanRequest struct {
	NewPlanID     uint   `json:"new_plan_id" binding:"required"`
	ChangeType    string `json:"change_type" binding:"required,oneof=upgrade downgrade"`
	EffectiveDate string `json:"effective_date" binding:"required,oneof=immediate period_end"`
}

// CreateSubscriptionResponse represents the response for subscription creation
type CreateSubscriptionResponse struct {
	Subscription *subdto.SubscriptionDTO      `json:"subscription"`
	Token        *subdto.SubscriptionTokenDTO `json:"token"`
}

// @Summary		Create subscription for self
// @Description	Create a new subscription for the authenticated user
// @Tags			subscriptions
// @Accept			json
// @Produce		json
// @Security		Bearer
// @Param			subscription	body		CreateSubscriptionRequest						true	"Subscription data"
// @Success		201				{object}	utils.APIResponse{data=CreateSubscriptionResponse}	"Subscription created successfully"
// @Failure		400				{object}	utils.APIResponse								"Bad request"
// @Failure		401				{object}	utils.APIResponse								"Unauthorized"
// @Failure		404				{object}	utils.APIResponse								"Plan not found"
// @Failure		500				{object}	utils.APIResponse								"Internal server error"
// @Router			/subscriptions [post]
func (h *SubscriptionHandler) CreateSubscription(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	var req CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for create subscription", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	autoRenew := true
	if req.AutoRenew != nil {
		autoRenew = *req.AutoRenew
	}

	startDate := time.Now()
	if req.StartDate != nil {
		startDate = *req.StartDate
	}

	cmd := usecases.CreateSubscriptionCommand{
		UserID:       userID.(uint),
		PlanID:       req.PlanID,
		BillingCycle: req.BillingCycle,
		StartDate:    startDate,
		AutoRenew:    autoRenew,
		PaymentInfo:  req.PaymentInfo,
	}

	result, err := h.createUseCase.Execute(c.Request.Context(), cmd)
	if err != nil {
		h.logger.Errorw("failed to create subscription", "error", err, "user_id", userID)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.CreatedResponse(c, gin.H{
		"subscription": result.Subscription,
		"token":        result.Token,
	}, "Subscription created successfully")
}

// @Summary		Get subscription by ID
// @Description	Get details of own subscription by ID
// @Tags			subscriptions
// @Accept			json
// @Produce		json
// @Security		Bearer
// @Param			id	path		int												true	"Subscription ID"
// @Success		200	{object}	utils.APIResponse{data=subdto.SubscriptionDTO}	"Subscription details"
// @Failure		400	{object}	utils.APIResponse								"Invalid subscription ID"
// @Failure		401	{object}	utils.APIResponse								"Unauthorized"
// @Failure		403	{object}	utils.APIResponse								"Access denied"
// @Failure		404	{object}	utils.APIResponse								"Subscription not found"
// @Failure		500	{object}	utils.APIResponse								"Internal server error"
// @Router			/subscriptions/{id} [get]
func (h *SubscriptionHandler) GetSubscription(c *gin.Context) {
	// Ownership already verified by middleware, subscription stored in context
	subscriptionID, exists := c.Get("subscription_id")
	if !exists {
		subscriptionID64, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "invalid subscription ID")
			return
		}
		subscriptionID = uint(subscriptionID64)
	}

	query := usecases.GetSubscriptionQuery{
		SubscriptionID: subscriptionID.(uint),
	}

	subscription, err := h.getUseCase.Execute(c.Request.Context(), query)
	if err != nil {
		h.logger.Errorw("failed to get subscription", "error", err, "subscription_id", subscriptionID)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", subscription)
}

// @Summary		List my subscriptions
// @Description	Get a paginated list of current user's subscriptions
// @Tags			subscriptions
// @Accept			json
// @Produce		json
// @Security		Bearer
// @Param			page		query		int																	false	"Page number"				default(1)
// @Param			page_size	query		int																	false	"Page size"					default(20)
// @Param			status		query		string																false	"Subscription status filter"	Enums(active,inactive,cancelled,expired,pending)
// @Success		200			{object}	utils.APIResponse{data=utils.ListResponse{items=[]subdto.SubscriptionDTO}}	"Subscriptions list"
// @Failure		400			{object}	utils.APIResponse														"Invalid query parameters"
// @Failure		401			{object}	utils.APIResponse														"Unauthorized"
// @Failure		500			{object}	utils.APIResponse														"Internal server error"
// @Router			/subscriptions [get]
func (h *SubscriptionHandler) ListUserSubscriptions(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pageSize := 20
	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	var status *string
	if statusStr := c.Query("status"); statusStr != "" {
		status = &statusStr
	}

	uid := userID.(uint)
	query := usecases.ListUserSubscriptionsQuery{
		UserID:   &uid,
		Status:   status,
		Page:     page,
		PageSize: pageSize,
	}

	result, err := h.listUserUseCase.Execute(c.Request.Context(), query)
	if err != nil {
		h.logger.Errorw("failed to list subscriptions", "error", err, "user_id", uid)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.ListSuccessResponse(c, result.Subscriptions, result.Total, result.Page, result.PageSize)
}

// @Summary		Cancel subscription
// @Description	Cancel own subscription
// @Tags			subscriptions
// @Accept			json
// @Produce		json
// @Security		Bearer
// @Param			id		path		int							true	"Subscription ID"
// @Param			cancel	body		CancelSubscriptionRequest	true	"Cancellation details"
// @Success		200		{object}	utils.APIResponse			"Subscription cancelled successfully"
// @Failure		400		{object}	utils.APIResponse			"Bad request"
// @Failure		401		{object}	utils.APIResponse			"Unauthorized"
// @Failure		403		{object}	utils.APIResponse			"Access denied"
// @Failure		404		{object}	utils.APIResponse			"Subscription not found"
// @Failure		500		{object}	utils.APIResponse			"Internal server error"
// @Router			/subscriptions/{id}/cancel [post]
func (h *SubscriptionHandler) CancelSubscription(c *gin.Context) {
	// Ownership already verified by middleware
	subscriptionID, exists := c.Get("subscription_id")
	if !exists {
		subscriptionID64, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "invalid subscription ID")
			return
		}
		subscriptionID = uint(subscriptionID64)
	}

	var req CancelSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for cancel subscription", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	immediate := false
	if req.Immediate != nil {
		immediate = *req.Immediate
	}

	cmd := usecases.CancelSubscriptionCommand{
		SubscriptionID: subscriptionID.(uint),
		Reason:         req.Reason,
		Immediate:      immediate,
	}

	if err := h.cancelUseCase.Execute(c.Request.Context(), cmd); err != nil {
		h.logger.Errorw("failed to cancel subscription", "error", err, "subscription_id", subscriptionID)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Subscription cancelled successfully", nil)
}

// @Summary		Change subscription plan
// @Description	Change the plan of own subscription (upgrade or downgrade)
// @Tags			subscriptions
// @Accept			json
// @Produce		json
// @Security		Bearer
// @Param			id		path		int					true	"Subscription ID"
// @Param			plan	body		ChangePlanRequest	true	"Plan change details"
// @Success		200		{object}	utils.APIResponse	"Plan changed successfully"
// @Failure		400		{object}	utils.APIResponse	"Bad request"
// @Failure		401		{object}	utils.APIResponse	"Unauthorized"
// @Failure		403		{object}	utils.APIResponse	"Access denied"
// @Failure		404		{object}	utils.APIResponse	"Subscription or plan not found"
// @Failure		500		{object}	utils.APIResponse	"Internal server error"
// @Router			/subscriptions/{id}/plan [patch]
func (h *SubscriptionHandler) ChangePlan(c *gin.Context) {
	// Ownership already verified by middleware
	subscriptionID, exists := c.Get("subscription_id")
	if !exists {
		subscriptionID64, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "invalid subscription ID")
			return
		}
		subscriptionID = uint(subscriptionID64)
	}

	var req ChangePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for change plan", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.ChangePlanCommand{
		SubscriptionID: subscriptionID.(uint),
		NewPlanID:      req.NewPlanID,
		ChangeType:     usecases.ChangeType(req.ChangeType),
		EffectiveDate:  usecases.EffectiveDate(req.EffectiveDate),
	}

	if err := h.changePlanUseCase.Execute(c.Request.Context(), cmd); err != nil {
		h.logger.Errorw("failed to change plan", "error", err, "subscription_id", subscriptionID)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Plan changed successfully", nil)
}
