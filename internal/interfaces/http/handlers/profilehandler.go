package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/user"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// ProfileHandler handles user profile-related HTTP requests
type ProfileHandler struct {
	userService *user.ServiceDDD
	logger      logger.Interface
}

// NewProfileHandler creates a new ProfileHandler
func NewProfileHandler(userService *user.ServiceDDD, log logger.Interface) *ProfileHandler {
	return &ProfileHandler{
		userService: userService,
		logger:      log,
	}
}

// UpdateProfile handles PATCH /users/me
func (h *ProfileHandler) UpdateProfile(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Parse request
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for update profile",
			"user_id", userID,
			"error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Log the request
	h.logger.Infow("update profile request",
		"user_id", userID,
		"has_name", req.Name != nil,
		"has_email", req.Email != nil)

	// Convert to application request
	appReq := req.ToApplicationRequest()

	// Update profile
	userResp, err := h.userService.UpdateProfile(c.Request.Context(), userID, *appReq)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "profile updated successfully", userResp)
}

// ChangePassword handles PUT /users/me/password
func (h *ProfileHandler) ChangePassword(c *gin.Context) {
	userID, err := utils.GetUserIDFromContext(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Parse request
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for change password",
			"user_id", userID,
			"error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Log the request (don't log passwords!)
	h.logger.Infow("change password request",
		"user_id", userID,
		"logout_all_devices", req.LogoutAllDevices)

	// Convert to application request
	appReq := req.ToApplicationRequest()

	// Change password
	err = h.userService.ChangePassword(c.Request.Context(), userID, *appReq)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "password changed successfully", nil)
}
