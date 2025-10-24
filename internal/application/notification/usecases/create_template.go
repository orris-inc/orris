package usecases

import (
	"context"
	"fmt"

	"orris/internal/application/notification/dto"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

type TemplateFactory interface {
	CreateTemplate(templateType, name, title, content string, variables []string) (NotificationTemplate, error)
}

type CreateTemplateUseCase struct {
	repo    NotificationTemplateRepository
	factory TemplateFactory
	logger  logger.Interface
}

func NewCreateTemplateUseCase(
	repo NotificationTemplateRepository,
	factory TemplateFactory,
	logger logger.Interface,
) *CreateTemplateUseCase {
	return &CreateTemplateUseCase{
		repo:    repo,
		factory: factory,
		logger:  logger,
	}
}

func (uc *CreateTemplateUseCase) Execute(ctx context.Context, req dto.CreateTemplateRequest) (*dto.TemplateResponse, error) {
	uc.logger.Infow("executing create template use case", "template_type", req.TemplateType, "name", req.Name)

	template, err := uc.factory.CreateTemplate(req.TemplateType, req.Name, req.Title, req.Content, req.Variables)
	if err != nil {
		uc.logger.Errorw("failed to create template entity", "error", err)
		return nil, errors.NewValidationError(fmt.Sprintf("failed to create template: %v", err))
	}

	if err := uc.repo.Create(ctx, template); err != nil {
		uc.logger.Errorw("failed to persist template", "error", err)
		return nil, fmt.Errorf("failed to save template: %w", err)
	}

	response := dto.ToTemplateResponse(template)

	uc.logger.Infow("template created successfully", "id", template.ID())
	return response, nil
}
