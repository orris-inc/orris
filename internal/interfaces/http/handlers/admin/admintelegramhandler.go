package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"

	telegramAdminApp "github.com/orris-inc/orris/internal/application/telegram/admin"
	"github.com/orris-inc/orris/internal/application/telegram/admin/dto"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// AdminTelegramHandler handles admin telegram-related HTTP requests
type AdminTelegramHandler struct {
	service *telegramAdminApp.ServiceDDD
	logger  logger.Interface
}

// NewAdminTelegramHandler creates a new admin telegram handler
func NewAdminTelegramHandler(service *telegramAdminApp.ServiceDDD, logger logger.Interface) *AdminTelegramHandler {
	return &AdminTelegramHandler{
		service: service,
		logger:  logger,
	}
}

// GetBindingStatus returns the current admin telegram binding status
// GET /admin/telegram/binding
func (h *AdminTelegramHandler) GetBindingStatus(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponseWithError(c, errors.NewUnauthorizedError("User not authenticated"))
		return
	}

	uid, ok := userID.(uint)
	if !ok {
		h.logger.Errorw("invalid user_id type", "user_id", userID)
		utils.ErrorResponseWithError(c, errors.NewInternalError("Internal error"))
		return
	}

	resp, err := h.service.GetBindingStatus(c.Request.Context(), uid)
	if err != nil {
		h.logger.Errorw("failed to get admin binding status", "user_id", uid, "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", resp)
}

// Unbind removes the admin telegram binding
// DELETE /admin/telegram/binding
func (h *AdminTelegramHandler) Unbind(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponseWithError(c, errors.NewUnauthorizedError("User not authenticated"))
		return
	}

	uid, ok := userID.(uint)
	if !ok {
		h.logger.Errorw("invalid user_id type", "user_id", userID)
		utils.ErrorResponseWithError(c, errors.NewInternalError("Internal error"))
		return
	}

	if err := h.service.Unbind(c.Request.Context(), uid); err != nil {
		h.logger.Errorw("failed to unbind admin telegram", "user_id", uid, "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Admin telegram unbound successfully", nil)
}

// UpdatePreferences updates admin notification preferences
// PATCH /admin/telegram/preferences
func (h *AdminTelegramHandler) UpdatePreferences(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.ErrorResponseWithError(c, errors.NewUnauthorizedError("User not authenticated"))
		return
	}

	uid, ok := userID.(uint)
	if !ok {
		h.logger.Errorw("invalid user_id type", "user_id", userID)
		utils.ErrorResponseWithError(c, errors.NewInternalError("Internal error"))
		return
	}

	var req dto.UpdateAdminPreferencesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Errorw("failed to parse update preferences request", "error", err)
		utils.ErrorResponseWithError(c, errors.NewValidationError("Invalid request body"))
		return
	}

	resp, err := h.service.UpdatePreferences(c.Request.Context(), uid, &req)
	if err != nil {
		h.logger.Errorw("failed to update admin preferences", "user_id", uid, "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Preferences updated successfully", resp)
}
