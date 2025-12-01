package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"orris/internal/application/subscription/usecases"
	"orris/internal/shared/authorization"
	"orris/internal/shared/logger"
	"orris/internal/shared/utils"
)

type SubscriptionHandler struct {
	createUseCase     *usecases.CreateSubscriptionUseCase
	getUseCase        *usecases.GetSubscriptionUseCase
	listUserUseCase   *usecases.ListUserSubscriptionsUseCase
	cancelUseCase     *usecases.CancelSubscriptionUseCase
	renewUseCase      *usecases.RenewSubscriptionUseCase
	changePlanUseCase *usecases.ChangePlanUseCase
	activateUseCase   *usecases.ActivateSubscriptionUseCase
	logger            logger.Interface
}

func NewSubscriptionHandler(
	createUC *usecases.CreateSubscriptionUseCase,
	getUC *usecases.GetSubscriptionUseCase,
	listUserUC *usecases.ListUserSubscriptionsUseCase,
	cancelUC *usecases.CancelSubscriptionUseCase,
	renewUC *usecases.RenewSubscriptionUseCase,
	changePlanUC *usecases.ChangePlanUseCase,
	activateUC *usecases.ActivateSubscriptionUseCase,
	logger logger.Interface,
) *SubscriptionHandler {
	return &SubscriptionHandler{
		createUseCase:     createUC,
		getUseCase:        getUC,
		listUserUseCase:   listUserUC,
		cancelUseCase:     cancelUC,
		renewUseCase:      renewUC,
		changePlanUseCase: changePlanUC,
		activateUseCase:   activateUC,
		logger:            logger,
	}
}

type CreateSubscriptionRequest struct {
	UserID       *uint                  `json:"user_id"` // Optional: Admin can specify target user ID
	PlanID       uint                   `json:"plan_id" binding:"required"`
	BillingCycle string                 `json:"billing_cycle" binding:"required,oneof=weekly monthly quarterly semi_annual yearly lifetime"`
	StartDate    *time.Time             `json:"start_date"`
	AutoRenew    *bool                  `json:"auto_renew"`
	PaymentInfo  map[string]interface{} `json:"payment_info"`
}

type CancelSubscriptionRequest struct {
	Reason    string `json:"reason" binding:"required"`
	Immediate *bool  `json:"immediate"`
}

// UpdateSubscriptionStatusRequest represents a unified request for subscription status changes
type UpdateSubscriptionStatusRequest struct {
	Status    string  `json:"status" binding:"required,oneof=active cancelled renewed"`
	Reason    *string `json:"reason"`    // Required when status is "cancelled"
	Immediate *bool   `json:"immediate"` // Optional for cancellation
}

type ChangePlanRequest struct {
	NewPlanID     uint   `json:"new_plan_id" binding:"required"`
	ChangeType    string `json:"change_type" binding:"required,oneof=upgrade downgrade"`
	EffectiveDate string `json:"effective_date" binding:"required,oneof=immediate period_end"`
}

// SubscriptionCreateResult represents the response for creating a subscription
type SubscriptionCreateResult struct {
	Subscription interface{} `json:"subscription"`
	Token        interface{} `json:"token"`
}

