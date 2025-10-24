package usecases

import (
	"context"
	"fmt"

	"orris/internal/application/notification/dto"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

type UpdateTemplateUseCase struct {
	repo   NotificationTemplateRepository
	logger logger.Interface
}

func NewUpdateTemplateUseCase(
	repo NotificationTemplateRepository,
	logger logger.Interface,
) *UpdateTemplateUseCase {
	return &UpdateTemplateUseCase{
		repo:   repo,
		logger: logger,
	}
}

func (uc *UpdateTemplateUseCase) Execute(ctx context.Context, id uint, req dto.UpdateTemplateRequest) (*dto.TemplateResponse, error) {
	uc.logger.Infow("executing update template use case", "id", id)

	template, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		uc.logger.Errorw("failed to find template", "id", id, "error", err)
		return nil, errors.NewNotFoundError("template not found")
	}

	if err := uc.repo.Update(ctx, template); err != nil {
		uc.logger.Errorw("failed to update template", "id", id, "error", err)
		return nil, fmt.Errorf("failed to update template: %w", err)
	}

	response := dto.ToTemplateResponse(template)

	uc.logger.Infow("template updated successfully", "id", id)
	return response, nil
}
