package http

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	adminUsecases "github.com/orris-inc/orris/internal/application/admin/usecases"
	forwardServices "github.com/orris-inc/orris/internal/application/forward/services"
	forwardUsecases "github.com/orris-inc/orris/internal/application/forward/usecases"
	nodeServices "github.com/orris-inc/orris/internal/application/node/services"
	nodeUsecases "github.com/orris-inc/orris/internal/application/node/usecases"
	notificationApp "github.com/orris-inc/orris/internal/application/notification"
	paymentGateway "github.com/orris-inc/orris/internal/application/payment/paymentgateway"
	paymentUsecases "github.com/orris-inc/orris/internal/application/payment/usecases"
	resourceUsecases "github.com/orris-inc/orris/internal/application/resource/usecases"
	settingApp "github.com/orris-inc/orris/internal/application/setting"
	settingUsecases "github.com/orris-inc/orris/internal/application/setting/usecases"
	subscriptionServices "github.com/orris-inc/orris/internal/application/subscription/services"
	subscriptionUsecases "github.com/orris-inc/orris/internal/application/subscription/usecases"
	telegramApp "github.com/orris-inc/orris/internal/application/telegram"
	telegramAdminApp "github.com/orris-inc/orris/internal/application/telegram/admin"
	telegramAdminUsecases "github.com/orris-inc/orris/internal/application/telegram/admin/usecases"
	"github.com/orris-inc/orris/internal/application/user/helpers"
	"github.com/orris-inc/orris/internal/application/user/usecases"
	"github.com/orris-inc/orris/internal/interfaces/adapters"
	"github.com/orris-inc/orris/internal/infrastructure/auth"
	"github.com/orris-inc/orris/internal/infrastructure/cache"
	"github.com/orris-inc/orris/internal/infrastructure/config"
	"github.com/orris-inc/orris/internal/infrastructure/email"
	infraPayment "github.com/orris-inc/orris/internal/infrastructure/payment"
	"github.com/orris-inc/orris/internal/infrastructure/pubsub"
	"github.com/orris-inc/orris/internal/infrastructure/repository"
	"github.com/orris-inc/orris/internal/infrastructure/scheduler"
	"github.com/orris-inc/orris/internal/infrastructure/services"
	telegramInfra "github.com/orris-inc/orris/internal/infrastructure/telegram"
	"github.com/orris-inc/orris/internal/infrastructure/template"
	"github.com/orris-inc/orris/internal/infrastructure/token"
	"github.com/orris-inc/orris/internal/interfaces/http/handlers"
	adminHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/admin"
	adminResourceGroupHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/admin/resourcegroup"
	adminSubscriptionHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/admin/subscription"
	forwardAgentAPIHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/forward/agent/api"
	forwardAgentCrudHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/forward/agent/crud"
	forwardAgentHubHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/forward/agent/hub"
	forwardRuleHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/forward/rule"
	forwardSubscriptionHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/forward/subscription"
	forwardUserHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/forward/user"
	nodeHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/node"
	telegramHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/telegram"
	ticketHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/ticket"
	"github.com/orris-inc/orris/internal/interfaces/http/middleware"
	"github.com/orris-inc/orris/internal/shared/biztime"
	shareddb "github.com/orris-inc/orris/internal/shared/db"
	"github.com/orris-inc/orris/internal/shared/goroutine"
	dto "github.com/orris-inc/orris/internal/shared/hubprotocol/forward"
	nodedto "github.com/orris-inc/orris/internal/shared/hubprotocol/node"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/services/markdown"
)

// ============================================================
// Section 1: Infrastructure - Redis, Repositories, Basic Services
// ============================================================

// initInfrastructure initializes Redis, all repositories, basic auth services,
// and early middlewares. Corresponds to Section 1 of the original NewRouter().
func (c *Container) initInfrastructure() {
	cfg := c.cfg
	log := c.log
	db := c.db

	// Initialize Redis client
	c.redis = initRedis(cfg, log)

	// Initialize all repositories
	c.repos = newRepositories(db, log)

	// Initialize auth services
	c.jwtSvc = auth.NewJWTService(cfg.Auth.JWT.Secret, cfg.Auth.JWT.AccessExpMinutes, cfg.Auth.JWT.RefreshExpDays)
	c.jwtService = &jwtServiceAdapter{c.jwtSvc}

	// Initialize HMAC-based agent token service for local token verification
	c.agentTokenSvc = auth.NewAgentTokenService(cfg.Forward.TokenSigningSecret)

	// Initialize early middlewares
	c.authMiddleware = middleware.NewAuthMiddleware(c.jwtSvc, c.repos.userRepo, cfg.Auth.Cookie, log)
	c.rateLimiter = middleware.NewRateLimiter(c.redis, 100, 1*time.Minute)

	// Initialize hourly traffic cache for Redis-based hourly data queries and daily aggregation
	c.hourlyTrafficCache = cache.NewRedisHourlyTrafficCache(c.redis, log)
}

// initRedis creates and tests the Redis client connection.
func initRedis(cfg *config.Config, log logger.Interface) *redis.Client {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.GetAddr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalw("failed to connect to Redis", "error", err)
	}
	log.Infow("Redis connection established successfully")

	return redisClient
}

// newRepositories creates all repository instances from the database connection.
func newRepositories(db *gorm.DB, log logger.Interface) *repositories {
	return &repositories{
		userRepo:                   repository.NewUserRepository(db, log),
		sessionRepo:                repository.NewSessionRepository(db),
		oauthRepo:                  repository.NewOAuthAccountRepository(db),
		subscriptionRepo:           repository.NewSubscriptionRepository(db, log),
		subscriptionPlanRepo:       repository.NewPlanRepository(db, log),
		subscriptionTokenRepo:      repository.NewSubscriptionTokenRepository(db, log),
		subscriptionUsageRepo:      repository.NewSubscriptionUsageRepository(db, log),
		subscriptionUsageStatsRepo: repository.NewSubscriptionUsageStatsRepository(db, log),
		planPricingRepo:            repository.NewPlanPricingRepository(db, log),
		paymentRepo:                repository.NewPaymentRepository(db),
		nodeRepoImpl:               repository.NewNodeRepository(db, log),
		forwardRuleRepo:            repository.NewForwardRuleRepository(db, log),
		forwardAgentRepo:           repository.NewForwardAgentRepository(db, log),
		resourceGroupRepo:          repository.NewResourceGroupRepository(db, log),
		announcementRepo:           repository.NewAnnouncementRepository(db),
		notificationRepo:           repository.NewNotificationRepository(db),
		templateRepo:               repository.NewNotificationTemplateRepository(db),
		userAnnouncementReadRepo:   repository.NewUserAnnouncementReadRepository(db),
		settingRepo:                repository.NewSystemSettingRepository(db, log),
		telegramBindingRepo:        repository.NewTelegramBindingRepository(db, log),
		adminBindingRepo:           repository.NewAdminTelegramBindingRepository(db, log),
	}
}

// ============================================================
// Section 2: Subscription - UseCases, Handlers, Scheduler Jobs
// ============================================================

