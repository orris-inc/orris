package usecases

import (
	"context"
	"fmt"

	"github.com/orris-inc/orris/internal/application/subscription/dto"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/shared/logger"
)

type ListSubscriptionTokensQuery struct {
	SubscriptionID uint
	ActiveOnly     bool
}

type ListSubscriptionTokensUseCase struct {
	tokenRepo subscription.SubscriptionTokenRepository
	logger    logger.Interface
}

func NewListSubscriptionTokensUseCase(
	tokenRepo subscription.SubscriptionTokenRepository,
	logger logger.Interface,
) *ListSubscriptionTokensUseCase {
	return &ListSubscriptionTokensUseCase{
		tokenRepo: tokenRepo,
		logger:    logger,
	}
}

func (uc *ListSubscriptionTokensUseCase) Execute(ctx context.Context, query ListSubscriptionTokensQuery) ([]*dto.SubscriptionTokenDTO, error) {
	var tokens []*subscription.SubscriptionToken
	var err error

	if query.ActiveOnly {
		tokens, err = uc.tokenRepo.GetActiveBySubscriptionID(ctx, query.SubscriptionID)
	} else {
		tokens, err = uc.tokenRepo.GetBySubscriptionID(ctx, query.SubscriptionID)
	}

	if err != nil {
		uc.logger.Errorw("failed to get tokens",
			"error", err,
			"subscription_id", query.SubscriptionID,
			"active_only", query.ActiveOnly,
		)
		return nil, fmt.Errorf("failed to get tokens: %w", err)
	}

	dtos := make([]*dto.SubscriptionTokenDTO, 0, len(tokens))
	for _, token := range tokens {
		dtos = append(dtos, dto.ToSubscriptionTokenDTO(token))
	}

	uc.logger.Debugw("tokens listed successfully",
		"subscription_id", query.SubscriptionID,
		"count", len(dtos),
		"active_only", query.ActiveOnly,
	)

	return dtos, nil
}
