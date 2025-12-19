package http

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	forwardServices "github.com/orris-inc/orris/internal/application/forward/services"
	forwardUsecases "github.com/orris-inc/orris/internal/application/forward/usecases"
	nodeUsecases "github.com/orris-inc/orris/internal/application/node/usecases"
	notificationApp "github.com/orris-inc/orris/internal/application/notification"
	paymentGateway "github.com/orris-inc/orris/internal/application/payment/paymentgateway"
	paymentUsecases "github.com/orris-inc/orris/internal/application/payment/usecases"
	resourceUsecases "github.com/orris-inc/orris/internal/application/resource/usecases"
	subscriptionUsecases "github.com/orris-inc/orris/internal/application/subscription/usecases"
	"github.com/orris-inc/orris/internal/application/user"
	"github.com/orris-inc/orris/internal/application/user/helpers"
	"github.com/orris-inc/orris/internal/application/user/usecases"
	"github.com/orris-inc/orris/internal/infrastructure/adapters"
	"github.com/orris-inc/orris/internal/infrastructure/auth"
	"github.com/orris-inc/orris/internal/infrastructure/cache"
	"github.com/orris-inc/orris/internal/infrastructure/config"
	"github.com/orris-inc/orris/internal/infrastructure/email"
	"github.com/orris-inc/orris/internal/infrastructure/repository"
	"github.com/orris-inc/orris/internal/infrastructure/services"
	"github.com/orris-inc/orris/internal/infrastructure/template"
	"github.com/orris-inc/orris/internal/infrastructure/token"
	"github.com/orris-inc/orris/internal/interfaces/http/handlers"
	adminHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/admin"
	agentHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/agent"
	forwardHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/forward"
	nodeHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/node"
	ticketHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/ticket"
	"github.com/orris-inc/orris/internal/interfaces/http/middleware"
	"github.com/orris-inc/orris/internal/interfaces/http/routes"
	"github.com/orris-inc/orris/internal/shared/authorization"
	"github.com/orris-inc/orris/internal/shared/logger"
	"github.com/orris-inc/orris/internal/shared/services/markdown"
)

