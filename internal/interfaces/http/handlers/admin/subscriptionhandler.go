// Package admin provides HTTP handlers for administrative operations.
package admin

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	subdto "github.com/orris-inc/orris/internal/application/subscription/dto"
	"github.com/orris-inc/orris/internal/application/subscription/usecases"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/constants"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// SubscriptionHandler handles admin subscription operations
type SubscriptionHandler struct {
	subscriptionRepo    subscription.SubscriptionRepository
	createUseCase       *usecases.CreateSubscriptionUseCase
	getUseCase          *usecases.GetSubscriptionUseCase
	listUseCase         *usecases.ListUserSubscriptionsUseCase
	cancelUseCase       *usecases.CancelSubscriptionUseCase
	deleteUseCase       *usecases.DeleteSubscriptionUseCase
	renewUseCase        *usecases.RenewSubscriptionUseCase
	changePlanUseCase   *usecases.ChangePlanUseCase
	activateUseCase     *usecases.ActivateSubscriptionUseCase
	suspendUseCase      *usecases.SuspendSubscriptionUseCase
	unsuspendUseCase    *usecases.UnsuspendSubscriptionUseCase
	resetUsageUseCase   *usecases.ResetSubscriptionUsageUseCase
	logger              logger.Interface
}

// NewSubscriptionHandler creates a new admin subscription handler
func NewSubscriptionHandler(
	subscriptionRepo subscription.SubscriptionRepository,
	createUC *usecases.CreateSubscriptionUseCase,
	getUC *usecases.GetSubscriptionUseCase,
	listUC *usecases.ListUserSubscriptionsUseCase,
	cancelUC *usecases.CancelSubscriptionUseCase,
	deleteUC *usecases.DeleteSubscriptionUseCase,
	renewUC *usecases.RenewSubscriptionUseCase,
	changePlanUC *usecases.ChangePlanUseCase,
	activateUC *usecases.ActivateSubscriptionUseCase,
	suspendUC *usecases.SuspendSubscriptionUseCase,
	unsuspendUC *usecases.UnsuspendSubscriptionUseCase,
	resetUsageUC *usecases.ResetSubscriptionUsageUseCase,
	logger logger.Interface,
) *SubscriptionHandler {
	return &SubscriptionHandler{
		subscriptionRepo:    subscriptionRepo,
		createUseCase:       createUC,
		getUseCase:          getUC,
		listUseCase:         listUC,
		cancelUseCase:       cancelUC,
		deleteUseCase:       deleteUC,
		renewUseCase:        renewUC,
		changePlanUseCase:   changePlanUC,
		activateUseCase:     activateUC,
		suspendUseCase:      suspendUC,
		unsuspendUseCase:    unsuspendUC,
		resetUsageUseCase:   resetUsageUC,
		logger:              logger,
	}
}

// CreateSubscriptionRequest represents the request to create a subscription
type CreateSubscriptionRequest struct {
	UserID       string                 `json:"user_id" binding:"required"` // Stripe-style SID (user_xxx)
	PlanID       string                 `json:"plan_id" binding:"required"` // Stripe-style SID (plan_xxx)
	BillingCycle string                 `json:"billing_cycle" binding:"required,oneof=weekly monthly quarterly semi_annual yearly lifetime"`
	StartDate    *time.Time             `json:"start_date"`
	AutoRenew    *bool                  `json:"auto_renew"`
	PaymentInfo  map[string]interface{} `json:"payment_info"`
	Activate     *bool                  `json:"activate"` // Whether to activate immediately, defaults to true for admin
}

// UpdateStatusRequest represents the request to update subscription status
type UpdateStatusRequest struct {
	Status    string  `json:"status" binding:"required,oneof=active cancelled suspended"`
	Reason    *string `json:"reason"`
	Immediate *bool   `json:"immediate"`
}

// SuspendRequest represents the request to suspend a subscription
type SuspendRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// RenewRequest represents the request to manually renew a subscription
type RenewRequest struct {
	BillingCycle *string `json:"billing_cycle,omitempty"` // Optional: weekly, monthly, quarterly, semi_annual, yearly, lifetime. If empty, uses current billing cycle.
}

// CreateSubscriptionResponse represents the response for subscription creation
type CreateSubscriptionResponse struct {
	Subscription *subdto.SubscriptionDTO      `json:"subscription"`
	Token        *subdto.SubscriptionTokenDTO `json:"token"`
}

