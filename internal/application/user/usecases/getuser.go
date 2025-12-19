package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/user/dto"
	domainUser "github.com/orris-inc/orris/internal/domain/user"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// GetUserUseCase handles the business logic for retrieving a user
type GetUserUseCase struct {
	userRepo domainUser.Repository
	logger   logger.Interface
}

// NewGetUserUseCase creates a new get user use case
func NewGetUserUseCase(userRepo domainUser.Repository, logger logger.Interface) *GetUserUseCase {
	return &GetUserUseCase{
		userRepo: userRepo,
		logger:   logger,
	}
}

// ExecuteByID retrieves a user by internal ID (for internal use)
func (uc *GetUserUseCase) ExecuteByID(ctx context.Context, internalID uint) (*dto.UserResponse, error) {
	uc.logger.Infow("executing get user by internal ID", "id", internalID)

	if internalID == 0 {
		return nil, errors.NewValidationError("user ID cannot be zero")
	}

	// Retrieve the user
	userEntity, err := uc.userRepo.GetByID(ctx, internalID)
	if err != nil {
		uc.logger.Errorw("failed to get user", "id", internalID, "error", err)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if userEntity == nil {
		uc.logger.Warnw("user not found", "id", internalID)
		return nil, errors.NewNotFoundError("user not found")
	}

	// Map to response DTO
	return uc.mapToResponse(userEntity), nil
}

// ExecuteBySID retrieves a user by external SID (Stripe-style ID)
func (uc *GetUserUseCase) ExecuteBySID(ctx context.Context, sid string) (*dto.UserResponse, error) {
	uc.logger.Infow("executing get user by SID", "sid", sid)

	if sid == "" {
		return nil, errors.NewValidationError("user SID cannot be empty")
	}

	// Retrieve the user
	userEntity, err := uc.userRepo.GetBySID(ctx, sid)
	if err != nil {
		uc.logger.Errorw("failed to get user", "sid", sid, "error", err)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if userEntity == nil {
		uc.logger.Warnw("user not found", "sid", sid)
		return nil, errors.NewNotFoundError("user not found")
	}

	// Map to response DTO
	return uc.mapToResponse(userEntity), nil
}

// ExecuteByEmail retrieves a user by email
func (uc *GetUserUseCase) ExecuteByEmail(ctx context.Context, email string) (*dto.UserResponse, error) {
	uc.logger.Infow("executing get user by email", "email", email)

	if email == "" {
		return nil, errors.NewValidationError("email cannot be empty")
	}

	// Use GetByEmail method to find user
	userEntity, err := uc.userRepo.GetByEmail(ctx, email)
	if err != nil {
		uc.logger.Errorw("failed to get user by email", "email", email, "error", err)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if userEntity == nil {
		uc.logger.Warnw("user not found", "email", email)
		return nil, errors.NewNotFoundError("user not found")
	}

	// Map to response DTO
	return uc.mapToResponse(userEntity), nil
}

// ExecuteList retrieves a paginated list of users
func (uc *GetUserUseCase) ExecuteList(ctx context.Context, request dto.ListUsersRequest) (*dto.ListUsersResponse, error) {
	uc.logger.Infow("executing list users", "page", request.Page, "pageSize", request.PageSize)

	// Validate and set defaults for pagination
	if request.Page <= 0 {
		request.Page = 1
	}
	if request.PageSize <= 0 {
		request.PageSize = 20
	}
	if request.PageSize > 100 {
		request.PageSize = 100
	}

	// Create filter from request
	filter := domainUser.ListFilter{
		Page:     request.Page,
		PageSize: request.PageSize,
		Email:    request.Email,
		Name:     request.Name,
		Status:   request.Status,
		Role:     request.Role,
		OrderBy:  request.OrderBy,
		Order:    request.Order,
	}

	// Retrieve users
	users, total, err := uc.userRepo.List(ctx, filter)
	if err != nil {
		uc.logger.Errorw("failed to list users", "error", err)
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	// Map to response DTOs
	userResponses := make([]*dto.UserResponse, len(users))
	for i, userEntity := range users {
		userResponses[i] = uc.mapToResponse(userEntity)
	}

	// Calculate pagination metadata
	totalPages := (int(total) + request.PageSize - 1) / request.PageSize

	response := &dto.ListUsersResponse{
		Users: userResponses,
		Pagination: dto.PaginationResponse{
			Page:       request.Page,
			PageSize:   request.PageSize,
			Total:      int(total),
			TotalPages: totalPages,
		},
	}

	uc.logger.Infow("users listed successfully", "count", len(users), "total", total)
	return response, nil
}

// mapToResponse maps a user entity to a response DTO
func (uc *GetUserUseCase) mapToResponse(userEntity *domainUser.User) *dto.UserResponse {
	displayInfo := userEntity.GetDisplayInfo()

	return &dto.UserResponse{
		ID:          userEntity.SID(),
		Email:       userEntity.Email().String(),
		Name:        userEntity.Name().String(),
		DisplayName: displayInfo.DisplayName,
		Initials:    displayInfo.Initials,
		Role:        string(userEntity.Role()),
		Status:      userEntity.Status().String(),
		CreatedAt:   userEntity.CreatedAt(),
		UpdatedAt:   userEntity.UpdatedAt(),
		Metadata: dto.UserMetadata{
			IsBusinessEmail:      userEntity.IsBusinessEmail(),
			CanPerformActions:    userEntity.CanPerformActions(),
			RequiresVerification: userEntity.RequiresVerification(),
			EmailDomain:          userEntity.Email().Domain(),
		},
	}
}
