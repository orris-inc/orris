package subscription

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/subscription/usecases"
	"github.com/orris-inc/orris/internal/domain/subscription/valueobjects"
	"github.com/orris-inc/orris/internal/shared/utils"
)

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

// ChangePlanRequest represents the request to change subscription plan
type ChangePlanRequest struct {
	NewPlanID     string `json:"new_plan_id" binding:"required"` // Stripe-style SID (plan_xxx)
	ChangeType    string `json:"change_type" binding:"required,oneof=upgrade downgrade"`
	EffectiveDate string `json:"effective_date" binding:"required,oneof=immediate period_end"`
}

func (h *Handler) UpdateStatus(c *gin.Context) {
	subscriptionID, err := h.ParseSubscriptionID(c)
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

func (h *Handler) ChangePlan(c *gin.Context) {
	subscriptionID, err := h.ParseSubscriptionID(c)
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

// Suspend suspends a subscription
func (h *Handler) Suspend(c *gin.Context) {
	subscriptionID, err := h.ParseSubscriptionID(c)
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
func (h *Handler) Unsuspend(c *gin.Context) {
	subscriptionID, err := h.ParseSubscriptionID(c)
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
func (h *Handler) ResetUsage(c *gin.Context) {
	subscriptionID, err := h.ParseSubscriptionID(c)
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
func (h *Handler) Renew(c *gin.Context) {
	subscriptionID, err := h.ParseSubscriptionID(c)
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