// initSubscription initializes subscription-related use cases, handlers,
// scheduler manager, and plan/payment components.
// Corresponds to Section 2 of the original NewRouter().
func (c *Container) initSubscription() {
	cfg := c.cfg
	log := c.log
	db := c.db
	repos := c.repos

	tokenGenerator := token.NewTokenGenerator()
	txMgr := shareddb.NewTransactionManager(db)

	subscriptionBaseURL := cfg.Subscription.GetBaseURL(cfg.Server.GetBaseURL())

	// Initialize use cases struct
	ucs := &allUseCases{}
	c.ucs = ucs

	ucs.createSubscriptionUC = subscriptionUsecases.NewCreateSubscriptionUseCase(
		repos.subscriptionRepo, repos.subscriptionPlanRepo, repos.subscriptionTokenRepo,
		repos.planPricingRepo, repos.userRepo, tokenGenerator, txMgr, log,
	)
	ucs.activateSubscriptionUC = subscriptionUsecases.NewActivateSubscriptionUseCase(repos.subscriptionRepo, log)
	ucs.getSubscriptionUC = subscriptionUsecases.NewGetSubscriptionUseCase(
		repos.subscriptionRepo, repos.subscriptionPlanRepo, repos.userRepo, log, subscriptionBaseURL,
	)
	ucs.listUserSubscriptionsUC = subscriptionUsecases.NewListUserSubscriptionsUseCase(
		repos.subscriptionRepo, repos.subscriptionPlanRepo, repos.userRepo, log, subscriptionBaseURL,
	)
	ucs.cancelSubscriptionUC = subscriptionUsecases.NewCancelSubscriptionUseCase(repos.subscriptionRepo, repos.subscriptionTokenRepo, log)
	ucs.suspendSubscriptionUC = subscriptionUsecases.NewSuspendSubscriptionUseCase(repos.subscriptionRepo, log)
	ucs.unsuspendSubscriptionUC = subscriptionUsecases.NewUnsuspendSubscriptionUseCase(repos.subscriptionRepo, log)
	ucs.resetSubscriptionUsageUC = subscriptionUsecases.NewResetSubscriptionUsageUseCase(repos.subscriptionRepo, log)
	ucs.deleteSubscriptionUC = subscriptionUsecases.NewDeleteSubscriptionUseCase(repos.subscriptionRepo, repos.subscriptionTokenRepo, txMgr, log)
	ucs.renewSubscriptionUC = subscriptionUsecases.NewRenewSubscriptionUseCase(repos.subscriptionRepo, repos.subscriptionPlanRepo, repos.planPricingRepo, log)
	ucs.changePlanUC = subscriptionUsecases.NewChangePlanUseCase(repos.subscriptionRepo, repos.subscriptionPlanRepo, log)
	ucs.getSubscriptionUsageStatsUC = subscriptionUsecases.NewGetSubscriptionUsageStatsUseCase(
		repos.subscriptionUsageRepo, repos.subscriptionUsageStatsRepo, c.hourlyTrafficCache,
		repos.nodeRepoImpl, repos.forwardRuleRepo, log,
	)
	ucs.resetSubscriptionLinkUC = subscriptionUsecases.NewResetSubscriptionLinkUseCase(
		repos.subscriptionRepo, repos.subscriptionPlanRepo, repos.userRepo, log, subscriptionBaseURL,
	)
	ucs.aggregateUsageUC = subscriptionUsecases.NewAggregateUsageUseCase(
		repos.subscriptionUsageRepo, repos.subscriptionUsageStatsRepo, c.hourlyTrafficCache, log,
	)

	// Initialize unified scheduler manager (gocron v2)
	schedulerManager, err := scheduler.NewSchedulerManager(log)
	if err != nil {
		log.Fatalw("failed to create scheduler manager", "error", err)
	}
	c.schedulerManager = schedulerManager

	// Register payment jobs (5 min interval)
	ucs.expirePaymentsUC = paymentUsecases.NewExpirePaymentsUseCase(repos.paymentRepo, repos.subscriptionRepo, log)
	ucs.cancelUnpaidSubsUC = paymentUsecases.NewCancelUnpaidSubscriptionsUseCase(repos.subscriptionRepo, repos.paymentRepo, log)
	ucs.retryActivationUC = paymentUsecases.NewRetrySubscriptionActivationUseCase(repos.paymentRepo, ucs.activateSubscriptionUC, log)
	if err := schedulerManager.RegisterPaymentJobs(ucs.expirePaymentsUC, ucs.cancelUnpaidSubsUC, ucs.retryActivationUC); err != nil {
		log.Warnw("failed to register payment jobs", "error", err)
	}

	// Register subscription jobs (24h interval)
	ucs.expireSubscriptionsUC = subscriptionUsecases.NewExpireSubscriptionsUseCase(repos.subscriptionRepo, log)
	if err := schedulerManager.RegisterSubscriptionJobs(ucs.expireSubscriptionsUC); err != nil {
		log.Warnw("failed to register subscription jobs", "error", err)
	}

	// Register usage aggregation jobs (cron-based)
	if err := schedulerManager.RegisterUsageAggregationJobs(ucs.aggregateUsageUC, scheduler.DefaultRetentionDays); err != nil {
		log.Warnw("failed to register usage aggregation jobs", "error", err)
	}

	// Plan use cases
	ucs.createPlanUC = subscriptionUsecases.NewCreatePlanUseCase(repos.subscriptionPlanRepo, repos.planPricingRepo, log)
	ucs.updatePlanUC = subscriptionUsecases.NewUpdatePlanUseCase(repos.subscriptionPlanRepo, repos.planPricingRepo, log)
	ucs.getPlanUC = subscriptionUsecases.NewGetPlanUseCase(repos.subscriptionPlanRepo, repos.planPricingRepo, log)
	ucs.listPlansUC = subscriptionUsecases.NewListPlansUseCase(repos.subscriptionPlanRepo, repos.planPricingRepo, log)
	ucs.getPublicPlansUC = subscriptionUsecases.NewGetPublicPlansUseCase(repos.subscriptionPlanRepo, repos.planPricingRepo, log)
	ucs.activatePlanUC = subscriptionUsecases.NewActivatePlanUseCase(repos.subscriptionPlanRepo, log)
	ucs.deactivatePlanUC = subscriptionUsecases.NewDeactivatePlanUseCase(repos.subscriptionPlanRepo, log)
	ucs.deletePlanUC = subscriptionUsecases.NewDeletePlanUseCase(
		repos.subscriptionPlanRepo, repos.subscriptionRepo, repos.planPricingRepo, txMgr, log,
	)
	ucs.getPlanPricingsUC = subscriptionUsecases.NewGetPlanPricingsUseCase(repos.subscriptionPlanRepo, repos.planPricingRepo, log)

	// Subscription token use cases
	ucs.generateTokenUC = subscriptionUsecases.NewGenerateSubscriptionTokenUseCase(repos.subscriptionRepo, repos.subscriptionTokenRepo, tokenGenerator, log)
	ucs.listTokensUC = subscriptionUsecases.NewListSubscriptionTokensUseCase(repos.subscriptionTokenRepo, log)
	ucs.revokeTokenUC = subscriptionUsecases.NewRevokeSubscriptionTokenUseCase(repos.subscriptionTokenRepo, log)
	ucs.refreshSubscriptionTokenUC = subscriptionUsecases.NewRefreshSubscriptionTokenUseCase(repos.subscriptionTokenRepo, repos.subscriptionRepo, tokenGenerator, log)

	// Resource group use cases
	ucs.createResourceGroupUC = resourceUsecases.NewCreateResourceGroupUseCase(repos.resourceGroupRepo, repos.subscriptionPlanRepo, log)
	ucs.getResourceGroupUC = resourceUsecases.NewGetResourceGroupUseCase(repos.resourceGroupRepo, repos.subscriptionPlanRepo, log)
	ucs.listResourceGroupsUC = resourceUsecases.NewListResourceGroupsUseCase(repos.resourceGroupRepo, repos.subscriptionPlanRepo, log)
	ucs.updateResourceGroupUC = resourceUsecases.NewUpdateResourceGroupUseCase(repos.resourceGroupRepo, repos.subscriptionPlanRepo, log)
	ucs.deleteResourceGroupUC = resourceUsecases.NewDeleteResourceGroupUseCase(repos.resourceGroupRepo, repos.forwardRuleRepo, repos.nodeRepoImpl, log)
	ucs.updateResourceGroupStatusUC = resourceUsecases.NewUpdateResourceGroupStatusUseCase(
		repos.resourceGroupRepo, repos.subscriptionPlanRepo, repos.nodeRepoImpl, repos.forwardRuleRepo, log,
	)

	// Initialize handlers struct and subscription/plan handlers
	hdlrs := &allHandlers{}
	c.hdlrs = hdlrs

	hdlrs.subscriptionHandler = handlers.NewSubscriptionHandler(
		ucs.createSubscriptionUC, ucs.getSubscriptionUC, ucs.listUserSubscriptionsUC,
		ucs.cancelSubscriptionUC, ucs.deleteSubscriptionUC, ucs.changePlanUC,
		ucs.getSubscriptionUsageStatsUC, ucs.resetSubscriptionLinkUC, log,
	)
	hdlrs.adminSubscriptionHandler = adminSubscriptionHandlers.NewHandler(
		repos.subscriptionRepo, ucs.createSubscriptionUC, ucs.getSubscriptionUC, ucs.listUserSubscriptionsUC,
		ucs.cancelSubscriptionUC, ucs.deleteSubscriptionUC, ucs.renewSubscriptionUC, ucs.changePlanUC,
		ucs.activateSubscriptionUC, ucs.suspendSubscriptionUC, ucs.unsuspendSubscriptionUC,
		ucs.resetSubscriptionUsageUC, log,
	)
	c.subscriptionOwnerMiddleware = middleware.NewSubscriptionOwnerMiddleware(repos.subscriptionRepo, log)

	hdlrs.planHandler = handlers.NewPlanHandler(
		ucs.createPlanUC, ucs.updatePlanUC, ucs.getPlanUC, ucs.listPlansUC,
		ucs.getPublicPlansUC, ucs.activatePlanUC, ucs.deactivatePlanUC, ucs.deletePlanUC, ucs.getPlanPricingsUC,
		log,
	)
	hdlrs.subscriptionTokenHandler = handlers.NewSubscriptionTokenHandler(
		ucs.generateTokenUC, ucs.listTokensUC, ucs.revokeTokenUC, ucs.refreshSubscriptionTokenUC,
		log,
	)
}

// ============================================================
// Section 3: Node - UseCases, Handlers, Middlewares
// ============================================================

