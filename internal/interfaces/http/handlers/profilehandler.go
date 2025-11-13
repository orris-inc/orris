package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"orris/internal/application/user"
	userdto "orris/internal/application/user/dto"
	"orris/internal/interfaces/dto"
	"orris/internal/shared/logger"
	"orris/internal/shared/utils"
)

var _ = userdto.UserResponse{} // ensure import is used for swagger

// ProfileHandler handles user profile-related HTTP requests
type ProfileHandler struct {
	userService *user.ServiceDDD
	logger      logger.Interface
}

// NewProfileHandler creates a new ProfileHandler
func NewProfileHandler(userService *user.ServiceDDD) *ProfileHandler {
	return &ProfileHandler{
		userService: userService,
		logger:      logger.NewLogger(),
	}
}

// UpdateProfile handles PATCH /users/me
//
//	@Summary		Update current user profile
//	@Description	Update the authenticated user's profile information (name, email)
//	@Tags			profile
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			profile	body		internal_application_user_dto.UpdateProfileRequest	true	"Profile update data"
//	@Success		200		{object}	utils.APIResponse									"Profile updated successfully"
//	@Failure		400		{object}	utils.APIResponse									"Bad request or validation error"
//	@Failure		401		{object}	utils.APIResponse									"Unauthorized"
//	@Failure		404		{object}	utils.APIResponse									"User not found"
//	@Failure		500		{object}	utils.APIResponse									"Internal server error"
//	@Router			/users/me [patch]
func (h *ProfileHandler) UpdateProfile(c *gin.Context) {
	// Get current user ID from context
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		h.logger.Error("user_id not found in context")
		utils.ErrorResponse(c, http.StatusUnauthorized, "authentication required")
		return
	}

	userID, ok := userIDInterface.(uint)
	if !ok {
		h.logger.Error("invalid user_id type in context")
		utils.ErrorResponse(c, http.StatusInternalServerError, "internal error")
		return
	}

	// Parse request
	var req dto.UpdateProfileRequest
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
//
//	@Summary		Change password
//	@Description	Change the authenticated user's password
//	@Tags			profile
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			password	body		internal_application_user_dto.ChangePasswordRequest	true	"Password change data"
//	@Success		200			{object}	utils.APIResponse									"Password changed successfully"
//	@Failure		400			{object}	utils.APIResponse									"Bad request or validation error (e.g., incorrect old password)"
//	@Failure		401			{object}	utils.APIResponse									"Unauthorized"
//	@Failure		404			{object}	utils.APIResponse									"User not found"
//	@Failure		500			{object}	utils.APIResponse									"Internal server error"
//	@Router			/users/me/password [put]
func (h *ProfileHandler) ChangePassword(c *gin.Context) {
	// Get current user ID from context
	userIDInterface, exists := c.Get("user_id")
	if !exists {
		h.logger.Error("user_id not found in context")
		utils.ErrorResponse(c, http.StatusUnauthorized, "authentication required")
		return
	}

	userID, ok := userIDInterface.(uint)
	if !ok {
		h.logger.Error("invalid user_id type in context")
		utils.ErrorResponse(c, http.StatusInternalServerError, "internal error")
		return
	}

	// Parse request
	var req dto.ChangePasswordRequest
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
	err := h.userService.ChangePassword(c.Request.Context(), userID, *appReq)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "password changed successfully", nil)
}
