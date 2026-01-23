package http

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
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
	"github.com/orris-inc/orris/internal/application/user"
	"github.com/orris-inc/orris/internal/application/user/helpers"
	"github.com/orris-inc/orris/internal/application/user/usecases"
	"github.com/orris-inc/orris/internal/domain/forward"
	"github.com/orris-inc/orris/internal/domain/node"
	"github.com/orris-inc/orris/internal/infrastructure/adapters"
	"github.com/orris-inc/orris/internal/infrastructure/auth"
	"github.com/orris-inc/orris/internal/infrastructure/cache"
	"github.com/orris-inc/orris/internal/infrastructure/config"
	"github.com/orris-inc/orris/internal/infrastructure/email"
	"github.com/orris-inc/orris/internal/infrastructure/pubsub"
	"github.com/orris-inc/orris/internal/infrastructure/repository"
	"github.com/orris-inc/orris/internal/infrastructure/scheduler"
	"github.com/orris-inc/orris/internal/infrastructure/services"
	telegramInfra "github.com/orris-inc/orris/internal/infrastructure/telegram"
	"github.com/orris-inc/orris/internal/infrastructure/template"
	"github.com/orris-inc/orris/internal/infrastructure/token"
	"github.com/orris-inc/orris/internal/interfaces/http/handlers"
	adminHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/admin"
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
	"github.com/orris-inc/orris/internal/interfaces/http/routes"
	"github.com/orris-inc/orris/internal/shared/authorization"
	"github.com/orris-inc/orris/internal/shared/biztime"
	shareddb "github.com/orris-inc/orris/internal/shared/db"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/services/markdown"
)

// Router represents the HTTP router configuration
type Router struct {
	engine                         *gin.Engine
	userHandler                    *handlers.UserHandler
	authHandler                    *handlers.AuthHandler
	passkeyHandler                 *handlers.PasskeyHandler
	profileHandler                 *handlers.ProfileHandler
	dashboardHandler               *handlers.DashboardHandler
	subscriptionHandler            *handlers.SubscriptionHandler
	adminSubscriptionHandler       *adminHandlers.SubscriptionHandler
	adminResourceGroupHandler      *adminHandlers.ResourceGroupHandler
	adminTrafficStatsHandler       *adminHandlers.TrafficStatsHandler
	adminTelegramHandler           *adminHandlers.AdminTelegramHandler
	adminNotificationService       *telegramAdminApp.ServiceDDD
	settingHandler                 *adminHandlers.SettingHandler
	settingService                 *settingApp.ServiceDDD
	planHandler                    *handlers.PlanHandler
	subscriptionTokenHandler       *handlers.SubscriptionTokenHandler
	paymentHandler                 *handlers.PaymentHandler
	nodeHandler                    *handlers.NodeHandler
	nodeSubscriptionHandler        *handlers.NodeSubscriptionHandler
	userNodeHandler                *nodeHandlers.UserNodeHandler
	agentHandler                   *nodeHandlers.AgentHandler
	ticketHandler                  *ticketHandlers.TicketHandler
	notificationHandler            *handlers.NotificationHandler
	telegramHandler                *telegramHandlers.Handler
	telegramService                *telegramApp.ServiceDDD
	telegramBotManager             *telegramInfra.BotServiceManager
	forwardRuleHandler             *forwardRuleHandlers.Handler
	forwardAgentHandler            *forwardAgentCrudHandlers.Handler
	forwardAgentVersionHandler     *forwardAgentCrudHandlers.VersionHandler
	forwardAgentSSEHandler         *forwardAgentCrudHandlers.ForwardAgentSSEHandler
	forwardAgentAPIHandler         *forwardAgentAPIHandlers.Handler
	userForwardRuleHandler         *forwardUserHandlers.Handler
	subscriptionForwardRuleHandler *forwardSubscriptionHandlers.Handler
	agentHub                       *services.AgentHub
	agentHubHandler                *forwardAgentHubHandlers.Handler
	nodeHubHandler                 *nodeHandlers.NodeHubHandler
	nodeVersionHandler             *nodeHandlers.NodeVersionHandler
	nodeSSEHandler                 *nodeHandlers.NodeSSEHandler
	adminHub                       *services.AdminHub
	configSyncService              *forwardServices.ConfigSyncService
	trafficLimitEnforcementSvc     *forwardServices.TrafficLimitEnforcementService
	forwardTrafficCache            cache.ForwardTrafficCache
	ruleTrafficBuffer              *forwardServices.RuleTrafficBuffer
	ruleTrafficFlushDone           chan struct{}
	subscriptionTrafficCache       cache.SubscriptionTrafficCache
	subscriptionTrafficBuffer      *nodeServices.SubscriptionTrafficBuffer
	subscriptionTrafficFlushDone   chan struct{}
	adminNotificationScheduler     *scheduler.AdminNotificationScheduler
	usageAggregationScheduler      *scheduler.UsageAggregationScheduler
	logger                         logger.Interface
	authMiddleware                 *middleware.AuthMiddleware
	subscriptionOwnerMiddleware    *middleware.SubscriptionOwnerMiddleware
	nodeTokenMiddleware            *middleware.NodeTokenMiddleware
	nodeOwnerMiddleware            *middleware.NodeOwnerMiddleware
	nodeQuotaMiddleware            *middleware.NodeQuotaMiddleware
	forwardAgentTokenMiddleware    *middleware.ForwardAgentTokenMiddleware
	forwardRuleOwnerMiddleware     *middleware.ForwardRuleOwnerMiddleware
	forwardQuotaMiddleware         *middleware.ForwardQuotaMiddleware
	rateLimiter                    *middleware.RateLimiter
	oauthManager                   *auth.OAuthServiceManager
	emailManager                   *email.EmailServiceManager
}

type jwtServiceAdapter struct {
	*auth.JWTService
}

func (a *jwtServiceAdapter) Generate(userUUID string, sessionID string, role authorization.UserRole) (*usecases.TokenPair, error) {
	pair, err := a.JWTService.Generate(userUUID, sessionID, role)
	if err != nil {
		return nil, err
	}
	return &usecases.TokenPair{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresIn:    pair.ExpiresIn,
	}, nil
}

type oauthClientAdapter struct {
	client interface {
		GetAuthURL(state string) (authURL string, codeVerifier string, err error)
		ExchangeCode(ctx context.Context, code string, codeVerifier string) (string, error)
		GetUserInfo(ctx context.Context, accessToken string) (*auth.OAuthUserInfo, error)
	}
}

func (a *oauthClientAdapter) GetAuthURL(state string) (string, string, error) {
	return a.client.GetAuthURL(state)
}

func (a *oauthClientAdapter) ExchangeCode(ctx context.Context, code string, codeVerifier string) (string, error) {
	return a.client.ExchangeCode(ctx, code, codeVerifier)
}