// initNode initializes node-related components including use cases, handlers,
// notification components, and node middleware.
// Corresponds to Section 3 of the original NewRouter().
func (c *Container) initNode() {
	cfg := c.cfg
	log := c.log
	db := c.db
	repos := c.repos
	ucs := c.ucs
	hdlrs := c.hdlrs

	// Initialize adapters
	c.nodeRepoAdapter = adapters.NewNodeRepositoryAdapter(repos.nodeRepoImpl, repos.forwardRuleRepo, db, log)
	c.tokenValidator = adapters.NewSubscriptionTokenValidatorAdapter(db, log)
	c.nodeStatusQuerier = adapters.NewNodeSystemStatusQuerierAdapter(c.redis, log)

	// Initialize GitHub release services for version checking
	c.forwardAgentReleaseService = services.NewGitHubReleaseService(services.GitHubRepoConfig{
		Owner: "orris-inc", Repo: "orris-client", AssetPrefix: "orris-client",
	}, log)
	c.nodeAgentReleaseService = services.NewGitHubReleaseService(services.GitHubRepoConfig{
		Owner: "orris-inc", Repo: "orrisp", AssetPrefix: "orrisp",
	}, log)

	// Initialize node use cases
	ucs.createNodeUC = nodeUsecases.NewCreateNodeUseCase(repos.nodeRepoImpl, repos.resourceGroupRepo, log)
	ucs.getNodeUC = nodeUsecases.NewGetNodeUseCase(repos.nodeRepoImpl, repos.resourceGroupRepo, c.nodeStatusQuerier, log)
	ucs.updateNodeUC = nodeUsecases.NewUpdateNodeUseCase(log, repos.nodeRepoImpl, repos.resourceGroupRepo)
	ucs.deleteNodeUC = nodeUsecases.NewDeleteNodeUseCase(repos.nodeRepoImpl, repos.forwardRuleRepo, log)
	ucs.listNodesUC = nodeUsecases.NewListNodesUseCase(repos.nodeRepoImpl, repos.resourceGroupRepo, repos.userRepo, c.nodeStatusQuerier, c.nodeAgentReleaseService, log)
	ucs.generateNodeTokenUC = nodeUsecases.NewGenerateNodeTokenUseCase(repos.nodeRepoImpl, log)
	ucs.generateNodeInstallScriptUC = nodeUsecases.NewGenerateNodeInstallScriptUseCase(repos.nodeRepoImpl, log)
	ucs.generateBatchInstallScriptUC = nodeUsecases.NewGenerateBatchInstallScriptUseCase(repos.nodeRepoImpl, log)

	// Initialize user node use cases
	ucs.createUserNodeUC = nodeUsecases.NewCreateUserNodeUseCase(repos.nodeRepoImpl, log)
	ucs.listUserNodesUC = nodeUsecases.NewListUserNodesUseCase(repos.nodeRepoImpl, log)
	ucs.getUserNodeUC = nodeUsecases.NewGetUserNodeUseCase(repos.nodeRepoImpl, log)
	ucs.updateUserNodeUC = nodeUsecases.NewUpdateUserNodeUseCase(repos.nodeRepoImpl, log)
	ucs.deleteUserNodeUC = nodeUsecases.NewDeleteUserNodeUseCase(repos.nodeRepoImpl, log)
	ucs.regenerateUserNodeTokenUC = nodeUsecases.NewRegenerateUserNodeTokenUseCase(repos.nodeRepoImpl, log)
	ucs.getUserNodeUsageUC = nodeUsecases.NewGetUserNodeUsageUseCase(repos.nodeRepoImpl, repos.subscriptionRepo, repos.subscriptionPlanRepo, log)
	ucs.getUserNodeInstallScriptUC = nodeUsecases.NewGetUserNodeInstallScriptUseCase(repos.nodeRepoImpl, log)
	ucs.getUserBatchInstallScriptUC = nodeUsecases.NewGetUserBatchInstallScriptUseCase(repos.nodeRepoImpl, log)

	// Initialize node authentication middleware
	ucs.validateNodeTokenUC = nodeUsecases.NewValidateNodeTokenUseCase(c.nodeRepoAdapter, log)
	c.nodeTokenMiddleware = middleware.NewNodeTokenMiddleware(ucs.validateNodeTokenUC, log)
	c.nodeOwnerMiddleware = middleware.NewNodeOwnerMiddleware(repos.nodeRepoImpl)
	c.nodeQuotaMiddleware = middleware.NewNodeQuotaMiddleware(repos.nodeRepoImpl, repos.subscriptionRepo, repos.subscriptionPlanRepo)

	// Initialize handlers
	apiBaseURL := cfg.Server.GetBaseURL()
	hdlrs.nodeHandler = handlers.NewNodeHandler(
		ucs.createNodeUC, ucs.getNodeUC, ucs.updateNodeUC, ucs.deleteNodeUC, ucs.listNodesUC,
		ucs.generateNodeTokenUC, ucs.generateNodeInstallScriptUC, ucs.generateBatchInstallScriptUC, apiBaseURL,
		log,
	)
	// Note: nodeSubscriptionHandler is created later after settingProvider is initialized
	hdlrs.userNodeHandler = nodeHandlers.NewUserNodeHandler(
		ucs.createUserNodeUC, ucs.listUserNodesUC, ucs.getUserNodeUC,
		ucs.updateUserNodeUC, ucs.deleteUserNodeUC, ucs.regenerateUserNodeTokenUC,
		ucs.getUserNodeUsageUC, ucs.getUserNodeInstallScriptUC, ucs.getUserBatchInstallScriptUC,
		apiBaseURL, log,
	)

	hdlrs.ticketHandler = ticketHandlers.NewTicketHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil, log)

	// Initialize notification components
	announcementRepoAdapter := adapters.NewAnnouncementRepositoryAdapter(repos.announcementRepo)
	notificationRepoAdapter := adapters.NewNotificationRepositoryAdapter(repos.notificationRepo)
	templateRepoAdapter := adapters.NewTemplateRepositoryAdapter(repos.templateRepo)
	userAnnouncementReadRepoAdapter := adapters.NewUserAnnouncementReadRepositoryAdapter(repos.userAnnouncementReadRepo)
	markdownService := markdown.NewMarkdownService()
	announcementFactory := adapters.NewAnnouncementFactoryAdapter()
	templateFactory := adapters.NewTemplateFactoryAdapter()

	notificationServiceDDD := notificationApp.NewServiceDDD(
		announcementRepoAdapter, notificationRepoAdapter, templateRepoAdapter,
		userAnnouncementReadRepoAdapter, announcementFactory, templateFactory,
		markdownService, log,
	)
	hdlrs.notificationHandler = handlers.NewNotificationHandler(notificationServiceDDD, repos.userRepo, log)

	// Initialize subscription template loader (needed by generateSubscriptionUC later)
	c.templateLoader = template.NewSubscriptionTemplateLoader(cfg.Subscription.TemplatesPath, log)
	if err := c.templateLoader.Load(); err != nil {
		log.Warnw("failed to load subscription templates, using defaults", "error", err)
	}
}

// ============================================================
// Section 4: Settings & Auth - OAuth, Email, Passkey
// ============================================================

// initSettingsAndAuth initializes setting service, OAuth/email managers with
// hot-reload support, and auth-related use cases and handlers.
// Corresponds to Section 4 of the original NewRouter().
func (c *Container) initSettingsAndAuth() {
	cfg := c.cfg
	log := c.log
	db := c.db
	repos := c.repos
	ucs := c.ucs
	hdlrs := c.hdlrs

	// Initialize System Setting components
	apiBaseURL := cfg.Server.GetBaseURL()
	settingProviderCfg := settingUsecases.SettingProviderConfig{
		TelegramConfig:      cfg.Telegram,
		GoogleOAuthConfig:   cfg.OAuth.Google,
		GitHubOAuthConfig:   cfg.OAuth.GitHub,
		EmailConfig:         cfg.Email,
		APIBaseURL:          apiBaseURL,
		SubscriptionBaseURL: cfg.Subscription.BaseURL,
		FrontendURL:         cfg.Server.FrontendCallbackURL,
		Timezone:            cfg.Server.Timezone,
	}
	c.settingServiceDDD = settingApp.NewServiceDDD(repos.settingRepo, settingProviderCfg, nil, log)
	hdlrs.settingHandler = adminHandlers.NewSettingHandler(c.settingServiceDDD, log)
	settingProvider := c.settingServiceDDD.GetSettingProvider()

	// Initialize subscription setting provider adapter for admin-configurable settings
	subscriptionSettingAdapter := adapters.NewSubscriptionSettingProviderAdapter(settingProvider)

	// Initialize generateSubscriptionUC (uses settingProvider)
	ucs.generateSubscriptionUC = nodeUsecases.NewGenerateSubscriptionUseCase(
		c.nodeRepoAdapter, c.tokenValidator, c.templateLoader,
		repos.subscriptionPlanRepo, repos.subscriptionUsageStatsRepo, c.hourlyTrafficCache,
		subscriptionSettingAdapter, log,
	)
	hdlrs.nodeSubscriptionHandler = handlers.NewNodeSubscriptionHandler(ucs.generateSubscriptionUC, log)

	// Create setting provider adapter to break reverse dependency from infrastructure to application
	c.settingProviderAdapt = &settingProviderAdapter{provider: settingProvider}

	// Initialize OAuthServiceManager for hot-reload support
	c.oauthManager = auth.NewOAuthServiceManager(c.settingProviderAdapt, log)
	if err := c.oauthManager.Initialize(context.Background()); err != nil {
		log.Warnw("oauth service manager initial config incomplete, will reinitialize on setting change", "error", err)
	}
	c.settingServiceDDD.Subscribe(c.oauthManager)

	// Initialize EmailServiceManager for hot-reload support
	c.emailManager = email.NewEmailServiceManager(c.settingProviderAdapt, log)
	if err := c.emailManager.Initialize(context.Background()); err != nil {
		log.Warnw("email service manager initial config incomplete, will reinitialize on setting change", "error", err)
	}
	c.settingServiceDDD.Subscribe(c.emailManager)

	// Create DynamicEmailService for use cases
	dynamicEmailSvc := email.NewDynamicEmailService(c.emailManager, log)

	// Inject EmailTester to break circular dependency
	c.settingServiceDDD.SetEmailTester(dynamicEmailSvc)

	// Create dynamic OAuth clients that fetch current client from manager
	dynamicGoogleClient := &dynamicOAuthClientAdapter{manager: c.oauthManager, provider: "google"}
	dynamicGitHubClient := &dynamicOAuthClientAdapter{manager: c.oauthManager, provider: "github"}

	// Initialize OAuth state store
	stateStore := cache.NewRedisStateStore(c.redis, "oauth:state:", 10*time.Minute)
	authHelper := helpers.NewAuthHelper(repos.userRepo, repos.sessionRepo, log)

	// Initialize password hasher (one instance for all auth use cases)
	hasher := auth.NewBcryptPasswordHasher(cfg.Auth.Password.BcryptCost)

	// Initialize auth-related use cases with dynamic services for hot-reload support
	ucs.registerUC = usecases.NewRegisterWithPasswordUseCase(repos.userRepo, hasher, dynamicEmailSvc, authHelper, c.settingServiceDDD, log)
	ucs.loginUC = usecases.NewLoginWithPasswordUseCase(repos.userRepo, repos.sessionRepo, hasher, c.jwtService, authHelper, c.settingServiceDDD, cfg.Auth.Session, log)
	ucs.verifyEmailUC = usecases.NewVerifyEmailUseCase(repos.userRepo, log)
	ucs.requestResetUC = usecases.NewRequestPasswordResetUseCase(repos.userRepo, dynamicEmailSvc, c.redis, log)
	ucs.resetPasswordUC = usecases.NewResetPasswordUseCase(repos.userRepo, repos.sessionRepo, hasher, dynamicEmailSvc, c.settingServiceDDD, log)
	ucs.adminResetPasswordUC = usecases.NewAdminResetPasswordUseCase(repos.userRepo, repos.sessionRepo, hasher, dynamicEmailSvc, c.settingServiceDDD, log)
	ucs.initiateOAuthUC = usecases.NewInitiateOAuthLoginUseCase(dynamicGoogleClient, dynamicGitHubClient, log, stateStore)
	ucs.handleOAuthUC = usecases.NewHandleOAuthCallbackUseCase(
		repos.userRepo, repos.oauthRepo, repos.sessionRepo,
		dynamicGoogleClient, dynamicGitHubClient, c.jwtService,
		ucs.initiateOAuthUC, authHelper, cfg.Auth.Session, log,
	)
	ucs.refreshTokenUC = usecases.NewRefreshTokenUseCase(repos.userRepo, repos.sessionRepo, c.jwtService, authHelper, cfg.Auth.Session, log)
	ucs.logoutUC = usecases.NewLogoutUseCase(repos.sessionRepo, log)

	hdlrs.authHandler = handlers.NewAuthHandler(
		ucs.registerUC, ucs.loginUC, ucs.verifyEmailUC, ucs.requestResetUC, ucs.resetPasswordUC,
		ucs.initiateOAuthUC, ucs.handleOAuthUC, ucs.refreshTokenUC, ucs.logoutUC, repos.userRepo, log,
		cfg.Auth.Cookie, cfg.Auth.JWT, cfg.Auth.Session,
		cfg.Server.FrontendCallbackURL, cfg.Server.AllowedOrigins,
		c.emailManager,
	)

	// Initialize Passkey (WebAuthn) components if configured
	if cfg.WebAuthn.IsConfigured() {
		webAuthnService, err := auth.NewWebAuthnService(cfg.WebAuthn)
		if err != nil {
			log.Warnw("failed to initialize WebAuthn service, passkey authentication disabled", "error", err)
		} else {
			passkeyRepo := repository.NewPasskeyCredentialRepository(db, log)
			passkeyChallengeStore := cache.NewPasskeyChallengeStore(c.redis)
			passkeySignupSessionStore := cache.NewPasskeySignupSessionStore(c.redis)

			startPasskeyRegistrationUC := usecases.NewStartPasskeyRegistrationUseCase(repos.userRepo, passkeyRepo, webAuthnService, passkeyChallengeStore, log)
			finishPasskeyRegistrationUC := usecases.NewFinishPasskeyRegistrationUseCase(repos.userRepo, passkeyRepo, webAuthnService, passkeyChallengeStore, log)
			startPasskeyAuthenticationUC := usecases.NewStartPasskeyAuthenticationUseCase(repos.userRepo, passkeyRepo, webAuthnService, passkeyChallengeStore, log)
			finishPasskeyAuthenticationUC := usecases.NewFinishPasskeyAuthenticationUseCase(repos.userRepo, passkeyRepo, repos.sessionRepo, webAuthnService, passkeyChallengeStore, c.jwtService, authHelper, cfg.Auth.Session, log)
			startPasskeySignupUC := usecases.NewStartPasskeySignupUseCase(repos.userRepo, webAuthnService, passkeyChallengeStore, passkeySignupSessionStore, log)
			finishPasskeySignupUC := usecases.NewFinishPasskeySignupUseCase(repos.userRepo, passkeyRepo, repos.sessionRepo, webAuthnService, passkeyChallengeStore, passkeySignupSessionStore, c.jwtService, authHelper, cfg.Auth.Session, log)
			listUserPasskeysUC := usecases.NewListUserPasskeysUseCase(passkeyRepo, log)
			deletePasskeyUC := usecases.NewDeletePasskeyUseCase(passkeyRepo, log)

			hdlrs.passkeyHandler = handlers.NewPasskeyHandler(
				startPasskeyRegistrationUC, finishPasskeyRegistrationUC,
				startPasskeyAuthenticationUC, finishPasskeyAuthenticationUC,
				startPasskeySignupUC, finishPasskeySignupUC,
				listUserPasskeysUC, deletePasskeyUC,
				log, cfg.Auth.Cookie, cfg.Auth.JWT, cfg.Auth.Session,
			)
			log.Infow("WebAuthn passkey authentication enabled")
		}
	}

	hdlrs.userHandler = handlers.NewUserHandler(c.userService, ucs.adminResetPasswordUC, log)

	// Create profile handler
	hdlrs.profileHandler = handlers.NewProfileHandler(c.userService, log)

	// Create dashboard handler
	ucs.getDashboardUC = usecases.NewGetDashboardUseCase(
		repos.subscriptionRepo, repos.subscriptionUsageStatsRepo,
		c.hourlyTrafficCache, repos.subscriptionPlanRepo, log,
	)
	hdlrs.dashboardHandler = handlers.NewDashboardHandler(ucs.getDashboardUC, log)

	// Payment handler
	var gateway paymentGateway.PaymentGateway = nil // Temporary placeholder until real implementation
	paymentConfig := paymentUsecases.PaymentConfig{
		NotifyURL: cfg.Server.GetBaseURL() + "/payments/callback",
	}
	paymentTxMgr := shareddb.NewTransactionManager(db)
	ucs.createPaymentUC = paymentUsecases.NewCreatePaymentUseCase(
		repos.paymentRepo, repos.subscriptionRepo, repos.subscriptionPlanRepo,
		repos.planPricingRepo, gateway, paymentTxMgr, log, paymentConfig,
	)
	ucs.handleCallbackUC = paymentUsecases.NewHandlePaymentCallbackUseCase(
		repos.paymentRepo, ucs.activateSubscriptionUC, gateway, log,
	)
	hdlrs.paymentHandler = handlers.NewPaymentHandler(ucs.createPaymentUC, ucs.handleCallbackUC, repos.subscriptionRepo, log)

	// Initialize USDT Service Manager for crypto payment support
	c.usdtServiceManager = infraPayment.NewUSDTServiceManager(db, repos.paymentRepo, repos.subscriptionRepo, log)
	usdtConfig := settingProvider.GetUSDTConfig(context.Background())
	if err := c.usdtServiceManager.Initialize(context.Background(), infraPayment.USDTConfig{
		Enabled:               usdtConfig.Enabled,
		POLReceivingAddresses: usdtConfig.POLReceivingAddresses,
		TRCReceivingAddresses: usdtConfig.TRCReceivingAddresses,
		PolygonScanAPIKey:     usdtConfig.PolygonScanAPIKey,
		TronGridAPIKey:        usdtConfig.TronGridAPIKey,
		PaymentTTLMinutes:     usdtConfig.PaymentTTLMinutes,
		POLConfirmations:      usdtConfig.POLConfirmations,
		TRCConfirmations:      usdtConfig.TRCConfirmations,
	}); err != nil {
		log.Warnw("failed to initialize USDT services, will retry on setting change", "error", err)
	}
	c.settingServiceDDD.Subscribe(c.usdtServiceManager)
	ucs.createPaymentUC.SetUSDTGatewayProvider(c.usdtServiceManager)
}

