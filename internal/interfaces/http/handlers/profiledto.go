package handlers

import (
	"github.com/orris-inc/orris/internal/application/user/dto"
)

// UpdateProfileRequest represents HTTP request to update profile
type UpdateProfileRequest struct {
	Name  *string `json:"name" binding:"omitempty,min=2,max=100"`
	Email *string `json:"email" binding:"omitempty,email"`
}

// ToApplicationRequest converts HTTP DTO to application DTO
func (r *UpdateProfileRequest) ToApplicationRequest() *dto.UpdateProfileRequest {
	return &dto.UpdateProfileRequest{
		Name:  r.Name,
		Email: r.Email,
	}
}

// ChangePasswordRequest represents HTTP request to change password
type ChangePasswordRequest struct {
	OldPassword      string `json:"old_password" binding:"required,min=8"`
	NewPassword      string `json:"new_password" binding:"required,min=8"`
	LogoutAllDevices bool   `json:"logout_all_devices"`
}

// ToApplicationRequest converts HTTP DTO to application DTO
func (r *ChangePasswordRequest) ToApplicationRequest() *dto.ChangePasswordRequest {
	return &dto.ChangePasswordRequest{
		OldPassword:      r.OldPassword,
		NewPassword:      r.NewPassword,
		LogoutAllDevices: r.LogoutAllDevices,
	}
}