func (a *oauthClientAdapter) GetUserInfo(ctx context.Context, accessToken string) (*usecases.OAuthUserInfo, error) {
	info, err := a.client.GetUserInfo(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	return &usecases.OAuthUserInfo{
		Email:         info.Email,
		Name:          info.Name,
		Picture:       info.Picture,
		EmailVerified: info.EmailVerified,
		Provider:      info.Provider,
		ProviderID:    info.ProviderID,
	}, nil
}

// dynamicOAuthClientAdapter wraps OAuthServiceManager to provide dynamic OAuth client access
// This adapter fetches the current OAuth client from manager on each call, enabling hot-reload support
type dynamicOAuthClientAdapter struct {
	manager  *auth.OAuthServiceManager
	provider string // "google" or "github"
}

func (a *dynamicOAuthClientAdapter) getClient() interface {
	GetAuthURL(state string) (authURL string, codeVerifier string, err error)
	ExchangeCode(ctx context.Context, code string, codeVerifier string) (string, error)
	GetUserInfo(ctx context.Context, accessToken string) (*auth.OAuthUserInfo, error)
} {
	switch a.provider {
	case "google":
		return a.manager.GetGoogleClient()
	case "github":
		return a.manager.GetGitHubClient()
	default:
		return nil
	}
}

func (a *dynamicOAuthClientAdapter) GetAuthURL(state string) (string, string, error) {
	client := a.getClient()
	if client == nil {
		return "", "", auth.ErrOAuthNotConfigured
	}
	return client.GetAuthURL(state)
}

func (a *dynamicOAuthClientAdapter) ExchangeCode(ctx context.Context, code string, codeVerifier string) (string, error) {
	client := a.getClient()
	if client == nil {
		return "", auth.ErrOAuthNotConfigured
	}
	return client.ExchangeCode(ctx, code, codeVerifier)
}

func (a *dynamicOAuthClientAdapter) GetUserInfo(ctx context.Context, accessToken string) (*usecases.OAuthUserInfo, error) {
	client := a.getClient()
	if client == nil {
		return nil, auth.ErrOAuthNotConfigured
	}
	info, err := client.GetUserInfo(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	return &usecases.OAuthUserInfo{
		Email:         info.Email,
		Name:          info.Name,
		Picture:       info.Picture,
		EmailVerified: info.EmailVerified,
		Provider:      info.Provider,
		ProviderID:    info.ProviderID,
	}, nil
}

// NewRouter creates a new HTTP router with all dependencies
func NewRouter(userService *user.ServiceDDD, db *gorm.DB, cfg *config.Config, log logger.Interface) *Router {
	engine := gin.New()

	userRepo := repository.NewUserRepository(db, log)
	sessionRepo := repository.NewSessionRepository(db)
	oauthRepo := repository.NewOAuthAccountRepository(db)

	hasher := auth.NewBcryptPasswordHasher(cfg.Auth.Password.BcryptCost)
	jwtSvc := auth.NewJWTService(cfg.Auth.JWT.Secret, cfg.Auth.JWT.AccessExpMinutes, cfg.Auth.JWT.RefreshExpDays)
	jwtService := &jwtServiceAdapter{jwtSvc}

	// Note: Email service and OAuth clients are now initialized later via
	// EmailServiceManager and OAuthServiceManager for hot-reload support.
	// See section after settingServiceDDD initialization.

	// Initialize Redis client for OAuth state storage and Asynq
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.GetAddr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Test Redis connection
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalw("failed to connect to Redis", "error", err)
	}
	log.Infow("Redis connection established successfully")

	// Create OAuth StateStore with 10 minute TTL
	stateStore := cache.NewRedisStateStore(
		redisClient,
		"oauth:state:",
		10*time.Minute,
	)

	// Initialize HMAC-based agent token service for local token verification
	agentTokenSvc := auth.NewAgentTokenService(cfg.Forward.TokenSigningSecret)

	authHelper := helpers.NewAuthHelper(userRepo, sessionRepo, log)

	// Note: Auth-related use cases (registerUC, loginUC, etc.) and authHandler
	// are initialized later after OAuthServiceManager and EmailServiceManager
	// are created for hot-reload support. See section after settingServiceDDD.

	// Declare variables for auth components that will be initialized later
	var authHandler *handlers.AuthHandler
	var passkeyHandler *handlers.PasskeyHandler
	var userHandler *handlers.UserHandler

	authMiddleware := middleware.NewAuthMiddleware(jwtSvc, userRepo, log)
	rateLimiter := middleware.NewRateLimiter(100, 1*time.Minute)

	subscriptionRepo := repository.NewSubscriptionRepository(db, log)
	subscriptionPlanRepo := repository.NewPlanRepository(db, log)
	subscriptionTokenRepo := repository.NewSubscriptionTokenRepository(db, log)
	subscriptionUsageRepo := repository.NewSubscriptionUsageRepository(db, log)
	subscriptionUsageStatsRepo := repository.NewSubscriptionUsageStatsRepository(db, log)
	planPricingRepo := repository.NewPlanPricingRepository(db, log)
	paymentRepo := repository.NewPaymentRepository(db)

	// Initialize node and forward repositories early (needed by subscription usage stats)
	nodeRepoImpl := repository.NewNodeRepository(db, log)
	forwardRuleRepo := repository.NewForwardRuleRepository(db, log)

	tokenGenerator := token.NewTokenGenerator()

	createSubscriptionUC := subscriptionUsecases.NewCreateSubscriptionUseCase(
		subscriptionRepo, subscriptionPlanRepo, subscriptionTokenRepo, planPricingRepo, userRepo, tokenGenerator, log,
	)
	activateSubscriptionUC := subscriptionUsecases.NewActivateSubscriptionUseCase(
		subscriptionRepo, log,
	)
	subscriptionBaseURL := cfg.Subscription.GetBaseURL(cfg.Server.GetBaseURL())
	getSubscriptionUC := subscriptionUsecases.NewGetSubscriptionUseCase(
		subscriptionRepo, subscriptionPlanRepo, userRepo, log, subscriptionBaseURL,
	)
	listUserSubscriptionsUC := subscriptionUsecases.NewListUserSubscriptionsUseCase(
		subscriptionRepo, subscriptionPlanRepo, userRepo, log, subscriptionBaseURL,
	)
	cancelSubscriptionUC := subscriptionUsecases.NewCancelSubscriptionUseCase(
		subscriptionRepo, subscriptionTokenRepo, log,
	)
	suspendSubscriptionUC := subscriptionUsecases.NewSuspendSubscriptionUseCase(
		subscriptionRepo, log,
	)
	unsuspendSubscriptionUC := subscriptionUsecases.NewUnsuspendSubscriptionUseCase(
		subscriptionRepo, log,
	)
	resetSubscriptionUsageUC := subscriptionUsecases.NewResetSubscriptionUsageUseCase(
		subscriptionRepo, log,
	)
	deleteSubscriptionUC := subscriptionUsecases.NewDeleteSubscriptionUseCase(
		subscriptionRepo, subscriptionTokenRepo, shareddb.NewTransactionManager(db), log,
	)
	renewSubscriptionUC := subscriptionUsecases.NewRenewSubscriptionUseCase(
		subscriptionRepo, subscriptionPlanRepo, planPricingRepo, log,
	)
	changePlanUC := subscriptionUsecases.NewChangePlanUseCase(
		subscriptionRepo, subscriptionPlanRepo, log,
	)

	// Initialize hourly traffic cache for Redis-based hourly data queries and daily aggregation
	hourlyTrafficCache := cache.NewRedisHourlyTrafficCache(redisClient, log)

	getSubscriptionUsageStatsUC := subscriptionUsecases.NewGetSubscriptionUsageStatsUseCase(
		subscriptionUsageRepo, subscriptionUsageStatsRepo, hourlyTrafficCache, nodeRepoImpl, forwardRuleRepo, log,
	)
	resetSubscriptionLinkUC := subscriptionUsecases.NewResetSubscriptionLinkUseCase(
		subscriptionRepo, subscriptionPlanRepo, userRepo, log, subscriptionBaseURL,
	)

	aggregateUsageUC := subscriptionUsecases.NewAggregateUsageUseCase(
		subscriptionUsageRepo, subscriptionUsageStatsRepo, hourlyTrafficCache, log,
	)

	// Initialize usage aggregation scheduler with default retention days (90 days)
	usageAggregationScheduler := scheduler.NewUsageAggregationScheduler(
		aggregateUsageUC,
		scheduler.DefaultRetentionDays,
		log,
	)

	createPlanUC := subscriptionUsecases.NewCreatePlanUseCase(
		subscriptionPlanRepo, planPricingRepo, log,
	)
	updatePlanUC := subscriptionUsecases.NewUpdatePlanUseCase(
		subscriptionPlanRepo, planPricingRepo, log,
	)
	getPlanUC := subscriptionUsecases.NewGetPlanUseCase(
		subscriptionPlanRepo, planPricingRepo, log,
	)
	listPlansUC := subscriptionUsecases.NewListPlansUseCase(
		subscriptionPlanRepo, planPricingRepo, log,
	)
	getPublicPlansUC := subscriptionUsecases.NewGetPublicPlansUseCase(
		subscriptionPlanRepo, planPricingRepo, log,
	)
	activatePlanUC := subscriptionUsecases.NewActivatePlanUseCase(
		subscriptionPlanRepo, log,
	)
	deactivatePlanUC := subscriptionUsecases.NewDeactivatePlanUseCase(
		subscriptionPlanRepo, log,
	)
	deletePlanUC := subscriptionUsecases.NewDeletePlanUseCase(
		subscriptionPlanRepo, subscriptionRepo, planPricingRepo, shareddb.NewTransactionManager(db), log,
	)
	getPlanPricingsUC := subscriptionUsecases.NewGetPlanPricingsUseCase(
		subscriptionPlanRepo, planPricingRepo, log,
	)

	generateTokenUC := subscriptionUsecases.NewGenerateSubscriptionTokenUseCase(
		subscriptionRepo, subscriptionTokenRepo, tokenGenerator, log,
	)
	listTokensUC := subscriptionUsecases.NewListSubscriptionTokensUseCase(
		subscriptionTokenRepo, log,
	)
	revokeTokenUC := subscriptionUsecases.NewRevokeSubscriptionTokenUseCase(
		subscriptionTokenRepo, log,
	)
	refreshSubscriptionTokenUC := subscriptionUsecases.NewRefreshSubscriptionTokenUseCase(
		subscriptionTokenRepo, subscriptionRepo, tokenGenerator, log,
	)

	subscriptionHandler := handlers.NewSubscriptionHandler(
		createSubscriptionUC, getSubscriptionUC, listUserSubscriptionsUC,
		cancelSubscriptionUC, deleteSubscriptionUC, changePlanUC, getSubscriptionUsageStatsUC,
		resetSubscriptionLinkUC, log,
	)
	adminSubscriptionHandler := adminHandlers.NewSubscriptionHandler(
		subscriptionRepo, createSubscriptionUC, getSubscriptionUC, listUserSubscriptionsUC,
		cancelSubscriptionUC, deleteSubscriptionUC, renewSubscriptionUC, changePlanUC,
		activateSubscriptionUC, suspendSubscriptionUC, unsuspendSubscriptionUC, resetSubscriptionUsageUC, log,
	)
	subscriptionOwnerMiddleware := middleware.NewSubscriptionOwnerMiddleware(subscriptionRepo, log)

	// Initialize resource group repository (handler initialized later after node and agent repos)
	resourceGroupRepo := repository.NewResourceGroupRepository(db, log)
	createResourceGroupUC := resourceUsecases.NewCreateResourceGroupUseCase(resourceGroupRepo, subscriptionPlanRepo, log)
	getResourceGroupUC := resourceUsecases.NewGetResourceGroupUseCase(resourceGroupRepo, subscriptionPlanRepo, log)
	listResourceGroupsUC := resourceUsecases.NewListResourceGroupsUseCase(resourceGroupRepo, subscriptionPlanRepo, log)
	updateResourceGroupUC := resourceUsecases.NewUpdateResourceGroupUseCase(resourceGroupRepo, subscriptionPlanRepo, log)
	deleteResourceGroupUC := resourceUsecases.NewDeleteResourceGroupUseCase(resourceGroupRepo, forwardRuleRepo, log)
	updateResourceGroupStatusUC := resourceUsecases.NewUpdateResourceGroupStatusUseCase(resourceGroupRepo, subscriptionPlanRepo, log)

	planHandler := handlers.NewPlanHandler(
		createPlanUC, updatePlanUC, getPlanUC, listPlansUC,
		getPublicPlansUC, activatePlanUC, deactivatePlanUC, deletePlanUC, getPlanPricingsUC,
	)
	subscriptionTokenHandler := handlers.NewSubscriptionTokenHandler(
		generateTokenUC, listTokensUC, revokeTokenUC, refreshSubscriptionTokenUC,
	)

	nodeRepo := adapters.NewNodeRepositoryAdapter(nodeRepoImpl, forwardRuleRepo, db, log)
	tokenValidator := adapters.NewSubscriptionTokenValidatorAdapter(db, log)

	// Initialize subscription template loader
	templateLoader := template.NewSubscriptionTemplateLoader(
		cfg.Subscription.TemplatesPath,
		log,
	)
	if err := templateLoader.Load(); err != nil {
		log.Warnw("failed to load subscription templates, using defaults", "error", err)
	}

	generateSubscriptionUC := nodeUsecases.NewGenerateSubscriptionUseCase(
		nodeRepo, tokenValidator, templateLoader, log,
	)

	// Initialize node system status querier adapter
	nodeStatusQuerier := adapters.NewNodeSystemStatusQuerierAdapter(redisClient, log)

	// Initialize GitHub release services for version checking
	// Forward agent uses orris-client repository
	forwardAgentReleaseService := services.NewGitHubReleaseService(services.GitHubRepoConfig{
		Owner:       "orris-inc",
		Repo:        "orris-client",
		AssetPrefix: "orris-client",
	}, log)
	// Node agent uses orrisp repository
	nodeAgentReleaseService := services.NewGitHubReleaseService(services.GitHubRepoConfig{
		Owner:       "orris-inc",
		Repo:        "orrisp",
		AssetPrefix: "orrisp",
	}, log)

	// Initialize node use cases
	createNodeUC := nodeUsecases.NewCreateNodeUseCase(nodeRepoImpl, log)
	getNodeUC := nodeUsecases.NewGetNodeUseCase(nodeRepoImpl, resourceGroupRepo, nodeStatusQuerier, log)
	updateNodeUC := nodeUsecases.NewUpdateNodeUseCase(log, nodeRepoImpl, resourceGroupRepo)
	deleteNodeUC := nodeUsecases.NewDeleteNodeUseCase(nodeRepoImpl, forwardRuleRepo, log)
	listNodesUC := nodeUsecases.NewListNodesUseCase(nodeRepoImpl, resourceGroupRepo, userRepo, nodeStatusQuerier, nodeAgentReleaseService, log)
	generateNodeTokenUC := nodeUsecases.NewGenerateNodeTokenUseCase(nodeRepoImpl, log)
	generateNodeInstallScriptUC := nodeUsecases.NewGenerateNodeInstallScriptUseCase(nodeRepoImpl, log)

	// Initialize user node use cases
	createUserNodeUC := nodeUsecases.NewCreateUserNodeUseCase(nodeRepoImpl, log)
	listUserNodesUC := nodeUsecases.NewListUserNodesUseCase(nodeRepoImpl, log)
	getUserNodeUC := nodeUsecases.NewGetUserNodeUseCase(nodeRepoImpl, log)
	updateUserNodeUC := nodeUsecases.NewUpdateUserNodeUseCase(nodeRepoImpl, log)
	deleteUserNodeUC := nodeUsecases.NewDeleteUserNodeUseCase(nodeRepoImpl, log)
	regenerateUserNodeTokenUC := nodeUsecases.NewRegenerateUserNodeTokenUseCase(nodeRepoImpl, log)
	getUserNodeUsageUC := nodeUsecases.NewGetUserNodeUsageUseCase(nodeRepoImpl, subscriptionRepo, subscriptionPlanRepo, log)
	getUserNodeInstallScriptUC := nodeUsecases.NewGetUserNodeInstallScriptUseCase(nodeRepoImpl, log)

	// Initialize node authentication middleware using the same node repository adapter
	validateNodeTokenUC := nodeUsecases.NewValidateNodeTokenUseCase(nodeRepo, log)
	nodeTokenMiddleware := middleware.NewNodeTokenMiddleware(validateNodeTokenUC, log)

	// Initialize node owner middleware
	nodeOwnerMiddleware := middleware.NewNodeOwnerMiddleware(nodeRepoImpl)

	// Initialize node quota middleware
	nodeQuotaMiddleware := middleware.NewNodeQuotaMiddleware(nodeRepoImpl, subscriptionRepo, subscriptionPlanRepo)

	// Initialize handlers
	// API URL for node install script generation
	apiBaseURL := cfg.Server.GetBaseURL()
	nodeHandler := handlers.NewNodeHandler(createNodeUC, getNodeUC, updateNodeUC, deleteNodeUC, listNodesUC, generateNodeTokenUC, generateNodeInstallScriptUC, apiBaseURL)
	nodeSubscriptionHandler := handlers.NewNodeSubscriptionHandler(generateSubscriptionUC)
	userNodeHandler := nodeHandlers.NewUserNodeHandler(
		createUserNodeUC,
		listUserNodesUC,
		getUserNodeUC,
		updateUserNodeUC,
		deleteUserNodeUC,
		regenerateUserNodeTokenUC,
		getUserNodeUsageUC,
		getUserNodeInstallScriptUC,
		apiBaseURL,
	)

	ticketHandler := ticketHandlers.NewTicketHandler(nil, nil, nil, nil, nil, nil, nil, nil, nil)

	announcementRepo := adapters.NewAnnouncementRepositoryAdapter(repository.NewAnnouncementRepository(db))
	notificationRepo := adapters.NewNotificationRepositoryAdapter(repository.NewNotificationRepository(db))
	templateRepo := adapters.NewTemplateRepositoryAdapter(repository.NewNotificationTemplateRepository(db))
	userRepoAdapter := adapters.NewUserRepositoryAdapter(userRepo)

	markdownService := markdown.NewMarkdownService()

	announcementFactory := adapters.NewAnnouncementFactoryAdapter()
	notificationFactory := adapters.NewNotificationFactoryAdapter()
	templateFactory := adapters.NewTemplateFactoryAdapter()

	notificationServiceDDD := notificationApp.NewServiceDDD(
		announcementRepo,
		notificationRepo,
		templateRepo,
		userRepoAdapter,
		announcementFactory,
		notificationFactory,
		templateFactory,
		markdownService,
		log,
	)

	notificationHandler := handlers.NewNotificationHandler(notificationServiceDDD, log)

	// Initialize System Setting components
	settingRepo := repository.NewSystemSettingRepository(db, log)
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
	settingServiceDDD := settingApp.NewServiceDDD(settingRepo, settingProviderCfg, nil, log)
	settingHandler := adminHandlers.NewSettingHandler(settingServiceDDD, log)
	settingProvider := settingServiceDDD.GetSettingProvider()

	// Initialize OAuthServiceManager for hot-reload support
	// Note: Initial failure is acceptable - will be re-initialized on setting changes
	oauthManager := auth.NewOAuthServiceManager(settingProvider, log)
	if err := oauthManager.Initialize(context.Background()); err != nil {
		log.Warnw("oauth service manager initial config incomplete, will reinitialize on setting change", "error", err)
	}
	settingServiceDDD.Subscribe(oauthManager)

	// Initialize EmailServiceManager for hot-reload support
	// Note: Initial failure is acceptable - will be re-initialized on setting changes
	emailManager := email.NewEmailServiceManager(settingProvider, log)
	if err := emailManager.Initialize(context.Background()); err != nil {
		log.Warnw("email service manager initial config incomplete, will reinitialize on setting change", "error", err)
	}
	settingServiceDDD.Subscribe(emailManager)

	// Create DynamicEmailService for use cases
	dynamicEmailSvc := email.NewDynamicEmailService(emailManager, log)

	// Inject EmailTester to break circular dependency
	settingServiceDDD.SetEmailTester(dynamicEmailSvc)

	// Create dynamic OAuth clients that fetch current client from manager
	dynamicGoogleClient := &dynamicOAuthClientAdapter{manager: oauthManager, provider: "google"}
	dynamicGitHubClient := &dynamicOAuthClientAdapter{manager: oauthManager, provider: "github"}

	// Initialize auth-related use cases with dynamic services for hot-reload support
	registerUC := usecases.NewRegisterWithPasswordUseCase(userRepo, hasher, dynamicEmailSvc, authHelper, log)
	loginUC := usecases.NewLoginWithPasswordUseCase(userRepo, sessionRepo, hasher, jwtService, authHelper, cfg.Auth.Session, log)
	verifyEmailUC := usecases.NewVerifyEmailUseCase(userRepo, log)
	requestResetUC := usecases.NewRequestPasswordResetUseCase(userRepo, dynamicEmailSvc, log)
	resetPasswordUC := usecases.NewResetPasswordUseCase(userRepo, sessionRepo, hasher, dynamicEmailSvc, log)
	adminResetPasswordUC := usecases.NewAdminResetPasswordUseCase(userRepo, sessionRepo, hasher, dynamicEmailSvc, log)
	initiateOAuthUC := usecases.NewInitiateOAuthLoginUseCase(dynamicGoogleClient, dynamicGitHubClient, log, stateStore)
	handleOAuthUC := usecases.NewHandleOAuthCallbackUseCase(userRepo, oauthRepo, sessionRepo, dynamicGoogleClient, dynamicGitHubClient, jwtService, initiateOAuthUC, authHelper, cfg.Auth.Session, log)
	refreshTokenUC := usecases.NewRefreshTokenUseCase(userRepo, sessionRepo, jwtService, authHelper, log)
	logoutUC := usecases.NewLogoutUseCase(sessionRepo, log)

	authHandler = handlers.NewAuthHandler(
		registerUC, loginUC, verifyEmailUC, requestResetUC, resetPasswordUC,
		initiateOAuthUC, handleOAuthUC, refreshTokenUC, logoutUC, userRepo, log,
		cfg.Auth.Cookie, cfg.Auth.JWT,
		cfg.Server.FrontendCallbackURL, cfg.Server.AllowedOrigins,
	)

	// Initialize Passkey (WebAuthn) components if configured
	if cfg.WebAuthn.IsConfigured() {
		webAuthnService, err := auth.NewWebAuthnService(cfg.WebAuthn)
		if err != nil {
			log.Warnw("failed to initialize WebAuthn service, passkey authentication disabled", "error", err)
		} else {
			passkeyRepo := repository.NewPasskeyCredentialRepository(db, log)
			passkeyChallengeStore := cache.NewPasskeyChallengeStore(redisClient)
			passkeySignupSessionStore := cache.NewPasskeySignupSessionStore(redisClient)

			startPasskeyRegistrationUC := usecases.NewStartPasskeyRegistrationUseCase(userRepo, passkeyRepo, webAuthnService, passkeyChallengeStore, log)
			finishPasskeyRegistrationUC := usecases.NewFinishPasskeyRegistrationUseCase(userRepo, passkeyRepo, webAuthnService, passkeyChallengeStore, log)
			startPasskeyAuthenticationUC := usecases.NewStartPasskeyAuthenticationUseCase(userRepo, passkeyRepo, webAuthnService, passkeyChallengeStore, log)
			finishPasskeyAuthenticationUC := usecases.NewFinishPasskeyAuthenticationUseCase(userRepo, passkeyRepo, sessionRepo, webAuthnService, passkeyChallengeStore, jwtService, authHelper, cfg.Auth.Session, log)
			startPasskeySignupUC := usecases.NewStartPasskeySignupUseCase(userRepo, webAuthnService, passkeyChallengeStore, passkeySignupSessionStore, log)
			finishPasskeySignupUC := usecases.NewFinishPasskeySignupUseCase(userRepo, passkeyRepo, sessionRepo, webAuthnService, passkeyChallengeStore, passkeySignupSessionStore, jwtService, authHelper, cfg.Auth.Session, log)
			listUserPasskeysUC := usecases.NewListUserPasskeysUseCase(passkeyRepo, log)
			deletePasskeyUC := usecases.NewDeletePasskeyUseCase(passkeyRepo, log)

			passkeyHandler = handlers.NewPasskeyHandler(
				startPasskeyRegistrationUC,
				finishPasskeyRegistrationUC,
				startPasskeyAuthenticationUC,
				finishPasskeyAuthenticationUC,
				startPasskeySignupUC,
				finishPasskeySignupUC,
				listUserPasskeysUC,
				deletePasskeyUC,
				log,
				cfg.Auth.Cookie,
				cfg.Auth.JWT,
			)
			log.Infow("WebAuthn passkey authentication enabled")
		}
	}

	userHandler = handlers.NewUserHandler(userService, adminResetPasswordUC)

	// Initialize Telegram notification components using BotServiceManager for hot-reload support
	var telegramHandler *telegramHandlers.Handler
	var telegramServiceDDD *telegramApp.ServiceDDD
	var telegramBotManager *telegramInfra.BotServiceManager

	// Initialize Telegram base components (regardless of initial config state)
	// These components don't depend on whether Telegram is currently configured
	telegramVerifyStore := cache.NewTelegramVerifyStore(redisClient)
	telegramBindingRepo := repository.NewTelegramBindingRepository(db, log)

	// Initialize Telegram ServiceDDD (initially without BotService, will be managed by BotServiceManager)
	telegramServiceDDD = telegramApp.NewServiceDDD(
		telegramBindingRepo,
		subscriptionRepo,
		subscriptionUsageRepo,
		subscriptionUsageStatsRepo,
		hourlyTrafficCache,
		subscriptionPlanRepo,
		telegramVerifyStore,
		nil, // BotService will be managed by BotServiceManager
		log,
	)

	// Create UpdateHandler for polling mode
	serviceAdapter := telegramInfra.NewServiceAdapter(
		telegramServiceDDD,
		func(ctx context.Context, telegramUserID int64, telegramUsername, verifyCode string) error {
			_, err := telegramServiceDDD.BindFromWebhook(ctx, telegramUserID, telegramUsername, verifyCode)
			return err
		},
		func(ctx context.Context, telegramUserID int64) (bool, error) {
			status, err := telegramServiceDDD.GetBindingStatusByTelegramID(ctx, telegramUserID)
			if err != nil {
				return false, err
			}
			return status.IsBound, nil
		},
	)
	updateHandler := telegramInfra.NewPollingUpdateHandler(serviceAdapter, log)

	// Create BotServiceManager with hot-reload support
	telegramBotManager = telegramInfra.NewBotServiceManager(settingProvider, updateHandler, log)

	// Inject BotServiceManager into ServiceAdapter (break circular dependency)
	serviceAdapter.SetBotServiceGetter(telegramBotManager)

	// Create DynamicBotService and inject into telegramServiceDDD
	// This allows the service to send messages via webhook mode with hot-reload support
	dynamicBotService := telegramInfra.NewDynamicBotService(telegramBotManager, log)
	telegramServiceDDD.SetBotService(dynamicBotService)

	// Subscribe BotServiceManager to setting changes for hot-reload
	settingServiceDDD.Subscribe(telegramBotManager)

	// Inject telegramTester to break circular dependency
	// BotServiceManager implements TelegramConnectionTester interface
	settingServiceDDD.SetTelegramTester(telegramBotManager)

	// Initialize Telegram Handler (webhook secret will be retrieved dynamically from SettingProvider)
	// For initial webhook secret, use env config as fallback
	initialWebhookSecret := cfg.Telegram.WebhookSecret
	telegramHandler = telegramHandlers.NewHandler(telegramServiceDDD, log, initialWebhookSecret)

	// Inject SettingProvider for hot-reload support of webhook secret from database
	telegramHandler.SetWebhookSecretProvider(settingProvider)

	log.Infow("telegram components initialized with hot-reload support")

	// Initialize admin notification components
	var adminTelegramHandler *adminHandlers.AdminTelegramHandler
	var adminNotificationServiceDDD *telegramAdminApp.ServiceDDD

	// Admin notification initialization - uses BotServiceManager's BotService when available
	adminVerifyStore := cache.NewAdminTelegramVerifyStore(redisClient)
	adminBindingRepo := repository.NewAdminTelegramBindingRepository(db, log)
	userRoleChecker := adapters.NewUserRoleCheckerAdapter(userRepo)

	// Create admin notification service with DynamicBotService for message sending
	// Note: dynamicBotService was already created above for telegramServiceDDD
	// Reusing it here enables hot-reload support for admin notifications as well
	adminNotificationServiceDDD = telegramAdminApp.NewServiceDDD(
		adminBindingRepo,
		adminVerifyStore,
		dynamicBotService,  // BotService - uses DynamicBotService for hot-reload
		telegramBotManager, // BotLinkProvider - get bot link from manager
		userRoleChecker,
		log,
	)

	adminTelegramHandler = adminHandlers.NewAdminTelegramHandler(adminNotificationServiceDDD, log)

	// Inject admin service into telegram handler for /adminbind command support (webhook mode)
	telegramHandler.SetAdminService(adminNotificationServiceDDD)

	// Inject admin binder into service adapter for /adminbind command support (polling mode)
	serviceAdapter.SetAdminBinder(adminNotificationServiceDDD)

	log.Infow("admin notification components initialized")

	// Create profile handler
	profileHandler := handlers.NewProfileHandler(userService)

	// Create dashboard handler
	getDashboardUC := usecases.NewGetDashboardUseCase(
		subscriptionRepo,
		subscriptionUsageStatsRepo,
		hourlyTrafficCache,
		subscriptionPlanRepo,
		log,
	)
	dashboardHandler := handlers.NewDashboardHandler(getDashboardUC, log)

	// TODO: Implement real payment gateway (Alipay/WeChat/Stripe)
	// Currently mock gateway is removed as per CLAUDE.md rule: "no mock data allowed"
	var gateway paymentGateway.PaymentGateway = nil // Temporary placeholder until real implementation
	paymentConfig := paymentUsecases.PaymentConfig{
		NotifyURL: cfg.Server.GetBaseURL() + "/payments/callback",
	}
	createPaymentUC := paymentUsecases.NewCreatePaymentUseCase(
		paymentRepo,
		subscriptionRepo,
		subscriptionPlanRepo,
		planPricingRepo,
		gateway,
		log,
		paymentConfig,
	)
	handleCallbackUC := paymentUsecases.NewHandlePaymentCallbackUseCase(
		paymentRepo,
		activateSubscriptionUC,
		gateway,
		log,
	)

	paymentHandler := handlers.NewPaymentHandler(createPaymentUC, handleCallbackUC, log)

	// Agent API handlers
	getNodeConfigUC := nodeUsecases.NewGetNodeConfigUseCase(nodeRepoImpl, log)
	getNodeSubscriptionsUC := nodeUsecases.NewGetNodeSubscriptionsUseCase(subscriptionRepo, nodeRepoImpl, log)

	// Initialize subscription traffic cache and buffer for RESTful agent traffic reporting
	// This is also used later by forwardQuotaMiddleware and nodeHubHandler
	// Note: subscriptionUsageRepo is passed for backward compatibility but no longer used.
	// Traffic data is now flushed to HourlyTrafficCache (Redis) instead of MySQL.
	subscriptionTrafficCache := cache.NewRedisSubscriptionTrafficCache(
		redisClient,
		hourlyTrafficCache,
		subscriptionUsageRepo,
		log,
	)
	subscriptionTrafficBuffer := nodeServices.NewSubscriptionTrafficBuffer(subscriptionTrafficCache, log)
	subscriptionTrafficBuffer.Start()

	// Initialize subscription quota cache for node traffic limit checking
	subscriptionQuotaCache := cache.NewRedisSubscriptionQuotaCache(redisClient, log)

	// Initialize quota cache sync service for managing quota cache
	quotaCacheSyncService := subscriptionServices.NewQuotaCacheSyncService(
		subscriptionRepo,
		subscriptionPlanRepo,
		subscriptionQuotaCache,
		log,
	)

	// Initialize node traffic limit enforcement service
	nodeTrafficLimitEnforcementSvc := nodeServices.NewNodeTrafficLimitEnforcementService(
		subscriptionRepo,
		subscriptionUsageStatsRepo,
		hourlyTrafficCache,
		subscriptionPlanRepo,
		subscriptionQuotaCache,
		log,
	)

	// Initialize adapters for node hub handler traffic limit checking
	nodeQuotaCacheAdapter := adapters.NewNodeSubscriptionQuotaCacheAdapter(subscriptionQuotaCache, log)
	nodeQuotaLoaderAdapter := adapters.NewNodeSubscriptionQuotaLoaderAdapter(
		subscriptionRepo,
		subscriptionPlanRepo,
		subscriptionQuotaCache,
		log,
	)
	nodeUsageReaderAdapter := adapters.NewNodeSubscriptionUsageReaderAdapter(
		hourlyTrafficCache,
		subscriptionUsageStatsRepo,
		log,
	)

	// Initialize agent report use cases with adapters
	subscriptionUsageRecorder := adapters.NewSubscriptionUsageRecorderAdapter(subscriptionTrafficBuffer, log)
	systemStatusUpdater := adapters.NewNodeSystemStatusUpdaterAdapter(redisClient, log)
	onlineSubscriptionTracker := adapters.NewOnlineSubscriptionTrackerAdapter(log)
	subscriptionIDResolver := adapters.NewSubscriptionIDResolverAdapter(subscriptionRepo, log)
	reportSubscriptionUsageUC := nodeUsecases.NewReportSubscriptionUsageUseCase(subscriptionUsageRecorder, subscriptionIDResolver, log)
	reportNodeStatusUC := nodeUsecases.NewReportNodeStatusUseCase(systemStatusUpdater, nodeRepoImpl, nodeRepoImpl, log)
	reportOnlineSubscriptionsUC := nodeUsecases.NewReportOnlineSubscriptionsUseCase(onlineSubscriptionTracker, subscriptionIDResolver, log)

	// Initialize RESTful Agent Handler
	agentHandler := nodeHandlers.NewAgentHandler(
		getNodeConfigUC,
		getNodeSubscriptionsUC,
		reportSubscriptionUsageUC,
		reportNodeStatusUC,
		reportOnlineSubscriptionsUC,
		log,
	)

	// Initialize forward agent repository (rule repo initialized earlier)
	forwardAgentRepo := repository.NewForwardAgentRepository(db, log)

	// Initialize admin notification processor and scheduler for offline alerts
	// (requires forwardAgentRepo, nodeRepoImpl, and other repos to be initialized)
	alertDeduplicator := cache.NewAlertDeduplicator(redisClient)
	adminNotificationProcessor := telegramAdminApp.NewAdminNotificationProcessor(
		adminBindingRepo,
		userRepo,
		subscriptionRepo,
		subscriptionUsageStatsRepo,
		hourlyTrafficCache,
		nodeRepoImpl,
		forwardAgentRepo,
		alertDeduplicator,
		&botServiceProviderAdapter{telegramBotManager},
		log,
	)
	adminNotificationScheduler := scheduler.NewAdminNotificationScheduler(adminNotificationProcessor, log)

	// Initialize mute notification service and inject into telegram handler
	muteNotificationUC := telegramAdminUsecases.NewMuteNotificationUseCase(forwardAgentRepo, nodeRepoImpl, log)
	telegramHandler.SetMuteService(muteNotificationUC)
	telegramHandler.SetCallbackAnswerer(dynamicBotService)

	// Inject mute service and callback answerer into service adapter for polling mode callback query handling
	serviceAdapter.SetMuteService(muteNotificationUC)
	serviceAdapter.SetCallbackAnswerer(dynamicBotService)

	// Initialize resource group membership use cases (need node and agent repos)
	manageNodesUC := resourceUsecases.NewManageResourceGroupNodesUseCase(resourceGroupRepo, nodeRepoImpl, subscriptionPlanRepo, log)
	manageAgentsUC := resourceUsecases.NewManageResourceGroupForwardAgentsUseCase(resourceGroupRepo, forwardAgentRepo, subscriptionPlanRepo, log)
	manageRulesUC := resourceUsecases.NewManageResourceGroupForwardRulesUseCase(resourceGroupRepo, forwardRuleRepo, subscriptionPlanRepo, log)

	// Initialize admin resource group handler
	adminResourceGroupHandler := adminHandlers.NewResourceGroupHandler(
		createResourceGroupUC, getResourceGroupUC, listResourceGroupsUC,
		updateResourceGroupUC, deleteResourceGroupUC, updateResourceGroupStatusUC,
		manageNodesUC, manageAgentsUC, manageRulesUC,
		subscriptionPlanRepo, log,
	)

	// Initialize admin traffic stats use cases (uses subscription_usage_stats table + Redis hourly buckets)
	getTrafficOverviewUC := adminUsecases.NewGetTrafficOverviewUseCase(
		subscriptionUsageStatsRepo, hourlyTrafficCache, subscriptionRepo, userRepo, nodeRepoImpl, forwardRuleRepo, log,
	)
	getUserTrafficStatsUC := adminUsecases.NewGetUserTrafficStatsUseCase(
		subscriptionUsageStatsRepo, hourlyTrafficCache, subscriptionRepo, userRepo, log,
	)
	getSubscriptionTrafficStatsUC := adminUsecases.NewGetSubscriptionTrafficStatsUseCase(
		subscriptionUsageStatsRepo, hourlyTrafficCache, subscriptionRepo, userRepo, subscriptionPlanRepo, log,
	)
	getAdminNodeTrafficStatsUC := adminUsecases.NewGetAdminNodeTrafficStatsUseCase(
		subscriptionUsageStatsRepo, hourlyTrafficCache, nodeRepoImpl, log,
	)
	getTrafficRankingUC := adminUsecases.NewGetTrafficRankingUseCase(
		subscriptionUsageStatsRepo, hourlyTrafficCache, subscriptionRepo, userRepo, log,
	)
	getTrafficTrendUC := adminUsecases.NewGetTrafficTrendUseCase(
		subscriptionUsageStatsRepo, hourlyTrafficCache, log,
	)

	// Initialize admin traffic stats handler
	adminTrafficStatsHandler := adminHandlers.NewTrafficStatsHandler(
		getTrafficOverviewUC,
		getUserTrafficStatsUC,
		getSubscriptionTrafficStatsUC,
		getAdminNodeTrafficStatsUC,
		getTrafficRankingUC,
		getTrafficTrendUC,
		log,
	)

	// Initialize forward rule components (configSyncService will be injected after creation)
	var createForwardRuleUC *forwardUsecases.CreateForwardRuleUseCase
	var getForwardRuleUC *forwardUsecases.GetForwardRuleUseCase
	var updateForwardRuleUC *forwardUsecases.UpdateForwardRuleUseCase
	var deleteForwardRuleUC *forwardUsecases.DeleteForwardRuleUseCase
	var listForwardRulesUC *forwardUsecases.ListForwardRulesUseCase
	var enableForwardRuleUC *forwardUsecases.EnableForwardRuleUseCase
	var disableForwardRuleUC *forwardUsecases.DisableForwardRuleUseCase
	var resetForwardTrafficUC *forwardUsecases.ResetForwardRuleTrafficUseCase

	// forwardRuleHandler will be initialized later after probeService is available

	// Initialize forward agent components

	createForwardAgentUC := forwardUsecases.NewCreateForwardAgentUseCase(forwardAgentRepo, agentTokenSvc, log)
	// Initialize forward agent status adapter early for getForwardAgentUC
	forwardAgentStatusAdapter := adapters.NewForwardAgentStatusAdapter(redisClient, log)
	ruleSyncStatusAdapter := adapters.NewRuleSyncStatusAdapter(redisClient, log)
	getForwardAgentUC := forwardUsecases.NewGetForwardAgentUseCase(forwardAgentRepo, forwardAgentStatusAdapter, log)
	// updateForwardAgentUC will be initialized later after configSyncService is available
	var updateForwardAgentUC *forwardUsecases.UpdateForwardAgentUseCase
	deleteForwardAgentUC := forwardUsecases.NewDeleteForwardAgentUseCase(forwardAgentRepo, forwardRuleRepo, log)
	listForwardAgentsUC := forwardUsecases.NewListForwardAgentsUseCase(forwardAgentRepo, forwardAgentStatusAdapter, forwardAgentReleaseService, log)
	enableForwardAgentUC := forwardUsecases.NewEnableForwardAgentUseCase(forwardAgentRepo, log)
	disableForwardAgentUC := forwardUsecases.NewDisableForwardAgentUseCase(forwardAgentRepo, log)
	regenerateForwardAgentTokenUC := forwardUsecases.NewRegenerateForwardAgentTokenUseCase(forwardAgentRepo, agentTokenSvc, log)
	validateForwardAgentTokenUC := forwardUsecases.NewValidateForwardAgentTokenUseCase(forwardAgentRepo, log)

	// Initialize agent last seen updater and agent info updater for status reporting
	agentLastSeenUpdater := adapters.NewAgentLastSeenUpdaterAdapter(forwardAgentRepo)
	agentInfoUpdater := adapters.NewAgentInfoUpdaterAdapter(forwardAgentRepo)
	getAgentStatusUC := forwardUsecases.NewGetAgentStatusUseCase(forwardAgentRepo, forwardAgentStatusAdapter, log)
	getRuleOverallStatusUC := forwardUsecases.NewGetRuleOverallStatusUseCase(forwardRuleRepo, forwardAgentRepo, ruleSyncStatusAdapter, log)
	getForwardAgentTokenUC := forwardUsecases.NewGetForwardAgentTokenUseCase(forwardAgentRepo, log)
	generateInstallScriptUC := forwardUsecases.NewGenerateInstallScriptUseCase(forwardAgentRepo, log)

	// Server base URL for forward agent install script
	serverBaseURL := cfg.Server.GetBaseURL()

	// forwardAgentHandler will be initialized later after updateForwardAgentUC is available
	var forwardAgentHandler *forwardAgentCrudHandlers.Handler

	reportAgentStatusUC := forwardUsecases.NewReportAgentStatusUseCase(
		forwardAgentRepo,
		forwardAgentStatusAdapter,
		forwardAgentStatusAdapter, // statusQuerier (same adapter implements both interfaces)
		agentLastSeenUpdater,
		agentInfoUpdater,
		log,
	)
	reportRuleSyncStatusUC := forwardUsecases.NewReportRuleSyncStatusUseCase(
		forwardAgentRepo,
		ruleSyncStatusAdapter,
		forwardRuleRepo,
		log,
	)

	// Initialize forward traffic recorder adapter for writing forward traffic to Redis HourlyTrafficCache
	forwardTrafficRecorder := adapters.NewForwardTrafficRecorderAdapter(
		hourlyTrafficCache,
		log,
	)

	// Initialize forward agent API handler for client to fetch rules and report traffic
	forwardAgentAPIHandler := forwardAgentAPIHandlers.NewHandler(forwardRuleRepo, forwardAgentRepo, nodeRepoImpl, reportAgentStatusUC, reportRuleSyncStatusUC, forwardAgentStatusAdapter, cfg.Forward.TokenSigningSecret, forwardTrafficRecorder, log)

	// Initialize forward agent token middleware
	forwardAgentTokenMiddleware := middleware.NewForwardAgentTokenMiddleware(validateForwardAgentTokenUC, log)

	// Initialize agent hub for forward agent WebSocket connections (probe functionality)
	agentHub := services.NewAgentHub(log, &services.AgentHubConfig{
		NodeStatusTimeoutMs: 5000, // 5 seconds timeout for node status
	})

	// Register forward status handler to process forward agent status updates
	forwardStatusHandler := adapters.NewForwardStatusHandler(reportAgentStatusUC, log)
	agentHub.RegisterStatusHandler(forwardStatusHandler)

	// Initialize and register probe service for forward domain
	probeService := forwardServices.NewProbeService(forwardRuleRepo, forwardAgentRepo, nodeRepoImpl, forwardAgentStatusAdapter, agentHub, cfg.Forward.TokenSigningSecret, log)
	agentHub.RegisterMessageHandler(probeService)

	// Initialize and register config sync service for forward domain
	configSyncService := forwardServices.NewConfigSyncService(forwardRuleRepo, forwardAgentRepo, nodeRepoImpl, forwardAgentStatusAdapter, cfg.Forward.TokenSigningSecret, agentHub, log)
	agentHub.RegisterMessageHandler(configSyncService)

	// Register rule sync status handler for WebSocket-based status reporting
	agentHub.RegisterMessageHandler(reportRuleSyncStatusUC)

	// Initialize forward traffic cache for real-time traffic updates
	forwardTrafficCache := cache.NewRedisForwardTrafficCache(
		redisClient,
		forwardRuleRepo,
		log,
	)

	// Initialize rule traffic buffer for batching traffic updates
	ruleTrafficBuffer := forwardServices.NewRuleTrafficBuffer(forwardTrafficCache, log)
	ruleTrafficBuffer.Start()

	// Initialize and register traffic message handler for WebSocket traffic updates
	trafficMessageHandler := services.NewTrafficMessageHandler(
		ruleTrafficBuffer,
		forwardRuleRepo,
		forwardTrafficRecorder,
		log,
	)
	agentHub.RegisterMessageHandler(trafficMessageHandler)

	// Create done channel for rule traffic flush scheduler
	ruleTrafficFlushDone := make(chan struct{})

	// Start rule traffic flush scheduler (Redis -> MySQL)
	go func() {
		ticker := time.NewTicker(2 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				ctx := context.Background()
				if err := forwardTrafficCache.FlushToDatabase(ctx); err != nil {
					log.Errorw("failed to flush rule traffic to database", "error", err)
				}
			case <-ruleTrafficFlushDone:
				return
			}
		}
	}()

	// Set port change notifier for exit agent port change detection
	reportAgentStatusUC.SetPortChangeNotifier(configSyncService)

	// Set node address change notifier for node IP change detection
	reportNodeStatusUC.SetAddressChangeNotifier(configSyncService)

	// Set node address change notifier for node update use case
	updateNodeUC.SetAddressChangeNotifier(configSyncService)

	// Now initialize updateForwardAgentUC with configSyncService for address change and config change notification
	updateForwardAgentUC = forwardUsecases.NewUpdateForwardAgentUseCase(forwardAgentRepo, resourceGroupRepo, configSyncService, configSyncService, log)

	// Now initialize forwardAgentHandler after updateForwardAgentUC is available
	forwardAgentHandler = forwardAgentCrudHandlers.NewHandler(
		createForwardAgentUC,
		getForwardAgentUC,
		listForwardAgentsUC,
		updateForwardAgentUC,
		deleteForwardAgentUC,
		enableForwardAgentUC,
		disableForwardAgentUC,
		regenerateForwardAgentTokenUC,
		getForwardAgentTokenUC,
		getAgentStatusUC,
		getRuleOverallStatusUC,
		generateInstallScriptUC,
		serverBaseURL,
	)

	// Initialize version handlers
	forwardAgentVersionHandler := forwardAgentCrudHandlers.NewVersionHandler(
		forwardAgentRepo,
		forwardAgentReleaseService,
		agentHub,
		log,
	)
	nodeVersionHandler := nodeHandlers.NewNodeVersionHandler(
		nodeRepoImpl,
		nodeAgentReleaseService,
		agentHub,
		log,
	)

	// Now initialize forward rule use cases with configSyncService
	createForwardRuleUC = forwardUsecases.NewCreateForwardRuleUseCase(forwardRuleRepo, forwardAgentRepo, nodeRepoImpl, resourceGroupRepo, subscriptionPlanRepo, configSyncService, log)
	getForwardRuleUC = forwardUsecases.NewGetForwardRuleUseCase(forwardRuleRepo, forwardAgentRepo, nodeRepoImpl, resourceGroupRepo, log)
	updateForwardRuleUC = forwardUsecases.NewUpdateForwardRuleUseCase(forwardRuleRepo, forwardAgentRepo, nodeRepoImpl, resourceGroupRepo, subscriptionPlanRepo, subscriptionRepo, configSyncService, log)
	deleteForwardRuleUC = forwardUsecases.NewDeleteForwardRuleUseCase(forwardRuleRepo, forwardTrafficCache, configSyncService, log)
	listForwardRulesUC = forwardUsecases.NewListForwardRulesUseCase(forwardRuleRepo, forwardAgentRepo, nodeRepoImpl, resourceGroupRepo, ruleSyncStatusAdapter, log)
	enableForwardRuleUC = forwardUsecases.NewEnableForwardRuleUseCase(forwardRuleRepo, configSyncService, log)
	disableForwardRuleUC = forwardUsecases.NewDisableForwardRuleUseCase(forwardRuleRepo, configSyncService, log)
	resetForwardTrafficUC = forwardUsecases.NewResetForwardRuleTrafficUseCase(forwardRuleRepo, log)

	// Initialize user forward rule use cases
	createUserForwardRuleUC := forwardUsecases.NewCreateUserForwardRuleUseCase(
		forwardRuleRepo,
		forwardAgentRepo,
		nodeRepoImpl,
		configSyncService,
		log,
	)
	listUserForwardRulesUC := forwardUsecases.NewListUserForwardRulesUseCase(
		forwardRuleRepo,
		forwardAgentRepo,
		nodeRepoImpl,
		ruleSyncStatusAdapter,
		log,
	)
	txMgr := shareddb.NewTransactionManager(db)
	reorderForwardRulesUC := forwardUsecases.NewReorderForwardRulesUseCase(
		forwardRuleRepo,
		txMgr,
		log,
	)
	getUserForwardUsageUC := forwardUsecases.NewGetUserForwardUsageUseCase(
		forwardRuleRepo,
		subscriptionRepo,
		subscriptionPlanRepo,
		subscriptionUsageRepo,
		subscriptionUsageStatsRepo,
		hourlyTrafficCache,
		log,
	)

	// Initialize traffic limit enforcement service
	trafficLimitEnforcementSvc := forwardServices.NewTrafficLimitEnforcementService(
		forwardRuleRepo,
		subscriptionRepo,
		subscriptionUsageRepo,
		subscriptionUsageStatsRepo,
		hourlyTrafficCache,
		subscriptionPlanRepo,
		log,
	)

	// Initialize list user forward agents use case
	listUserForwardAgentsUC := forwardUsecases.NewListUserForwardAgentsUseCase(
		forwardAgentRepo,
		subscriptionRepo,
		subscriptionPlanRepo,
		resourceGroupRepo,
		log,
	)

	// Initialize batch forward rule use case (needed by both admin and user handlers)
	batchForwardRuleUC := forwardUsecases.NewBatchForwardRuleUseCase(
		forwardRuleRepo,
		createForwardRuleUC,
		createUserForwardRuleUC,
		deleteForwardRuleUC,
		enableForwardRuleUC,
		disableForwardRuleUC,
		updateForwardRuleUC,
		txMgr,
		log,
	)

	// Initialize user forward rule handler
	userForwardRuleHandler := forwardUserHandlers.NewHandler(
		createUserForwardRuleUC,
		listUserForwardRulesUC,
		getUserForwardUsageUC,
		updateForwardRuleUC,  // reuse existing
		deleteForwardRuleUC,  // reuse existing
		enableForwardRuleUC,  // reuse existing
		disableForwardRuleUC, // reuse existing
		getForwardRuleUC,     // reuse existing
		listUserForwardAgentsUC,
		reorderForwardRulesUC, // reuse existing
		batchForwardRuleUC,
	)

	// Initialize subscription forward rule use cases
	createSubscriptionForwardRuleUC := forwardUsecases.NewCreateSubscriptionForwardRuleUseCase(
		forwardRuleRepo,
		forwardAgentRepo,
		nodeRepoImpl,
		configSyncService,
		log,
	)
	listSubscriptionForwardRulesUC := forwardUsecases.NewListSubscriptionForwardRulesUseCase(
		forwardRuleRepo,
		forwardAgentRepo,
		nodeRepoImpl,
		subscriptionRepo,
		resourceGroupRepo,
		ruleSyncStatusAdapter,
		log,
	)
	getSubscriptionForwardUsageUC := forwardUsecases.NewGetSubscriptionForwardUsageUseCase(
		forwardRuleRepo,
		subscriptionRepo,
		subscriptionPlanRepo,
		subscriptionUsageRepo,
		subscriptionUsageStatsRepo,
		hourlyTrafficCache,
		log,
	)

	// Initialize subscription forward rule handler
	subscriptionForwardRuleHandler := forwardSubscriptionHandlers.NewHandler(
		createSubscriptionForwardRuleUC,
		listSubscriptionForwardRulesUC,
		getSubscriptionForwardUsageUC,
		updateForwardRuleUC,   // reuse existing
		deleteForwardRuleUC,   // reuse existing
		enableForwardRuleUC,   // reuse existing
		disableForwardRuleUC,  // reuse existing
		getForwardRuleUC,      // reuse existing
		reorderForwardRulesUC, // reuse existing
	)

	// Note: External forward rules have been merged into forward_rules table with rule_type='external'
	// The separate externalforward module has been removed.

	// Initialize forward rule owner middleware
	forwardRuleOwnerMiddleware := middleware.NewForwardRuleOwnerMiddleware(
		forwardRuleRepo,
		log,
	)

	// forwardQuotaMiddleware will be initialized after subscriptionTrafficCache is created
	var forwardQuotaMiddleware *middleware.ForwardQuotaMiddleware

	// Initialize forward rule handler (after probeService is available)
	forwardRuleHandler := forwardRuleHandlers.NewHandler(
		createForwardRuleUC,
		getForwardRuleUC,
		updateForwardRuleUC,
		deleteForwardRuleUC,
		listForwardRulesUC,
		enableForwardRuleUC,
		disableForwardRuleUC,
		resetForwardTrafficUC,
		reorderForwardRulesUC,
		batchForwardRuleUC,
		probeService,
	)

	// Initialize agent hub handler
	agentHubHandler := forwardAgentHubHandlers.NewHandler(agentHub, forwardAgentRepo, log)

	// Initialize node status handler and register to agent hub
	nodeStatusHandler := adapters.NewNodeStatusHandler(systemStatusUpdater, nodeRepoImpl, log)
	agentHub.RegisterNodeStatusHandler(nodeStatusHandler)

	// Initialize node config sync service for pushing config to node agents
	nodeConfigSyncService := nodeServices.NewNodeConfigSyncService(nodeRepoImpl, agentHub, log)

	// Initialize subscription sync service for pushing subscription changes to node agents
	subscriptionSyncService := nodeServices.NewSubscriptionSyncService(nodeRepoImpl, subscriptionRepo, resourceGroupRepo, agentHub, log)

	// Initialize Redis Pub/Sub event bus for cross-instance subscription synchronization
	subscriptionEventBus := pubsub.NewRedisSubscriptionEventBus(redisClient, log)

	// Set event publisher on subscription sync service for cross-instance sync
	subscriptionSyncService.SetEventPublisher(subscriptionEventBus)

	// Set deactivation notifier on node traffic limit enforcement service
	nodeTrafficLimitEnforcementSvc.SetDeactivationNotifier(subscriptionSyncService)

	// Initialize subscription event handler for processing events from other instances
	subscriptionEventHandler := nodeServices.NewSubscriptionEventHandler(
		subscriptionRepo,
		subscriptionSyncService,
		log,
	)

	// Start subscription event subscriber in background
	subscriptionEventHandler.StartSubscriber(context.Background(), subscriptionEventBus)

	// Initialize admin hub for SSE connections to frontend (must be before callbacks)
	adminHub := services.NewAdminHub(log, &services.AdminHubConfig{
		StatusThrottleMs: 1000, // 1 second throttle for node status updates
		AgentBroadcastMs: 1000, // 1 second interval for aggregated agent status broadcast
		NodeBroadcastMs:  1000, // 1 second interval for aggregated node status broadcast
	})

	// Set AdminHub on nodeStatusHandler for SSE broadcasting
	nodeStatusHandler.SetAdminHub(adminHub, &nodeSIDResolverAdapter{repo: nodeRepoImpl})

	// Set AdminHub on forwardStatusHandler for SSE broadcasting
	forwardStatusHandler.SetAdminHub(adminHub, &agentSIDResolverAdapter{repo: forwardAgentRepo})

	// Set AgentStatusQuerier on AdminHub for aggregated SSE broadcasting
	agentStatusQuerierAdapter := adapters.NewAgentStatusQuerierAdapter(forwardAgentRepo, forwardAgentStatusAdapter, log)
	adminHub.SetAgentStatusQuerier(agentStatusQuerierAdapter)

	// Set NodeStatusQuerier on AdminHub for aggregated SSE broadcasting
	nodeStatusQuerierAdapter := adapters.NewNodeStatusQuerierAdapter(nodeRepoImpl, nodeStatusQuerier, log)
	adminHub.SetNodeStatusQuerier(nodeStatusQuerierAdapter)

	// Set OnNodeOnline callback to sync config, broadcast SSE event, and send Telegram notification
	agentHub.SetOnNodeOnline(func(nodeID uint) {
		ctx := context.Background()

		// Sync config to node
		if err := nodeConfigSyncService.FullSyncToNode(ctx, nodeID); err != nil {
			log.Warnw("failed to sync config to node on connect",
				"node_id", nodeID,
				"error", err,
			)
		}

		// Get node info
		n, err := nodeRepoImpl.GetByID(ctx, nodeID)
		if err != nil {
			log.Warnw("failed to get node for SSE broadcast",
				"node_id", nodeID,
				"error", err,
			)
			return
		}
		if n == nil {
			return
		}

		// Broadcast SSE event
		adminHub.BroadcastNodeOnline(n.SID(), n.Name())

		// Send Telegram notification (real-time online alert)
		cmd := telegramAdminApp.NotifyNodeOnlineCommand{
			NodeID:           nodeID,
			NodeSID:          n.SID(),
			NodeName:         n.Name(),
			MuteNotification: n.MuteNotification(),
		}
		if err := adminNotificationServiceDDD.NotifyNodeOnline(ctx, cmd); err != nil {
			log.Errorw("failed to send node online notification",
				"node_sid", n.SID(),
				"error", err,
			)
		}
	})

	// Set OnNodeOffline callback to broadcast SSE event and send Telegram notification
	agentHub.SetOnNodeOffline(func(nodeID uint) {
		ctx := context.Background()

		// Get node info
		n, err := nodeRepoImpl.GetByID(ctx, nodeID)
		if err != nil {
			log.Warnw("failed to get node for offline notification",
				"node_id", nodeID,
				"error", err,
			)
			return
		}
		if n == nil {
			return
		}

		// Broadcast SSE event
		adminHub.BroadcastNodeOffline(n.SID(), n.Name())

		// Send Telegram notification (real-time offline alert)
		var lastSeenAt time.Time
		if n.LastSeenAt() != nil {
			lastSeenAt = *n.LastSeenAt()
		} else {
			lastSeenAt = biztime.NowUTC()
		}
		cmd := telegramAdminApp.NotifyNodeOfflineCommand{
			NodeID:           nodeID,
			NodeSID:          n.SID(),
			NodeName:         n.Name(),
			LastSeenAt:       lastSeenAt,
			OfflineMinutes:   0, // Just disconnected
			MuteNotification: n.MuteNotification(),
		}
		if err := adminNotificationServiceDDD.NotifyNodeOffline(ctx, cmd); err != nil {
			log.Errorw("failed to send node offline notification",
				"node_sid", n.SID(),
				"error", err,
			)
		}
	})

	// Set OnAgentOnline callback to sync config, broadcast SSE event, and send Telegram notification
	agentHub.SetOnAgentOnline(func(agentID uint) {
		ctx := context.Background()

		// Sync config to forward agent
		if err := configSyncService.FullSyncToAgent(ctx, agentID); err != nil {
			log.Warnw("failed to sync config to agent on connect",
				"agent_id", agentID,
				"error", err,
			)
		}

		// Get agent info
		agent, err := forwardAgentRepo.GetByID(ctx, agentID)
		if err != nil {
			log.Warnw("failed to get agent for SSE broadcast",
				"agent_id", agentID,
				"error", err,
			)
			return
		}
		if agent == nil {
			return
		}

		// Broadcast SSE event
		adminHub.BroadcastForwardAgentOnline(agent.SID(), agent.Name())

		// Send Telegram notification (real-time online alert)
		cmd := telegramAdminApp.NotifyAgentOnlineCommand{
			AgentID:          agentID,
			AgentSID:         agent.SID(),
			AgentName:        agent.Name(),
			MuteNotification: agent.MuteNotification(),
		}
		if err := adminNotificationServiceDDD.NotifyAgentOnline(ctx, cmd); err != nil {
			log.Errorw("failed to send agent online notification",
				"agent_sid", agent.SID(),
				"error", err,
			)
		}
	})

	// Set OnAgentOffline callback to broadcast SSE event and send Telegram notification
	agentHub.SetOnAgentOffline(func(agentID uint) {
		ctx := context.Background()

		// Get agent info
		agent, err := forwardAgentRepo.GetByID(ctx, agentID)
		if err != nil {
			log.Warnw("failed to get agent for offline notification",
				"agent_id", agentID,
				"error", err,
			)
			return
		}
		if agent == nil {
			return
		}

		// Broadcast SSE event
		adminHub.BroadcastForwardAgentOffline(agent.SID(), agent.Name())

		// Send Telegram notification (real-time offline alert)
		cmd := telegramAdminApp.NotifyAgentOfflineCommand{
			AgentID:          agentID,
			AgentSID:         agent.SID(),
			AgentName:        agent.Name(),
			LastSeenAt:       biztime.NowUTC(), // Use current time as agent doesn't track LastSeenAt
			OfflineMinutes:   0,                // Just disconnected
			MuteNotification: agent.MuteNotification(),
		}
		if err := adminNotificationServiceDDD.NotifyAgentOffline(ctx, cmd); err != nil {
			log.Errorw("failed to send agent offline notification",
				"agent_sid", agent.SID(),
				"error", err,
			)
		}
	})

	// Set config change notifier for node update use case
	updateNodeUC.SetConfigChangeNotifier(nodeConfigSyncService)

	// Set subscription change notifier for subscription use cases
	createSubscriptionUC.SetSubscriptionNotifier(subscriptionSyncService)
	activateSubscriptionUC.SetSubscriptionNotifier(subscriptionSyncService)
	cancelSubscriptionUC.SetSubscriptionNotifier(subscriptionSyncService)
	suspendSubscriptionUC.SetSubscriptionNotifier(subscriptionSyncService)
	unsuspendSubscriptionUC.SetSubscriptionNotifier(subscriptionSyncService)
	unsuspendSubscriptionUC.SetQuotaCacheManager(quotaCacheSyncService)
	resetSubscriptionUsageUC.SetSubscriptionNotifier(subscriptionSyncService)
	resetSubscriptionUsageUC.SetQuotaCacheManager(quotaCacheSyncService)
	renewSubscriptionUC.SetSubscriptionNotifier(subscriptionSyncService)

	// Initialize QuotaService for unified quota calculation
	quotaService := subscriptionUsecases.NewQuotaService(
		subscriptionRepo,
		subscriptionUsageStatsRepo,
		hourlyTrafficCache,
		subscriptionPlanRepo,
		log,
	)

	// Initialize forward quota middleware with QuotaService for unified quota check
	forwardQuotaMiddleware = middleware.NewForwardQuotaMiddleware(
		forwardRuleRepo,
		subscriptionRepo,
		subscriptionPlanRepo,
		quotaService,
		log,
	)

	// Create done channel for subscription traffic flush scheduler
	subscriptionTrafficFlushDone := make(chan struct{})

	// Start subscription traffic flush scheduler (Redis -> MySQL)
	go func() {
		ticker := time.NewTicker(2 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				ctx := context.Background()
				if err := subscriptionTrafficCache.FlushToDatabase(ctx); err != nil {
					log.Errorw("failed to flush subscription traffic to database", "error", err)
				}
			case <-subscriptionTrafficFlushDone:
				return
			}
		}
	}()

	// Initialize node hub handler with traffic buffer support
	nodeHubHandler := nodeHandlers.NewNodeHubHandler(agentHub, nodeRepoImpl, subscriptionTrafficBuffer, subscriptionIDResolver, log)
	nodeHubHandler.SetAddressChangeNotifier(configSyncService)
	nodeHubHandler.SetIPUpdater(nodeRepoImpl)
	nodeHubHandler.SetSubscriptionSyncer(subscriptionSyncService)
	nodeHubHandler.SetTrafficEnforcer(nodeTrafficLimitEnforcementSvc)
	nodeHubHandler.SetQuotaCache(nodeQuotaCacheAdapter)
	nodeHubHandler.SetQuotaLoader(nodeQuotaLoaderAdapter)
	nodeHubHandler.SetUsageReader(nodeUsageReaderAdapter)

	// Initialize node SSE handler
	nodeSSEHandler := nodeHandlers.NewNodeSSEHandler(adminHub, log)

	// Initialize forward agent SSE handler
	forwardAgentSSEHandler := forwardAgentCrudHandlers.NewForwardAgentSSEHandler(adminHub, log)

	return &Router{
		engine:                         engine,
		userHandler:                    userHandler,
		authHandler:                    authHandler,
		passkeyHandler:                 passkeyHandler,
		profileHandler:                 profileHandler,
		dashboardHandler:               dashboardHandler,
		subscriptionHandler:            subscriptionHandler,
		adminSubscriptionHandler:       adminSubscriptionHandler,
		adminResourceGroupHandler:      adminResourceGroupHandler,
		adminTrafficStatsHandler:       adminTrafficStatsHandler,
		adminTelegramHandler:           adminTelegramHandler,
		adminNotificationService:       adminNotificationServiceDDD,
		settingHandler:                 settingHandler,
		settingService:                 settingServiceDDD,
		planHandler:                    planHandler,
		subscriptionTokenHandler:       subscriptionTokenHandler,
		paymentHandler:                 paymentHandler,
		nodeHandler:                    nodeHandler,
		nodeSubscriptionHandler:        nodeSubscriptionHandler,
		userNodeHandler:                userNodeHandler,
		agentHandler:                   agentHandler,
		ticketHandler:                  ticketHandler,
		notificationHandler:            notificationHandler,
		telegramHandler:                telegramHandler,
		telegramService:                telegramServiceDDD,
		telegramBotManager:             telegramBotManager,
		forwardRuleHandler:             forwardRuleHandler,
		forwardAgentHandler:            forwardAgentHandler,
		forwardAgentVersionHandler:     forwardAgentVersionHandler,
		forwardAgentSSEHandler:         forwardAgentSSEHandler,
		forwardAgentAPIHandler:         forwardAgentAPIHandler,
		userForwardRuleHandler:         userForwardRuleHandler,
		subscriptionForwardRuleHandler: subscriptionForwardRuleHandler,
		agentHub:                       agentHub,
		agentHubHandler:                agentHubHandler,
		nodeHubHandler:                 nodeHubHandler,
		nodeVersionHandler:             nodeVersionHandler,
		nodeSSEHandler:                 nodeSSEHandler,
		adminHub:                       adminHub,
		configSyncService:              configSyncService,
		trafficLimitEnforcementSvc:     trafficLimitEnforcementSvc,
		forwardTrafficCache:            forwardTrafficCache,
		ruleTrafficBuffer:              ruleTrafficBuffer,
		ruleTrafficFlushDone:           ruleTrafficFlushDone,
		subscriptionTrafficCache:       subscriptionTrafficCache,
		subscriptionTrafficBuffer:      subscriptionTrafficBuffer,
		subscriptionTrafficFlushDone:   subscriptionTrafficFlushDone,
		adminNotificationScheduler:     adminNotificationScheduler,
		usageAggregationScheduler:      usageAggregationScheduler,
		logger:                         log,
		authMiddleware:                 authMiddleware,
		subscriptionOwnerMiddleware:    subscriptionOwnerMiddleware,
		nodeTokenMiddleware:            nodeTokenMiddleware,
		nodeOwnerMiddleware:            nodeOwnerMiddleware,
		nodeQuotaMiddleware:            nodeQuotaMiddleware,
		forwardAgentTokenMiddleware:    forwardAgentTokenMiddleware,
		forwardRuleOwnerMiddleware:     forwardRuleOwnerMiddleware,
		forwardQuotaMiddleware:         forwardQuotaMiddleware,
		rateLimiter:                    rateLimiter,
		oauthManager:                   oauthManager,
		emailManager:                   emailManager,
	}
}

