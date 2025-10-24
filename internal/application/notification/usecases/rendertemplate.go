package usecases

import (
	"context"
	"fmt"

	"orris/internal/application/notification/dto"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

type RenderTemplateUseCase struct {
	repo            NotificationTemplateRepository
	markdownService dto.MarkdownService
	logger          logger.Interface
}

func NewRenderTemplateUseCase(
	repo NotificationTemplateRepository,
	markdownService dto.MarkdownService,
	logger logger.Interface,
) *RenderTemplateUseCase {
	return &RenderTemplateUseCase{
		repo:            repo,
		markdownService: markdownService,
		logger:          logger,
	}
}

func (uc *RenderTemplateUseCase) Execute(ctx context.Context, req dto.RenderTemplateRequest) (*dto.RenderTemplateResponse, error) {
	uc.logger.Infow("executing render template use case", "template_type", req.TemplateType)

	template, err := uc.repo.FindByType(ctx, req.TemplateType)
	if err != nil {
		uc.logger.Errorw("failed to find template", "template_type", req.TemplateType, "error", err)
		return nil, errors.NewNotFoundError("template", req.TemplateType)
	}

	title, content, err := template.Render(req.Data)
	if err != nil {
		uc.logger.Errorw("failed to render template", "template_type", req.TemplateType, "error", err)
		return nil, fmt.Errorf("failed to render template: %w", err)
	}

	contentHTML := ""
	if uc.markdownService != nil {
		html, err := uc.markdownService.ToHTML(content)
		if err == nil {
			contentHTML = html
		} else {
			uc.logger.Warnw("failed to convert markdown to html", "error", err)
		}
	}

	return &dto.RenderTemplateResponse{
		Title:       title,
		Content:     content,
		ContentHTML: contentHTML,
	}, nil
}
