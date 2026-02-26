package http

import (
	"context"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	forwardServices "github.com/orris-inc/orris/internal/application/forward/services"
	nodeServices "github.com/orris-inc/orris/internal/application/node/services"
	nodeUsecases "github.com/orris-inc/orris/internal/application/node/usecases"
	settingApp "github.com/orris-inc/orris/internal/application/setting"
	subscriptionServices "github.com/orris-inc/orris/internal/application/subscription/services"
	telegramApp "github.com/orris-inc/orris/internal/application/telegram"
	telegramAdminApp "github.com/orris-inc/orris/internal/application/telegram/admin"
	"github.com/orris-inc/orris/internal/application/user"
	"github.com/orris-inc/orris/internal/infrastructure/auth"
	"github.com/orris-inc/orris/internal/infrastructure/cache"
	"github.com/orris-inc/orris/internal/infrastructure/config"
	"github.com/orris-inc/orris/internal/infrastructure/email"
	infraPayment "github.com/orris-inc/orris/internal/infrastructure/payment"
	"github.com/orris-inc/orris/internal/infrastructure/pubsub"
	"github.com/orris-inc/orris/internal/infrastructure/scheduler"
	"github.com/orris-inc/orris/internal/infrastructure/services"
	telegramInfra "github.com/orris-inc/orris/internal/infrastructure/telegram"
	"github.com/orris-inc/orris/internal/infrastructure/template"
	"github.com/orris-inc/orris/internal/interfaces/adapters"
	"github.com/orris-inc/orris/internal/interfaces/http/middleware"
	"github.com/orris-inc/orris/internal/shared/logger"
)

// Container holds all infrastructure components, repositories, use cases, handlers,
// and background services. It is responsible for wiring everything together and
// providing a Shutdown() method for graceful termination.
type Container struct {
	// Core infrastructure
	engine *gin.Engine
	db     *gorm.DB
	cfg    *config.Config
	log    logger.Interface
	redis  *redis.Client

	// Repositories
	repos *repositories

	// Use cases
	ucs *allUseCases

	// Handlers
	hdlrs *allHandlers

	// Middlewares
	authMiddleware              *middleware.AuthMiddleware
	subscriptionOwnerMiddleware *middleware.SubscriptionOwnerMiddleware
	nodeTokenMiddleware         *middleware.NodeTokenMiddleware
	nodeOwnerMiddleware         *middleware.NodeOwnerMiddleware
	nodeQuotaMiddleware         *middleware.NodeQuotaMiddleware
	forwardAgentTokenMiddleware *middleware.ForwardAgentTokenMiddleware
	forwardRuleOwnerMiddleware  *middleware.ForwardRuleOwnerMiddleware
	forwardQuotaMiddleware      *middleware.ForwardQuotaMiddleware
	rateLimiter                 *middleware.RateLimiter

	// Auth & setting infrastructure services
	jwtSvc               *auth.JWTService
	jwtService           *jwtServiceAdapter
	agentTokenSvc        *auth.AgentTokenService
	oauthManager         *auth.OAuthServiceManager
	emailManager         *email.EmailServiceManager
	settingServiceDDD    *settingApp.ServiceDDD
	settingProviderAdapt *settingProviderAdapter

	// Caches
	hourlyTrafficCache       cache.HourlyTrafficCache
	subscriptionQuotaCache   *cache.RedisSubscriptionQuotaCache
	forwardTrafficCache      cache.ForwardTrafficCache
	subscriptionTrafficCache cache.SubscriptionTrafficCache

	// Background services and hubs
	schedulerManager             *scheduler.SchedulerManager
	usdtServiceManager           *infraPayment.USDTServiceManager
	agentHub                     *services.AgentHub
	adminHub                     *services.AdminHub
	configSyncService            *forwardServices.ConfigSyncService
	trafficLimitEnforcementSvc   *forwardServices.TrafficLimitEnforcementService
	ruleTrafficBuffer            *forwardServices.RuleTrafficBuffer
	ruleTrafficFlushDone         chan struct{}
	subscriptionTrafficBuffer    *nodeServices.SubscriptionTrafficBuffer
	subscriptionTrafficFlushDone chan struct{}

	// Hub event bus for cross-instance WebSocket/SSE relay
	hubEventBus         *pubsub.RedisHubEventBus
	hubEventBusCancel   context.CancelFunc
	hubEventBusCancelMu sync.Mutex

	// Telegram
	telegramServiceDDD          *telegramApp.ServiceDDD
	telegramBotManager          *telegramInfra.BotServiceManager
	adminNotificationServiceDDD *telegramAdminApp.ServiceDDD

	// Cross-cutting adapters and services (created in one section, used in another)
	nodeRepoAdapter                *adapters.NodeRepositoryAdapter
	tokenValidator                 *adapters.SubscriptionTokenValidatorAdapter
	templateLoader                 *template.SubscriptionTemplateLoader
	nodeStatusQuerier              *adapters.NodeSystemStatusQuerierAdapter
	forwardAgentReleaseService     *services.GitHubReleaseService
	nodeAgentReleaseService        *services.GitHubReleaseService
	serviceAdapter                 *telegramInfra.ServiceAdapter
	dynamicBotService              *telegramInfra.DynamicBotService
	nodeStatusHandler              *adapters.NodeStatusHandler
	forwardStatusHandler           *adapters.ForwardStatusHandler
	nodeConfigSyncService          *nodeServices.NodeConfigSyncService
	subscriptionSyncService        *nodeServices.SubscriptionSyncService
	quotaCacheSyncService          *subscriptionServices.QuotaCacheSyncService
	alertStateManager              *cache.AlertStateManager
	nodeTrafficLimitEnforcementSvc *nodeServices.NodeTrafficLimitEnforcementService
	systemStatusUpdater            nodeUsecases.NodeSystemStatusUpdater
	subscriptionIDResolver         nodeUsecases.SubscriptionIDResolver
	nodeQuotaCacheAdapter          *adapters.NodeSubscriptionQuotaCacheAdapter
	nodeQuotaLoaderAdapter         *adapters.NodeSubscriptionQuotaLoaderAdapter
	nodeUsageReaderAdapter         *adapters.NodeSubscriptionUsageReaderAdapter
	onlineSubscriptionTracker      *adapters.OnlineSubscriptionTrackerAdapter

	// User service (passed from outside)
	userService *user.ServiceDDD
}

// NewContainer creates a new Container with all dependencies wired together.
// The initialization order follows the original NewRouter() logic to preserve
// the complex inter-dependencies between components.
func NewContainer(userService *user.ServiceDDD, db *gorm.DB, cfg *config.Config, log logger.Interface) *Container {
	c := &Container{
		engine:      gin.New(),
		db:          db,
		cfg:         cfg,
		log:         log,
		userService: userService,
	}

	// Section 1: Infrastructure - Redis, Repositories, Basic Services
	c.initInfrastructure()

	// Section 2: Subscription - UseCases, Handlers, Scheduler Jobs
	c.initSubscription()

	// Section 3: Node - UseCases, Handlers, Middlewares
	c.initNode()

	// Section 4: Settings & Auth - OAuth, Email, Passkey
	c.initSettingsAndAuth()

	// Section 5: Telegram - Bot, Notifications, Admin Alerts
	c.initTelegram()

	// Section 6: Forward - Agents, Rules, Traffic, AgentHub
	c.initForward()

	// Section 7: Callbacks & Notifiers - Event Handlers, Sync
	c.initCallbacksAndNotifiers()

	// Section 8: Final remaining handlers and middlewares
	c.initRemainingHandlers()

	return c
}

