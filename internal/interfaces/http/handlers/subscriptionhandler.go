package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	subdto "github.com/orris-inc/orris/internal/application/subscription/dto"
	"github.com/orris-inc/orris/internal/application/subscription/usecases"
	"github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/constants"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

var (
	_ = subdto.SubscriptionDTO{}
	_ = subdto.SubscriptionTokenDTO{}
)

// SubscriptionHandler handles user subscription operations
type SubscriptionHandler struct {
	createUseCase        *usecases.CreateSubscriptionUseCase
	getUseCase           *usecases.GetSubscriptionUseCase
	listUserUseCase      *usecases.ListUserSubscriptionsUseCase
	cancelUseCase        *usecases.CancelSubscriptionUseCase
	deleteUseCase        *usecases.DeleteSubscriptionUseCase
	changePlanUseCase    *usecases.ChangePlanUseCase
	getUsageStatsUseCase *usecases.GetSubscriptionUsageStatsUseCase
	resetLinkUseCase     *usecases.ResetSubscriptionLinkUseCase
	logger               logger.Interface
}

// NewSubscriptionHandler creates a new user subscription handler
func NewSubscriptionHandler(
	createUC *usecases.CreateSubscriptionUseCase,
	getUC *usecases.GetSubscriptionUseCase,
	listUserUC *usecases.ListUserSubscriptionsUseCase,
	cancelUC *usecases.CancelSubscriptionUseCase,
	deleteUC *usecases.DeleteSubscriptionUseCase,
	changePlanUC *usecases.ChangePlanUseCase,
	getUsageStatsUC *usecases.GetSubscriptionUsageStatsUseCase,
	resetLinkUC *usecases.ResetSubscriptionLinkUseCase,
	logger logger.Interface,
) *SubscriptionHandler {
	return &SubscriptionHandler{
		createUseCase:        createUC,
		getUseCase:           getUC,
		listUserUseCase:      listUserUC,
		cancelUseCase:        cancelUC,
		deleteUseCase:        deleteUC,
		changePlanUseCase:    changePlanUC,
		getUsageStatsUseCase: getUsageStatsUC,
		resetLinkUseCase:     resetLinkUC,
		logger:               logger,
	}
}

// CreateSubscriptionRequest represents the request to create a subscription for self
type CreateSubscriptionRequest struct {
	PlanID       string                 `json:"plan_id" binding:"required"` // Stripe-style plan SID (plan_xxx)
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
	NewPlanID     string `json:"new_plan_id" binding:"required"` // Stripe-style plan SID (plan_xxx)
	ChangeType    string `json:"change_type" binding:"required,oneof=upgrade downgrade"`
	EffectiveDate string `json:"effective_date" binding:"required,oneof=immediate period_end"`
}

// CreateSubscriptionResponse represents the response for subscription creation
type CreateSubscriptionResponse struct {
	Subscription *subdto.SubscriptionDTO      `json:"subscription"`
	Token        *subdto.SubscriptionTokenDTO `json:"token"`
}

