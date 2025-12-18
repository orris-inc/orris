package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/orris-inc/orris/internal/application/user"
	"github.com/orris-inc/orris/internal/application/user/usecases"
	"github.com/orris-inc/orris/internal/interfaces/dto"
	"github.com/orris-inc/orris/internal/shared/constants"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/utils"
)

// UserHandler handles HTTP requests for user operations
type UserHandler struct {
	userService          *user.ServiceDDD
	adminResetPasswordUC *usecases.AdminResetPasswordUseCase
	logger               logger.Interface
}

// NewUserHandler creates a new user handler
func NewUserHandler(
	userService *user.ServiceDDD,
	adminResetPasswordUC *usecases.AdminResetPasswordUseCase,
) *UserHandler {
	return &UserHandler{
		userService:          userService,
		adminResetPasswordUC: adminResetPasswordUC,
		logger:               logger.NewLogger(),
	}
}

// CreateUser handles POST /users
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for create user", "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Convert to application request
	appReq := req.ToApplicationRequest()

	// Create user
	userResp, err := h.userService.CreateUser(c.Request.Context(), *appReq)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.CreatedResponse(c, userResp, "User created successfully")
}

// GetUser handles GET /users/:id
func (h *UserHandler) GetUser(c *gin.Context) {
	// Parse user ID
	userID, err := dto.ParseUserID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Get user
	userResp, err := h.userService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", userResp)
}

// UpdateUser handles PATCH /users/:id
func (h *UserHandler) UpdateUser(c *gin.Context) {
	// Log access control information
	currentUserID, _ := c.Get("user_id")
	userRole := c.GetString(constants.ContextKeyUserRole)

	// Parse user ID
	userID, err := dto.ParseUserID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	h.logger.Infow("update user request",
		"current_user_id", currentUserID,
		constants.ContextKeyUserRole, userRole,
		"target_user_id", userID)

	var req dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for update user",
			"user_id", userID,
			"error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Convert to application request
	appReq := req.ToApplicationRequest()

	// Update user
	userResp, err := h.userService.UpdateUser(c.Request.Context(), userID, *appReq)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "User updated successfully", userResp)
}

// DeleteUser handles DELETE /users/:id
func (h *UserHandler) DeleteUser(c *gin.Context) {
	// Parse user ID
	userID, err := dto.ParseUserID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	// Delete user
	if err := h.userService.DeleteUser(c.Request.Context(), userID); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.NoContentResponse(c)
}

// ListUsers handles GET /users
func (h *UserHandler) ListUsers(c *gin.Context) {
	// Parse query parameters
	req, err := dto.ParseListUsersRequest(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	// List users
	response, err := h.userService.ListUsers(c.Request.Context(), *req)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.ListSuccessResponse(c, response.Users, int64(response.Pagination.Total), response.Pagination.Page, response.Pagination.PageSize)
}

// GetUserByEmail handles GET /users/email/:email
func (h *UserHandler) GetUserByEmail(c *gin.Context) {
	email := c.Param("email")
	if email == "" {
		utils.ErrorResponseWithError(c, errors.NewValidationError("Email parameter is required"))
		return
	}

	// Get user by email
	userResp, err := h.userService.GetUserByEmail(c.Request.Context(), email)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "", userResp)
}

// AdminResetPassword handles PATCH /users/:id/password
func (h *UserHandler) AdminResetPassword(c *gin.Context) {
	// Parse user ID
	userID, err := dto.ParseUserID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	var req dto.AdminResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warnw("invalid request body for admin reset password", "user_id", userID, "error", err)
		utils.ErrorResponseWithError(c, err)
		return
	}

	cmd := usecases.AdminResetPasswordCommand{
		UserID:      userID,
		NewPassword: req.Password,
	}

	if err := h.adminResetPasswordUC.Execute(c.Request.Context(), cmd); err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Password reset successfully", nil)
}

// HealthCheck handles GET /health for user service health check
func (h *UserHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "orris",
	})
}
