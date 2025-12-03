package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	subdto "github.com/orris-inc/orris/internal/application/subscription/dto"
	"github.com/orris-inc/orris/internal/application/subscription/usecases"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

var (
	_ = subdto.SubscriptionDTO{}
	_ = subdto.SubscriptionTokenDTO{}
)

// SubscriptionHandler handles user subscription operations
type SubscriptionHandler struct {
	createUseCase          *usecases.CreateSubscriptionUseCase
	getUseCase             *usecases.GetSubscriptionUseCase
	listUserUseCase        *usecases.ListUserSubscriptionsUseCase
	cancelUseCase          *usecases.CancelSubscriptionUseCase
	changePlanUseCase      *usecases.ChangePlanUseCase
	getTrafficStatsUseCase *usecases.GetSubscriptionTrafficStatsUseCase
	logger                 logger.Interface
}

// NewSubscriptionHandler creates a new user subscription handler
func NewSubscriptionHandler(
	createUC *usecases.CreateSubscriptionUseCase,
	getUC *usecases.GetSubscriptionUseCase,
	listUserUC *usecases.ListUserSubscriptionsUseCase,
	cancelUC *usecases.CancelSubscriptionUseCase,
	changePlanUC *usecases.ChangePlanUseCase,
	getTrafficStatsUC *usecases.GetSubscriptionTrafficStatsUseCase,
	logger logger.Interface,
) *SubscriptionHandler {
	return &SubscriptionHandler{
		createUseCase:          createUC,
		getUseCase:             getUC,
		listUserUseCase:        listUserUC,
		cancelUseCase:          cancelUC,
		changePlanUseCase:      changePlanUC,
		getTrafficStatsUseCase: getTrafficStatsUC,
		logger:                 logger,
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

// UpdateStatusRequest represents the request to update subscription status
type UpdateStatusRequest struct {
	Status    string  `json:"status" binding:"required,oneof=cancelled"`
	Reason    *string `json:"reason"`
	Immediate *bool   `json:"immediate"`
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

func (h *SubscriptionHandler) UpdateStatus(c *gin.Context) {
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

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for update subscription status", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	switch req.Status {
	case "cancelled":
		reason := ""
		if req.Reason != nil {
			reason = *req.Reason
		}
		if reason == "" {
			utils.ErrorResponse(c, http.StatusBadRequest, "reason is required for cancellation")
			return
		}

		immediate := false
		if req.Immediate != nil {
			immediate = *req.Immediate
		}

		cmd := usecases.CancelSubscriptionCommand{
			SubscriptionID: subscriptionID.(uint),
			Reason:         reason,
			Immediate:      immediate,
		}

		if err := h.cancelUseCase.Execute(c.Request.Context(), cmd); err != nil {
			h.logger.Errorw("failed to cancel subscription", "error", err, "subscription_id", subscriptionID)
			utils.ErrorResponseWithError(c, err)
			return
		}
		utils.SuccessResponse(c, http.StatusOK, "Subscription cancelled successfully", nil)

	default:
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid status value")
	}
}

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

// GetTrafficStats handles GET /subscriptions/:id/traffic-stats
func (h *SubscriptionHandler) GetTrafficStats(c *gin.Context) {
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

	// Parse query parameters
	fromStr := c.Query("from")
	toStr := c.Query("to")

	if fromStr == "" || toStr == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "from and to query parameters are required")
		return
	}

	from, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid from time format, use RFC3339")
		return
	}

	to, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid to time format, use RFC3339")
		return
	}

	granularity := c.Query("granularity")

	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pageSize := 100
	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 1000 {
			pageSize = ps
		}
	}

	query := usecases.GetSubscriptionTrafficStatsQuery{
		SubscriptionID: subscriptionID.(uint),
		From:           from,
		To:             to,
		Granularity:    granularity,
		Page:           page,
		PageSize:       pageSize,
	}

	result, err := h.getTrafficStatsUseCase.Execute(c.Request.Context(), query)
	if err != nil {
		h.logger.Errorw("failed to get subscription traffic stats", "error", err, "subscription_id", subscriptionID)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}
