package handlers

import (
	"time"

	"github.com/orris-inc/orris/internal/domain/user"
)

// RegisterResponse represents the response for user registration.
type RegisterResponse struct {
	UserID                    string `json:"user_id"`
	Email                     string `json:"email"`
	RequiresEmailVerification bool   `json:"requires_email_verification"`
}

// toRegisterResponse converts a user and email verification flag to RegisterResponse.
func toRegisterResponse(u *user.User, requiresEmailVerification bool) *RegisterResponse {
	return &RegisterResponse{
		UserID:                    u.SID(),
		Email:                     u.Email().String(),
		RequiresEmailVerification: requiresEmailVerification,
	}
}

// LoginResponse represents the response for user login.
type LoginResponse struct {
	User      *UserInfoResponse `json:"user"`
	ExpiresIn int64             `json:"expires_in"`
}

// UserInfoResponse represents user information in API responses.
type UserInfoResponse struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	Initials    string    `json:"initials"`
	Role        string    `json:"role"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// toUserInfoResponse converts UserDisplayInfo to UserInfoResponse.
func toUserInfoResponse(info user.UserDisplayInfo) *UserInfoResponse {
	return &UserInfoResponse{
		ID:          info.ID,
		Email:       info.Email,
		DisplayName: info.DisplayName,
		Initials:    info.Initials,
		Role:        info.Role,
		Status:      info.Status,
		CreatedAt:   info.CreatedAt,
	}
}

// RefreshTokenResponse represents the response for token refresh.
type RefreshTokenResponse struct {
	ExpiresIn int64 `json:"expires_in"`
}
