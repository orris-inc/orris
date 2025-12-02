// Package admin provides HTTP handlers for administrative operations.
package admin

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"orris/internal/application/subscription/usecases"
	"orris/internal/shared/logger"
	"orris/internal/shared/utils"
)

// SubscriptionHandler handles admin subscription operations
type SubscriptionHandler struct {
	createUseCase     *usecases.CreateSubscriptionUseCase
	getUseCase        *usecases.GetSubscriptionUseCase
	listUseCase       *usecases.ListUserSubscriptionsUseCase
	cancelUseCase     *usecases.CancelSubscriptionUseCase
	renewUseCase      *usecases.RenewSubscriptionUseCase
	changePlanUseCase *usecases.ChangePlanUseCase
	activateUseCase   *usecases.ActivateSubscriptionUseCase
	logger            logger.Interface
}

// NewSubscriptionHandler creates a new admin subscription handler
func NewSubscriptionHandler(
	createUC *usecases.CreateSubscriptionUseCase,
	getUC *usecases.GetSubscriptionUseCase,
	listUC *usecases.ListUserSubscriptionsUseCase,
	cancelUC *usecases.CancelSubscriptionUseCase,
	renewUC *usecases.RenewSubscriptionUseCase,
	changePlanUC *usecases.ChangePlanUseCase,
	activateUC *usecases.ActivateSubscriptionUseCase,
	logger logger.Interface,
) *SubscriptionHandler {
	return &SubscriptionHandler{
		createUseCase:     createUC,
		getUseCase:        getUC,
		listUseCase:       listUC,
		cancelUseCase:     cancelUC,
		renewUseCase:      renewUC,
		changePlanUseCase: changePlanUC,
		activateUseCase:   activateUC,
		logger:            logger,
	}
}

// CreateSubscriptionRequest represents the request to create a subscription
type CreateSubscriptionRequest struct {
	UserID       uint                   `json:"user_id" binding:"required"`
	PlanID       uint                   `json:"plan_id" binding:"required"`
	BillingCycle string                 `json:"billing_cycle" binding:"required,oneof=weekly monthly quarterly semi_annual yearly lifetime"`
	StartDate    *time.Time             `json:"start_date"`
	AutoRenew    *bool                  `json:"auto_renew"`
	PaymentInfo  map[string]interface{} `json:"payment_info"`
}

// UpdateStatusRequest represents the request to update subscription status
type UpdateStatusRequest struct {
	Status    string  `json:"status" binding:"required,oneof=active cancelled renewed"`
	Reason    *string `json:"reason"`
	Immediate *bool   `json:"immediate"`
}

// ChangePlanRequest represents the request to change subscription plan
type ChangePlanRequest struct {
	NewPlanID     uint   `json:"new_plan_id" binding:"required"`
	ChangeType    string `json:"change_type" binding:"required,oneof=upgrade downgrade"`
	EffectiveDate string `json:"effective_date" binding:"required,oneof=immediate period_end"`
}

// @Summary		Create subscription for user (Admin)
// @Description	Admin creates a subscription for any user
// @Tags			admin-subscriptions
// @Accept			json
// @Produce		json
// @Security		Bearer
// @Param			subscription	body		CreateSubscriptionRequest	true	"Subscription data"
// @Success		201				{object}	utils.APIResponse			"Subscription created successfully"
// @Failure		400				{object}	utils.APIResponse			"Bad request"
// @Failure		401				{object}	utils.APIResponse			"Unauthorized"
// @Failure		403				{object}	utils.APIResponse			"Forbidden - admin only"
// @Failure		404				{object}	utils.APIResponse			"User or plan not found"
// @Failure		500				{object}	utils.APIResponse			"Internal server error"
// @Router			/admin/subscriptions [post]
func (h *SubscriptionHandler) Create(c *gin.Context) {
	var req CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for admin create subscription", "error", err)
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
		UserID:       req.UserID,
		PlanID:       req.PlanID,
		BillingCycle: req.BillingCycle,
		StartDate:    startDate,
		AutoRenew:    autoRenew,
		PaymentInfo:  req.PaymentInfo,
	}

	result, err := h.createUseCase.Execute(c.Request.Context(), cmd)
	if err != nil {
		h.logger.Errorw("failed to create subscription", "error", err, "user_id", req.UserID)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.CreatedResponse(c, gin.H{
		"subscription": result.Subscription,
		"token":        result.Token,
	}, "Subscription created successfully")
}

