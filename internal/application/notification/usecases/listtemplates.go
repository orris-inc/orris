package usecases

import (
	"context"

	"orris/internal/application/notification/dto"
	"orris/internal/shared/logger"
)

type ListTemplatesUseCase struct {
	repo   NotificationTemplateRepository
	logger logger.Interface
}

func NewListTemplatesUseCase(
	repo NotificationTemplateRepository,
	logger logger.Interface,
) *ListTemplatesUseCase {
	return &ListTemplatesUseCase{
		repo:   repo,
		logger: logger,
	}
}

func (uc *ListTemplatesUseCase) Execute(ctx context.Context) ([]*dto.TemplateResponse, error) {
	uc.logger.Infow("executing list templates use case")

	templates, err := uc.repo.FindAll(ctx)
	if err != nil {
		uc.logger.Errorw("failed to list templates", "error", err)
		return nil, err
	}

	responses := dto.ToTemplateResponseList(templates)

	return responses, nil
}
