package dto

import (
	"time"
)

// CreateUserRequest represents the request to create a user
type CreateUserRequest struct {
	Email string `json:"email" binding:"required,email"`
	Name  string `json:"name" binding:"required,min=2,max=100"`
}

// UpdateUserRequest represents the request to update a user
type UpdateUserRequest struct {
	Email  *string `json:"email,omitempty" binding:"omitempty,email"`
	Name   *string `json:"name,omitempty" binding:"omitempty,min=2,max=100"`
	Status *string `json:"status,omitempty" binding:"omitempty,oneof=active inactive pending suspended"`
}

// ListUsersRequest represents the request to list users
type ListUsersRequest struct {
	Page     int    `json:"page" form:"page"`
	PageSize int    `json:"page_size" form:"page_size"`
	Email    string `json:"email,omitempty" form:"email"`
	Name     string `json:"name,omitempty" form:"name"`
	Status   string `json:"status,omitempty" form:"status"`
	OrderBy  string `json:"order_by,omitempty" form:"order_by"`
	Order    string `json:"order,omitempty" form:"order" binding:"omitempty,oneof=asc desc"`
}

// UserResponse represents the response for a user
type UserResponse struct {
	ID          uint         `json:"id"`
	Email       string       `json:"email"`
	Name        string       `json:"name"`
	DisplayName string       `json:"display_name"`
	Initials    string       `json:"initials"`
	Status      string       `json:"status"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
	Metadata    UserMetadata `json:"metadata,omitempty"`
}

// UserMetadata contains additional user metadata
type UserMetadata struct {
	IsBusinessEmail      bool   `json:"is_business_email"`
	CanPerformActions    bool   `json:"can_perform_actions"`
	RequiresVerification bool   `json:"requires_verification"`
	EmailDomain          string `json:"email_domain"`
}

// ListUsersResponse represents the response for listing users
type ListUsersResponse struct {
	Users      []*UserResponse    `json:"users"`
	Pagination PaginationResponse `json:"pagination"`
}

// PaginationResponse represents pagination metadata
type PaginationResponse struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