// ChangePlanRequest represents the request to change subscription plan
type ChangePlanRequest struct {
	NewPlanID     string `json:"new_plan_id" binding:"required"` // Stripe-style SID (plan_xxx)
	ChangeType    string `json:"change_type" binding:"required,oneof=upgrade downgrade"`
	EffectiveDate string `json:"effective_date" binding:"required,oneof=immediate period_end"`
}

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

	// Admin-created subscriptions are activated by default
	activate := true
	if req.Activate != nil {
		activate = *req.Activate
	}

	startDate := biztime.NowUTC()
	if req.StartDate != nil {
		startDate = req.StartDate.UTC()
	}

	cmd := usecases.CreateSubscriptionCommand{
		UserSID:             req.UserID,
		PlanSID:             req.PlanID,
		BillingCycle:        req.BillingCycle,
		StartDate:           startDate,
		AutoRenew:           autoRenew,
		PaymentInfo:         req.PaymentInfo,
		ActivateImmediately: activate,
	}

	result, err := h.createUseCase.Execute(c.Request.Context(), cmd)
	if err != nil {
		h.logger.Errorw("failed to create subscription", "error", err, "user_id", req.UserID)
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Convert domain entities to DTOs for proper JSON serialization
	subscriptionDTO := subdto.ToSubscriptionDTO(result.Subscription, nil, nil, "")
	tokenDTO := subdto.ToSubscriptionTokenDTOWithPlainToken(result.Token, result.Subscription.SID(), result.PlainToken)

	utils.CreatedResponse(c, CreateSubscriptionResponse{
		Subscription: subscriptionDTO,
		Token:        tokenDTO,
	}, "Subscription created successfully")
}

// allowedSortByFields defines valid sort_by parameter values for subscription list
var allowedSortByFields = map[string]bool{
	"id": true, "sid": true, "user_id": true, "plan_id": true,
	"status": true, "billing_cycle": true, "start_date": true,
	"end_date": true, "created_at": true, "updated_at": true,
}

