package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"orris/internal/application/user"
	userdto "orris/internal/application/user/dto"
	"orris/internal/interfaces/dto"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
	"orris/internal/shared/utils"
)

var _ = userdto.UserResponse{} // ensure import is used for swagger

// UserHandler handles HTTP requests for user operations
type UserHandler struct {
	userService *user.ServiceDDD
	logger      logger.Interface
}

// NewUserHandler creates a new user handler
func NewUserHandler(userService *user.ServiceDDD) *UserHandler {
	return &UserHandler{
		userService: userService,
		logger:      logger.NewLogger(),
	}
}

// CreateUser handles POST /users
//
//	@Summary		Create a new user
//	@Description	Create a new user with the input data
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			user	body		internal_application_user_dto.CreateUserRequest	true	"User data"
//	@Success		201		{object}	utils.APIResponse								"User created successfully"
//	@Failure		400		{object}	utils.APIResponse								"Bad request"
//	@Failure		401		{object}	utils.APIResponse								"Unauthorized"
//	@Failure		403		{object}	utils.APIResponse								"Forbidden - Requires admin role"
//	@Failure		409		{object}	utils.APIResponse								"Email already exists"
//	@Failure		500		{object}	utils.APIResponse								"Internal server error"
//	@Router			/users [post]
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
//
//	@Summary		Get user by ID
//	@Description	Get details of a user by their ID
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id	path		int					true	"User ID"
//	@Success		200	{object}	utils.APIResponse	"User details"
//	@Failure		400	{object}	utils.APIResponse	"Invalid user ID"
//	@Failure		401	{object}	utils.APIResponse	"Unauthorized"
//	@Failure		403	{object}	utils.APIResponse	"Forbidden - Requires admin role"
//	@Failure		404	{object}	utils.APIResponse	"User not found"
//	@Failure		500	{object}	utils.APIResponse	"Internal server error"
//	@Router			/users/{id} [get]
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
//
//	@Summary		Update user (partial update)
//	@Description	Partially update user information by ID. All fields are optional, at least one must be provided. Only admins can update users. Supports updating email, name, status, and role.
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id		path		int												true	"User ID"
//	@Param			user	body		internal_application_user_dto.UpdateUserRequest	true	"User update data (all fields optional)"
//	@Success		200		{object}	utils.APIResponse								"User updated successfully"
//	@Failure		400		{object}	utils.APIResponse								"Bad request"
//	@Failure		401		{object}	utils.APIResponse								"Unauthorized"
//	@Failure		403		{object}	utils.APIResponse								"Forbidden - Requires admin role"
//	@Failure		404		{object}	utils.APIResponse								"User not found"
//	@Failure		409		{object}	utils.APIResponse								"Email already exists"
//	@Failure		500		{object}	utils.APIResponse								"Internal server error"
//	@Router			/users/{id} [patch]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	// Log access control information
	currentUserID, _ := c.Get("user_id")
	userRole := c.GetString("user_role")

	// Parse user ID
	userID, err := dto.ParseUserID(c)
	if err != nil {
		utils.ErrorResponseWithError(c, err)
		return
	}

	h.logger.Infow("update user request",
		"current_user_id", currentUserID,
		"user_role", userRole,
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
//
//	@Summary		Delete user
//	@Description	Delete a user by ID
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			id	path	int	true	"User ID"
//	@Success		204	"User deleted successfully"
//	@Failure		400	{object}	utils.APIResponse	"Invalid user ID"
//	@Failure		401	{object}	utils.APIResponse	"Unauthorized"
//	@Failure		403	{object}	utils.APIResponse	"Forbidden - Requires admin role"
//	@Failure		404	{object}	utils.APIResponse	"User not found"
//	@Failure		500	{object}	utils.APIResponse	"Internal server error"
//	@Router			/users/{id} [delete]
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
//
//	@Summary		List users
//	@Description	Get a paginated list of users with optional filtering
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			page		query		int											false	"Page number"			default(1)
//	@Param			page_size	query		int											false	"Page size"				default(20)
//	@Param			status		query		string										false	"User status filter"	Enums(active,inactive,pending,suspended)
//	@Param			role		query		string										false	"User role filter"		Enums(user,admin)
//	@Success		200			{object}	utils.APIResponse{data=utils.ListResponse}	"Users list"
//	@Failure		400			{object}	utils.APIResponse							"Invalid query parameters"
//	@Failure		401			{object}	utils.APIResponse							"Unauthorized"
//	@Failure		403			{object}	utils.APIResponse							"Forbidden - Requires admin role"
//	@Failure		500			{object}	utils.APIResponse							"Internal server error"
//	@Router			/users [get]
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
//
//	@Summary		Get user by email
//	@Description	Get user details by email address
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			email	path		string				true	"User email address"
//	@Success		200		{object}	utils.APIResponse	"User details"
//	@Failure		400		{object}	utils.APIResponse	"Invalid email"
//	@Failure		401		{object}	utils.APIResponse	"Unauthorized"
//	@Failure		403		{object}	utils.APIResponse	"Forbidden - Requires admin role"
//	@Failure		404		{object}	utils.APIResponse	"User not found"
//	@Failure		500		{object}	utils.APIResponse	"Internal server error"
//	@Router			/users/email/{email} [get]
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

// HealthCheck handles GET /health for user service health check
//
//	@Summary		Health check
//	@Description	Check if the service is healthy
//	@Tags			health
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string]interface{}	"Service is healthy"
//	@Router			/health [get]
func (h *UserHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "orris",
	})
}