// SubscriptionResponse represents the response for subscription details
type SubscriptionResponse struct {
	ID                 uint       `json:"id"`
	UserID             uint       `json:"user_id"`
	PlanID             uint       `json:"plan_id"`
	Status             string     `json:"status"`
	StartDate          time.Time  `json:"start_date"`
	EndDate            time.Time  `json:"end_date"`
	AutoRenew          bool       `json:"auto_renew"`
	CurrentPeriodStart time.Time  `json:"current_period_start"`
	CurrentPeriodEnd   time.Time  `json:"current_period_end"`
	IsExpired          bool       `json:"is_expired"`
	IsActive           bool       `json:"is_active"`
	CancelledAt        *time.Time `json:"cancelled_at,omitempty"`
	CancelReason       *string    `json:"cancel_reason,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// @Summary		Create a new subscription with billing cycle selection
// @Description	Create a new subscription for the authenticated user with the specified plan and billing cycle. Admin users can optionally specify a user_id to create subscription for any user.
// @Tags			subscriptions
// @Accept			json
// @Produce		json
// @Security		Bearer
// @Param			subscription	body		CreateSubscriptionRequest							true	"Subscription data with billing cycle (weekly/monthly/quarterly/semi_annual/yearly/lifetime). For admin: optionally specify user_id to create subscription for another user."
// @Success		201				{object}	utils.APIResponse{data=SubscriptionCreateResult}	"Subscription created successfully"
// @Failure		400				{object}	utils.APIResponse									"Bad request"
// @Failure		401				{object}	utils.APIResponse									"Unauthorized"
// @Failure		403				{object}	utils.APIResponse									"Forbidden - only admin can create subscription for other users"
// @Failure		404				{object}	utils.APIResponse									"Plan not found or user not found"
// @Failure		500				{object}	utils.APIResponse									"Internal server error"
// @Router			/subscriptions [post]
func (h *SubscriptionHandler) CreateSubscription(c *gin.Context) {
	currentUserID, exists := c.Get("user_id")
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

	// Determine target user ID based on role and request
	var targetUserID uint
	userRole := c.GetString("user_role")

	if req.UserID != nil && *req.UserID != 0 {
		// Request specifies a user_id
		if userRole != string(authorization.RoleAdmin) {
			h.logger.Warnw("non-admin user attempted to create subscription for another user",
				"current_user_id", currentUserID,
				"target_user_id", *req.UserID,
			)
			utils.ErrorResponse(c, http.StatusForbidden, "only admin can create subscription for other users")
			return
		}
		targetUserID = *req.UserID
		h.logger.Infow("admin creating subscription for user",
			"admin_id", currentUserID,
			"target_user_id", targetUserID,
		)
	} else {
		// No user_id specified, create for current user
		targetUserID = currentUserID.(uint)
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
		UserID:       targetUserID,
		PlanID:       req.PlanID,
		BillingCycle: req.BillingCycle,
		StartDate:    startDate,
		AutoRenew:    autoRenew,
		PaymentInfo:  req.PaymentInfo,
	}

	result, err := h.createUseCase.Execute(c.Request.Context(), cmd)
	if err != nil {
		h.logger.Errorw("failed to create subscription", "error", err, "target_user_id", targetUserID)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.CreatedResponse(c, gin.H{
		"subscription": result.Subscription,
		"token":        result.Token,
	}, "Subscription created successfully")
}

// @Summary		Get subscription by ID
// @Description	Get details of a specific subscription by its ID
// @Tags			subscriptions
// @Accept			json
// @Produce		json
// @Security		Bearer
// @Param			id	path		int												true	"Subscription ID"
// @Success		200	{object}	utils.APIResponse{data=SubscriptionResponse}	"Subscription details"
// @Failure		400	{object}	utils.APIResponse								"Invalid subscription ID"
// @Failure		401	{object}	utils.APIResponse								"Unauthorized"
// @Failure		403	{object}	utils.APIResponse								"Access denied"
// @Failure		404	{object}	utils.APIResponse								"Subscription not found"
// @Failure		500	{object}	utils.APIResponse								"Internal server error"
// @Router			/subscriptions/{id} [get]
func (h *SubscriptionHandler) GetSubscription(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

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

	if subscription.UserID != userID.(uint) {
		utils.ErrorResponse(c, http.StatusForbidden, "access denied")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", subscription)
}

// @Summary		List user subscriptions
// @Description	Get a paginated list of subscriptions for the authenticated user
// @Tags			subscriptions
// @Accept			json
// @Produce		json
// @Security		Bearer
// @Param			page		query		int											false	"Page number"					default(1)
// @Param			page_size	query		int											false	"Page size"						default(20)
// @Param			status		query		string										false	"Subscription status filter"	Enums(active,inactive,cancelled,expired,pending)
// @Success		200			{object}	utils.APIResponse{data=utils.ListResponse}	"Subscriptions list"
// @Failure		400			{object}	utils.APIResponse							"Invalid query parameters"
// @Failure		401			{object}	utils.APIResponse							"Unauthorized"
// @Failure		500			{object}	utils.APIResponse							"Internal server error"
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

	query := usecases.ListUserSubscriptionsQuery{
		UserID:   userID.(uint),
		Status:   status,
		Page:     page,
		PageSize: pageSize,
	}

	result, err := h.listUserUseCase.Execute(c.Request.Context(), query)
	if err != nil {
		h.logger.Errorw("failed to list user subscriptions", "error", err, "user_id", userID)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.ListSuccessResponse(c, result.Subscriptions, result.Total, result.Page, result.PageSize)
}

// @Summary		Change subscription plan
// @Description	Change the plan of an existing subscription (upgrade or downgrade)
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
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

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

	query := usecases.GetSubscriptionQuery{
		SubscriptionID: uint(subscriptionID),
	}

	subscription, err := h.getUseCase.Execute(c.Request.Context(), query)
	if err != nil {
		h.logger.Errorw("failed to get subscription", "error", err, "subscription_id", subscriptionID)
		utils.ErrorResponseWithError(c, err)
		return
	}

	if subscription.UserID != userID.(uint) {
		utils.ErrorResponse(c, http.StatusForbidden, "access denied")
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

// @Summary		Update subscription status
// @Description	Update subscription status (activate, cancel, or renew)
// @Tags			subscriptions
// @Accept			json
// @Produce		json
// @Security		Bearer
// @Param			id		path		int									true	"Subscription ID"
// @Param			status	body		UpdateSubscriptionStatusRequest		true	"Status update details"
// @Success		200		{object}	utils.APIResponse					"Subscription status updated successfully"
// @Failure		400		{object}	utils.APIResponse					"Bad request"
// @Failure		401		{object}	utils.APIResponse					"Unauthorized"
// @Failure		403		{object}	utils.APIResponse					"Access denied"
// @Failure		404		{object}	utils.APIResponse					"Subscription not found"
// @Failure		500		{object}	utils.APIResponse					"Internal server error"
// @Router			/subscriptions/{id}/status [patch]
func (h *SubscriptionHandler) UpdateSubscriptionStatus(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponse(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	subscriptionID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid subscription ID")
		return
	}

	var req UpdateSubscriptionStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for update subscription status", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Verify ownership
	query := usecases.GetSubscriptionQuery{
		SubscriptionID: uint(subscriptionID),
	}

	subscription, err := h.getUseCase.Execute(c.Request.Context(), query)
	if err != nil {
		h.logger.Errorw("failed to get subscription", "error", err, "subscription_id", subscriptionID)
		utils.ErrorResponseWithError(c, err)
		return
	}

	if subscription.UserID != userID.(uint) {
		utils.ErrorResponse(c, http.StatusForbidden, "access denied")
		return
	}

	// Execute the appropriate action based on status
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