// @Summary		List all subscriptions (Admin)
// @Description	Get a paginated list of all subscriptions with optional filters
// @Tags			admin-subscriptions
// @Accept			json
// @Produce		json
// @Security		Bearer
// @Param			page		query		int											false	"Page number"				default(1)
// @Param			page_size	query		int											false	"Page size"					default(20)
// @Param			status		query		string										false	"Subscription status filter"	Enums(active,inactive,cancelled,expired,pending)
// @Param			user_id		query		int											false	"Filter by user ID"
// @Success		200			{object}	utils.APIResponse{data=utils.ListResponse}	"Subscriptions list"
// @Failure		400			{object}	utils.APIResponse							"Invalid query parameters"
// @Failure		401			{object}	utils.APIResponse							"Unauthorized"
// @Failure		403			{object}	utils.APIResponse							"Forbidden - admin only"
// @Failure		500			{object}	utils.APIResponse							"Internal server error"
// @Router			/admin/subscriptions [get]
func (h *SubscriptionHandler) List(c *gin.Context) {
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

	var userID *uint
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		if uid, err := strconv.ParseUint(userIDStr, 10, 64); err == nil {
			uidVal := uint(uid)
			userID = &uidVal
		}
	}

	query := usecases.ListUserSubscriptionsQuery{
		UserID:   userID,
		Status:   status,
		Page:     page,
		PageSize: pageSize,
	}

	result, err := h.listUseCase.Execute(c.Request.Context(), query)
	if err != nil {
		h.logger.Errorw("failed to list subscriptions", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.ListSuccessResponse(c, result.Subscriptions, result.Total, result.Page, result.PageSize)
}

// @Summary		Get subscription by ID (Admin)
// @Description	Get details of any subscription
// @Tags			admin-subscriptions
// @Accept			json
// @Produce		json
// @Security		Bearer
// @Param			id	path		int					true	"Subscription ID"
// @Success		200	{object}	utils.APIResponse	"Subscription details"
// @Failure		400	{object}	utils.APIResponse	"Invalid subscription ID"
// @Failure		401	{object}	utils.APIResponse	"Unauthorized"
// @Failure		403	{object}	utils.APIResponse	"Forbidden - admin only"
// @Failure		404	{object}	utils.APIResponse	"Subscription not found"
// @Failure		500	{object}	utils.APIResponse	"Internal server error"
// @Router			/admin/subscriptions/{id} [get]
func (h *SubscriptionHandler) Get(c *gin.Context) {
	subscriptionID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid subscription ID")
		return
	}

	query := usecases.GetSubscriptionQuery{
		SubscriptionID: uint(subscriptionID),
	}

	subscription, err := h.getUseCase.Execute(c.Request.Context(), query)
	if err != nil {
		h.logger.Errorw("failed to get subscription", "error", err, "subscription_id", subscriptionID)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", subscription)
}

// @Summary		Update subscription status (Admin)
// @Description	Update subscription status (activate, cancel, or renew)
// @Tags			admin-subscriptions
// @Accept			json
// @Produce		json
// @Security		Bearer
// @Param			id		path		int					true	"Subscription ID"
// @Param			status	body		UpdateStatusRequest	true	"Status update details"
// @Success		200		{object}	utils.APIResponse	"Subscription status updated successfully"
// @Failure		400		{object}	utils.APIResponse	"Bad request"
// @Failure		401		{object}	utils.APIResponse	"Unauthorized"
// @Failure		403		{object}	utils.APIResponse	"Forbidden - admin only"
// @Failure		404		{object}	utils.APIResponse	"Subscription not found"
// @Failure		500		{object}	utils.APIResponse	"Internal server error"
// @Router			/admin/subscriptions/{id}/status [patch]
func (h *SubscriptionHandler) UpdateStatus(c *gin.Context) {
	subscriptionID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid subscription ID")
		return
	}

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for update subscription status", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	switch req.Status {
	case "active":
		cmd := usecases.ActivateSubscriptionCommand{
			SubscriptionID: uint(subscriptionID),
		}
		if err := h.activateUseCase.Execute(c.Request.Context(), cmd); err != nil {
			h.logger.Errorw("failed to activate subscription", "error", err, "subscription_id", subscriptionID)
			utils.ErrorResponseWithError(c, err)
			return
		}
		utils.SuccessResponse(c, http.StatusOK, "Subscription activated successfully", nil)

	case "cancelled":
		if req.Reason == nil || *req.Reason == "" {
			utils.ErrorResponse(c, http.StatusBadRequest, "reason is required for cancellation")
			return
		}
		immediate := false
		if req.Immediate != nil {
			immediate = *req.Immediate
		}
		cmd := usecases.CancelSubscriptionCommand{
			SubscriptionID: uint(subscriptionID),
			Reason:         *req.Reason,
			Immediate:      immediate,
		}
		if err := h.cancelUseCase.Execute(c.Request.Context(), cmd); err != nil {
			h.logger.Errorw("failed to cancel subscription", "error", err, "subscription_id", subscriptionID)
			utils.ErrorResponseWithError(c, err)
			return
		}
		utils.SuccessResponse(c, http.StatusOK, "Subscription cancelled successfully", nil)

	case "renewed":
		cmd := usecases.RenewSubscriptionCommand{
			SubscriptionID: uint(subscriptionID),
			IsAutoRenew:    false,
		}
		if err := h.renewUseCase.Execute(c.Request.Context(), cmd); err != nil {
			h.logger.Errorw("failed to renew subscription", "error", err, "subscription_id", subscriptionID)
			utils.ErrorResponseWithError(c, err)
			return
		}
		utils.SuccessResponse(c, http.StatusOK, "Subscription renewed successfully", nil)

	default:
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid status value")
	}
}

// @Summary		Change subscription plan (Admin)
// @Description	Change the plan of any subscription
// @Tags			admin-subscriptions
// @Accept			json
// @Produce		json
// @Security		Bearer
// @Param			id		path		int					true	"Subscription ID"
// @Param			plan	body		ChangePlanRequest	true	"Plan change details"
// @Success		200		{object}	utils.APIResponse	"Plan changed successfully"
// @Failure		400		{object}	utils.APIResponse	"Bad request"
// @Failure		401		{object}	utils.APIResponse	"Unauthorized"
// @Failure		403		{object}	utils.APIResponse	"Forbidden - admin only"
// @Failure		404		{object}	utils.APIResponse	"Subscription or plan not found"
// @Failure		500		{object}	utils.APIResponse	"Internal server error"
// @Router			/admin/subscriptions/{id}/plan [patch]
func (h *SubscriptionHandler) ChangePlan(c *gin.Context) {
	subscriptionID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid subscription ID")
		return
	}

	var req ChangePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for change plan", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.ChangePlanCommand{
		SubscriptionID: uint(subscriptionID),
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
