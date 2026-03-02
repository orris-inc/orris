// Package subscription provides HTTP handlers for admin subscription operations.
package subscription

import (
	"encoding/json"
	"fmt"
	"io"
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
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// Handler handles admin subscription operations
type Handler struct {
	subscriptionRepo  subscription.SubscriptionRepository
	createUseCase     *usecases.CreateSubscriptionUseCase
	getUseCase        *usecases.GetSubscriptionUseCase
	listUseCase       *usecases.ListUserSubscriptionsUseCase
	cancelUseCase     *usecases.CancelSubscriptionUseCase
	deleteUseCase     *usecases.DeleteSubscriptionUseCase
	renewUseCase      *usecases.RenewSubscriptionUseCase
	changePlanUseCase *usecases.ChangePlanUseCase
	activateUseCase   *usecases.ActivateSubscriptionUseCase
	suspendUseCase    *usecases.SuspendSubscriptionUseCase
	unsuspendUseCase  *usecases.UnsuspendSubscriptionUseCase
	resetUsageUseCase *usecases.ResetSubscriptionUsageUseCase
	updateUseCase     *usecases.UpdateSubscriptionUseCase
	logger            logger.Interface
}

// NewHandler creates a new admin subscription handler
func NewHandler(
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
	updateUC *usecases.UpdateSubscriptionUseCase,
	logger logger.Interface,
) *Handler {
	return &Handler{
		subscriptionRepo:  subscriptionRepo,
		createUseCase:     createUC,
		getUseCase:        getUC,
		listUseCase:       listUC,
		cancelUseCase:     cancelUC,
		deleteUseCase:     deleteUC,
		renewUseCase:      renewUC,
		changePlanUseCase: changePlanUC,
		activateUseCase:   activateUC,
		suspendUseCase:    suspendUC,
		unsuspendUseCase:  unsuspendUC,
		resetUsageUseCase: resetUsageUC,
		updateUseCase:     updateUC,
		logger:            logger,
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

// CreateSubscriptionResponse represents the response for subscription creation
type CreateSubscriptionResponse struct {
	Subscription *subdto.SubscriptionDTO      `json:"subscription"`
	Token        *subdto.SubscriptionTokenDTO `json:"token"`
}

// allowedSortByFields defines valid sort_by parameter values for subscription list
var allowedSortByFields = map[string]bool{
	"id": true, "sid": true, "user_id": true, "plan_id": true,
	"status": true, "billing_cycle": true, "start_date": true,
	"end_date": true, "created_at": true, "updated_at": true,
}

// ParseSubscriptionID parses subscription ID from URL parameter.
// Only accepts Stripe-style SID format (sub_xxx) to prevent ID enumeration.
func (h *Handler) ParseSubscriptionID(c *gin.Context) (uint, error) {
	idStr := c.Param("id")

	// Only accept Stripe-style SID format (sub_xxx) to prevent ID enumeration
	if !strings.HasPrefix(idStr, id.PrefixSubscription+"_") {
		return 0, fmt.Errorf("invalid subscription ID format: expected sub_xxx")
	}

	sub, err := h.subscriptionRepo.GetBySID(c.Request.Context(), idStr)
	if err != nil {
		return 0, err
	}
	if sub == nil {
		return 0, nil
	}
	return sub.ID(), nil
}

func (h *Handler) Create(c *gin.Context) {
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
	// Newly created subscription has 0 online devices; set device_limit from plan
	opts := []subdto.SubscriptionDTOOption{subdto.WithOnlineDeviceCount(0)}
	if result.Plan != nil && result.Plan.Features() != nil {
		if deviceLimit, err := result.Plan.Features().GetDeviceLimit(); err == nil {
			opts = append(opts, subdto.WithDeviceLimit(deviceLimit))
		}
	}
	subscriptionDTO := subdto.ToSubscriptionDTO(result.Subscription, nil, nil, "", opts...)
	tokenDTO := subdto.ToSubscriptionTokenDTOWithPlainToken(result.Token, result.Subscription.SID(), result.PlainToken)

	utils.CreatedResponse(c, CreateSubscriptionResponse{
		Subscription: subscriptionDTO,
		Token:        tokenDTO,
	}, "Subscription created successfully")
}

func (h *Handler) List(c *gin.Context) {
	p := utils.ParsePagination(c)

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
		Page:          p.Page,
		PageSize:      p.PageSize,
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

func (h *Handler) Get(c *gin.Context) {
	subscriptionID, err := h.ParseSubscriptionID(c)
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

func (h *Handler) Delete(c *gin.Context) {
	subscriptionID, err := h.ParseSubscriptionID(c)
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

	utils.NoContentResponse(c)
}

// updateSubscriptionRawRequest is used for raw JSON parsing to detect explicit null vs absent fields.
type updateSubscriptionRawRequest struct {
	StartDate      *time.Time       `json:"start_date"`
	EndDate        *time.Time       `json:"end_date"`
	DataLimitBytes *json.RawMessage `json:"data_limit_bytes"` // RawMessage to distinguish null from absent
	DataUsedBytes  *uint64          `json:"data_used_bytes"`
}

func (h *Handler) Update(c *gin.Context) {
	subscriptionID, err := h.ParseSubscriptionID(c)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid subscription ID")
		return
	}
	if subscriptionID == 0 {
		utils.ErrorResponse(c, http.StatusNotFound, "subscription not found")
		return
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "failed to read request body")
		return
	}

	var raw updateSubscriptionRawRequest
	if err := json.Unmarshal(body, &raw); err != nil {
		h.logger.Warnw("invalid request body for admin update subscription", "error", err)
		utils.ErrorResponse(c, http.StatusBadRequest, "invalid JSON")
		return
	}

	// Parse DataLimitBytes: nil RawMessage = key absent, null RawMessage = explicit null
	var dataLimitBytes *uint64
	clearDataLimit := false
	if raw.DataLimitBytes != nil {
		// Key is present in JSON
		if string(*raw.DataLimitBytes) == "null" {
			// Explicit null → clear the override
			clearDataLimit = true
		} else {
			// Has a value → parse as uint64
			var limit uint64
			if err := json.Unmarshal(*raw.DataLimitBytes, &limit); err != nil {
				utils.ErrorResponse(c, http.StatusBadRequest, "invalid data_limit_bytes value")
				return
			}
			dataLimitBytes = &limit
		}
	}

	// Check if at least one field is provided
	if raw.StartDate == nil && raw.EndDate == nil && raw.DataLimitBytes == nil && raw.DataUsedBytes == nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "at least one field must be provided")
		return
	}

	cmd := usecases.UpdateSubscriptionCommand{
		SubscriptionID: subscriptionID,
		StartDate:      raw.StartDate,
		EndDate:        raw.EndDate,
		DataLimitBytes: dataLimitBytes,
		ClearDataLimit: clearDataLimit,
		DataUsedBytes:  raw.DataUsedBytes,
	}

	if err := h.updateUseCase.Execute(c.Request.Context(), cmd); err != nil {
		h.logger.Errorw("failed to update subscription", "error", err, "subscription_id", subscriptionID)
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Return updated subscription
	query := usecases.GetSubscriptionQuery{SubscriptionID: subscriptionID}
	sub, err := h.getUseCase.Execute(c.Request.Context(), query)
	if err != nil {
		h.logger.Errorw("failed to get updated subscription", "error", err, "subscription_id", subscriptionID)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Subscription updated successfully", sub)
}
