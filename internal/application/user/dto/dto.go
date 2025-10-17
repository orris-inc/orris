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

// UserEventResponse represents a user event for external consumers
type UserEventResponse struct {
	EventType  string                 `json:"event_type"`
	UserID     uint                   `json:"user_id"`
	OccurredAt time.Time              `json:"occurred_at"`
	Data       map[string]interface{} `json:"data"`
}

// ActivateUserRequest represents the request to activate a user
type ActivateUserRequest struct {
	UserID uint `json:"user_id" binding:"required"`
}

// DeactivateUserRequest represents the request to deactivate a user
type DeactivateUserRequest struct {
	UserID uint   `json:"user_id" binding:"required"`
	Reason string `json:"reason,omitempty"`
}

// SuspendUserRequest represents the request to suspend a user
type SuspendUserRequest struct {
	UserID uint   `json:"user_id" binding:"required"`
	Reason string `json:"reason" binding:"required"`
}

// DeleteUserRequest represents the request to delete a user
type DeleteUserRequest struct {
	UserID uint `json:"user_id" binding:"required"`
}

// SearchUsersRequest represents a complex search request
type SearchUsersRequest struct {
	EmailDomain      string   `json:"email_domain,omitempty"`
	IsBusinessEmail  *bool    `json:"is_business_email,omitempty"`
	Statuses         []string `json:"statuses,omitempty"`
	CreatedAfter     *time.Time `json:"created_after,omitempty"`
	CreatedBefore    *time.Time `json:"created_before,omitempty"`
	Page             int      `json:"page"`
	PageSize         int      `json:"page_size"`
}

// BulkUserOperationRequest represents a bulk operation on users
type BulkUserOperationRequest struct {
	UserIDs   []uint `json:"user_ids" binding:"required,min=1"`
	Operation string `json:"operation" binding:"required,oneof=activate deactivate suspend delete"`
	Reason    string `json:"reason,omitempty"`
}

// BulkUserOperationResponse represents the response for bulk operations
type BulkUserOperationResponse struct {
	Successful []uint   `json:"successful"`
	Failed     []uint   `json:"failed"`
	Errors     []string `json:"errors,omitempty"`
}

// UserStatisticsResponse represents user statistics
type UserStatisticsResponse struct {
	TotalUsers       int            `json:"total_users"`
	ActiveUsers      int            `json:"active_users"`
	PendingUsers     int            `json:"pending_users"`
	InactiveUsers    int            `json:"inactive_users"`
	SuspendedUsers   int            `json:"suspended_users"`
	BusinessUsers    int            `json:"business_users"`
	UsersByStatus    map[string]int `json:"users_by_status"`
	UsersByDomain    map[string]int `json:"users_by_domain"`
	RecentSignups    int            `json:"recent_signups"`
}

// UserImportRequest represents a request to import users
type UserImportRequest struct {
	Users           []CreateUserRequest `json:"users" binding:"required,min=1"`
	SkipValidation  bool                `json:"skip_validation,omitempty"`
	AutoActivate    bool                `json:"auto_activate,omitempty"`
}

// UserImportResponse represents the response for user import
type UserImportResponse struct {
	TotalProcessed   int      `json:"total_processed"`
	SuccessfulImports int      `json:"successful_imports"`
	FailedImports     int      `json:"failed_imports"`
	ImportedUserIDs   []uint   `json:"imported_user_ids"`
	Errors            []string `json:"errors,omitempty"`
}

// UserExportRequest represents a request to export users
type UserExportRequest struct {
	Format   string   `json:"format" binding:"required,oneof=json csv xlsx"`
	Statuses []string `json:"statuses,omitempty"`
	Fields   []string `json:"fields,omitempty"`
}

// UserValidationRequest represents a request to validate user data
type UserValidationRequest struct {
	Email string `json:"email" binding:"required"`
	Name  string `json:"name" binding:"required"`
}

// UserValidationResponse represents the response for user validation
type UserValidationResponse struct {
	IsValid           bool     `json:"is_valid"`
	Errors            []string `json:"errors,omitempty"`
	Suggestions       []string `json:"suggestions,omitempty"`
	IsBusinessEmail   bool     `json:"is_business_email"`
	EmailDomain       string   `json:"email_domain"`
}