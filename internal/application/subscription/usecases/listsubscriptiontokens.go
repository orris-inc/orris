package usecases

import (
	"context"
	"fmt"
	"time"

	"orris/internal/domain/subscription"
	"orris/internal/shared/logger"
)

type ListSubscriptionTokensQuery struct {
	SubscriptionID uint
	ActiveOnly     bool
}

type SubscriptionTokenDTO struct {
	ID             uint
	SubscriptionID uint
	Name           string
	Prefix         string
	Scope          string
	ExpiresAt      *time.Time
	LastUsedAt     *time.Time
	UsageCount     uint64
	IsActive       bool
	CreatedAt      time.Time
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

func (uc *ListSubscriptionTokensUseCase) Execute(ctx context.Context, query ListSubscriptionTokensQuery) ([]*SubscriptionTokenDTO, error) {
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

	dtos := make([]*SubscriptionTokenDTO, 0, len(tokens))
	for _, token := range tokens {
		dtos = append(dtos, uc.toDTO(token))
	}

	uc.logger.Infow("tokens listed successfully",
		"subscription_id", query.SubscriptionID,
		"count", len(dtos),
		"active_only", query.ActiveOnly,
	)

	return dtos, nil
}

func (uc *ListSubscriptionTokensUseCase) toDTO(token *subscription.SubscriptionToken) *SubscriptionTokenDTO {
	return &SubscriptionTokenDTO{
		ID:             token.ID(),
		SubscriptionID: token.SubscriptionID(),
		Name:           token.Name(),
		Prefix:         token.Prefix(),
		Scope:          token.Scope().String(),
		ExpiresAt:      token.ExpiresAt(),
		LastUsedAt:     token.LastUsedAt(),
		UsageCount:     token.UsageCount(),
		IsActive:       token.IsActive(),
		CreatedAt:      token.CreatedAt(),
	}
}
