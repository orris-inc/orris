package usecases

import (
	"context"
	"time"

	"golang.org/x/sync/errgroup"

	dto "github.com/orris-inc/orris/internal/application/admin/dto"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/domain/user"
	"github.com/orris-inc/orris/internal/infrastructure/cache"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/errors"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// GetAdminDashboardUseCase handles retrieving admin dashboard snapshot.
type GetAdminDashboardUseCase struct {
	userRepo           user.Repository
	subscriptionRepo   subscription.SubscriptionRepository
	nodeRepo           node.NodeRepository
	forwardRuleRepo    forward.RuleQuerier
	forwardAgentRepo   forward.AgentRepository
	hourlyTrafficCache cache.HourlyTrafficCache
	logger             logger.Interface
}

// NewGetAdminDashboardUseCase creates a new GetAdminDashboardUseCase.
func NewGetAdminDashboardUseCase(
	userRepo user.Repository,
	subscriptionRepo subscription.SubscriptionRepository,
	nodeRepo node.NodeRepository,
	forwardRuleRepo forward.RuleQuerier,
	forwardAgentRepo forward.AgentRepository,
	hourlyTrafficCache cache.HourlyTrafficCache,
	log logger.Interface,
) *GetAdminDashboardUseCase {
	return &GetAdminDashboardUseCase{
		userRepo:           userRepo,
		subscriptionRepo:   subscriptionRepo,
		nodeRepo:           nodeRepo,
		forwardRuleRepo:    forwardRuleRepo,
		forwardAgentRepo:   forwardAgentRepo,
		hourlyTrafficCache: hourlyTrafficCache,
		logger:             log,
	}
}

// Execute retrieves the admin dashboard snapshot.
func (uc *GetAdminDashboardUseCase) Execute(ctx context.Context) (*dto.AdminDashboardResponse, error) {
	uc.logger.Debugw("fetching admin dashboard")

	now := biztime.NowUTC()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	weekday := now.Weekday()
	if weekday == time.Sunday {
		weekday = 7
	}
	weekStart := todayStart.AddDate(0, 0, -int(weekday-time.Monday))
	onlineThreshold := now.Add(-5 * time.Minute)

	var (
		totalUsers    int64
		newToday      int64
		newThisWeek   int64
		activeSubs    int64
		expiredSubs   int64
		suspendedSubs int64
		pendingSubs   int64
		expiring7Days int64
		totalNodes    int64
		onlineNodes   int64
		totalRules    int64
		totalAgents   int64
		onlineAgents  int64
		trafficUpload   uint64
		trafficDownload uint64
		trafficTotal    uint64
	)

	g, gctx := errgroup.WithContext(ctx)

	// Users: total
	g.Go(func() error {
		_, total, err := uc.userRepo.List(gctx, user.ListFilter{Page: 1, PageSize: 1})
		if err != nil {
			return errors.NewInternalError("failed to count total users")
		}
		totalUsers = total
		return nil
	})

	// Users: new today
	g.Go(func() error {
		ts := todayStart
		_, total, err := uc.userRepo.List(gctx, user.ListFilter{Page: 1, PageSize: 1, CreatedAfter: &ts})
		if err != nil {
			return errors.NewInternalError("failed to count new users today")
		}
		newToday = total
		return nil
	})

	// Users: new this week
	g.Go(func() error {
		ws := weekStart
		_, total, err := uc.userRepo.List(gctx, user.ListFilter{Page: 1, PageSize: 1, CreatedAfter: &ws})
		if err != nil {
			return errors.NewInternalError("failed to count new users this week")
		}
		newThisWeek = total
		return nil
	})

	// Subscriptions: active
	g.Go(func() error {
		count, err := uc.subscriptionRepo.CountByStatus(gctx, "active")
		if err != nil {
			return errors.NewInternalError("failed to count active subscriptions")
		}
		activeSubs = count
		return nil
	})

	// Subscriptions: expired
	g.Go(func() error {
		count, err := uc.subscriptionRepo.CountByStatus(gctx, "expired")
		if err != nil {
			return errors.NewInternalError("failed to count expired subscriptions")
		}
		expiredSubs = count
		return nil
	})

	// Subscriptions: suspended
	g.Go(func() error {
		count, err := uc.subscriptionRepo.CountByStatus(gctx, "suspended")
		if err != nil {
			return errors.NewInternalError("failed to count suspended subscriptions")
		}
		suspendedSubs = count
		return nil
	})

	// Subscriptions: pending_payment
	g.Go(func() error {
		count, err := uc.subscriptionRepo.CountByStatus(gctx, "pending_payment")
		if err != nil {
			return errors.NewInternalError("failed to count pending payment subscriptions")
		}
		pendingSubs = count
		return nil
	})

	// Subscriptions: expiring in 7 days
	g.Go(func() error {
		subs, err := uc.subscriptionRepo.FindExpiringSubscriptions(gctx, 7)
		if err != nil {
			return errors.NewInternalError("failed to find expiring subscriptions")
		}
		expiring7Days = int64(len(subs))
		return nil
	})

	// Nodes: total
	g.Go(func() error {
		nf := node.NodeFilter{}
		nf.Page = 1
		nf.PageSize = 1
		_, total, err := uc.nodeRepo.List(gctx, nf)
		if err != nil {
			return errors.NewInternalError("failed to count total nodes")
		}
		totalNodes = total
		return nil
	})

	// Nodes: online
	g.Go(func() error {
		count, err := uc.nodeRepo.CountByLastSeenAfter(gctx, onlineThreshold)
		if err != nil {
			return errors.NewInternalError("failed to count online nodes")
		}
		onlineNodes = count
		return nil
	})

	// Forward rules: total
	g.Go(func() error {
		_, total, err := uc.forwardRuleRepo.List(gctx, forward.ListFilter{Page: 1, PageSize: 1})
		if err != nil {
			return errors.NewInternalError("failed to count forward rules")
		}
		totalRules = total
		return nil
	})

	// Forward agents: total
	g.Go(func() error {
		_, total, err := uc.forwardAgentRepo.List(gctx, forward.AgentListFilter{Page: 1, PageSize: 1})
		if err != nil {
			return errors.NewInternalError("failed to count forward agents")
		}
		totalAgents = total
		return nil
	})

	// Forward agents: online
	g.Go(func() error {
		count, err := uc.forwardAgentRepo.CountByLastSeenAfter(gctx, onlineThreshold)
		if err != nil {
			return errors.NewInternalError("failed to count online forward agents")
		}
		onlineAgents = count
		return nil
	})

	// Traffic today
	g.Go(func() error {
		traffic, err := uc.hourlyTrafficCache.GetPlatformTotalTraffic(gctx, "", todayStart, now)
		if err != nil {
			uc.logger.Warnw("failed to get today's traffic from cache, defaulting to zero", "error", err)
			return nil
		}
		trafficUpload = traffic.Upload
		trafficDownload = traffic.Download
		trafficTotal = traffic.Total
		return nil
	})

	if err := g.Wait(); err != nil {
		uc.logger.Errorw("failed to fetch admin dashboard data", "error", err)
		return nil, err
	}

	resp := &dto.AdminDashboardResponse{
		Users: dto.DashboardUsersSection{
			Total:       totalUsers,
			NewToday:    newToday,
			NewThisWeek: newThisWeek,
		},
		Subscriptions: dto.DashboardSubscriptionsSection{
			Active:          activeSubs,
			Expired:         expiredSubs,
			Suspended:       suspendedSubs,
			PendingPayment:  pendingSubs,
			ExpiringIn7Days: expiring7Days,
		},
		Nodes: dto.DashboardNodesSection{
			Total:   totalNodes,
			Online:  onlineNodes,
			Offline: totalNodes - onlineNodes,
		},
		Forward: dto.DashboardForwardSection{
			TotalRules:   totalRules,
			TotalAgents:  totalAgents,
			OnlineAgents: onlineAgents,
		},
		TrafficToday: dto.DashboardTrafficSection{
			Upload:   trafficUpload,
			Download: trafficDownload,
			Total:    trafficTotal,
		},
	}

	uc.logger.Debugw("admin dashboard fetched successfully",
		"total_users", totalUsers,
		"active_subs", activeSubs,
		"total_nodes", totalNodes,
	)

	return resp, nil
}
