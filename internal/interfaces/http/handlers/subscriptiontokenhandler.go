package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/subscription/usecases"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

type SubscriptionTokenHandler struct {
	generateTokenUC *usecases.GenerateSubscriptionTokenUseCase
	listTokensUC    *usecases.ListSubscriptionTokensUseCase
	revokeTokenUC   *usecases.RevokeSubscriptionTokenUseCase
	refreshTokenUC  *usecases.RefreshSubscriptionTokenUseCase
	logger          logger.Interface
}

func NewSubscriptionTokenHandler(
	generateTokenUC *usecases.GenerateSubscriptionTokenUseCase,
	listTokensUC *usecases.ListSubscriptionTokensUseCase,
	revokeTokenUC *usecases.RevokeSubscriptionTokenUseCase,
	refreshTokenUC *usecases.RefreshSubscriptionTokenUseCase,
) *SubscriptionTokenHandler {
	return &SubscriptionTokenHandler{
		generateTokenUC: generateTokenUC,
		listTokensUC:    listTokensUC,
		revokeTokenUC:   revokeTokenUC,
		refreshTokenUC:  refreshTokenUC,
		logger:          logger.NewLogger(),
	}
}

type GenerateTokenRequest struct {
	Name      string     `json:"name" binding:"required"`
	Scope     string     `json:"scope" binding:"required"`
	ExpiresAt *time.Time `json:"expires_at"`
}

func (h *SubscriptionTokenHandler) GenerateToken(c *gin.Context) {
	subscriptionID, err := parseSubscriptionID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	var req GenerateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for generate token",
			"subscription_id", subscriptionID,
			"error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Convert ExpiresAt to UTC if provided
	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		utc := req.ExpiresAt.UTC()
		expiresAt = &utc
	}

	cmd := usecases.GenerateSubscriptionTokenCommand{
		SubscriptionID: subscriptionID,
		Name:           req.Name,
		Scope:          req.Scope,
		ExpiresAt:      expiresAt,
	}

	result, err := h.generateTokenUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.CreatedResponse(c, result, "Subscription token generated successfully")
}

func (h *SubscriptionTokenHandler) ListTokens(c *gin.Context) {
	subscriptionID, err := parseSubscriptionID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	activeOnly := false
	if activeOnlyStr := c.Query("active_only"); activeOnlyStr != "" {
		activeOnly, err = strconv.ParseBool(activeOnlyStr)
		if err != nil {
			utils.ErrorResponseWithError(c, errors.NewValidationError("Invalid active_only parameter"))
			return
		}
	}

	query := usecases.ListSubscriptionTokensQuery{
		SubscriptionID: subscriptionID,
		ActiveOnly:     activeOnly,
	}

	result, err := h.listTokensUC.Execute(c.Request.Context(), query)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}

func (h *SubscriptionTokenHandler) RevokeToken(c *gin.Context) {
	tokenID, err := parseTokenID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.RevokeSubscriptionTokenCommand{
		TokenID: tokenID,
	}

	if err := h.revokeTokenUC.Execute(c.Request.Context(), cmd); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Subscription token revoked successfully", nil)
}

func (h *SubscriptionTokenHandler) RefreshToken(c *gin.Context) {
	tokenID, err := parseTokenID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.RefreshSubscriptionTokenCommand{
		OldTokenID: tokenID,
	}

	result, err := h.refreshTokenUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Subscription token refreshed successfully", result)
}

func parseSubscriptionID(c *gin.Context) (uint, error) {
	idStr := c.Param("id")
	if idStr == "" {
		return 0, errors.NewValidationError("Subscription ID is required")
	}

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, errors.NewValidationError("Invalid subscription ID format")
	}

	if id == 0 {
		return 0, errors.NewValidationError("Subscription ID cannot be zero")
	}

	return uint(id), nil
}

func parseTokenID(c *gin.Context) (uint, error) {
	idStr := c.Param("token_id")
	if idStr == "" {
		return 0, errors.NewValidationError("Token ID is required")
	}

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, errors.NewValidationError("Invalid token ID format")
	}

	if id == 0 {
		return 0, errors.NewValidationError("Token ID cannot be zero")
	}

	return uint(id), nil
}
