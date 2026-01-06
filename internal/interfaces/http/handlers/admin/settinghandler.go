// Package admin provides HTTP handlers for administrative operations.
package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"

	settingApp "github.com/orris-inc/orris/internal/application/setting"
	"github.com/orris-inc/orris/internal/application/setting/dto"
	"github.com/orris-inc/orris/internal/domain/setting"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// SettingHandler handles system settings admin API operations
type SettingHandler struct {
	service *settingApp.ServiceDDD
	logger  logger.Interface
}

// NewSettingHandler creates a new setting handler
func NewSettingHandler(service *settingApp.ServiceDDD, logger logger.Interface) *SettingHandler {
	return &SettingHandler{
		service: service,
		logger:  logger,
	}
}

// GetCategorySettings retrieves all settings in a category
// GET /admin/settings/:category
func (h *SettingHandler) GetCategorySettings(c *gin.Context) {
	category := c.Param("category")
	if category == "" {
		h.logger.Warnw("missing category parameter")
		utils.ErrorResponse(c, http.StatusBadRequest, "category parameter is required")
		return
	}

	result, err := h.service.GetByCategory(c.Request.Context(), category)
	if err != nil {
		if err == setting.ErrSettingNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "category not found or has no settings")
			return
		}
		h.logger.Errorw("failed to get category settings", "category", category, "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}

// UpdateCategorySettings batch updates settings in a category
// PUT /admin/settings/:category
func (h *SettingHandler) UpdateCategorySettings(c *gin.Context) {
	category := c.Param("category")
	if category == "" {
		h.logger.Warnw("missing category parameter")
		utils.ErrorResponse(c, http.StatusBadRequest, "category parameter is required")
		return
	}

	var req dto.UpdateCategorySettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for update category settings", "category", category, "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		h.logger.Errorw("user_id not found in context")
		utils.ErrorResponse(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	uid, ok := userID.(uint)
	if !ok {
		h.logger.Errorw("invalid user_id type", "user_id", userID)
		utils.ErrorResponse(c, http.StatusInternalServerError, "internal error")
		return
	}

	if err := h.service.UpdateCategorySettings(c.Request.Context(), category, req, uid); err != nil {
		if err == setting.ErrSettingNotFound {
			utils.ErrorResponse(c, http.StatusNotFound, "setting not found")
			return
		}
		if err == setting.ErrInvalidSettingKey {
			utils.ErrorResponse(c, http.StatusBadRequest, "invalid setting key")
			return
		}
		if err == setting.ErrInvalidValueType {
			utils.ErrorResponse(c, http.StatusBadRequest, "invalid value type")
			return
		}
		h.logger.Errorw("failed to update category settings", "category", category, "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Settings updated successfully", nil)
}

// GetTelegramConfig retrieves Telegram configuration
// GET /admin/settings/telegram/config
func (h *SettingHandler) GetTelegramConfig(c *gin.Context) {
	result, err := h.service.GetTelegramConfig(c.Request.Context())
	if err != nil {
		h.logger.Errorw("failed to get telegram config", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}

// UpdateTelegramConfig updates Telegram configuration
// PUT /admin/settings/telegram/config
func (h *SettingHandler) UpdateTelegramConfig(c *gin.Context) {
	var req dto.UpdateTelegramConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for update telegram config", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Get user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		h.logger.Errorw("user_id not found in context")
		utils.ErrorResponse(c, http.StatusUnauthorized, "user not authenticated")
		return
	}

	uid, ok := userID.(uint)
	if !ok {
		h.logger.Errorw("invalid user_id type", "user_id", userID)
		utils.ErrorResponse(c, http.StatusInternalServerError, "internal error")
		return
	}

	if err := h.service.UpdateTelegramConfig(c.Request.Context(), req, uid); err != nil {
		if err == setting.ErrInvalidValueType {
			utils.ErrorResponse(c, http.StatusBadRequest, "invalid value type")
			return
		}
		h.logger.Errorw("failed to update telegram config", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Telegram configuration updated successfully", nil)
}

// TestTelegramConnectionRequest represents the request body for testing telegram connection
type TestTelegramConnectionRequest struct {
	BotToken string `json:"bot_token"` // Optional: token to test (if empty, uses saved token)
}

// TestTelegramConnection tests the Telegram bot connection
// POST /admin/settings/telegram/test
func (h *SettingHandler) TestTelegramConnection(c *gin.Context) {
	var req TestTelegramConnectionRequest
	// Ignore bind error - bot_token is optional
	_ = c.ShouldBindJSON(&req)

	result, err := h.service.TestTelegramConnection(c.Request.Context(), req.BotToken)
	if err != nil {
		h.logger.Errorw("failed to test telegram connection", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", result)
}
