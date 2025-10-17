package dto

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"orris/internal/application/user/dto"
	"orris/internal/shared/constants"
	"orris/internal/shared/errors"
	"orris/internal/shared/utils"
)

// CreateUserRequest represents HTTP request to create a user
type CreateUserRequest struct {
	Email string `json:"email" binding:"required" validate:"required,email"`
	Name  string `json:"name" binding:"required" validate:"required,min=2,max=100"`
}

// UpdateUserRequest represents HTTP request to update a user
type UpdateUserRequest struct {
	Email string `json:"email" binding:"required" validate:"required,email"`
	Name  string `json:"name" binding:"required" validate:"required,min=2,max=100"`
}

// ToApplicationRequest converts HTTP DTO to application layer DTO
func (r *CreateUserRequest) ToApplicationRequest() *dto.CreateUserRequest {
	return &dto.CreateUserRequest{
		Email: r.Email,
		Name:  r.Name,
	}
}

// ToApplicationRequest converts HTTP DTO to application layer DTO
func (r *UpdateUserRequest) ToApplicationRequest() *dto.UpdateUserRequest {
	return &dto.UpdateUserRequest{
		Email: &r.Email,
		Name:  &r.Name,
	}
}

// ParseListUsersRequest parses query parameters for listing users
func ParseListUsersRequest(c *gin.Context) (*dto.ListUsersRequest, error) {
	req := &dto.ListUsersRequest{
		Page:     constants.DefaultPage,
		PageSize: constants.DefaultPageSize,
	}

	// Parse page
	if pageStr := c.Query("page"); pageStr != "" {
		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			return nil, errors.NewValidationError("Invalid page parameter")
		}
		req.Page = page
	}

	// Parse page_size
	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		pageSize, err := strconv.Atoi(pageSizeStr)
		if err != nil || pageSize < 1 {
			return nil, errors.NewValidationError("Invalid page_size parameter")
		}
		if pageSize > constants.MaxPageSize {
			pageSize = constants.MaxPageSize
		}
		req.PageSize = pageSize
	}

	// Parse filters
	req.Email = c.Query("email")
	req.Name = c.Query("name")
	req.Status = c.Query("status")
	req.OrderBy = c.Query("order_by")
	req.Order = c.Query("order")

	// Validate request
	if err := utils.ValidateStruct(req); err != nil {
		return nil, err
	}

	return req, nil
}

// ParseUserID parses user ID from URL parameter
func ParseUserID(c *gin.Context) (uint, error) {
	idStr := c.Param("id")
	if idStr == "" {
		return 0, errors.NewValidationError("User ID is required")
	}

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, errors.NewValidationError("Invalid user ID format")
	}

	if id == 0 {
		return 0, errors.NewValidationError("User ID cannot be zero")
	}

	return uint(id), nil
}