func (h *SubscriptionHandler) List(c *gin.Context) {
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pageSize := constants.DefaultPageSize
	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= constants.MaxPageSize {
			pageSize = ps
		}
	}

	// Validate status parameter against allowed values
	var status *string
	if statusStr := c.Query("status"); statusStr != "" {
		if valueobjects.ValidStatuses[valueobjects.SubscriptionStatus(statusStr)] {
			status = &statusStr
		} else {
			utils.ErrorResponse(c, http.StatusBadRequest, "invalid status value")
			return
		}
	}

	var userID *uint
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		if uid, err := strconv.ParseUint(userIDStr, 10, 64); err == nil {
			uidVal := uint(uid)
			userID = &uidVal
		}
	}

	var planID *uint
	if planIDStr := c.Query("plan_id"); planIDStr != "" {
		if pid, err := strconv.ParseUint(planIDStr, 10, 64); err == nil {
			pidVal := uint(pid)
			planID = &pidVal
		}
	}

	// Validate billing_cycle parameter against allowed values
	var billingCycle *string
	if billingCycleStr := c.Query("billing_cycle"); billingCycleStr != "" {
		if valueobjects.ValidBillingCycles[valueobjects.BillingCycle(billingCycleStr)] {
			billingCycle = &billingCycleStr
		} else {
			utils.ErrorResponse(c, http.StatusBadRequest, "invalid billing_cycle value")
			return
		}
	}

	var createdFrom *time.Time
	if createdFromStr := c.Query("created_from"); createdFromStr != "" {
		if t, err := time.Parse(time.RFC3339, createdFromStr); err == nil {
			createdFrom = &t
		} else {
			utils.ErrorResponse(c, http.StatusBadRequest, "invalid created_from format, use RFC3339")
			return
		}
	}

	var createdTo *time.Time
	if createdToStr := c.Query("created_to"); createdToStr != "" {
		if t, err := time.Parse(time.RFC3339, createdToStr); err == nil {
			createdTo = &t
		} else {
			utils.ErrorResponse(c, http.StatusBadRequest, "invalid created_to format, use RFC3339")
			return
		}
	}

	var expiresBefore *time.Time
	if expiresBeforeStr := c.Query("expires_before"); expiresBeforeStr != "" {
		if t, err := time.Parse(time.RFC3339, expiresBeforeStr); err == nil {
			expiresBefore = &t
		} else {
			utils.ErrorResponse(c, http.StatusBadRequest, "invalid expires_before format, use RFC3339")
			return
		}
	}

	// Validate sort_by parameter against whitelist
	var sortBy string
	if sortByStr := c.Query("sort_by"); sortByStr != "" {
		if allowedSortByFields[sortByStr] {
			sortBy = sortByStr
		} else {
			utils.ErrorResponse(c, http.StatusBadRequest, "invalid sort_by value")
			return
		}
	}

	// Validate sort_order parameter
	var sortDesc *bool
	if sortOrderStr := c.Query("sort_order"); sortOrderStr != "" {
		if sortOrderStr != "asc" && sortOrderStr != "desc" {
			utils.ErrorResponse(c, http.StatusBadRequest, "invalid sort_order value, use 'asc' or 'desc'")
			return
		}
		desc := sortOrderStr == "desc"
		sortDesc = &desc
	}

	query := usecases.ListUserSubscriptionsQuery{
		UserID:        userID,
		PlanID:        planID,
		Status:        status,
		BillingCycle:  billingCycle,
		CreatedFrom:   createdFrom,
		CreatedTo:     createdTo,
		ExpiresBefore: expiresBefore,
		Page:          page,
		PageSize:      pageSize,
		SortBy:        sortBy,
		SortDesc:      sortDesc,
	}

	result, err := h.listUseCase.Execute(c.Request.Context(), query)
	if err != nil {
		h.logger.Errorw("failed to list subscriptions", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.ListSuccessResponse(c, result.Subscriptions, result.Total, result.Page, result.PageSize)
}

// parseSubscriptionID parses subscription ID from URL parameter, supporting both Stripe-style (sub_xxx) and numeric IDs
func (h *SubscriptionHandler) parseSubscriptionID(c *gin.Context) (uint, error) {
	idStr := c.Param("id")

	// Check if ID is Stripe-style (sub_xxx)
	if strings.HasPrefix(idStr, id.PrefixSubscription+"_") {
		sub, err := h.subscriptionRepo.GetBySID(c.Request.Context(), idStr)
		if err != nil {
			return 0, err
		}
		if sub == nil {
			return 0, nil
		}
		return sub.ID(), nil
	}

	// Try parsing as numeric ID
	subscriptionID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(subscriptionID), nil
}

func (h *SubscriptionHandler) Get(c *gin.Context) {
	subscriptionID, err := h.parseSubscriptionID(c)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid subscription ID")
		return
	}
	if subscriptionID == 0 {
		utils.ErrorResponse(c, http.StatusNotFound, "subscription not found")
		return
	}

	query := usecases.GetSubscriptionQuery{
		SubscriptionID: subscriptionID,
	}

	sub, err := h.getUseCase.Execute(c.Request.Context(), query)
	if err != nil {
		h.logger.Errorw("failed to get subscription", "error", err, "subscription_id", subscriptionID)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", sub)
}

func (h *SubscriptionHandler) UpdateStatus(c *gin.Context) {
	subscriptionID, err := h.parseSubscriptionID(c)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid subscription ID")
		return
	}
	if subscriptionID == 0 {
		utils.ErrorResponse(c, http.StatusNotFound, "subscription not found")
		return
	}

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for update subscription status", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	switch req.Status {
	case string(valueobjects.StatusActive):
		cmd := usecases.ActivateSubscriptionCommand{
			SubscriptionID: subscriptionID,
		}
		if err := h.activateUseCase.Execute(c.Request.Context(), cmd); err != nil {
			h.logger.Errorw("failed to activate subscription", "error", err, "subscription_id", subscriptionID)
			utils.ErrorResponseWithError(c, err)
			return
		}
		utils.SuccessResponse(c, http.StatusOK, "Subscription activated successfully", nil)

	case string(valueobjects.StatusCancelled):
		if req.Reason == nil || *req.Reason == "" {
			utils.ErrorResponse(c, http.StatusBadRequest, "reason is required for cancellation")
			return
		}
		immediate := false
		if req.Immediate != nil {
			immediate = *req.Immediate
		}
		cmd := usecases.CancelSubscriptionCommand{
			SubscriptionID: subscriptionID,
			Reason:         *req.Reason,
			Immediate:      immediate,
		}
		if err := h.cancelUseCase.Execute(c.Request.Context(), cmd); err != nil {
			h.logger.Errorw("failed to cancel subscription", "error", err, "subscription_id", subscriptionID)
			utils.ErrorResponseWithError(c, err)
			return
		}
		utils.SuccessResponse(c, http.StatusOK, "Subscription cancelled successfully", nil)

	case string(valueobjects.StatusSuspended):
		if req.Reason == nil || *req.Reason == "" {
			utils.ErrorResponse(c, http.StatusBadRequest, "reason is required for suspension")
			return
		}
		cmd := usecases.SuspendSubscriptionCommand{
			SubscriptionID: subscriptionID,
			Reason:         *req.Reason,
		}
		if err := h.suspendUseCase.Execute(c.Request.Context(), cmd); err != nil {
			h.logger.Errorw("failed to suspend subscription", "error", err, "subscription_id", subscriptionID)
			utils.ErrorResponseWithError(c, err)
			return
		}
		utils.SuccessResponse(c, http.StatusOK, "Subscription suspended successfully", nil)

	default:
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid status value")
	}
}

