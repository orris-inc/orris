package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/domain/resource"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// DeleteResourceGroupUseCase handles deleting a resource group
type DeleteResourceGroupUseCase struct {
	repo   resource.Repository
	logger logger.Interface
}

// NewDeleteResourceGroupUseCase creates a new DeleteResourceGroupUseCase
func NewDeleteResourceGroupUseCase(repo resource.Repository, logger logger.Interface) *DeleteResourceGroupUseCase {
	return &DeleteResourceGroupUseCase{
		repo:   repo,
		logger: logger,
	}
}

// Execute soft deletes a resource group by its internal ID
func (uc *DeleteResourceGroupUseCase) Execute(ctx context.Context, id uint) error {
	// Check if resource group exists
	group, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		uc.logger.Errorw("failed to get resource group", "error", err, "id", id)
		return fmt.Errorf("failed to get resource group: %w", err)
	}
	if group == nil {
		return resource.ErrGroupNotFound
	}

	// Delete resource group
	if err := uc.repo.Delete(ctx, id); err != nil {
		uc.logger.Errorw("failed to delete resource group", "error", err, "id", id)
		return fmt.Errorf("failed to delete resource group: %w", err)
	}

	uc.logger.Infow("resource group deleted successfully", "id", id, "sid", group.SID())

	return nil
}
