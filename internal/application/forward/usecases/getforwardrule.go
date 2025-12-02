package usecases

import (
	"context"
	"fmt"

	"orris/internal/application/forward/dto"
	"orris/internal/domain/forward"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

// GetForwardRuleQuery represents the input for getting a forward rule.
type GetForwardRuleQuery struct {
	ID uint
}

// GetForwardRuleUseCase handles getting a single forward rule.
type GetForwardRuleUseCase struct {
	repo   forward.Repository
	logger logger.Interface
}

// NewGetForwardRuleUseCase creates a new GetForwardRuleUseCase.
func NewGetForwardRuleUseCase(
	repo forward.Repository,
	logger logger.Interface,
) *GetForwardRuleUseCase {
	return &GetForwardRuleUseCase{
		repo:   repo,
		logger: logger,
	}
}

// Execute retrieves a forward rule by ID.
func (uc *GetForwardRuleUseCase) Execute(ctx context.Context, query GetForwardRuleQuery) (*dto.ForwardRuleDTO, error) {
	uc.logger.Infow("executing get forward rule use case", "id", query.ID)

	if query.ID == 0 {
		return nil, errors.NewValidationError("rule ID is required")
	}

	rule, err := uc.repo.GetByID(ctx, query.ID)
	if err != nil {
		uc.logger.Errorw("failed to get forward rule", "id", query.ID, "error", err)
		return nil, fmt.Errorf("failed to get forward rule: %w", err)
	}
	if rule == nil {
		return nil, errors.NewNotFoundError("forward rule", fmt.Sprintf("%d", query.ID))
	}

	return dto.ToForwardRuleDTO(rule), nil
}