func (h *SubscriptionHandler) ChangePlan(c *gin.Context) {
	subscriptionID, err := h.parseSubscriptionID(c)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid subscription ID")
		return
	}
	if subscriptionID == 0 {
		utils.ErrorResponse(c, http.StatusNotFound, "subscription not found")
		return
	}

	var req ChangePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for change plan", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.ChangePlanCommand{
		SubscriptionID: subscriptionID,
		NewPlanSID:     req.NewPlanID, // Use Stripe-style SID
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

func (h *SubscriptionHandler) Delete(c *gin.Context) {
	subscriptionID, err := h.parseSubscriptionID(c)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid subscription ID")
		return
	}
	if subscriptionID == 0 {
		utils.ErrorResponse(c, http.StatusNotFound, "subscription not found")
		return
	}

	if err := h.deleteUseCase.Execute(c.Request.Context(), subscriptionID); err != nil {
		h.logger.Errorw("failed to delete subscription", "error", err, "subscription_id", subscriptionID)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Subscription deleted successfully", nil)
}

// Suspend suspends a subscription
func (h *SubscriptionHandler) Suspend(c *gin.Context) {
	subscriptionID, err := h.parseSubscriptionID(c)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid subscription ID")
		return
	}
	if subscriptionID == 0 {
		utils.ErrorResponse(c, http.StatusNotFound, "subscription not found")
		return
	}

	var req SuspendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for suspend subscription", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.SuspendSubscriptionCommand{
		SubscriptionID: subscriptionID,
		Reason:         req.Reason,
	}

	if err := h.suspendUseCase.Execute(c.Request.Context(), cmd); err != nil {
		h.logger.Errorw("failed to suspend subscription", "error", err, "subscription_id", subscriptionID)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Subscription suspended successfully", nil)
}

// Unsuspend reactivates a suspended subscription
func (h *SubscriptionHandler) Unsuspend(c *gin.Context) {
	subscriptionID, err := h.parseSubscriptionID(c)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid subscription ID")
		return
	}
	if subscriptionID == 0 {
		utils.ErrorResponse(c, http.StatusNotFound, "subscription not found")
		return
	}

	cmd := usecases.UnsuspendSubscriptionCommand{
		SubscriptionID: subscriptionID,
	}

	if err := h.unsuspendUseCase.Execute(c.Request.Context(), cmd); err != nil {
		h.logger.Errorw("failed to unsuspend subscription", "error", err, "subscription_id", subscriptionID)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Subscription unsuspended successfully", nil)
}

// ResetUsage resets a subscription's traffic usage
func (h *SubscriptionHandler) ResetUsage(c *gin.Context) {
	subscriptionID, err := h.parseSubscriptionID(c)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid subscription ID")
		return
	}
	if subscriptionID == 0 {
		utils.ErrorResponse(c, http.StatusNotFound, "subscription not found")
		return
	}

	cmd := usecases.ResetSubscriptionUsageCommand{
		SubscriptionID: subscriptionID,
	}

	if err := h.resetUsageUseCase.Execute(c.Request.Context(), cmd); err != nil {
		h.logger.Errorw("failed to reset subscription usage", "error", err, "subscription_id", subscriptionID)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Subscription usage reset successfully", nil)
}

// Renew manually renews a subscription for another billing period
func (h *SubscriptionHandler) Renew(c *gin.Context) {
	subscriptionID, err := h.parseSubscriptionID(c)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid subscription ID")
		return
	}
	if subscriptionID == 0 {
		utils.ErrorResponse(c, http.StatusNotFound, "subscription not found")
		return
	}

	var req RenewRequest
	// Allow empty body - billing_cycle is optional
	_ = c.ShouldBindJSON(&req)

	var billingCycle string
	if req.BillingCycle != nil {
		billingCycle = *req.BillingCycle
	}

	cmd := usecases.RenewSubscriptionCommand{
		SubscriptionID: subscriptionID,
		BillingCycle:   billingCycle, // Optional: if empty, uses subscription's current billing cycle
		IsAutoRenew:    false,
	}

	if err := h.renewUseCase.Execute(c.Request.Context(), cmd); err != nil {
		h.logger.Errorw("failed to renew subscription", "error", err, "subscription_id", subscriptionID)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Subscription renewed successfully", nil)
}