// SetupRoutes configures all HTTP routes
func (r *Router) SetupRoutes(cfg *config.Config) {
	r.engine.Use(middleware.Logger())
	r.engine.Use(middleware.Recovery())
	r.engine.Use(middleware.CORS(cfg.Server.AllowedOrigins))

	r.engine.GET("/health", r.userHandler.HealthCheck)
	r.engine.GET("/version", r.userHandler.Version)

	auth := r.engine.Group("/auth")
	{
		auth.POST("/register", r.rateLimiter.Limit(), r.authHandler.Register)
		auth.POST("/login", r.rateLimiter.Limit(), r.authHandler.Login)
		auth.POST("/verify-email", r.authHandler.VerifyEmail)
		auth.GET("/verify-email", r.authHandler.VerifyEmail)
		auth.POST("/forgot-password", r.rateLimiter.Limit(), r.authHandler.ForgotPassword)
		auth.POST("/reset-password", r.authHandler.ResetPassword)

		auth.GET("/oauth/:provider", r.authHandler.InitiateOAuth)
		auth.GET("/oauth/:provider/callback", r.authHandler.HandleOAuthCallback)

		auth.POST("/refresh", r.authHandler.RefreshToken)
		auth.POST("/logout", r.authMiddleware.RequireAuth(), r.authHandler.Logout)
		auth.GET("/me", r.authMiddleware.RequireAuth(), r.authHandler.GetCurrentUser)

		// Passkey (WebAuthn) authentication routes
		if r.passkeyHandler != nil {
			auth.POST("/passkey/register/start", r.authMiddleware.RequireAuth(), r.passkeyHandler.StartRegistration)
			auth.POST("/passkey/register/finish", r.authMiddleware.RequireAuth(), r.passkeyHandler.FinishRegistration)
			auth.POST("/passkey/login/start", r.rateLimiter.Limit(), r.passkeyHandler.StartAuthentication)
			auth.POST("/passkey/login/finish", r.rateLimiter.Limit(), r.passkeyHandler.FinishAuthentication)
			// Passkey signup (new user registration without password)
			auth.POST("/passkey/signup/start", r.rateLimiter.Limit(), r.passkeyHandler.StartSignup)
			auth.POST("/passkey/signup/finish", r.rateLimiter.Limit(), r.passkeyHandler.FinishSignup)
		}
	}

	users := r.engine.Group("/users")
	users.Use(r.authMiddleware.RequireAuth())
	{
		// IMPORTANT: Register specific paths BEFORE parameterized paths to avoid route conflicts

		// Collection operations (no ID parameter)
		users.POST("", authorization.RequireAdmin(), r.userHandler.CreateUser)
		users.GET("", authorization.RequireAdmin(), r.userHandler.ListUsers)

		// Specific named endpoints (must come BEFORE /:id to avoid conflicts)
		users.PATCH("/me", r.profileHandler.UpdateProfile)
		users.PUT("/me/password", r.profileHandler.ChangePassword)
		users.GET("/me/dashboard", r.dashboardHandler.GetDashboard)

		// Passkey management routes
		if r.passkeyHandler != nil {
			users.GET("/me/passkeys", r.passkeyHandler.ListPasskeys)
			users.DELETE("/me/passkeys/:id", r.passkeyHandler.DeletePasskey)
		}

		users.GET("/email/:email", authorization.RequireAdmin(), r.userHandler.GetUserByEmail)

		// Generic parameterized routes (must come LAST)
		users.GET("/:id", authorization.RequireAdmin(), r.userHandler.GetUser)
		users.PATCH("/:id", authorization.RequireAdmin(), r.userHandler.UpdateUser)
		users.DELETE("/:id", authorization.RequireAdmin(), r.userHandler.DeleteUser)
		users.PATCH("/:id/password", authorization.RequireAdmin(), r.userHandler.AdminResetPassword)
	}

	// Admin subscription routes - full CRUD for all subscriptions
	adminSubscriptions := r.engine.Group("/admin/subscriptions")
	adminSubscriptions.Use(r.authMiddleware.RequireAuth(), authorization.RequireAdmin())
	{
		adminSubscriptions.POST("", r.adminSubscriptionHandler.Create)
		adminSubscriptions.GET("", r.adminSubscriptionHandler.List)
		adminSubscriptions.GET("/:id", r.adminSubscriptionHandler.Get)
		adminSubscriptions.PATCH("/:id/status", r.adminSubscriptionHandler.UpdateStatus)
		adminSubscriptions.PATCH("/:id/plan", r.adminSubscriptionHandler.ChangePlan)
		adminSubscriptions.POST("/:id/suspend", r.adminSubscriptionHandler.Suspend)
		adminSubscriptions.POST("/:id/unsuspend", r.adminSubscriptionHandler.Unsuspend)
		adminSubscriptions.POST("/:id/reset-usage", r.adminSubscriptionHandler.ResetUsage)
		adminSubscriptions.DELETE("/:id", r.adminSubscriptionHandler.Delete)
	}

	// Admin resource group routes - full CRUD for resource groups
	adminResourceGroups := r.engine.Group("/admin/resource-groups")
	adminResourceGroups.Use(r.authMiddleware.RequireAuth(), authorization.RequireAdmin())
	{
		adminResourceGroups.POST("", r.adminResourceGroupHandler.Create)
		adminResourceGroups.GET("", r.adminResourceGroupHandler.List)
		adminResourceGroups.GET("/:id", r.adminResourceGroupHandler.Get)
		adminResourceGroups.PATCH("/:id", r.adminResourceGroupHandler.Update)
		adminResourceGroups.DELETE("/:id", r.adminResourceGroupHandler.Delete)
		adminResourceGroups.POST("/:id/activate", r.adminResourceGroupHandler.Activate)
		adminResourceGroups.POST("/:id/deactivate", r.adminResourceGroupHandler.Deactivate)

		// Node membership management
		adminResourceGroups.POST("/:id/nodes", r.adminResourceGroupHandler.AddNodes)
		adminResourceGroups.DELETE("/:id/nodes", r.adminResourceGroupHandler.RemoveNodes)
		adminResourceGroups.GET("/:id/nodes", r.adminResourceGroupHandler.ListNodes)

		// Forward agent membership management
		adminResourceGroups.POST("/:id/forward-agents", r.adminResourceGroupHandler.AddForwardAgents)
		adminResourceGroups.DELETE("/:id/forward-agents", r.adminResourceGroupHandler.RemoveForwardAgents)
		adminResourceGroups.GET("/:id/forward-agents", r.adminResourceGroupHandler.ListForwardAgents)

		// Forward rule membership management
		adminResourceGroups.POST("/:id/forward-rules", r.adminResourceGroupHandler.AddForwardRules)
		adminResourceGroups.DELETE("/:id/forward-rules", r.adminResourceGroupHandler.RemoveForwardRules)
		adminResourceGroups.GET("/:id/forward-rules", r.adminResourceGroupHandler.ListForwardRules)
	}

	// Admin traffic stats routes
	adminTrafficStats := r.engine.Group("/admin/traffic-stats")
	adminTrafficStats.Use(r.authMiddleware.RequireAuth(), authorization.RequireAdmin())
	{
		adminTrafficStats.GET("/overview", r.adminTrafficStatsHandler.GetOverview)
		adminTrafficStats.GET("/users", r.adminTrafficStatsHandler.GetUserStats)
		adminTrafficStats.GET("/subscriptions", r.adminTrafficStatsHandler.GetSubscriptionStats)
		adminTrafficStats.GET("/nodes", r.adminTrafficStatsHandler.GetNodeStats)
		adminTrafficStats.GET("/ranking/users", r.adminTrafficStatsHandler.GetUserRanking)
		adminTrafficStats.GET("/ranking/subscriptions", r.adminTrafficStatsHandler.GetSubscriptionRanking)
		adminTrafficStats.GET("/trend", r.adminTrafficStatsHandler.GetTrend)
	}

	// Admin telegram routes (only if handler is initialized)
	if r.adminTelegramHandler != nil {
		adminTelegram := r.engine.Group("/admin/telegram")
		adminTelegram.Use(r.authMiddleware.RequireAuth(), authorization.RequireAdmin())
		{
			adminTelegram.GET("/binding", r.adminTelegramHandler.GetBindingStatus)
			adminTelegram.DELETE("/binding", r.adminTelegramHandler.Unbind)
			adminTelegram.PATCH("/preferences", r.adminTelegramHandler.UpdatePreferences)
		}
	}

	// Note: Admin external forward rules routes have been removed.
	// External forward rules are now managed as forward_rules with rule_type='external'.

	// User subscription routes - only own subscriptions
	subscriptions := r.engine.Group("/subscriptions")
	subscriptions.Use(r.authMiddleware.RequireAuth())
	{
		// Collection operations (no ownership check needed)
		subscriptions.POST("", r.subscriptionHandler.CreateSubscription)
		subscriptions.GET("", r.subscriptionHandler.ListUserSubscriptions)

		// Operations on specific subscription (ownership verified by middleware)
		// :sid is subscription SID (sub_xxx format)
		subscriptionWithOwnership := subscriptions.Group("/:sid")
		subscriptionWithOwnership.Use(r.subscriptionOwnerMiddleware.RequireOwnership())
		{
			subscriptionWithOwnership.GET("", r.subscriptionHandler.GetSubscription)
			subscriptionWithOwnership.PATCH("/status", r.subscriptionHandler.UpdateStatus)
			subscriptionWithOwnership.PATCH("/plan", r.subscriptionHandler.ChangePlan)
			subscriptionWithOwnership.PUT("/link", r.subscriptionHandler.ResetLink)
			subscriptionWithOwnership.DELETE("", r.subscriptionHandler.DeleteSubscription)

			// Token sub-resource endpoints
			// :token_id is token SID (subtk_xxx format)
			subscriptionWithOwnership.POST("/tokens/:token_id/refresh", r.subscriptionTokenHandler.RefreshToken)
			subscriptionWithOwnership.DELETE("/tokens/:token_id", r.subscriptionTokenHandler.RevokeToken)
			subscriptionWithOwnership.POST("/tokens", r.subscriptionTokenHandler.GenerateToken)
			subscriptionWithOwnership.GET("/tokens", r.subscriptionTokenHandler.ListTokens)

			// Traffic statistics endpoint
			subscriptionWithOwnership.GET("/traffic-stats", r.subscriptionHandler.GetTrafficStats)
		}
	}

	payments := r.engine.Group("/payments")
	{
		payments.POST("/callback", r.paymentHandler.HandleCallback)

		paymentsProtected := payments.Group("")
		paymentsProtected.Use(r.authMiddleware.RequireAuth())
		{
			paymentsProtected.POST("", r.paymentHandler.CreatePayment)
		}
	}

	plans := r.engine.Group("/plans")
	{
		// IMPORTANT: Register specific paths BEFORE parameterized paths to avoid route conflicts
		// e.g., /public must come before /:id, /activate before /:id, etc.

		// Public endpoints (no authentication required)
		plans.GET("/public", r.planHandler.GetPublicPlans)

		// Protected endpoints (read operations)
		plansProtected := plans.Group("")
		plansProtected.Use(r.authMiddleware.RequireAuth())
		{
			// Read operations - available to all authenticated users
			plansProtected.GET("", r.planHandler.ListPlans)
			plansProtected.GET("/:id", r.planHandler.GetPlan)
			plansProtected.GET("/:id/pricings", r.planHandler.GetPlanPricings)
		}

		// Admin-only endpoints (write operations)
		plansAdmin := plans.Group("")
		plansAdmin.Use(r.authMiddleware.RequireAuth())
		plansAdmin.Use(authorization.RequireAdmin())
		{
			// Collection operations (no ID parameter)
			plansAdmin.POST("", r.planHandler.CreatePlan)

			// Specific action endpoints (must come BEFORE /:id to avoid conflicts)
			// Using PATCH for state changes as per RESTful best practices
			plansAdmin.PATCH("/:id/status", r.planHandler.UpdatePlanStatus)

			// Generic parameterized routes (must come LAST)
			plansAdmin.PUT("/:id", r.planHandler.UpdatePlan)
			plansAdmin.DELETE("/:id", r.planHandler.DeletePlan)
		}
	}

	routes.SetupNodeRoutes(r.engine, &routes.NodeRouteConfig{
		NodeHandler:         r.nodeHandler,
		NodeHubHandler:      r.nodeHubHandler,
		NodeVersionHandler:  r.nodeVersionHandler,
		NodeSSEHandler:      r.nodeSSEHandler,
		UserNodeHandler:     r.userNodeHandler,
		SubscriptionHandler: r.nodeSubscriptionHandler,
		AuthMiddleware:      r.authMiddleware,
		NodeTokenMW:         r.nodeTokenMiddleware,
		NodeOwnerMW:         r.nodeOwnerMiddleware,
		NodeQuotaMW:         r.nodeQuotaMiddleware,
		RateLimiter:         r.rateLimiter,
	})

	// RESTful Agent API - Modern API with resource-oriented design
	agentAPI := r.engine.Group("/agents")
	agentAPI.Use(r.nodeTokenMiddleware.RequireNodeTokenHeader())
	{
		// GET /agents/:nodesid/config - Get node configuration
		agentAPI.GET("/:nodesid/config", r.agentHandler.GetConfig)

		// GET /agents/:nodesid/subscriptions - Get active subscriptions for node
		agentAPI.GET("/:nodesid/subscriptions", r.agentHandler.GetSubscriptions)

		// POST /agents/:nodesid/traffic - Report subscription traffic data
		agentAPI.POST("/:nodesid/traffic", r.agentHandler.ReportTraffic)

		// PUT /agents/:nodesid/status - Update node system status
		agentAPI.PUT("/:nodesid/status", r.agentHandler.UpdateStatus)

		// PUT /agents/:nodesid/online-subscriptions - Update online subscriptions list
		agentAPI.PUT("/:nodesid/online-subscriptions", r.agentHandler.UpdateOnlineSubscriptions)
	}

	routes.SetupTicketRoutes(r.engine, &routes.TicketRouteConfig{
		TicketHandler:  r.ticketHandler,
		AuthMiddleware: r.authMiddleware,
	})

	routes.SetupNotificationRoutes(r.engine, &routes.NotificationRouteConfig{
		NotificationHandler: r.notificationHandler,
		AuthMiddleware:      r.authMiddleware,
	})

	// Setup Telegram routes (only if handler is initialized)
	if r.telegramHandler != nil {
		routes.SetupTelegramRoutes(r.engine, &routes.TelegramRouteConfig{
			Handler:        r.telegramHandler,
			AuthMiddleware: r.authMiddleware,
		})
	}

	routes.SetupForwardRoutes(r.engine, &routes.ForwardRouteConfig{
		ForwardRuleHandler:          r.forwardRuleHandler,
		ForwardAgentHandler:         r.forwardAgentHandler,
		ForwardAgentVersionHandler:  r.forwardAgentVersionHandler,
		ForwardAgentSSEHandler:      r.forwardAgentSSEHandler,
		ForwardAgentHubHandler:      r.agentHubHandler,
		ForwardAgentAPIHandler:      r.forwardAgentAPIHandler,
		UserForwardHandler:          r.userForwardRuleHandler,
		AuthMiddleware:              r.authMiddleware,
		ForwardAgentTokenMiddleware: r.forwardAgentTokenMiddleware,
		ForwardRuleOwnerMiddleware:  r.forwardRuleOwnerMiddleware,
		ForwardQuotaMiddleware:      r.forwardQuotaMiddleware,
	})

	// Subscription-scoped forward rules routes
	routes.SetupSubscriptionForwardRoutes(r.engine, &routes.SubscriptionForwardRouteConfig{
		SubscriptionForwardHandler:  r.subscriptionForwardRuleHandler,
		AuthMiddleware:              r.authMiddleware,
		SubscriptionOwnerMiddleware: r.subscriptionOwnerMiddleware,
		ForwardRuleOwnerMiddleware:  r.forwardRuleOwnerMiddleware,
		ForwardQuotaMiddleware:      r.forwardQuotaMiddleware,
	})

	// Note: External forward rules routes have been removed.
	// External rules are now part of forward_rules with rule_type='external'.

	routes.SetupAgentHubRoutes(r.engine, &routes.AgentHubRouteConfig{
		HubHandler:                  r.agentHubHandler,
		NodeHubHandler:              r.nodeHubHandler,
		ForwardAgentTokenMiddleware: r.forwardAgentTokenMiddleware,
		NodeTokenMiddleware:         r.nodeTokenMiddleware,
	})

	// Setup Setting routes (Admin only)
	if r.settingHandler != nil {
		routes.SetupSettingRoutes(r.engine, &routes.SettingRouteConfig{
			Handler:        r.settingHandler,
			AuthMiddleware: r.authMiddleware,
		})
	}
}

