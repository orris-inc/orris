package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/setting/dto"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// GetSecuritySettings retrieves security settings
// GET /admin/settings/security
func (h *SettingHandler) GetSecuritySettings(c *gin.Context) {
	result, err := h.service.GetSecuritySettings(c.Request.Context())
	if err != nil {
		h.logger.Errorw("failed to get security settings", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "", result)
}

// UpdateSecuritySettings updates security settings
// PUT /admin/settings/security
func (h *SettingHandler) UpdateSecuritySettings(c *gin.Context) {
	var req dto.UpdateSecuritySettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	if err := h.service.UpdateSecuritySettings(c.Request.Context(), req, userID); err != nil {
		h.logger.Errorw("failed to update security settings", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Security settings updated successfully", nil)
}

// GetRegistrationSettings retrieves registration settings
// GET /admin/settings/registration
func (h *SettingHandler) GetRegistrationSettings(c *gin.Context) {
	result, err := h.service.GetRegistrationSettings(c.Request.Context())
	if err != nil {
		h.logger.Errorw("failed to get registration settings", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "", result)
}

// UpdateRegistrationSettings updates registration settings
// PUT /admin/settings/registration
func (h *SettingHandler) UpdateRegistrationSettings(c *gin.Context) {
	var req dto.UpdateRegistrationSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	if err := h.service.UpdateRegistrationSettings(c.Request.Context(), req, userID); err != nil {
		h.logger.Errorw("failed to update registration settings", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Registration settings updated successfully", nil)
}

// GetPublicRegistration retrieves public registration settings (no auth required)
// GET /registration-settings
func (h *SettingHandler) GetPublicRegistration(c *gin.Context) {
	result, err := h.service.GetPublicRegistration(c.Request.Context())
	if err != nil {
		h.logger.Errorw("failed to get public registration settings", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "", result)
}

// GetLegalSettings retrieves legal settings
// GET /admin/settings/legal
func (h *SettingHandler) GetLegalSettings(c *gin.Context) {
	result, err := h.service.GetLegalSettings(c.Request.Context())
	if err != nil {
		h.logger.Errorw("failed to get legal settings", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "", result)
}

// UpdateLegalSettings updates legal settings
// PUT /admin/settings/legal
func (h *SettingHandler) UpdateLegalSettings(c *gin.Context) {
	var req dto.UpdateLegalSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	if err := h.service.UpdateLegalSettings(c.Request.Context(), req, userID); err != nil {
		h.logger.Errorw("failed to update legal settings", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Legal settings updated successfully", nil)
}

// GetPublicLegal retrieves public legal URLs (no auth required)
// GET /legal
func (h *SettingHandler) GetPublicLegal(c *gin.Context) {
	result, err := h.service.GetPublicLegal(c.Request.Context())
	if err != nil {
		h.logger.Errorw("failed to get public legal settings", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "", result)
}

// GetPublicPasswordPolicy retrieves public password policy (no auth required)
// GET /password-policy
func (h *SettingHandler) GetPublicPasswordPolicy(c *gin.Context) {
	result, err := h.service.GetPublicPasswordPolicy(c.Request.Context())
	if err != nil {
		h.logger.Errorw("failed to get public password policy", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}
	utils.SuccessResponse(c, http.StatusOK, "", result)
}