func (h *SubscriptionHandler) CreateSubscription(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
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

	startDate := biztime.NowUTC()
	if req.StartDate != nil {
		startDate = req.StartDate.UTC()
	}

	// Validate plan SID format
	if err := id.ValidatePrefix(req.PlanID, id.PrefixPlan); err != nil {
		h.logger.Warnw("invalid plan ID format", "plan_id", req.PlanID, "error", err)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid plan ID format, expected plan_xxxxx")
		return
	}

	cmd := usecases.CreateSubscriptionCommand{
		UserID:       userID,
		PlanSID:      req.PlanID,
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

	// Convert domain entities to DTOs for proper JSON serialization
	subscriptionDTO := subdto.ToSubscriptionDTO(result.Subscription, nil, nil, "")
	tokenDTO := subdto.ToSubscriptionTokenDTOWithPlainToken(result.Token, result.Subscription.SID(), result.PlainToken)

	utils.CreatedResponse(c, CreateSubscriptionResponse{
		Subscription: subscriptionDTO,
		Token:        tokenDTO,
	}, "Subscription created successfully")
}

func (h *SubscriptionHandler) GetSubscription(c *gin.Context) {
	// Ownership already verified by middleware, subscription stored in context
	subscriptionID, err := utils.GetSubscriptionIDFromContext(c)
	if err != nil {
		// Fallback: parse SID from URL parameter
		sidStr := c.Param("id")
		if err := id.ValidatePrefix(sidStr, id.PrefixSubscription); err != nil {
			utils.ErrorResponse(c, http.StatusBadRequest, "invalid subscription ID format, expected sub_xxxxx")
			return
		}
		// Use SID-based query
		subscription, err := h.getUseCase.ExecuteBySID(c.Request.Context(), sidStr)
		if err != nil {
			h.logger.Errorw("failed to get subscription by SID", "error", err, "subscription_sid", sidStr)
			utils.ErrorResponseWithError(c, err)
			return
		}
		utils.SuccessResponse(c, http.StatusOK, "", subscription)
		return
	}

	query := usecases.GetSubscriptionQuery{
		SubscriptionID: subscriptionID,
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
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	pg := utils.ParsePagination(c)

	var status *string
	if statusStr := c.Query("status"); statusStr != "" {
		status = &statusStr
	}

	query := usecases.ListUserSubscriptionsQuery{
		UserID:   &userID,
		Status:   status,
		Page:     pg.Page,
		PageSize: pg.PageSize,
	}

	result, err := h.listUserUseCase.Execute(c.Request.Context(), query)
	if err != nil {
		h.logger.Errorw("failed to list subscriptions", "error", err, "user_id", userID)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.ListSuccessResponse(c, result.Subscriptions, result.Total, result.Page, result.PageSize)
}

func (h *SubscriptionHandler) UpdateStatus(c *gin.Context) {
	subscriptionID, err := utils.GetSubscriptionIDFromContext(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for update subscription status", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	switch req.Status {
	case string(valueobjects.StatusCancelled):
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
			SubscriptionID: subscriptionID,
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
	subscriptionID, err := utils.GetSubscriptionIDFromContext(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	var req ChangePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for change plan", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Validate new plan SID format
	if err := id.ValidatePrefix(req.NewPlanID, id.PrefixPlan); err != nil {
		h.logger.Warnw("invalid new plan ID format", "new_plan_id", req.NewPlanID, "error", err)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid new_plan_id format, expected plan_xxxxx")
		return
	}

	cmd := usecases.ChangePlanCommand{
		SubscriptionID: subscriptionID,
		NewPlanSID:     req.NewPlanID,
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

// GetTrafficStats handles GET /subscriptions/:sid/traffic-stats
func (h *SubscriptionHandler) GetTrafficStats(c *gin.Context) {
	subscriptionID, err := utils.GetSubscriptionIDFromContext(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
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

	// Traffic stats: default page size is MaxPageSize, upper limit is 1000
	pg := utils.ParsePaginationWithLimits(c, constants.MaxPageSize, 1000)

	query := usecases.GetSubscriptionUsageStatsQuery{
		SubscriptionID: subscriptionID,
		From:           from,
		To:             to,
		Granularity:    granularity,
		Page:           pg.Page,
		PageSize:       pg.PageSize,
	}

	result, err := h.getUsageStatsUseCase.Execute(c.Request.Context(), query)
	if err != nil {
		h.logger.Errorw("failed to get subscription traffic stats", "error", err, "subscription_id", subscriptionID)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}

// ResetLink handles PUT /subscriptions/:sid/link
// Resets the subscription link by generating a new UUID
func (h *SubscriptionHandler) ResetLink(c *gin.Context) {
	subscriptionID, err := utils.GetSubscriptionIDFromContext(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.ResetSubscriptionLinkCommand{
		SubscriptionID: subscriptionID,
	}

	result, err := h.resetLinkUseCase.Execute(c.Request.Context(), cmd)
	if err != nil {
		h.logger.Errorw("failed to reset subscription link", "error", err, "subscription_id", subscriptionID)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Subscription link reset successfully", result)
}

// DeleteSubscription handles DELETE /subscriptions/:sid
func (h *SubscriptionHandler) DeleteSubscription(c *gin.Context) {
	subscriptionID, err := utils.GetSubscriptionIDFromContext(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	if err := h.deleteUseCase.Execute(c.Request.Context(), subscriptionID); err != nil {
		h.logger.Errorw("failed to delete subscription", "error", err, "subscription_id", subscriptionID)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Subscription deleted successfully", nil)
}