// GetEngine returns the Gin engine
func (r *Router) GetEngine() *gin.Engine {
	return r.engine
}

// Run starts the HTTP server
func (r *Router) Run(addr string) error {
	return r.engine.Run(addr)
}

// Shutdown gracefully shuts down the router
func (r *Router) Shutdown() {
	// Stop admin notification scheduler first
	if r.adminNotificationScheduler != nil {
		r.adminNotificationScheduler.Stop()
	}

	// Stop usage aggregation scheduler
	if r.usageAggregationScheduler != nil {
		r.usageAggregationScheduler.Stop()
	}

	// Stop telegram bot service manager if running
	if r.telegramBotManager != nil {
		r.telegramBotManager.Stop()
	}

	// Close all SSE connections first to allow HTTP server shutdown to proceed quickly
	if r.adminHub != nil {
		r.adminHub.Shutdown()
	}

	// Stop rule traffic flush scheduler goroutine
	if r.ruleTrafficFlushDone != nil {
		close(r.ruleTrafficFlushDone)
	}

	// Stop rule traffic buffer (flushes remaining data to Redis)
	if r.ruleTrafficBuffer != nil {
		r.ruleTrafficBuffer.Stop()
	}

	// Final flush forward traffic from Redis to MySQL
	if r.forwardTrafficCache != nil {
		ctx := context.Background()
		if err := r.forwardTrafficCache.FlushToDatabase(ctx); err != nil {
			r.logger.Errorw("failed to flush forward traffic to database on shutdown", "error", err)
		}
	}

	// Stop subscription traffic flush scheduler goroutine
	if r.subscriptionTrafficFlushDone != nil {
		close(r.subscriptionTrafficFlushDone)
	}

	// Stop subscription traffic buffer (flushes remaining data to Redis)
	if r.subscriptionTrafficBuffer != nil {
		r.subscriptionTrafficBuffer.Stop()
	}

	// Final flush subscription traffic from Redis to MySQL
	if r.subscriptionTrafficCache != nil {
		ctx := context.Background()
		if err := r.subscriptionTrafficCache.FlushToDatabase(ctx); err != nil {
			r.logger.Errorw("failed to flush subscription traffic to database on shutdown", "error", err)
		}
	}
}