// ============================================================
// Section 5: Telegram - Bot, Notifications, Admin Alerts
// ============================================================

// initTelegram initializes telegram bot, service, admin notification components.
// Corresponds to Section 5 of the original NewRouter().
func (c *Container) initTelegram() {
	cfg := c.cfg
	log := c.log
	repos := c.repos
	hdlrs := c.hdlrs

	// Initialize Telegram base components
	telegramVerifyStore := cache.NewTelegramVerifyStore(c.redis)

	// Initialize Telegram ServiceDDD (initially without BotService)
	c.telegramServiceDDD = telegramApp.NewServiceDDD(
		repos.telegramBindingRepo, repos.subscriptionRepo,
		repos.subscriptionUsageStatsRepo, c.hourlyTrafficCache,
		repos.subscriptionPlanRepo, telegramVerifyStore,
		nil, // BotService will be managed by BotServiceManager
		log,
	)

	// Declare admin service variable early for closure capture
	// (c.adminNotificationServiceDDD will be set later in this function)

	// Create UpdateHandler for polling mode
	c.serviceAdapter = telegramInfra.NewServiceAdapter(
		c.telegramServiceDDD,
		func(ctx context.Context, telegramUserID int64, telegramUsername, verifyCode string) error {
			_, err := c.telegramServiceDDD.BindFromWebhook(ctx, telegramUserID, telegramUsername, verifyCode)
			return err
		},
		func(ctx context.Context, telegramUserID int64) (bool, error) {
			status, err := c.telegramServiceDDD.GetBindingStatusByTelegramID(ctx, telegramUserID)
			if err != nil {
				return false, err
			}
			return status.IsBound, nil
		},
		func(ctx context.Context, telegramUserID int64, language string) error {
			return c.telegramServiceDDD.UpdateBindingLanguage(ctx, telegramUserID, language)
		},
		func(ctx context.Context, telegramUserID int64, language string) error {
			if c.adminNotificationServiceDDD != nil {
				return c.adminNotificationServiceDDD.UpdateAdminBindingLanguage(ctx, telegramUserID, language)
			}
			return nil
		},
	)
	updateHandler := telegramInfra.NewPollingUpdateHandler(c.serviceAdapter, log)

	// Create BotServiceManager with hot-reload support
	c.telegramBotManager = telegramInfra.NewBotServiceManager(c.settingProviderAdapt, updateHandler, log)

	// Inject polling offset store for offset persistence across restarts
	pollingOffsetStore := cache.NewPollingOffsetStore(c.redis)
	c.telegramBotManager.SetOffsetStore(pollingOffsetStore)

	// Inject BotServiceManager into ServiceAdapter (break circular dependency)
	c.serviceAdapter.SetBotServiceGetter(c.telegramBotManager)

	// Create DynamicBotService and inject into telegramServiceDDD
	c.dynamicBotService = telegramInfra.NewDynamicBotService(c.telegramBotManager, log)
	c.telegramServiceDDD.SetBotService(c.dynamicBotService)

	// Subscribe BotServiceManager to setting changes for hot-reload
	c.settingServiceDDD.Subscribe(c.telegramBotManager)

	// Inject telegramTester to break circular dependency
	c.settingServiceDDD.SetTelegramTester(c.telegramBotManager)

	// Initialize Telegram Handler
	initialWebhookSecret := cfg.Telegram.WebhookSecret
	hdlrs.telegramHandler = telegramHandlers.NewHandler(c.telegramServiceDDD, log, initialWebhookSecret)

	// Inject SettingProvider for hot-reload support of webhook secret from database
	settingProvider := c.settingServiceDDD.GetSettingProvider()
	hdlrs.telegramHandler.SetWebhookSecretProvider(settingProvider)

	log.Infow("telegram components initialized with hot-reload support")

	// Initialize admin notification components
	adminVerifyStore := cache.NewAdminTelegramVerifyStore(c.redis)
	userRoleChecker := adapters.NewUserRoleCheckerAdapter(repos.userRepo)

	c.adminNotificationServiceDDD = telegramAdminApp.NewServiceDDD(
		repos.adminBindingRepo, adminVerifyStore,
		c.dynamicBotService, c.telegramBotManager,
		userRoleChecker, log,
	)

	hdlrs.adminTelegramHandler = adminHandlers.NewAdminTelegramHandler(c.adminNotificationServiceDDD, log)

	// Inject admin service into telegram handler for /adminbind command support (webhook mode)
	hdlrs.telegramHandler.SetAdminService(c.adminNotificationServiceDDD)

	// Inject admin binder into service adapter for /adminbind command support (polling mode)
	c.serviceAdapter.SetAdminBinder(c.adminNotificationServiceDDD)

	log.Infow("admin notification components initialized")
}

// ============================================================
// Section 6: Forward - Agents, Rules, Traffic, AgentHub
// ============================================================

