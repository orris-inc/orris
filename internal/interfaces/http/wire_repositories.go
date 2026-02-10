package http

import (
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/domain/notification"
	"github.com/orris-inc/orris/internal/domain/resource"
	"github.com/orris-inc/orris/internal/domain/setting"
	"github.com/orris-inc/orris/internal/domain/subscription"
	"github.com/orris-inc/orris/internal/domain/user"
	"github.com/orris-inc/orris/internal/infrastructure/repository"
)

// repositories holds all repository instances used by the application.
// Types match the return types of the repository constructors.
type repositories struct {
	userRepo                   user.Repository
	sessionRepo                user.SessionRepository
	oauthRepo                  user.OAuthAccountRepository
	subscriptionRepo           subscription.SubscriptionRepository
	subscriptionPlanRepo       subscription.PlanRepository
	subscriptionTokenRepo      subscription.SubscriptionTokenRepository
	subscriptionUsageRepo      subscription.SubscriptionUsageRepository
	subscriptionUsageStatsRepo subscription.SubscriptionUsageStatsRepository
	planPricingRepo            subscription.PlanPricingRepository
	paymentRepo                *repository.PaymentRepository
	nodeRepoImpl               node.NodeRepository
	forwardRuleRepo            forward.Repository
	forwardAgentRepo           forward.AgentRepository
	resourceGroupRepo          resource.Repository
	announcementRepo           notification.AnnouncementRepository
	notificationRepo           notification.NotificationRepository
	templateRepo               notification.NotificationTemplateRepository
	userAnnouncementReadRepo   *repository.UserAnnouncementReadRepository
	settingRepo                setting.Repository
	telegramBindingRepo        *repository.TelegramBindingRepository
	adminBindingRepo           *repository.AdminTelegramBindingRepository
}