// GetTelegramService returns the telegram service for scheduler use
func (r *Router) GetTelegramService() *telegramApp.ServiceDDD {
	return r.telegramService
}

// GetAdminNotificationService returns the admin notification service for scheduler use
func (r *Router) GetAdminNotificationService() *telegramAdminApp.ServiceDDD {
	return r.adminNotificationService
}

// GetTelegramBotManager returns the telegram bot service manager
func (r *Router) GetTelegramBotManager() *telegramInfra.BotServiceManager {
	return r.telegramBotManager
}

// GetSettingService returns the setting service
func (r *Router) GetSettingService() *settingApp.ServiceDDD {
	return r.settingService
}

// StartTelegramPolling starts the telegram bot service using BotServiceManager
func (r *Router) StartTelegramPolling(ctx context.Context) error {
	if r.telegramBotManager == nil {
		return nil
	}
	if err := r.telegramBotManager.Start(ctx); err != nil {
		return err
	}

	// Start admin notification scheduler after telegram bot manager
	if r.adminNotificationScheduler != nil {
		r.adminNotificationScheduler.Start(ctx)
	}

	return nil
}

// StartUsageAggregationScheduler starts the usage aggregation scheduler
func (r *Router) StartUsageAggregationScheduler(ctx context.Context) {
	if r.usageAggregationScheduler != nil {
		r.usageAggregationScheduler.Start(ctx)
	}
}

