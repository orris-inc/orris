package usecases

import (
	"context"
	"time"

	"orris/internal/domain/node"
	"orris/internal/domain/subscription"
	"orris/internal/shared/errors"
	"orris/internal/shared/logger"
)

type CheckTrafficLimitQuery struct {
	NodeID         uint
	UserID         *uint
	SubscriptionID uint
}

type TrafficLimitResult struct {
	Exceeded       bool   `json:"exceeded"`
	TotalTraffic   uint64 `json:"total_traffic"`
	TrafficLimit   uint64 `json:"traffic_limit"`
	RemainingBytes uint64 `json:"remaining_bytes"`
	UsagePercent   float64 `json:"usage_percent"`
}

type CheckTrafficLimitUseCase struct {
	trafficRepo      node.NodeTrafficRepository
	subscriptionRepo subscription.SubscriptionRepository
	planRepo         subscription.SubscriptionPlanRepository
	logger           logger.Interface
}

func NewCheckTrafficLimitUseCase(
	trafficRepo node.NodeTrafficRepository,
	subscriptionRepo subscription.SubscriptionRepository,
	planRepo subscription.SubscriptionPlanRepository,
	logger logger.Interface,
) *CheckTrafficLimitUseCase {
	return &CheckTrafficLimitUseCase{
		trafficRepo:      trafficRepo,
		subscriptionRepo: subscriptionRepo,
		planRepo:         planRepo,
		logger:           logger,
	}
}

func (uc *CheckTrafficLimitUseCase) Execute(
	ctx context.Context,
	query CheckTrafficLimitQuery,
) (*TrafficLimitResult, error) {
	uc.logger.Infow("checking traffic limit",
		"node_id", query.NodeID,
		"subscription_id", query.SubscriptionID,
	)

	if err := uc.validateQuery(query); err != nil {
		uc.logger.Errorw("invalid traffic limit query", "error", err)
		return nil, err
	}

	sub, err := uc.subscriptionRepo.GetByID(ctx, query.SubscriptionID)
	if err != nil {
		uc.logger.Errorw("failed to get subscription", "error", err)
		return nil, errors.NewNotFoundError("subscription not found")
	}

	plan, err := uc.planRepo.GetByID(ctx, sub.PlanID())
	if err != nil {
		uc.logger.Errorw("failed to get subscription plan", "error", err)
		return nil, errors.NewNotFoundError("subscription plan not found")
	}

	trafficLimit := plan.StorageLimit()
	if trafficLimit == 0 {
		uc.logger.Infow("no traffic limit configured",
			"subscription_id", query.SubscriptionID,
		)
		return &TrafficLimitResult{
			Exceeded:       false,
			TotalTraffic:   0,
			TrafficLimit:   0,
			RemainingBytes: 0,
			UsagePercent:   0,
		}, nil
	}

	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	endOfMonth := startOfMonth.AddDate(0, 1, 0)

	summary, err := uc.trafficRepo.GetTotalTraffic(ctx, query.NodeID, startOfMonth, endOfMonth)
	if err != nil {
		uc.logger.Errorw("failed to get total traffic", "error", err)
		return nil, errors.NewInternalError("failed to get traffic statistics")
	}

	totalTraffic := uint64(0)
	if summary != nil {
		totalTraffic = summary.Total
	}

	exceeded := totalTraffic >= trafficLimit
	remainingBytes := uint64(0)
	if !exceeded {
		remainingBytes = trafficLimit - totalTraffic
	}

	usagePercent := float64(0)
	if trafficLimit > 0 {
		usagePercent = float64(totalTraffic) / float64(trafficLimit) * 100
	}

	result := &TrafficLimitResult{
		Exceeded:       exceeded,
		TotalTraffic:   totalTraffic,
		TrafficLimit:   trafficLimit,
		RemainingBytes: remainingBytes,
		UsagePercent:   usagePercent,
	}

	if exceeded {
		uc.logger.Warnw("traffic limit exceeded",
			"node_id", query.NodeID,
			"subscription_id", query.SubscriptionID,
			"total_traffic", totalTraffic,
			"limit", trafficLimit,
		)
	} else {
		uc.logger.Infow("traffic limit check completed",
			"exceeded", false,
			"usage_percent", usagePercent,
		)
	}

	return result, nil
}

func (uc *CheckTrafficLimitUseCase) validateQuery(query CheckTrafficLimitQuery) error {
	if query.NodeID == 0 {
		return errors.NewValidationError("node ID is required")
	}

	if query.SubscriptionID == 0 {
		return errors.NewValidationError("subscription ID is required")
	}

	return nil
}
