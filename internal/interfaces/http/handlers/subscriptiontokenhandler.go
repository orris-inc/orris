package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"orris/internal/application/subscription/usecases"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
	"orris/internal/shared/utils"
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

// @Summary		Generate subscription token
// @Description	Generate a new API token for a subscription with specified scope and expiration
// @Tags			subscription-tokens
// @Accept			json
// @Produce		json
// @Security		Bearer
// @Param			subscription_id	path		int						true	"Subscription ID"
// @Param			token			body		GenerateTokenRequest	true	"Token generation data"
// @Success		201				{object}	utils.APIResponse		"Subscription token generated successfully"
// @Failure		400				{object}	utils.APIResponse		"Bad request"
// @Failure		401				{object}	utils.APIResponse		"Unauthorized"
// @Failure		403				{object}	utils.APIResponse		"Access denied"
// @Failure		404				{object}	utils.APIResponse		"Subscription not found"
// @Failure		500				{object}	utils.APIResponse		"Internal server error"
// @Router			/subscriptions/{id}/tokens [post]
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

	cmd := usecases.GenerateSubscriptionTokenCommand{
		SubscriptionID: subscriptionID,
		Name:           req.Name,
		Scope:          req.Scope,
		ExpiresAt:      req.ExpiresAt,
	}

	result, err := h.generateTokenUC.Execute(c.Request.Context(), cmd)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.CreatedResponse(c, result, "Subscription token generated successfully")
}

// @Summary		List subscription tokens
// @Description	Get all tokens associated with a subscription
// @Tags			subscription-tokens
// @Accept			json
// @Produce		json
// @Security		Bearer
// @Param			subscription_id	path		int					true	"Subscription ID"
// @Param			active_only		query		bool				false	"Show only active tokens"	default(false)
// @Success		200				{object}	utils.APIResponse	"Subscription tokens list"
// @Failure		400				{object}	utils.APIResponse	"Invalid subscription ID"
// @Failure		401				{object}	utils.APIResponse	"Unauthorized"
// @Failure		403				{object}	utils.APIResponse	"Access denied"
// @Failure		404				{object}	utils.APIResponse	"Subscription not found"
// @Failure		500				{object}	utils.APIResponse	"Internal server error"
// @Router			/subscriptions/{id}/tokens [get]
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

// @Summary		Revoke subscription token
// @Description	Revoke an existing subscription token to invalidate it
// @Tags			subscription-tokens
// @Accept			json
// @Produce		json
// @Security		Bearer
// @Param			subscription_id	path		int					true	"Subscription ID"
// @Param			token_id		path		int					true	"Token ID"
// @Success		200				{object}	utils.APIResponse	"Subscription token revoked successfully"
// @Failure		400				{object}	utils.APIResponse	"Invalid token ID"
// @Failure		401				{object}	utils.APIResponse	"Unauthorized"
// @Failure		403				{object}	utils.APIResponse	"Access denied"
// @Failure		404				{object}	utils.APIResponse	"Token not found"
// @Failure		500				{object}	utils.APIResponse	"Internal server error"
// @Router			/subscriptions/{id}/tokens/{token_id} [delete]
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

// @Summary		Refresh subscription token
// @Description	Refresh an existing subscription token to extend its validity
// @Tags			subscription-tokens
// @Accept			json
// @Produce		json
// @Security		Bearer
// @Param			subscription_id	path		int					true	"Subscription ID"
// @Param			token_id		path		int					true	"Token ID"
// @Success		200				{object}	utils.APIResponse	"Subscription token refreshed successfully"
// @Failure		400				{object}	utils.APIResponse	"Invalid token ID"
// @Failure		401				{object}	utils.APIResponse	"Unauthorized"
// @Failure		403				{object}	utils.APIResponse	"Access denied"
// @Failure		404				{object}	utils.APIResponse	"Token not found"
// @Failure		500				{object}	utils.APIResponse	"Internal server error"
// @Router			/subscriptions/{id}/tokens/{token_id}/refresh [post]
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