// nodeSIDResolverAdapter adapts node repository for SID resolution.
type nodeSIDResolverAdapter struct {
	repo node.NodeRepository
}

// GetSIDByID resolves node internal ID to Stripe-style SID.
func (a *nodeSIDResolverAdapter) GetSIDByID(nodeID uint) (string, bool) {
	ctx := context.Background()
	n, err := a.repo.GetByID(ctx, nodeID)
	if err != nil || n == nil {
		return "", false
	}
	return n.SID(), true
}

// agentSIDResolverAdapter adapts forward agent repository for SID resolution.
type agentSIDResolverAdapter struct {
	repo forward.AgentRepository
}

// GetSIDByID resolves forward agent internal ID to Stripe-style SID and name.
func (a *agentSIDResolverAdapter) GetSIDByID(agentID uint) (string, string, bool) {
	ctx := context.Background()
	agent, err := a.repo.GetByID(ctx, agentID)
	if err != nil || agent == nil {
		return "", "", false
	}
	return agent.SID(), agent.Name(), true
}

// botServiceProviderAdapter adapts BotServiceManager to satisfy telegramAdminApp.BotServiceProvider interface.
// The interface expects GetBotService() to return usecases.TelegramMessageSender,
// but BotServiceManager returns *BotService which implements the same method signature.
type botServiceProviderAdapter struct {
	manager *telegramInfra.BotServiceManager
}

// GetBotService returns the BotService as TelegramMessageSender interface.
func (a *botServiceProviderAdapter) GetBotService() telegramAdminUsecases.TelegramMessageSender {
	bs := a.manager.GetBotService()
	if bs == nil {
		return nil
	}
	return bs
}