// Router represents the HTTP router configuration
type Router struct {
	engine                      *gin.Engine
	userHandler                 *handlers.UserHandler
	authHandler                 *handlers.AuthHandler
	profileHandler              *handlers.ProfileHandler
	subscriptionHandler         *handlers.SubscriptionHandler
	adminSubscriptionHandler    *adminHandlers.SubscriptionHandler
	adminResourceGroupHandler   *adminHandlers.ResourceGroupHandler
	planHandler                 *handlers.PlanHandler
	subscriptionTokenHandler    *handlers.SubscriptionTokenHandler
	paymentHandler              *handlers.PaymentHandler
	nodeHandler                 *handlers.NodeHandler
	nodeSubscriptionHandler     *handlers.NodeSubscriptionHandler
	agentHandler                *nodeHandlers.AgentHandler
	ticketHandler               *ticketHandlers.TicketHandler
	notificationHandler         *handlers.NotificationHandler
	forwardRuleHandler          *forwardHandlers.ForwardHandler
	forwardAgentHandler         *forwardHandlers.ForwardAgentHandler
	forwardAgentAPIHandler      *forwardHandlers.AgentHandler
	userForwardRuleHandler      *forwardHandlers.UserForwardRuleHandler
	agentHub                    *services.AgentHub
	agentHubHandler             *agentHandlers.HubHandler
	configSyncService           *forwardServices.ConfigSyncService
	trafficLimitEnforcementSvc  *forwardServices.TrafficLimitEnforcementService
	authMiddleware              *middleware.AuthMiddleware
	subscriptionOwnerMiddleware *middleware.SubscriptionOwnerMiddleware
	nodeTokenMiddleware         *middleware.NodeTokenMiddleware
	forwardAgentTokenMiddleware *middleware.ForwardAgentTokenMiddleware
	forwardRuleOwnerMiddleware  *middleware.ForwardRuleOwnerMiddleware
	forwardQuotaMiddleware      *middleware.ForwardQuotaMiddleware
	rateLimiter                 *middleware.RateLimiter
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

// NewRouter creates a new HTTP router with all dependencies
func NewRouter(userService *user.ServiceDDD, db *gorm.DB, cfg *config.Config, log logger.Interface) *Router {
	engine := gin.New()

	userRepo := repository.NewUserRepositoryDDD(db, log)
	sessionRepo := repository.NewSessionRepository(db)
	oauthRepo := repository.NewOAuthAccountRepository(db)

	hasher := auth.NewBcryptPasswordHasher(cfg.Auth.Password.BcryptCost)
	jwtSvc := auth.NewJWTService(cfg.Auth.JWT.Secret, cfg.Auth.JWT.AccessExpMinutes, cfg.Auth.JWT.RefreshExpDays)
	jwtService := &jwtServiceAdapter{jwtSvc}

	emailCfg := email.SMTPConfig{
		Host:        cfg.Email.SMTPHost,
		Port:        cfg.Email.SMTPPort,
		Username:    cfg.Email.SMTPUser,
		Password:    cfg.Email.SMTPPassword,
		FromAddress: cfg.Email.FromAddress,
		FromName:    cfg.Email.FromName,
	}
	emailService := email.NewSMTPEmailService(emailCfg)

	googleBase := auth.NewGoogleOAuthClient(auth.GoogleOAuthConfig{
		ClientID:     cfg.OAuth.Google.ClientID,
		ClientSecret: cfg.OAuth.Google.ClientSecret,
		RedirectURL:  cfg.OAuth.Google.GetRedirectURL(cfg.Server.GetBaseURL()),
	})
	googleClient := &oauthClientAdapter{googleBase}

	githubBase := auth.NewGitHubOAuthClient(auth.GitHubOAuthConfig{
		ClientID:     cfg.OAuth.GitHub.ClientID,
		ClientSecret: cfg.OAuth.GitHub.ClientSecret,
		RedirectURL:  cfg.OAuth.GitHub.GetRedirectURL(cfg.Server.GetBaseURL()),
	})
	githubClient := &oauthClientAdapter{githubBase}

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

	registerUC := usecases.NewRegisterWithPasswordUseCase(userRepo, hasher, emailService, authHelper, log)
	loginUC := usecases.NewLoginWithPasswordUseCase(userRepo, sessionRepo, hasher, jwtService, authHelper, cfg.Auth.Session, log)
	verifyEmailUC := usecases.NewVerifyEmailUseCase(userRepo, log)
	requestResetUC := usecases.NewRequestPasswordResetUseCase(userRepo, emailService, log)
	resetPasswordUC := usecases.NewResetPasswordUseCase(userRepo, sessionRepo, hasher, emailService, log)
	adminResetPasswordUC := usecases.NewAdminResetPasswordUseCase(userRepo, sessionRepo, hasher, emailService, log)
	initiateOAuthUC := usecases.NewInitiateOAuthLoginUseCase(googleClient, githubClient, log, stateStore)
	handleOAuthUC := usecases.NewHandleOAuthCallbackUseCase(userRepo, oauthRepo, sessionRepo, googleClient, githubClient, jwtService, initiateOAuthUC, authHelper, cfg.Auth.Session, log)
	refreshTokenUC := usecases.NewRefreshTokenUseCase(sessionRepo, jwtService, authHelper, log)
	logoutUC := usecases.NewLogoutUseCase(sessionRepo, log)

	authHandler := handlers.NewAuthHandler(
		registerUC, loginUC, verifyEmailUC, requestResetUC, resetPasswordUC,
		initiateOAuthUC, handleOAuthUC, refreshTokenUC, logoutUC, userRepo, log,
		cfg.Auth.Cookie, cfg.Auth.JWT,
		cfg.Server.FrontendCallbackURL, cfg.Server.AllowedOrigins,
	)

	userHandler := handlers.NewUserHandler(userService, adminResetPasswordUC)

	authMiddleware := middleware.NewAuthMiddleware(jwtSvc, userRepo, log)
	rateLimiter := middleware.NewRateLimiter(100, 1*time.Minute)

	subscriptionRepo := repository.NewSubscriptionRepository(db, log)
	subscriptionPlanRepo := repository.NewPlanRepository(db, log)
	subscriptionTokenRepo := repository.NewSubscriptionTokenRepository(db, log)
	subscriptionUsageRepo := repository.NewSubscriptionUsageRepository(db, log)
	planPricingRepo := repository.NewPlanPricingRepository(db, log)
	paymentRepo := repository.NewPaymentRepository(db)

	tokenGenerator := token.NewTokenGenerator()

	createSubscriptionUC := subscriptionUsecases.NewCreateSubscriptionUseCase(
		subscriptionRepo, subscriptionPlanRepo, subscriptionTokenRepo, planPricingRepo, userRepo, tokenGenerator, log,
	)
	activateSubscriptionUC := subscriptionUsecases.NewActivateSubscriptionUseCase(
		subscriptionRepo, log,
	)
	subscriptionBaseURL := cfg.Subscription.GetBaseURL(cfg.Server.GetBaseURL())
	getSubscriptionUC := subscriptionUsecases.NewGetSubscriptionUseCase(
		subscriptionRepo, subscriptionPlanRepo, log, subscriptionBaseURL,
	)
	listUserSubscriptionsUC := subscriptionUsecases.NewListUserSubscriptionsUseCase(
		subscriptionRepo, subscriptionPlanRepo, log, subscriptionBaseURL,
	)
	cancelSubscriptionUC := subscriptionUsecases.NewCancelSubscriptionUseCase(
		subscriptionRepo, subscriptionTokenRepo, log,
	)
	renewSubscriptionUC := subscriptionUsecases.NewRenewSubscriptionUseCase(
		subscriptionRepo, subscriptionPlanRepo, planPricingRepo, log,
	)
	changePlanUC := subscriptionUsecases.NewChangePlanUseCase(
		subscriptionRepo, subscriptionPlanRepo, log,
	)
	getSubscriptionUsageStatsUC := subscriptionUsecases.NewGetSubscriptionUsageStatsUseCase(
		subscriptionUsageRepo, log,
	)
	resetSubscriptionLinkUC := subscriptionUsecases.NewResetSubscriptionLinkUseCase(
		subscriptionRepo, subscriptionPlanRepo, log, subscriptionBaseURL,
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
		cancelSubscriptionUC, changePlanUC, getSubscriptionUsageStatsUC,
		resetSubscriptionLinkUC, log,
	)
	adminSubscriptionHandler := adminHandlers.NewSubscriptionHandler(
		createSubscriptionUC, getSubscriptionUC, listUserSubscriptionsUC,
		cancelSubscriptionUC, renewSubscriptionUC, changePlanUC,
		activateSubscriptionUC, log,
	)
	subscriptionOwnerMiddleware := middleware.NewSubscriptionOwnerMiddleware(subscriptionRepo, log)

	// Initialize resource group repository (handler initialized later after node and agent repos)
	resourceGroupRepo := repository.NewResourceGroupRepository(db, log)
	createResourceGroupUC := resourceUsecases.NewCreateResourceGroupUseCase(resourceGroupRepo, subscriptionPlanRepo, log)
	getResourceGroupUC := resourceUsecases.NewGetResourceGroupUseCase(resourceGroupRepo, subscriptionPlanRepo, log)
	listResourceGroupsUC := resourceUsecases.NewListResourceGroupsUseCase(resourceGroupRepo, subscriptionPlanRepo, log)
	updateResourceGroupUC := resourceUsecases.NewUpdateResourceGroupUseCase(resourceGroupRepo, subscriptionPlanRepo, log)
	deleteResourceGroupUC := resourceUsecases.NewDeleteResourceGroupUseCase(resourceGroupRepo, log)
	updateResourceGroupStatusUC := resourceUsecases.NewUpdateResourceGroupStatusUseCase(resourceGroupRepo, subscriptionPlanRepo, log)

	planHandler := handlers.NewPlanHandler(
		createPlanUC, updatePlanUC, getPlanUC, listPlansUC,
		getPublicPlansUC, activatePlanUC, deactivatePlanUC, getPlanPricingsUC,
	)
	subscriptionTokenHandler := handlers.NewSubscriptionTokenHandler(
		generateTokenUC, listTokensUC, revokeTokenUC, refreshSubscriptionTokenUC,
	)

	nodeRepoImpl := repository.NewNodeRepository(db, log)
	nodeRepo := adapters.NewNodeRepositoryAdapter(nodeRepoImpl, db, log)
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

	// Initialize node use cases
	createNodeUC := nodeUsecases.NewCreateNodeUseCase(nodeRepoImpl, log)
	getNodeUC := nodeUsecases.NewGetNodeUseCase(nodeRepoImpl, nodeStatusQuerier, log)
	updateNodeUC := nodeUsecases.NewUpdateNodeUseCase(log, nodeRepoImpl, resourceGroupRepo)
	deleteNodeUC := nodeUsecases.NewDeleteNodeUseCase(nodeRepoImpl, log)
	listNodesUC := nodeUsecases.NewListNodesUseCase(nodeRepoImpl, nodeStatusQuerier, log)
	generateNodeTokenUC := nodeUsecases.NewGenerateNodeTokenUseCase(nodeRepoImpl, log)
	generateNodeInstallScriptUC := nodeUsecases.NewGenerateNodeInstallScriptUseCase(nodeRepoImpl, log)

	// Initialize node authentication middleware using the same node repository adapter
	validateNodeTokenUC := nodeUsecases.NewValidateNodeTokenUseCase(nodeRepo, log)
	nodeTokenMiddleware := middleware.NewNodeTokenMiddleware(validateNodeTokenUC, log)

	// Initialize handlers
	// API URL for node install script generation
	apiBaseURL := cfg.Server.GetBaseURL()
	nodeHandler := handlers.NewNodeHandler(createNodeUC, getNodeUC, updateNodeUC, deleteNodeUC, listNodesUC, generateNodeTokenUC, generateNodeInstallScriptUC, apiBaseURL)
	nodeSubscriptionHandler := handlers.NewNodeSubscriptionHandler(generateSubscriptionUC)

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

	// Create profile handler
	profileHandler := handlers.NewProfileHandler(userService)

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

	// Initialize agent report use cases with adapters
	subscriptionUsageRecorder := adapters.NewSubscriptionUsageRecorderAdapter(subscriptionUsageRepo, log)
	systemStatusUpdater := adapters.NewNodeSystemStatusUpdaterAdapter(redisClient, log)
	onlineSubscriptionTracker := adapters.NewOnlineSubscriptionTrackerAdapter(log)
	reportSubscriptionUsageUC := nodeUsecases.NewReportSubscriptionUsageUseCase(subscriptionUsageRecorder, log)
	reportNodeStatusUC := nodeUsecases.NewReportNodeStatusUseCase(systemStatusUpdater, nodeRepoImpl, log)
	reportOnlineSubscriptionsUC := nodeUsecases.NewReportOnlineSubscriptionsUseCase(onlineSubscriptionTracker, log)

	// Initialize RESTful Agent Handler
	agentHandler := nodeHandlers.NewAgentHandler(
		getNodeConfigUC,
		getNodeSubscriptionsUC,
		reportSubscriptionUsageUC,
		reportNodeStatusUC,
		reportOnlineSubscriptionsUC,
		log,
	)

	// Initialize forward repositories (agent repo needed by rule use cases)
	forwardAgentRepo := repository.NewForwardAgentRepository(db, log)
	forwardRuleRepo := repository.NewForwardRuleRepository(db, log)

	// Initialize resource group membership use cases (need node and agent repos)
	manageNodesUC := resourceUsecases.NewManageResourceGroupNodesUseCase(resourceGroupRepo, nodeRepoImpl, log)
	manageAgentsUC := resourceUsecases.NewManageResourceGroupForwardAgentsUseCase(resourceGroupRepo, forwardAgentRepo, log)

	// Initialize admin resource group handler
	adminResourceGroupHandler := adminHandlers.NewResourceGroupHandler(
		createResourceGroupUC, getResourceGroupUC, listResourceGroupsUC,
		updateResourceGroupUC, deleteResourceGroupUC, updateResourceGroupStatusUC,
		manageNodesUC, manageAgentsUC,
		subscriptionPlanRepo, log,
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
	getForwardAgentUC := forwardUsecases.NewGetForwardAgentUseCase(forwardAgentRepo, forwardAgentStatusAdapter, log)
	updateForwardAgentUC := forwardUsecases.NewUpdateForwardAgentUseCase(forwardAgentRepo, resourceGroupRepo, log)
	deleteForwardAgentUC := forwardUsecases.NewDeleteForwardAgentUseCase(forwardAgentRepo, log)
	listForwardAgentsUC := forwardUsecases.NewListForwardAgentsUseCase(forwardAgentRepo, forwardAgentStatusAdapter, log)
	enableForwardAgentUC := forwardUsecases.NewEnableForwardAgentUseCase(forwardAgentRepo, log)
	disableForwardAgentUC := forwardUsecases.NewDisableForwardAgentUseCase(forwardAgentRepo, log)
	regenerateForwardAgentTokenUC := forwardUsecases.NewRegenerateForwardAgentTokenUseCase(forwardAgentRepo, agentTokenSvc, log)
	validateForwardAgentTokenUC := forwardUsecases.NewValidateForwardAgentTokenUseCase(forwardAgentRepo, log)

	// Initialize agent last seen updater for status reporting
	agentLastSeenUpdater := adapters.NewAgentLastSeenUpdaterAdapter(forwardAgentRepo)
	getAgentStatusUC := forwardUsecases.NewGetAgentStatusUseCase(forwardAgentRepo, forwardAgentStatusAdapter, log)
	getForwardAgentTokenUC := forwardUsecases.NewGetForwardAgentTokenUseCase(forwardAgentRepo, log)
	generateInstallScriptUC := forwardUsecases.NewGenerateInstallScriptUseCase(forwardAgentRepo, log)

	// Server base URL for forward agent install script
	serverBaseURL := cfg.Server.GetBaseURL()

	forwardAgentHandler := forwardHandlers.NewForwardAgentHandler(
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
		generateInstallScriptUC,
		serverBaseURL,
	)
	reportAgentStatusUC := forwardUsecases.NewReportAgentStatusUseCase(
		forwardAgentRepo,
		forwardAgentStatusAdapter,
		forwardAgentStatusAdapter, // statusQuerier (same adapter implements both interfaces)
		agentLastSeenUpdater,
		log,
	)

	// Initialize forward agent API handler for client to fetch rules and report traffic
	forwardAgentAPIHandler := forwardHandlers.NewAgentHandler(forwardRuleRepo, forwardAgentRepo, nodeRepoImpl, reportAgentStatusUC, forwardAgentStatusAdapter, cfg.Forward.TokenSigningSecret, log)

	// Initialize forward agent token middleware
	forwardAgentTokenMiddleware := middleware.NewForwardAgentTokenMiddleware(validateForwardAgentTokenUC, log)

	// Initialize agent hub for forward agent WebSocket connections (probe functionality)
	agentHub := services.NewAgentHub(log)

	// Register forward status handler to process forward agent status updates
	forwardStatusHandler := adapters.NewForwardStatusHandler(forwardAgentStatusAdapter, log)
	agentHub.RegisterStatusHandler(forwardStatusHandler)

	// Initialize and register probe service for forward domain
	probeService := forwardServices.NewProbeService(forwardRuleRepo, forwardAgentRepo, nodeRepoImpl, forwardAgentStatusAdapter, agentHub, log)
	agentHub.RegisterMessageHandler(probeService)

	// Initialize and register config sync service for forward domain
	configSyncService := forwardServices.NewConfigSyncService(forwardRuleRepo, forwardAgentRepo, nodeRepoImpl, forwardAgentStatusAdapter, cfg.Forward.TokenSigningSecret, agentHub, log)
	agentHub.RegisterMessageHandler(configSyncService)

	// Set port change notifier for exit agent port change detection
	reportAgentStatusUC.SetPortChangeNotifier(configSyncService)

	// Now initialize forward rule use cases with configSyncService
	createForwardRuleUC = forwardUsecases.NewCreateForwardRuleUseCase(forwardRuleRepo, forwardAgentRepo, nodeRepoImpl, configSyncService, log)
	getForwardRuleUC = forwardUsecases.NewGetForwardRuleUseCase(forwardRuleRepo, forwardAgentRepo, nodeRepoImpl, log)
	updateForwardRuleUC = forwardUsecases.NewUpdateForwardRuleUseCase(forwardRuleRepo, forwardAgentRepo, nodeRepoImpl, configSyncService, log)
	deleteForwardRuleUC = forwardUsecases.NewDeleteForwardRuleUseCase(forwardRuleRepo, configSyncService, log)
	listForwardRulesUC = forwardUsecases.NewListForwardRulesUseCase(forwardRuleRepo, forwardAgentRepo, nodeRepoImpl, log)
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
		log,
	)
	getUserForwardUsageUC := forwardUsecases.NewGetUserForwardUsageUseCase(
		forwardRuleRepo,
		subscriptionRepo,
		subscriptionPlanRepo,
		log,
	)

	// Initialize traffic limit enforcement service
	trafficLimitEnforcementSvc := forwardServices.NewTrafficLimitEnforcementService(
		forwardRuleRepo,
		subscriptionRepo,
		subscriptionPlanRepo,
		log,
	)

	// Initialize user forward rule handler
	userForwardRuleHandler := forwardHandlers.NewUserForwardRuleHandler(
		createUserForwardRuleUC,
		listUserForwardRulesUC,
		getUserForwardUsageUC,
		updateForwardRuleUC,  // reuse existing
		deleteForwardRuleUC,  // reuse existing
		enableForwardRuleUC,  // reuse existing
		disableForwardRuleUC, // reuse existing
		getForwardRuleUC,     // reuse existing
	)

	// Initialize forward rule owner middleware
	forwardRuleOwnerMiddleware := middleware.NewForwardRuleOwnerMiddleware(
		forwardRuleRepo,
		log,
	)

	// Initialize forward quota middleware
	forwardQuotaMiddleware := middleware.NewForwardQuotaMiddleware(
		forwardRuleRepo,
		subscriptionRepo,
		subscriptionPlanRepo,
		log,
	)

	// Initialize forward rule handler (after probeService is available)
	forwardRuleHandler := forwardHandlers.NewForwardHandler(
		createForwardRuleUC,
		getForwardRuleUC,
		updateForwardRuleUC,
		deleteForwardRuleUC,
		listForwardRulesUC,
		enableForwardRuleUC,
		disableForwardRuleUC,
		resetForwardTrafficUC,
		probeService,
	)

	// Initialize agent hub handler with config sync service
	agentHubHandler := agentHandlers.NewHubHandler(agentHub, configSyncService, log)

	return &Router{
		engine:                      engine,
		userHandler:                 userHandler,
		authHandler:                 authHandler,
		profileHandler:              profileHandler,
		subscriptionHandler:         subscriptionHandler,
		adminSubscriptionHandler:    adminSubscriptionHandler,
		adminResourceGroupHandler:   adminResourceGroupHandler,
		planHandler:                 planHandler,
		subscriptionTokenHandler:    subscriptionTokenHandler,
		paymentHandler:              paymentHandler,
		nodeHandler:                 nodeHandler,
		nodeSubscriptionHandler:     nodeSubscriptionHandler,
		agentHandler:                agentHandler,
		ticketHandler:               ticketHandler,
		notificationHandler:         notificationHandler,
		forwardRuleHandler:          forwardRuleHandler,
		forwardAgentHandler:         forwardAgentHandler,
		forwardAgentAPIHandler:      forwardAgentAPIHandler,
		userForwardRuleHandler:      userForwardRuleHandler,
		agentHub:                    agentHub,
		agentHubHandler:             agentHubHandler,
		configSyncService:           configSyncService,
		trafficLimitEnforcementSvc:  trafficLimitEnforcementSvc,
		authMiddleware:              authMiddleware,
		subscriptionOwnerMiddleware: subscriptionOwnerMiddleware,
		nodeTokenMiddleware:         nodeTokenMiddleware,
		forwardAgentTokenMiddleware: forwardAgentTokenMiddleware,
		forwardRuleOwnerMiddleware:  forwardRuleOwnerMiddleware,
		forwardQuotaMiddleware:      forwardQuotaMiddleware,
		rateLimiter:                 rateLimiter,
	}
}

// SetupRoutes configures all HTTP routes
func (r *Router) SetupRoutes(cfg *config.Config) {
	r.engine.Use(middleware.Logger())
	r.engine.Use(middleware.Recovery())
	r.engine.Use(middleware.CORS(cfg.Server.AllowedOrigins))

	r.engine.GET("/health", r.userHandler.HealthCheck)

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
	}

	// Admin resource group routes - full CRUD for resource groups
	adminResourceGroups := r.engine.Group("/admin/resource-groups")
	adminResourceGroups.Use(r.authMiddleware.RequireAuth(), authorization.RequireAdmin())
	{
		adminResourceGroups.POST("", r.adminResourceGroupHandler.Create)
		adminResourceGroups.GET("", r.adminResourceGroupHandler.List)
		adminResourceGroups.GET("/:id", r.adminResourceGroupHandler.Get)
		adminResourceGroups.GET("/sid/:sid", r.adminResourceGroupHandler.GetBySID)
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
	}

	// User subscription routes - only own subscriptions
	subscriptions := r.engine.Group("/subscriptions")
	subscriptions.Use(r.authMiddleware.RequireAuth())
	{
		// Collection operations (no ownership check needed)
		subscriptions.POST("", r.subscriptionHandler.CreateSubscription)
		subscriptions.GET("", r.subscriptionHandler.ListUserSubscriptions)

		// Operations on specific subscription (ownership verified by middleware)
		subscriptionWithOwnership := subscriptions.Group("/:id")
		subscriptionWithOwnership.Use(r.subscriptionOwnerMiddleware.RequireOwnership())
		{
			subscriptionWithOwnership.GET("", r.subscriptionHandler.GetSubscription)
			subscriptionWithOwnership.PATCH("/status", r.subscriptionHandler.UpdateStatus)
			subscriptionWithOwnership.PATCH("/plan", r.subscriptionHandler.ChangePlan)
			subscriptionWithOwnership.PUT("/link", r.subscriptionHandler.ResetLink)

			// Token sub-resource endpoints (using /action path due to Gin framework limitation with colon format)
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

		// Protected endpoints
		plansProtected := plans.Group("")
		plansProtected.Use(r.authMiddleware.RequireAuth())
		{
			// Collection operations (no ID parameter)
			plansProtected.POST("", r.planHandler.CreatePlan)
			plansProtected.GET("", r.planHandler.ListPlans)

			// Specific action endpoints (must come BEFORE /:id to avoid conflicts)
			// Using PATCH for state changes as per RESTful best practices
			plansProtected.PATCH("/:id/status", r.planHandler.UpdatePlanStatus)
			plansProtected.GET("/:id/pricings", r.planHandler.GetPlanPricings)

			// Generic parameterized routes (must come LAST)
			plansProtected.GET("/:id", r.planHandler.GetPlan)
			plansProtected.PUT("/:id", r.planHandler.UpdatePlan)
		}
	}

	routes.SetupNodeRoutes(r.engine, &routes.NodeRouteConfig{
		NodeHandler:         r.nodeHandler,
		SubscriptionHandler: r.nodeSubscriptionHandler,
		AuthMiddleware:      r.authMiddleware,
		NodeTokenMW:         r.nodeTokenMiddleware,
		RateLimiter:         r.rateLimiter,
	})

	// RESTful Agent API - Modern API with resource-oriented design
	agentAPI := r.engine.Group("/agents")
	agentAPI.Use(r.nodeTokenMiddleware.RequireNodeTokenHeader())
	{
		// GET /agents/:id/config - Get node configuration
		agentAPI.GET("/:id/config", r.agentHandler.GetConfig)

		// GET /agents/:id/subscriptions - Get active subscriptions for node
		agentAPI.GET("/:id/subscriptions", r.agentHandler.GetSubscriptions)

		// POST /agents/:id/traffic - Report subscription traffic data
		agentAPI.POST("/:id/traffic", r.agentHandler.ReportTraffic)

		// PUT /agents/:id/status - Update node system status
		agentAPI.PUT("/:id/status", r.agentHandler.UpdateStatus)

		// PUT /agents/:id/online-subscriptions - Update online subscriptions list
		agentAPI.PUT("/:id/online-subscriptions", r.agentHandler.UpdateOnlineSubscriptions)
	}

	routes.SetupTicketRoutes(r.engine, &routes.TicketRouteConfig{
		TicketHandler:  r.ticketHandler,
		AuthMiddleware: r.authMiddleware,
	})

	routes.SetupNotificationRoutes(r.engine, &routes.NotificationRouteConfig{
		NotificationHandler: r.notificationHandler,
		AuthMiddleware:      r.authMiddleware,
	})

	routes.SetupForwardRoutes(r.engine, &routes.ForwardRouteConfig{
		ForwardRuleHandler:          r.forwardRuleHandler,
		ForwardAgentHandler:         r.forwardAgentHandler,
		ForwardAgentAPIHandler:      r.forwardAgentAPIHandler,
		UserForwardHandler:          r.userForwardRuleHandler,
		AuthMiddleware:              r.authMiddleware,
		ForwardAgentTokenMiddleware: r.forwardAgentTokenMiddleware,
		ForwardRuleOwnerMiddleware:  r.forwardRuleOwnerMiddleware,
		ForwardQuotaMiddleware:      r.forwardQuotaMiddleware,
	})

	routes.SetupAgentHubRoutes(r.engine, &routes.AgentHubRouteConfig{
		HubHandler:                  r.agentHubHandler,
		ForwardAgentTokenMiddleware: r.forwardAgentTokenMiddleware,
	})
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
	// Reserved for future cleanup tasks
}
