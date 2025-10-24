package usecases

import (
	"context"
	"fmt"

	"orris/internal/application/user/dto"
	domainUser "orris/internal/domain/user"
	"orris/internal/domain/user/specifications"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

// GetUserUseCase handles the business logic for retrieving a user
type GetUserUseCase struct {
	userRepo domainUser.RepositoryWithSpecifications
	logger   logger.Interface
}

// NewGetUserUseCase creates a new get user use case
func NewGetUserUseCase(userRepo domainUser.RepositoryWithSpecifications, logger logger.Interface) *GetUserUseCase {
	return &GetUserUseCase{
		userRepo: userRepo,
		logger:   logger,
	}
}

// ExecuteByID retrieves a user by ID
func (uc *GetUserUseCase) ExecuteByID(ctx context.Context, id uint) (*dto.UserResponse, error) {
	uc.logger.Infow("executing get user by ID", "id", id)
	
	if id == 0 {
		return nil, errors.NewValidationError("user ID cannot be zero")
	}
	
	// Retrieve the user
	userEntity, err := uc.userRepo.GetByID(ctx, id)
	if err != nil {
		uc.logger.Errorw("failed to get user", "id", id, "error", err)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	
	if userEntity == nil {
		uc.logger.Warnw("user not found", "id", id)
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
	
	// Use specification to find user
	emailSpec := specifications.NewEmailSpecification(email)
	users, err := uc.userRepo.FindBySpecification(ctx, emailSpec, 1)
	if err != nil {
		uc.logger.Errorw("failed to get user by email", "email", email, "error", err)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	
	if len(users) == 0 {
		uc.logger.Warnw("user not found", "email", email)
		return nil, errors.NewNotFoundError("user not found")
	}
	
	// Map to response DTO
	return uc.mapToResponse(users[0]), nil
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

// ExecuteBySpecification retrieves users matching a specification
func (uc *GetUserUseCase) ExecuteBySpecification(ctx context.Context, spec specifications.Specification, limit int) ([]*dto.UserResponse, error) {
	uc.logger.Infow("executing get users by specification")
	
	// Retrieve users matching specification
	users, err := uc.userRepo.FindBySpecification(ctx, spec, limit)
	if err != nil {
		uc.logger.Errorw("failed to find users by specification", "error", err)
		return nil, fmt.Errorf("failed to find users: %w", err)
	}
	
	// Map to response DTOs
	responses := make([]*dto.UserResponse, len(users))
	for i, userEntity := range users {
		responses[i] = uc.mapToResponse(userEntity)
	}
	
	uc.logger.Infow("users found by specification", "count", len(users))
	return responses, nil
}

// mapToResponse maps a user entity to a response DTO
func (uc *GetUserUseCase) mapToResponse(userEntity *domainUser.User) *dto.UserResponse {
	displayInfo := userEntity.GetDisplayInfo()
	
	return &dto.UserResponse{
		ID:          userEntity.ID(),
		Email:       userEntity.Email().String(),
		Name:        userEntity.Name().String(),
		DisplayName: displayInfo.DisplayName,
		Initials:    displayInfo.Initials,
		Status:      userEntity.Status().String(),
		CreatedAt:   userEntity.CreatedAt(),
		UpdatedAt:   userEntity.UpdatedAt(),
		Metadata: dto.UserMetadata{
			IsBusinessEmail:       userEntity.IsBusinessEmail(),
			CanPerformActions:     userEntity.CanPerformActions(),
			RequiresVerification:  userEntity.RequiresVerification(),
			EmailDomain:           userEntity.Email().Domain(),
		},
	}
}