// initForward initializes forward agent/rule components, agent hub,
// traffic caches, and related services.
// Corresponds to Section 6 of the original NewRouter().
func (c *Container) initForward() {
	cfg := c.cfg
	log := c.log
	db := c.db
	repos := c.repos
	ucs := c.ucs
	hdlrs := c.hdlrs

	// Agent API handler use cases
	ucs.getNodeConfigUC = nodeUsecases.NewGetNodeConfigUseCase(repos.nodeRepoImpl, log)
	ucs.getNodeSubscriptionsUC = nodeUsecases.NewGetNodeSubscriptionsUseCase(repos.subscriptionRepo, repos.nodeRepoImpl, log)

	// Initialize subscription traffic cache and buffer for RESTful agent traffic reporting
	c.subscriptionTrafficCache = cache.NewRedisSubscriptionTrafficCache(
		c.redis, c.hourlyTrafficCache, repos.subscriptionUsageRepo, log,
	)
	c.subscriptionTrafficBuffer = nodeServices.NewSubscriptionTrafficBuffer(c.subscriptionTrafficCache, log)
	c.subscriptionTrafficBuffer.Start()

	// Initialize subscription quota cache for node traffic limit checking
	c.subscriptionQuotaCache = cache.NewRedisSubscriptionQuotaCache(c.redis, log)

	// Initialize quota cache sync service
	c.quotaCacheSyncService = subscriptionServices.NewQuotaCacheSyncService(
		repos.subscriptionRepo, repos.subscriptionPlanRepo, c.subscriptionQuotaCache, log,
	)

	// Initialize node traffic limit enforcement service
	c.nodeTrafficLimitEnforcementSvc = nodeServices.NewNodeTrafficLimitEnforcementService(
		repos.subscriptionRepo, repos.subscriptionUsageStatsRepo,
		c.hourlyTrafficCache, repos.subscriptionPlanRepo, c.subscriptionQuotaCache, log,
	)

	// Initialize adapters for node hub handler traffic limit checking
	c.nodeQuotaCacheAdapter = adapters.NewNodeSubscriptionQuotaCacheAdapter(c.subscriptionQuotaCache, log)
	c.nodeQuotaLoaderAdapter = adapters.NewNodeSubscriptionQuotaLoaderAdapter(
		repos.subscriptionRepo, repos.subscriptionPlanRepo, c.subscriptionQuotaCache, log,
	)
	c.nodeUsageReaderAdapter = adapters.NewNodeSubscriptionUsageReaderAdapter(
		c.hourlyTrafficCache, repos.subscriptionUsageStatsRepo, log,
	)

	// Initialize agent report use cases with adapters
	subscriptionUsageRecorder := adapters.NewSubscriptionUsageRecorderAdapter(c.subscriptionTrafficBuffer, log)
	c.systemStatusUpdater = adapters.NewNodeSystemStatusUpdaterAdapter(c.redis, log)
	onlineSubscriptionTracker := adapters.NewOnlineSubscriptionTrackerAdapter(log)
	c.subscriptionIDResolver = adapters.NewSubscriptionIDResolverAdapter(repos.subscriptionRepo, log)
	ucs.reportSubscriptionUsageUC = nodeUsecases.NewReportSubscriptionUsageUseCase(subscriptionUsageRecorder, c.subscriptionIDResolver, log)
	ucs.reportNodeStatusUC = nodeUsecases.NewReportNodeStatusUseCase(c.systemStatusUpdater, repos.nodeRepoImpl, repos.nodeRepoImpl, log)
	ucs.reportOnlineSubscriptionsUC = nodeUsecases.NewReportOnlineSubscriptionsUseCase(onlineSubscriptionTracker, c.subscriptionIDResolver, log)

	// Initialize RESTful Agent Handler
	hdlrs.agentHandler = nodeHandlers.NewAgentHandler(
		ucs.getNodeConfigUC, ucs.getNodeSubscriptionsUC,
		ucs.reportSubscriptionUsageUC, ucs.reportNodeStatusUC,
		ucs.reportOnlineSubscriptionsUC, log,
	)

	// Initialize admin notification processor and scheduler for offline alerts
	c.alertStateManager = cache.NewAlertStateManager(c.redis)
	adminNotificationProcessor := telegramAdminApp.NewAdminNotificationProcessor(
		repos.adminBindingRepo, repos.userRepo, repos.subscriptionRepo,
		repos.subscriptionUsageStatsRepo, c.hourlyTrafficCache,
		repos.nodeRepoImpl, repos.forwardAgentRepo,
		c.alertStateManager, &botServiceProviderAdapter{c.telegramBotManager}, log,
	)
	if err := c.schedulerManager.RegisterAdminNotificationJobs(adminNotificationProcessor); err != nil {
		log.Warnw("failed to register admin notification jobs", "error", err)
	}

	// Create alert state clearer adapter and inject into delete use cases
	alertStateClearer := adapters.NewAlertStateClearerAdapter(c.alertStateManager)
	ucs.deleteNodeUC.WithAlertStateClearer(alertStateClearer)

	// Initialize mute notification service
	ucs.muteNotificationUC = telegramAdminUsecases.NewMuteNotificationUseCase(repos.forwardAgentRepo, repos.nodeRepoImpl, log)
	hdlrs.telegramHandler.SetMuteService(ucs.muteNotificationUC)
	hdlrs.telegramHandler.SetCallbackAnswerer(c.dynamicBotService)

	// Inject mute service and callback answerer into service adapter for polling mode
	c.serviceAdapter.SetMuteService(ucs.muteNotificationUC)
	c.serviceAdapter.SetCallbackAnswerer(c.dynamicBotService)

	// Initialize resource group membership use cases
	ucs.manageNodesUC = resourceUsecases.NewManageResourceGroupNodesUseCase(repos.resourceGroupRepo, repos.nodeRepoImpl, repos.subscriptionPlanRepo, log)
	ucs.manageAgentsUC = resourceUsecases.NewManageResourceGroupForwardAgentsUseCase(repos.resourceGroupRepo, repos.forwardAgentRepo, repos.subscriptionPlanRepo, log)
	ucs.manageRulesUC = resourceUsecases.NewManageResourceGroupForwardRulesUseCase(repos.resourceGroupRepo, repos.forwardRuleRepo, repos.subscriptionPlanRepo, log)

	// Initialize admin resource group handler
	hdlrs.adminResourceGroupHandler = adminResourceGroupHandlers.NewHandler(
		ucs.createResourceGroupUC, ucs.getResourceGroupUC, ucs.listResourceGroupsUC,
		ucs.updateResourceGroupUC, ucs.deleteResourceGroupUC, ucs.updateResourceGroupStatusUC,
		ucs.manageNodesUC, ucs.manageAgentsUC, ucs.manageRulesUC,
		repos.subscriptionPlanRepo, log,
	)

	// Initialize admin traffic stats use cases
	ucs.getTrafficOverviewUC = adminUsecases.NewGetTrafficOverviewUseCase(
		repos.subscriptionUsageStatsRepo, c.hourlyTrafficCache, repos.subscriptionRepo,
		repos.userRepo, repos.nodeRepoImpl, repos.forwardRuleRepo, log,
	)
	ucs.getUserTrafficStatsUC = adminUsecases.NewGetUserTrafficStatsUseCase(
		repos.subscriptionUsageStatsRepo, c.hourlyTrafficCache, repos.subscriptionRepo, repos.userRepo, log,
	)
	ucs.getSubscriptionTrafficStatsUC = adminUsecases.NewGetSubscriptionTrafficStatsUseCase(
		repos.subscriptionUsageStatsRepo, c.hourlyTrafficCache, repos.subscriptionRepo,
		repos.userRepo, repos.subscriptionPlanRepo, log,
	)
	ucs.getAdminNodeTrafficStatsUC = adminUsecases.NewGetAdminNodeTrafficStatsUseCase(
		repos.subscriptionUsageStatsRepo, c.hourlyTrafficCache, repos.nodeRepoImpl, log,
	)
	ucs.getTrafficRankingUC = adminUsecases.NewGetTrafficRankingUseCase(
		repos.subscriptionUsageStatsRepo, c.hourlyTrafficCache, repos.subscriptionRepo, repos.userRepo, log,
	)
	ucs.getTrafficTrendUC = adminUsecases.NewGetTrafficTrendUseCase(
		repos.subscriptionUsageStatsRepo, c.hourlyTrafficCache, log,
	)
	hdlrs.adminTrafficStatsHandler = adminHandlers.NewTrafficStatsHandler(
		ucs.getTrafficOverviewUC, ucs.getUserTrafficStatsUC,
		ucs.getSubscriptionTrafficStatsUC, ucs.getAdminNodeTrafficStatsUC,
		ucs.getTrafficRankingUC, ucs.getTrafficTrendUC, log,
	)

	// Initialize forward agent components
	forwardAgentStatusAdapter := adapters.NewForwardAgentStatusAdapter(c.redis, log)
	ruleSyncStatusAdapter := adapters.NewRuleSyncStatusAdapter(c.redis, log)

	ucs.createForwardAgentUC = forwardUsecases.NewCreateForwardAgentUseCase(repos.forwardAgentRepo, repos.resourceGroupRepo, c.agentTokenSvc, log)
	ucs.getForwardAgentUC = forwardUsecases.NewGetForwardAgentUseCase(repos.forwardAgentRepo, forwardAgentStatusAdapter, log)
	ucs.deleteForwardAgentUC = forwardUsecases.NewDeleteForwardAgentUseCase(repos.forwardAgentRepo, repos.forwardRuleRepo, log)
	ucs.deleteForwardAgentUC.WithAlertStateClearer(alertStateClearer)
	ucs.listForwardAgentsUC = forwardUsecases.NewListForwardAgentsUseCase(repos.forwardAgentRepo, repos.resourceGroupRepo, forwardAgentStatusAdapter, c.forwardAgentReleaseService, log)
	ucs.enableForwardAgentUC = forwardUsecases.NewEnableForwardAgentUseCase(repos.forwardAgentRepo, log)
	ucs.disableForwardAgentUC = forwardUsecases.NewDisableForwardAgentUseCase(repos.forwardAgentRepo, log)
	ucs.regenerateForwardAgentTokenUC = forwardUsecases.NewRegenerateForwardAgentTokenUseCase(repos.forwardAgentRepo, c.agentTokenSvc, log)
	ucs.validateForwardAgentTokenUC = forwardUsecases.NewValidateForwardAgentTokenUseCase(repos.forwardAgentRepo, log)

	// Initialize agent last seen updater and agent info updater
	agentLastSeenUpdater := adapters.NewAgentLastSeenUpdaterAdapter(repos.forwardAgentRepo)
	agentInfoUpdater := adapters.NewAgentInfoUpdaterAdapter(repos.forwardAgentRepo)
	ucs.getAgentStatusUC = forwardUsecases.NewGetAgentStatusUseCase(repos.forwardAgentRepo, forwardAgentStatusAdapter, log)
	ucs.getRuleOverallStatusUC = forwardUsecases.NewGetRuleOverallStatusUseCase(repos.forwardRuleRepo, repos.forwardAgentRepo, ruleSyncStatusAdapter, log)
	ucs.getForwardAgentTokenUC = forwardUsecases.NewGetForwardAgentTokenUseCase(repos.forwardAgentRepo, log)
	ucs.generateInstallScriptUC = forwardUsecases.NewGenerateInstallScriptUseCase(repos.forwardAgentRepo, log)

	serverBaseURL := cfg.Server.GetBaseURL()

	ucs.reportAgentStatusUC = forwardUsecases.NewReportAgentStatusUseCase(
		repos.forwardAgentRepo, forwardAgentStatusAdapter, forwardAgentStatusAdapter,
		agentLastSeenUpdater, agentInfoUpdater, log,
	)
	ucs.reportRuleSyncStatusUC = forwardUsecases.NewReportRuleSyncStatusUseCase(
		repos.forwardAgentRepo, ruleSyncStatusAdapter, repos.forwardRuleRepo, log,
	)

	// Initialize forward traffic recorder adapter
	forwardTrafficRecorder := adapters.NewForwardTrafficRecorderAdapter(c.hourlyTrafficCache, log)

	// Initialize forward agent API handler
	hdlrs.forwardAgentAPIHandler = forwardAgentAPIHandlers.NewHandler(
		repos.forwardRuleRepo, repos.forwardAgentRepo, repos.nodeRepoImpl,
		ucs.reportAgentStatusUC, ucs.reportRuleSyncStatusUC, forwardAgentStatusAdapter,
		cfg.Forward.TokenSigningSecret, forwardTrafficRecorder, log,
	)

	// Initialize forward agent token middleware
	c.forwardAgentTokenMiddleware = middleware.NewForwardAgentTokenMiddleware(ucs.validateForwardAgentTokenUC, log)

	// Initialize agent hub for forward agent WebSocket connections
	c.agentHub = services.NewAgentHub(log, &services.AgentHubConfig{
		NodeStatusTimeoutMs: 5000,
	})

	// Initialize Hub Event Bus for cross-instance command relay
	c.hubEventBus = pubsub.NewRedisHubEventBus(c.redis, log)
	c.agentHub.SetEventBus(c.hubEventBus)

	// Register forward status handler
	c.forwardStatusHandler = adapters.NewForwardStatusHandler(ucs.reportAgentStatusUC, log)
	c.agentHub.RegisterStatusHandler(c.forwardStatusHandler)

	// Initialize and register probe service
	probeService := forwardServices.NewProbeService(
		repos.forwardRuleRepo, repos.forwardAgentRepo, repos.nodeRepoImpl,
		forwardAgentStatusAdapter, c.agentHub, cfg.Forward.TokenSigningSecret, log,
	)
	c.agentHub.RegisterMessageHandler(probeService)

	// Initialize and register config sync service
	c.configSyncService = forwardServices.NewConfigSyncService(
		repos.forwardRuleRepo, repos.forwardAgentRepo, repos.nodeRepoImpl,
		forwardAgentStatusAdapter, cfg.Forward.TokenSigningSecret, c.agentHub, log,
	)
	c.agentHub.RegisterMessageHandler(c.configSyncService)

	// Register rule sync status handler for WebSocket-based status reporting
	c.agentHub.RegisterMessageHandler(ucs.reportRuleSyncStatusUC)

	// Initialize forward traffic cache
	c.forwardTrafficCache = cache.NewRedisForwardTrafficCache(c.redis, repos.forwardRuleRepo, log)

	// Initialize rule traffic buffer
	c.ruleTrafficBuffer = forwardServices.NewRuleTrafficBuffer(c.forwardTrafficCache, log)
	c.ruleTrafficBuffer.Start()

	// Initialize and register traffic message handler
	trafficMessageHandler := services.NewTrafficMessageHandler(
		c.ruleTrafficBuffer, repos.forwardRuleRepo, forwardTrafficRecorder, log,
	)
	c.agentHub.RegisterMessageHandler(trafficMessageHandler)

	// Initialize and register tunnel health handler
	tunnelHealthHandler := forwardServices.NewTunnelHealthHandler(log)
	c.agentHub.RegisterMessageHandler(tunnelHealthHandler)

	// Create done channel for rule traffic flush scheduler
	c.ruleTrafficFlushDone = make(chan struct{})

	// Start rule traffic flush scheduler (Redis -> MySQL)
	goroutine.SafeGo(log, "rule-traffic-flush-scheduler", func() {
		ticker := time.NewTicker(2 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				ctx := context.Background()
				if err := c.forwardTrafficCache.FlushToDatabase(ctx); err != nil {
					log.Errorw("failed to flush rule traffic to database", "error", err)
				}
			case <-c.ruleTrafficFlushDone:
				return
			}
		}
	})

	// Set port change notifier for exit agent port change detection
	ucs.reportAgentStatusUC.SetPortChangeNotifier(c.configSyncService)

	// Set node address change notifier for node IP change detection
	ucs.reportNodeStatusUC.SetAddressChangeNotifier(c.configSyncService)

	// Set node address change notifier for node update use case
	ucs.updateNodeUC.SetAddressChangeNotifier(c.configSyncService)

	// Now initialize updateForwardAgentUC with configSyncService
	ucs.updateForwardAgentUC = forwardUsecases.NewUpdateForwardAgentUseCase(
		repos.forwardAgentRepo, repos.resourceGroupRepo, c.configSyncService, c.configSyncService, log,
	)

	// Now initialize forwardAgentHandler after updateForwardAgentUC is available
	hdlrs.forwardAgentHandler = forwardAgentCrudHandlers.NewHandler(
		ucs.createForwardAgentUC, ucs.getForwardAgentUC, ucs.listForwardAgentsUC,
		ucs.updateForwardAgentUC, ucs.deleteForwardAgentUC,
		ucs.enableForwardAgentUC, ucs.disableForwardAgentUC,
		ucs.regenerateForwardAgentTokenUC, ucs.getForwardAgentTokenUC,
		ucs.getAgentStatusUC, ucs.getRuleOverallStatusUC,
		ucs.generateInstallScriptUC, serverBaseURL, log,
	)

	// Initialize version handlers
	hdlrs.forwardAgentVersionHandler = forwardAgentCrudHandlers.NewVersionHandler(
		repos.forwardAgentRepo, c.forwardAgentReleaseService, c.agentHub, log,
	)
	hdlrs.nodeVersionHandler = nodeHandlers.NewNodeVersionHandler(
		repos.nodeRepoImpl, c.nodeAgentReleaseService, c.agentHub, log,
	)

	// Now initialize forward rule use cases with configSyncService
	ucs.createForwardRuleUC = forwardUsecases.NewCreateForwardRuleUseCase(
		repos.forwardRuleRepo, repos.forwardAgentRepo, repos.nodeRepoImpl,
		repos.resourceGroupRepo, repos.subscriptionPlanRepo, c.configSyncService, log,
	)
	ucs.getForwardRuleUC = forwardUsecases.NewGetForwardRuleUseCase(
		repos.forwardRuleRepo, repos.forwardAgentRepo, repos.nodeRepoImpl, repos.resourceGroupRepo, log,
	)
	ucs.updateForwardRuleUC = forwardUsecases.NewUpdateForwardRuleUseCase(
		repos.forwardRuleRepo, repos.forwardAgentRepo, repos.nodeRepoImpl,
		repos.resourceGroupRepo, repos.subscriptionPlanRepo, repos.subscriptionRepo, c.configSyncService, log,
	)
	ucs.deleteForwardRuleUC = forwardUsecases.NewDeleteForwardRuleUseCase(repos.forwardRuleRepo, c.forwardTrafficCache, c.configSyncService, log)
	ucs.listForwardRulesUC = forwardUsecases.NewListForwardRulesUseCase(
		repos.forwardRuleRepo, repos.forwardAgentRepo, repos.nodeRepoImpl,
		repos.resourceGroupRepo, ruleSyncStatusAdapter, log,
	)
	ucs.enableForwardRuleUC = forwardUsecases.NewEnableForwardRuleUseCase(repos.forwardRuleRepo, c.configSyncService, log)
	ucs.disableForwardRuleUC = forwardUsecases.NewDisableForwardRuleUseCase(repos.forwardRuleRepo, c.configSyncService, log)
	ucs.resetForwardTrafficUC = forwardUsecases.NewResetForwardRuleTrafficUseCase(repos.forwardRuleRepo, log)

	txMgr := shareddb.NewTransactionManager(db)
	ucs.reorderForwardRulesUC = forwardUsecases.NewReorderForwardRulesUseCase(repos.forwardRuleRepo, txMgr, log)

	// Initialize user forward rule use cases
	ucs.createUserForwardRuleUC = forwardUsecases.NewCreateUserForwardRuleUseCase(
		repos.forwardRuleRepo, repos.forwardAgentRepo, repos.nodeRepoImpl, c.configSyncService, log,
	)
	ucs.listUserForwardRulesUC = forwardUsecases.NewListUserForwardRulesUseCase(
		repos.forwardRuleRepo, repos.forwardAgentRepo, repos.nodeRepoImpl, ruleSyncStatusAdapter, log,
	)
	ucs.getUserForwardUsageUC = forwardUsecases.NewGetUserForwardUsageUseCase(
		repos.forwardRuleRepo, repos.subscriptionRepo, repos.subscriptionPlanRepo,
		repos.subscriptionUsageRepo, repos.subscriptionUsageStatsRepo, c.hourlyTrafficCache, log,
	)

	// Initialize traffic limit enforcement service
	c.trafficLimitEnforcementSvc = forwardServices.NewTrafficLimitEnforcementService(
		repos.forwardRuleRepo, repos.subscriptionRepo, repos.subscriptionUsageRepo,
		repos.subscriptionUsageStatsRepo, c.hourlyTrafficCache, repos.subscriptionPlanRepo, log,
	)

	// Initialize list user forward agents use case
	ucs.listUserForwardAgentsUC = forwardUsecases.NewListUserForwardAgentsUseCase(
		repos.forwardAgentRepo, repos.subscriptionRepo, repos.subscriptionPlanRepo, repos.resourceGroupRepo, log,
	)

	// Initialize batch forward rule use case
	ucs.batchForwardRuleUC = forwardUsecases.NewBatchForwardRuleUseCase(
		repos.forwardRuleRepo, ucs.createForwardRuleUC, ucs.createUserForwardRuleUC,
		ucs.deleteForwardRuleUC, ucs.enableForwardRuleUC, ucs.disableForwardRuleUC,
		ucs.updateForwardRuleUC, txMgr, log,
	)

	// Initialize user forward rule handler
	hdlrs.userForwardRuleHandler = forwardUserHandlers.NewHandler(
		ucs.createUserForwardRuleUC, ucs.listUserForwardRulesUC, ucs.getUserForwardUsageUC,
		ucs.updateForwardRuleUC, ucs.deleteForwardRuleUC,
		ucs.enableForwardRuleUC, ucs.disableForwardRuleUC,
		ucs.getForwardRuleUC, ucs.listUserForwardAgentsUC,
		ucs.reorderForwardRulesUC, ucs.batchForwardRuleUC,
		log,
	)

	// Initialize subscription forward rule use cases
	ucs.createSubscriptionForwardRuleUC = forwardUsecases.NewCreateSubscriptionForwardRuleUseCase(
		repos.forwardRuleRepo, repos.forwardAgentRepo, repos.nodeRepoImpl, c.configSyncService, log,
	)
	ucs.listSubscriptionForwardRulesUC = forwardUsecases.NewListSubscriptionForwardRulesUseCase(
		repos.forwardRuleRepo, repos.forwardAgentRepo, repos.nodeRepoImpl,
		repos.subscriptionRepo, repos.resourceGroupRepo, ruleSyncStatusAdapter, log,
	)
	ucs.getSubscriptionForwardUsageUC = forwardUsecases.NewGetSubscriptionForwardUsageUseCase(
		repos.forwardRuleRepo, repos.subscriptionRepo, repos.subscriptionPlanRepo,
		repos.subscriptionUsageRepo, repos.subscriptionUsageStatsRepo, c.hourlyTrafficCache, log,
	)

	// Initialize subscription forward rule handler
	hdlrs.subscriptionForwardRuleHandler = forwardSubscriptionHandlers.NewHandler(
		ucs.createSubscriptionForwardRuleUC, ucs.listSubscriptionForwardRulesUC,
		ucs.getSubscriptionForwardUsageUC, ucs.updateForwardRuleUC, ucs.deleteForwardRuleUC,
		ucs.enableForwardRuleUC, ucs.disableForwardRuleUC, ucs.getForwardRuleUC,
		ucs.reorderForwardRulesUC, log,
	)

	// Initialize forward rule owner middleware
	c.forwardRuleOwnerMiddleware = middleware.NewForwardRuleOwnerMiddleware(repos.forwardRuleRepo, log)

	// Initialize forward rule handler (after probeService is available)
	hdlrs.forwardRuleHandler = forwardRuleHandlers.NewHandler(
		ucs.createForwardRuleUC, ucs.getForwardRuleUC, ucs.updateForwardRuleUC,
		ucs.deleteForwardRuleUC, ucs.listForwardRulesUC,
		ucs.enableForwardRuleUC, ucs.disableForwardRuleUC,
		ucs.resetForwardTrafficUC, ucs.reorderForwardRulesUC,
		ucs.batchForwardRuleUC, probeService, log,
	)

	// Initialize agent hub handler
	hdlrs.agentHubHandler = forwardAgentHubHandlers.NewHandler(c.agentHub, repos.forwardAgentRepo, log)

	// Initialize node status handler and register to agent hub
	c.nodeStatusHandler = adapters.NewNodeStatusHandler(c.systemStatusUpdater, repos.nodeRepoImpl, log)
	c.agentHub.RegisterNodeStatusHandler(c.nodeStatusHandler)

	// Initialize node config sync service
	c.nodeConfigSyncService = nodeServices.NewNodeConfigSyncService(repos.nodeRepoImpl, c.agentHub, log)

	// Initialize subscription sync service
	c.subscriptionSyncService = nodeServices.NewSubscriptionSyncService(repos.nodeRepoImpl, repos.subscriptionRepo, repos.resourceGroupRepo, c.agentHub, log)

	// Initialize Redis Pub/Sub event bus
	subscriptionEventBus := pubsub.NewRedisSubscriptionEventBus(c.redis, log)
	c.subscriptionSyncService.SetEventPublisher(subscriptionEventBus)

	// Set subscription syncer on resource group use cases
	ucs.manageNodesUC.SetNodeSubscriptionSyncer(c.subscriptionSyncService)
	ucs.manageRulesUC.SetNodeSubscriptionSyncer(c.subscriptionSyncService)
	ucs.deleteResourceGroupUC.SetNodeSubscriptionSyncer(c.subscriptionSyncService)
	ucs.updateResourceGroupStatusUC.SetNodeSubscriptionSyncer(c.subscriptionSyncService)

	// Set deactivation notifier on node traffic limit enforcement service
	c.nodeTrafficLimitEnforcementSvc.SetDeactivationNotifier(c.subscriptionSyncService)

	// Initialize subscription event handler
	subscriptionEventHandler := nodeServices.NewSubscriptionEventHandler(
		repos.subscriptionRepo, c.subscriptionSyncService, log,
	)
	subscriptionEventHandler.StartSubscriber(context.Background(), subscriptionEventBus)

	// Initialize admin hub for SSE connections
	c.adminHub = services.NewAdminHub(log, &services.AdminHubConfig{
		MaxConnsPerUser:  20,
		StatusThrottleMs: 1000,
		AgentBroadcastMs: 1000,
		NodeBroadcastMs:  1000,
	})
	c.adminHub.SetEventBus(c.hubEventBus)
}

