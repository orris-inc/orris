package usecases

import (
	"context"

	"github.com/orris-inc/orris/internal/application/forward/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// GetForwardChainUseCase handles retrieving a forward chain by ID.
type GetForwardChainUseCase struct {
	repo   forward.ChainRepository
	logger logger.Interface
}

// NewGetForwardChainUseCase creates a new GetForwardChainUseCase.
func NewGetForwardChainUseCase(
	repo forward.ChainRepository,
	logger logger.Interface,
) *GetForwardChainUseCase {
	return &GetForwardChainUseCase{
		repo:   repo,
		logger: logger,
	}
}

// Execute retrieves a forward chain by ID.
func (uc *GetForwardChainUseCase) Execute(ctx context.Context, id uint) (*dto.ForwardChainDTO, error) {
	uc.logger.Infow("executing get forward chain use case", "id", id)

	if id == 0 {
		return nil, errors.NewValidationError("chain ID is required")
	}

	chain, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		uc.logger.Errorw("failed to get forward chain", "id", id, "error", err)
		return nil, err
	}

	if chain == nil {
		return nil, errors.NewNotFoundError("forward chain", string(rune(id)))
	}

	return dto.ToForwardChainDTO(chain), nil
}
