package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	entitlementApp "github.com/orris-inc/orris/internal/application/entitlement"
	"github.com/orris-inc/orris/internal/application/entitlement/usecases"
	"github.com/orris-inc/orris/internal/shared/constants"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// EntitlementHandler handles HTTP requests for user entitlement operations
type EntitlementHandler struct {
	getUserEntitlementsUC *usecases.GetUserEntitlementsUseCase
	service               *entitlementApp.ServiceImpl
	logger                logger.Interface
}

// NewEntitlementHandler creates a new entitlement handler
func NewEntitlementHandler(
	getUserEntitlementsUC *usecases.GetUserEntitlementsUseCase,
	service *entitlementApp.ServiceImpl,
	logger logger.Interface,
) *EntitlementHandler {
	return &EntitlementHandler{
		getUserEntitlementsUC: getUserEntitlementsUC,
		service:               service,
		logger:                logger,
	}
}

// GetMyEntitlements handles GET /users/me/entitlements
// Get all entitlements for the current user
// Query parameters:
//   - resource_type: filter by resource type (node, forward_agent, etc.)
func (h *EntitlementHandler) GetMyEntitlements(c *gin.Context) {
	// Get current user ID from context
	userID, exists := c.Get(constants.ContextKeyUserID)
	if !exists {
		h.logger.Warnw("user ID not found in context")
		utils.ErrorResponse(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	uid, ok := userID.(uint)
	if !ok {
		h.logger.Warnw("invalid user ID type in context", "user_id", userID)
		utils.ErrorResponse(c, http.StatusInternalServerError, "invalid user ID")
		return
	}

	// Get optional resource type filter
	resourceType := c.Query("resource_type")

	// If resource_type is provided, use ExecuteAccessibleResources
	if resourceType != "" {
		result, err := h.getUserEntitlementsUC.ExecuteAccessibleResources(c.Request.Context(), uid, resourceType)
		if err != nil {
			h.logger.Errorw("failed to get accessible resources",
				"error", err,
				"user_id", uid,
				"resource_type", resourceType,
			)
			utils.ErrorResponseWithError(c, err)
			return
		}
		utils.SuccessResponse(c, http.StatusOK, "", result)
		return
	}

	// Execute use case to get all entitlements
	entitlements, err := h.getUserEntitlementsUC.Execute(c.Request.Context(), uid)
	if err != nil {
		h.logger.Errorw("failed to get user entitlements", "error", err, "user_id", uid)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", gin.H{
		"entitlements": entitlements,
	})
}