// ============================================================
// Section 7: Callbacks & Notifiers - Event Handlers, Sync
// ============================================================

// initCallbacksAndNotifiers wires up online/offline callbacks, SSE broadcasting,
// and config/subscription change notifiers.
// Corresponds to Section 7 of the original NewRouter().
func (c *Container) initCallbacksAndNotifiers() {
	log := c.log
	repos := c.repos
	ucs := c.ucs

	// Create cancelable context for hub event bus subscribers
	hubCtx, hubCancel := context.WithCancel(context.Background())
	c.hubEventBusCancelMu.Lock()
	c.hubEventBusCancel = hubCancel
	c.hubEventBusCancelMu.Unlock()

	// Start Hub PubSub subscribers for cross-instance command relay
	goroutine.SafeGo(log, "hub-agent-cmd-subscriber", func() {
		if err := c.hubEventBus.SubscribeAgentCommands(hubCtx, func(agentID uint, cmd *dto.CommandData) {
			c.agentHub.HandleRemoteAgentCommand(agentID, cmd)
		}); err != nil {
			logSubscriberExit(log, "hub agent command subscriber", err)
		}
	})
	goroutine.SafeGo(log, "hub-node-cmd-subscriber", func() {
		if err := c.hubEventBus.SubscribeNodeCommands(hubCtx, func(nodeID uint, cmd *nodedto.NodeCommandData) {
			c.agentHub.HandleRemoteNodeCommand(nodeID, cmd)
		}); err != nil {
			logSubscriberExit(log, "hub node command subscriber", err)
		}
	})
	goroutine.SafeGo(log, "hub-status-subscriber", func() {
		if err := c.hubEventBus.SubscribeStatusEvents(hubCtx, func(event pubsub.HubStatusEvent) {
			switch event.Type {
			case pubsub.HubEventAgentOnline:
				c.adminHub.BroadcastForwardAgentOnline(event.AgentSID, event.AgentName)
			case pubsub.HubEventAgentOffline:
				c.adminHub.BroadcastForwardAgentOffline(event.AgentSID, event.AgentName)
			case pubsub.HubEventNodeOnline:
				c.adminHub.BroadcastNodeOnline(event.NodeSID, event.NodeName)
			case pubsub.HubEventNodeOffline:
				c.adminHub.BroadcastNodeOffline(event.NodeSID, event.NodeName)
			}
		}); err != nil {
			logSubscriberExit(log, "hub status subscriber", err)
		}
	})

	// Set AdminHub on nodeStatusHandler for SSE broadcasting
	c.nodeStatusHandler.SetAdminHub(c.adminHub, &nodeSIDResolverAdapter{repo: repos.nodeRepoImpl})

	// Set AdminHub on forwardStatusHandler for SSE broadcasting
	c.forwardStatusHandler.SetAdminHub(c.adminHub, &agentSIDResolverAdapter{repo: repos.forwardAgentRepo})

	// Set AgentStatusQuerier on AdminHub for aggregated SSE broadcasting
	agentStatusQuerierAdapter := adapters.NewAgentStatusQuerierAdapter(repos.forwardAgentRepo, adapters.NewForwardAgentStatusAdapter(c.redis, log), log)
	c.adminHub.SetAgentStatusQuerier(agentStatusQuerierAdapter)

	// Set NodeStatusQuerier on AdminHub for aggregated SSE broadcasting
	nodeStatusQuerierAdapter := adapters.NewNodeStatusQuerierAdapter(repos.nodeRepoImpl, c.nodeStatusQuerier, log)
	c.adminHub.SetNodeStatusQuerier(nodeStatusQuerierAdapter)

	// Set OnNodeOnline callback to sync config, broadcast SSE event, and send recovery notification if needed
	c.agentHub.SetOnNodeOnline(func(nodeID uint) {
		ctx := context.Background()

		// Sync config to node
		if err := c.nodeConfigSyncService.FullSyncToNode(ctx, nodeID); err != nil {
			log.Warnw("failed to sync config to node on connect",
				"node_id", nodeID, "error", err)
		}

		// Get node info
		n, err := repos.nodeRepoImpl.GetByID(ctx, nodeID)
		if err != nil {
			log.Warnw("failed to get node for SSE broadcast",
				"node_id", nodeID, "error", err)
			return
		}
		if n == nil {
			return
		}

		// Broadcast SSE event locally
		c.adminHub.BroadcastNodeOnline(n.SID(), n.Name())

		// Publish status event for cross-instance SSE relay
		if c.hubEventBus != nil {
			if err := c.hubEventBus.PublishStatusEvent(ctx, pubsub.HubStatusEvent{
				Type:     pubsub.HubEventNodeOnline,
				NodeID:   nodeID,
				NodeSID:  n.SID(),
				NodeName: n.Name(),
			}); err != nil {
				log.Warnw("failed to publish node online status event", "error", err)
			}
		}

		// Check if node was in Firing state (offline alert was sent)
		wasFiring, firedAt, err := c.alertStateManager.TransitionToNormal(ctx, cache.AlertResourceTypeNode, nodeID)
		if err != nil {
			log.Warnw("failed to transition node alert state to normal",
				"node_id", nodeID, "error", err)
		}

		if wasFiring {
			var downtimeMinutes int64
			if firedAt != nil {
				downtimeMinutes = int64(biztime.NowUTC().Sub(*firedAt).Minutes())
			}
			cmd := telegramAdminApp.NotifyNodeRecoveryCommand{
				NodeID: nodeID, NodeSID: n.SID(), NodeName: n.Name(),
				OnlineAt: biztime.NowUTC(), DowntimeMinutes: downtimeMinutes,
				MuteNotification: n.MuteNotification(),
			}
			if err := c.adminNotificationServiceDDD.NotifyNodeRecovery(ctx, cmd); err != nil {
				log.Errorw("failed to send node recovery notification",
					"node_sid", n.SID(), "error", err)
			}
		}
	})

	// Set OnNodeOffline callback to broadcast SSE event only
	c.agentHub.SetOnNodeOffline(func(nodeID uint) {
		ctx := context.Background()
		n, err := repos.nodeRepoImpl.GetByID(ctx, nodeID)
		if err != nil {
			log.Warnw("failed to get node for offline broadcast",
				"node_id", nodeID, "error", err)
			return
		}
		if n == nil {
			return
		}
		c.adminHub.BroadcastNodeOffline(n.SID(), n.Name())

		// Publish status event for cross-instance SSE relay
		if c.hubEventBus != nil {
			if err := c.hubEventBus.PublishStatusEvent(ctx, pubsub.HubStatusEvent{
				Type:     pubsub.HubEventNodeOffline,
				NodeID:   nodeID,
				NodeSID:  n.SID(),
				NodeName: n.Name(),
			}); err != nil {
				log.Warnw("failed to publish node offline status event", "error", err)
			}
		}
	})

	// Set OnAgentOnline callback to sync config, broadcast SSE event, and send recovery notification if needed
	c.agentHub.SetOnAgentOnline(func(agentID uint) {
		ctx := context.Background()

		if err := c.configSyncService.FullSyncToAgent(ctx, agentID); err != nil {
			log.Warnw("failed to sync config to agent on connect",
				"agent_id", agentID, "error", err)
		}

		if err := c.configSyncService.NotifyExitPortChange(ctx, agentID); err != nil {
			log.Warnw("failed to notify entry agents of exit agent online",
				"agent_id", agentID, "error", err)
		}

		agent, err := repos.forwardAgentRepo.GetByID(ctx, agentID)
		if err != nil {
			log.Warnw("failed to get agent for SSE broadcast",
				"agent_id", agentID, "error", err)
			return
		}
		if agent == nil {
			return
		}

		c.adminHub.BroadcastForwardAgentOnline(agent.SID(), agent.Name())

		// Publish status event for cross-instance SSE relay
		if c.hubEventBus != nil {
			if err := c.hubEventBus.PublishStatusEvent(ctx, pubsub.HubStatusEvent{
				Type:      pubsub.HubEventAgentOnline,
				AgentID:   agentID,
				AgentSID:  agent.SID(),
				AgentName: agent.Name(),
			}); err != nil {
				log.Warnw("failed to publish agent online status event", "error", err)
			}
		}

		wasFiring, firedAt, err := c.alertStateManager.TransitionToNormal(ctx, cache.AlertResourceTypeAgent, agentID)
		if err != nil {
			log.Warnw("failed to transition agent alert state to normal",
				"agent_id", agentID, "error", err)
		}

		if wasFiring {
			var downtimeMinutes int64
			if firedAt != nil {
				downtimeMinutes = int64(biztime.NowUTC().Sub(*firedAt).Minutes())
			}
			cmd := telegramAdminApp.NotifyAgentRecoveryCommand{
				AgentID: agentID, AgentSID: agent.SID(), AgentName: agent.Name(),
				OnlineAt: biztime.NowUTC(), DowntimeMinutes: downtimeMinutes,
				MuteNotification: agent.MuteNotification(),
			}
			if err := c.adminNotificationServiceDDD.NotifyAgentRecovery(ctx, cmd); err != nil {
				log.Errorw("failed to send agent recovery notification",
					"agent_sid", agent.SID(), "error", err)
			}
		}
	})

	// Set OnAgentOffline callback to broadcast SSE event only
	c.agentHub.SetOnAgentOffline(func(agentID uint) {
		ctx := context.Background()

		if err := c.configSyncService.NotifyExitPortChange(ctx, agentID); err != nil {
			log.Warnw("failed to notify entry agents of exit agent offline",
				"agent_id", agentID, "error", err)
		}

		agent, err := repos.forwardAgentRepo.GetByID(ctx, agentID)
		if err != nil {
			log.Warnw("failed to get agent for offline broadcast",
				"agent_id", agentID, "error", err)
			return
		}
		if agent == nil {
			return
		}
		c.adminHub.BroadcastForwardAgentOffline(agent.SID(), agent.Name())

		// Publish status event for cross-instance SSE relay
		if c.hubEventBus != nil {
			if err := c.hubEventBus.PublishStatusEvent(ctx, pubsub.HubStatusEvent{
				Type:      pubsub.HubEventAgentOffline,
				AgentID:   agentID,
				AgentSID:  agent.SID(),
				AgentName: agent.Name(),
			}); err != nil {
				log.Warnw("failed to publish agent offline status event", "error", err)
			}
		}
	})

	// Set config change notifier for node update use case
	ucs.updateNodeUC.SetConfigChangeNotifier(c.nodeConfigSyncService)

	// Set subscription change notifier for subscription use cases
	ucs.createSubscriptionUC.SetSubscriptionNotifier(c.subscriptionSyncService)
	ucs.activateSubscriptionUC.SetSubscriptionNotifier(c.subscriptionSyncService)
	ucs.cancelSubscriptionUC.SetSubscriptionNotifier(c.subscriptionSyncService)
	ucs.suspendSubscriptionUC.SetSubscriptionNotifier(c.subscriptionSyncService)
	ucs.unsuspendSubscriptionUC.SetSubscriptionNotifier(c.subscriptionSyncService)
	ucs.unsuspendSubscriptionUC.SetQuotaCacheManager(c.quotaCacheSyncService)
	ucs.resetSubscriptionUsageUC.SetSubscriptionNotifier(c.subscriptionSyncService)
	ucs.resetSubscriptionUsageUC.SetQuotaCacheManager(c.quotaCacheSyncService)
	ucs.renewSubscriptionUC.SetSubscriptionNotifier(c.subscriptionSyncService)
}

// ============================================================
// Section 8: Final remaining handlers and middlewares
// ============================================================

// initRemainingHandlers initializes the final handlers and middlewares that
// depend on components from previous sections.
// Corresponds to Section 8 of the original NewRouter().
func (c *Container) initRemainingHandlers() {
	log := c.log
	repos := c.repos
	ucs := c.ucs
	hdlrs := c.hdlrs

	// Initialize QuotaService for unified quota calculation
	ucs.quotaService = subscriptionUsecases.NewQuotaService(
		repos.subscriptionRepo, repos.subscriptionUsageStatsRepo,
		c.hourlyTrafficCache, repos.subscriptionPlanRepo, log,
	)

	// Initialize forward quota middleware with QuotaService
	c.forwardQuotaMiddleware = middleware.NewForwardQuotaMiddleware(
		repos.forwardRuleRepo, repos.subscriptionRepo, repos.subscriptionPlanRepo,
		ucs.quotaService, log,
	)

	// Create done channel for subscription traffic flush scheduler
	c.subscriptionTrafficFlushDone = make(chan struct{})

	// Start subscription traffic flush scheduler (Redis -> MySQL)
	goroutine.SafeGo(log, "subscription-traffic-flush-scheduler", func() {
		ticker := time.NewTicker(2 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				ctx := context.Background()
				if err := c.subscriptionTrafficCache.FlushToDatabase(ctx); err != nil {
					log.Errorw("failed to flush subscription traffic to database", "error", err)
				}
			case <-c.subscriptionTrafficFlushDone:
				return
			}
		}
	})

	// Initialize node hub handler with traffic buffer support
	hdlrs.nodeHubHandler = nodeHandlers.NewNodeHubHandler(
		c.agentHub, repos.nodeRepoImpl, c.subscriptionTrafficBuffer,
		c.subscriptionIDResolver, log,
	)
	hdlrs.nodeHubHandler.SetAddressChangeNotifier(c.configSyncService)
	hdlrs.nodeHubHandler.SetIPUpdater(repos.nodeRepoImpl)
	hdlrs.nodeHubHandler.SetSubscriptionSyncer(c.subscriptionSyncService)
	hdlrs.nodeHubHandler.SetTrafficEnforcer(c.nodeTrafficLimitEnforcementSvc)
	hdlrs.nodeHubHandler.SetQuotaCache(c.nodeQuotaCacheAdapter)
	hdlrs.nodeHubHandler.SetQuotaLoader(c.nodeQuotaLoaderAdapter)
	hdlrs.nodeHubHandler.SetUsageReader(c.nodeUsageReaderAdapter)

	// Initialize node SSE handler
	hdlrs.nodeSSEHandler = nodeHandlers.NewNodeSSEHandler(c.adminHub, log)

	// Initialize forward agent SSE handler
	hdlrs.forwardAgentSSEHandler = forwardAgentCrudHandlers.NewForwardAgentSSEHandler(c.adminHub, log)
}

// logSubscriberExit logs a hub subscriber exit at the appropriate level.
// Context cancellation during shutdown is expected and logged at INFO;
// unexpected errors are logged at ERROR.
func logSubscriberExit(log logger.Interface, name string, err error) {
	if errors.Is(err, context.Canceled) {
		log.Infow(name+" stopped", "reason", "context canceled")
		return
	}
	log.Errorw(name+" failed", "error", err)
}
