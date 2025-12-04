package http

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	forwardUsecases "github.com/orris-inc/orris/internal/application/forward/usecases"
	nodeUsecases "github.com/orris-inc/orris/internal/application/node/usecases"
	notificationApp "github.com/orris-inc/orris/internal/application/notification"
	paymentGateway "github.com/orris-inc/orris/internal/application/payment/payment_gateway"
	paymentUsecases "github.com/orris-inc/orris/internal/application/payment/usecases"
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
	"github.com/orris-inc/orris/internal/infrastructure/token"
	"github.com/orris-inc/orris/internal/interfaces/http/handlers"
	adminHandlers "github.com/orris-inc/orris/internal/interfaces/http/handlers/admin"
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
	subscriptionPlanHandler     *handlers.SubscriptionPlanHandler
	subscriptionTokenHandler    *handlers.SubscriptionTokenHandler
	paymentHandler              *handlers.PaymentHandler
	nodeHandler                 *handlers.NodeHandler
	nodeGroupHandler            *handlers.NodeGroupHandler
	nodeSubscriptionHandler     *handlers.NodeSubscriptionHandler
	agentHandler                *nodeHandlers.AgentHandler
	ticketHandler               *ticketHandlers.TicketHandler
	notificationHandler         *handlers.NotificationHandler
	forwardRuleHandler          *forwardHandlers.ForwardHandler
	forwardAgentHandler         *forwardHandlers.ForwardAgentHandler
	forwardChainHandler         *forwardHandlers.ForwardChainHandler
	forwardAgentAPIHandler      *forwardHandlers.AgentHandler
	authMiddleware              *middleware.AuthMiddleware
	subscriptionOwnerMiddleware *middleware.SubscriptionOwnerMiddleware
	nodeTokenMiddleware         *middleware.NodeTokenMiddleware
	forwardAgentTokenMiddleware *middleware.ForwardAgentTokenMiddleware
	rateLimiter                 *middleware.RateLimiter
}

type jwtServiceAdapter struct {
	*auth.JWTService
}

func (a *jwtServiceAdapter) Generate(userID uint, sessionID string, role authorization.UserRole) (*usecases.TokenPair, error) {
	pair, err := a.JWTService.Generate(userID, sessionID, role)
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

	userHandler := handlers.NewUserHandler(userService)

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

	authHelper := helpers.NewAuthHelper(userRepo, sessionRepo, log)

	registerUC := usecases.NewRegisterWithPasswordUseCase(userRepo, hasher, emailService, authHelper, log)
	loginUC := usecases.NewLoginWithPasswordUseCase(userRepo, sessionRepo, hasher, jwtService, authHelper, cfg.Auth.Session, log)
	verifyEmailUC := usecases.NewVerifyEmailUseCase(userRepo, log)
	requestResetUC := usecases.NewRequestPasswordResetUseCase(userRepo, emailService, log)
	resetPasswordUC := usecases.NewResetPasswordUseCase(userRepo, sessionRepo, hasher, emailService, log)
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

	authMiddleware := middleware.NewAuthMiddleware(jwtSvc, log)
	rateLimiter := middleware.NewRateLimiter(100, 1*time.Minute)

	subscriptionRepo := repository.NewSubscriptionRepository(db, log)
	subscriptionPlanRepo := repository.NewSubscriptionPlanRepository(db, log)
	subscriptionTokenRepo := repository.NewSubscriptionTokenRepository(db, log)
	subscriptionTrafficRepo := repository.NewSubscriptionTrafficRepository(db, log)
	planPricingRepo := repository.NewPlanPricingRepository(db, log)
	paymentRepo := repository.NewPaymentRepository(db)

	tokenGenerator := token.NewTokenGenerator()

	createSubscriptionUC := subscriptionUsecases.NewCreateSubscriptionUseCase(
		subscriptionRepo, subscriptionPlanRepo, subscriptionTokenRepo, planPricingRepo, userRepo, tokenGenerator, log,
	)
	activateSubscriptionUC := subscriptionUsecases.NewActivateSubscriptionUseCase(
		subscriptionRepo, log,
	)
	getSubscriptionUC := subscriptionUsecases.NewGetSubscriptionUseCase(
		subscriptionRepo, subscriptionPlanRepo, log,
	)
	listUserSubscriptionsUC := subscriptionUsecases.NewListUserSubscriptionsUseCase(
		subscriptionRepo, subscriptionPlanRepo, log,
	)
	cancelSubscriptionUC := subscriptionUsecases.NewCancelSubscriptionUseCase(
		subscriptionRepo, subscriptionTokenRepo, log,
	)
	renewSubscriptionUC := subscriptionUsecases.NewRenewSubscriptionUseCase(
		subscriptionRepo, subscriptionPlanRepo, log,
	)
	changePlanUC := subscriptionUsecases.NewChangePlanUseCase(
		subscriptionRepo, subscriptionPlanRepo, log,
	)
	getSubscriptionTrafficStatsUC := subscriptionUsecases.NewGetSubscriptionTrafficStatsUseCase(
		subscriptionTrafficRepo, log,
	)

	createPlanUC := subscriptionUsecases.NewCreateSubscriptionPlanUseCase(
		subscriptionPlanRepo, log,
	)
	updatePlanUC := subscriptionUsecases.NewUpdateSubscriptionPlanUseCase(
		subscriptionPlanRepo, log,
	)
	getPlanUC := subscriptionUsecases.NewGetSubscriptionPlanUseCase(
		subscriptionPlanRepo, log,
	)
	listPlansUC := subscriptionUsecases.NewListSubscriptionPlansUseCase(
		subscriptionPlanRepo, log,
	)
	getPublicPlansUC := subscriptionUsecases.NewGetPublicPlansUseCase(
		subscriptionPlanRepo, planPricingRepo, log,
	)
	activatePlanUC := subscriptionUsecases.NewActivateSubscriptionPlanUseCase(
		subscriptionPlanRepo, log,
	)
	deactivatePlanUC := subscriptionUsecases.NewDeactivateSubscriptionPlanUseCase(
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
		cancelSubscriptionUC, changePlanUC, getSubscriptionTrafficStatsUC, log,
	)
	adminSubscriptionHandler := adminHandlers.NewSubscriptionHandler(
		createSubscriptionUC, getSubscriptionUC, listUserSubscriptionsUC,
		cancelSubscriptionUC, renewSubscriptionUC, changePlanUC,
		activateSubscriptionUC, log,
	)
	subscriptionOwnerMiddleware := middleware.NewSubscriptionOwnerMiddleware(subscriptionRepo, log)
	subscriptionPlanHandler := handlers.NewSubscriptionPlanHandler(
		createPlanUC, updatePlanUC, getPlanUC, listPlansUC,
		getPublicPlansUC, activatePlanUC, deactivatePlanUC, getPlanPricingsUC,
	)
	subscriptionTokenHandler := handlers.NewSubscriptionTokenHandler(
		generateTokenUC, listTokensUC, revokeTokenUC, refreshSubscriptionTokenUC,
	)

	nodeRepoImpl := repository.NewNodeRepository(db, log)
	nodeRepo := adapters.NewNodeRepositoryAdapter(nodeRepoImpl, db, log)
	nodeGroupRepoImpl := repository.NewNodeGroupRepository(db, log)
	tokenValidator := adapters.NewSubscriptionTokenValidatorAdapter(db, log)
	generateSubscriptionUC := nodeUsecases.NewGenerateSubscriptionUseCase(
		nodeRepo, tokenValidator, log,
	)

	// Initialize node system status querier adapter
	nodeStatusQuerier := adapters.NewNodeSystemStatusQuerierAdapter(redisClient, log)

	// Initialize node use cases
	createNodeUC := nodeUsecases.NewCreateNodeUseCase(nodeRepoImpl, log)
	getNodeUC := nodeUsecases.NewGetNodeUseCase(nodeRepoImpl, nodeStatusQuerier, log)
	updateNodeUC := nodeUsecases.NewUpdateNodeUseCase(log, nodeRepoImpl)
	deleteNodeUC := nodeUsecases.NewDeleteNodeUseCase(nodeRepoImpl, nodeGroupRepoImpl, log)
	listNodesUC := nodeUsecases.NewListNodesUseCase(nodeRepoImpl, nodeStatusQuerier, log)
	generateNodeTokenUC := nodeUsecases.NewGenerateNodeTokenUseCase(nodeRepoImpl, log)

	// Initialize node authentication middleware using the same node repository adapter
	validateNodeTokenUC := nodeUsecases.NewValidateNodeTokenUseCase(nodeRepo, log)
	nodeTokenMiddleware := middleware.NewNodeTokenMiddleware(validateNodeTokenUC, log)

	// Initialize NodeGroup use cases
	createNodeGroupUC := nodeUsecases.NewCreateNodeGroupUseCase(nodeGroupRepoImpl, log)
	getNodeGroupUC := nodeUsecases.NewGetNodeGroupUseCase(nodeGroupRepoImpl, log)
	updateNodeGroupUC := nodeUsecases.NewUpdateNodeGroupUseCase(nodeGroupRepoImpl, log)
	deleteNodeGroupUC := nodeUsecases.NewDeleteNodeGroupUseCase(nodeGroupRepoImpl, log)
	listNodeGroupsUC := nodeUsecases.NewListNodeGroupsUseCase(nodeGroupRepoImpl, log)
	addNodeToGroupUC := nodeUsecases.NewAddNodeToGroupUseCase(nodeRepoImpl, nodeGroupRepoImpl, log)
	removeNodeFromGroupUC := nodeUsecases.NewRemoveNodeFromGroupUseCase(nodeGroupRepoImpl, log)
	batchAddNodesToGroupUC := nodeUsecases.NewBatchAddNodesToGroupUseCase(nodeRepoImpl, nodeGroupRepoImpl, log)
	batchRemoveNodesFromGroupUC := nodeUsecases.NewBatchRemoveNodesFromGroupUseCase(nodeGroupRepoImpl, log)
	listGroupNodesUC := nodeUsecases.NewListGroupNodesUseCase(nodeRepoImpl, nodeGroupRepoImpl, log)
	associateGroupWithPlanUC := nodeUsecases.NewAssociateGroupWithPlanUseCase(nodeGroupRepoImpl, subscriptionPlanRepo, log)
	disassociateGroupFromPlanUC := nodeUsecases.NewDisassociateGroupFromPlanUseCase(nodeGroupRepoImpl, log)

	// Initialize handlers
	nodeHandler := handlers.NewNodeHandler(createNodeUC, getNodeUC, updateNodeUC, deleteNodeUC, listNodesUC, generateNodeTokenUC)
	nodeGroupHandler := handlers.NewNodeGroupHandler(
		createNodeGroupUC,
		getNodeGroupUC,
		updateNodeGroupUC,
		deleteNodeGroupUC,
		listNodeGroupsUC,
		addNodeToGroupUC,
		removeNodeFromGroupUC,
		batchAddNodesToGroupUC,
		batchRemoveNodesFromGroupUC,
		listGroupNodesUC,
		associateGroupWithPlanUC,
		disassociateGroupFromPlanUC,
	)
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
	getNodeSubscriptionsUC := nodeUsecases.NewGetNodeSubscriptionsUseCase(subscriptionRepo, log)

	// Initialize agent report use cases with adapters
	subscriptionTrafficRecorder := adapters.NewSubscriptionTrafficRecorderAdapter(subscriptionTrafficRepo, log)
	systemStatusUpdater := adapters.NewNodeSystemStatusUpdaterAdapter(redisClient, log)
	onlineSubscriptionTracker := adapters.NewOnlineSubscriptionTrackerAdapter(log)
	reportSubscriptionTrafficUC := nodeUsecases.NewReportSubscriptionTrafficUseCase(subscriptionTrafficRecorder, log)
	reportNodeStatusUC := nodeUsecases.NewReportNodeStatusUseCase(systemStatusUpdater, nodeRepoImpl, log)
	reportOnlineSubscriptionsUC := nodeUsecases.NewReportOnlineSubscriptionsUseCase(onlineSubscriptionTracker, log)

	// Initialize RESTful Agent Handler
	agentHandler := nodeHandlers.NewAgentHandler(
		getNodeConfigUC,
		getNodeSubscriptionsUC,
		reportSubscriptionTrafficUC,
		reportNodeStatusUC,
		reportOnlineSubscriptionsUC,
		log,
	)

	// Initialize forward rule components
	forwardRuleRepo := repository.NewForwardRuleRepository(db, log)

	createForwardRuleUC := forwardUsecases.NewCreateForwardRuleUseCase(forwardRuleRepo, log)
	getForwardRuleUC := forwardUsecases.NewGetForwardRuleUseCase(forwardRuleRepo, log)
	updateForwardRuleUC := forwardUsecases.NewUpdateForwardRuleUseCase(forwardRuleRepo, log)
	deleteForwardRuleUC := forwardUsecases.NewDeleteForwardRuleUseCase(forwardRuleRepo, log)
	listForwardRulesUC := forwardUsecases.NewListForwardRulesUseCase(forwardRuleRepo, log)
	enableForwardRuleUC := forwardUsecases.NewEnableForwardRuleUseCase(forwardRuleRepo, log)
	disableForwardRuleUC := forwardUsecases.NewDisableForwardRuleUseCase(forwardRuleRepo, log)
	resetForwardTrafficUC := forwardUsecases.NewResetForwardRuleTrafficUseCase(forwardRuleRepo, log)

	forwardRuleHandler := forwardHandlers.NewForwardHandler(
		createForwardRuleUC,
		getForwardRuleUC,
		updateForwardRuleUC,
		deleteForwardRuleUC,
		listForwardRulesUC,
		enableForwardRuleUC,
		disableForwardRuleUC,
		resetForwardTrafficUC,
	)

	// Initialize forward agent components
	forwardAgentRepo := repository.NewForwardAgentRepository(db, log)

	createForwardAgentUC := forwardUsecases.NewCreateForwardAgentUseCase(forwardAgentRepo, log)
	getForwardAgentUC := forwardUsecases.NewGetForwardAgentUseCase(forwardAgentRepo, log)
	updateForwardAgentUC := forwardUsecases.NewUpdateForwardAgentUseCase(forwardAgentRepo, log)
	deleteForwardAgentUC := forwardUsecases.NewDeleteForwardAgentUseCase(forwardAgentRepo, log)
	listForwardAgentsUC := forwardUsecases.NewListForwardAgentsUseCase(forwardAgentRepo, log)
	enableForwardAgentUC := forwardUsecases.NewEnableForwardAgentUseCase(forwardAgentRepo, log)
	disableForwardAgentUC := forwardUsecases.NewDisableForwardAgentUseCase(forwardAgentRepo, log)
	regenerateForwardAgentTokenUC := forwardUsecases.NewRegenerateForwardAgentTokenUseCase(forwardAgentRepo, log)
	validateForwardAgentTokenUC := forwardUsecases.NewValidateForwardAgentTokenUseCase(forwardAgentRepo, log)

	forwardAgentHandler := forwardHandlers.NewForwardAgentHandler(
		createForwardAgentUC,
		getForwardAgentUC,
		listForwardAgentsUC,
		updateForwardAgentUC,
		deleteForwardAgentUC,
		enableForwardAgentUC,
		disableForwardAgentUC,
		regenerateForwardAgentTokenUC,
	)

	// Initialize forward agent API handler for client to fetch rules and report traffic
	forwardAgentAPIHandler := forwardHandlers.NewAgentHandler(forwardRuleRepo, log)

	// Initialize forward agent token middleware
	forwardAgentTokenMiddleware := middleware.NewForwardAgentTokenMiddleware(validateForwardAgentTokenUC, log)

	// Initialize forward chain components
	forwardChainRepo := repository.NewForwardChainRepository(db, log)

	createForwardChainUC := forwardUsecases.NewCreateForwardChainUseCase(forwardChainRepo, forwardRuleRepo, forwardAgentRepo, log)
	getForwardChainUC := forwardUsecases.NewGetForwardChainUseCase(forwardChainRepo, log)
	listForwardChainsUC := forwardUsecases.NewListForwardChainsUseCase(forwardChainRepo, log)
	enableForwardChainUC := forwardUsecases.NewEnableForwardChainUseCase(forwardChainRepo, forwardRuleRepo, log)
	disableForwardChainUC := forwardUsecases.NewDisableForwardChainUseCase(forwardChainRepo, forwardRuleRepo, log)
	deleteForwardChainUC := forwardUsecases.NewDeleteForwardChainUseCase(forwardChainRepo, forwardRuleRepo, log)

	forwardChainHandler := forwardHandlers.NewForwardChainHandler(
		createForwardChainUC,
		getForwardChainUC,
		listForwardChainsUC,
		enableForwardChainUC,
		disableForwardChainUC,
		deleteForwardChainUC,
	)

	return &Router{
		engine:                      engine,
		userHandler:                 userHandler,
		authHandler:                 authHandler,
		profileHandler:              profileHandler,
		subscriptionHandler:         subscriptionHandler,
		adminSubscriptionHandler:    adminSubscriptionHandler,
		subscriptionPlanHandler:     subscriptionPlanHandler,
		subscriptionTokenHandler:    subscriptionTokenHandler,
		paymentHandler:              paymentHandler,
		nodeHandler:                 nodeHandler,
		nodeGroupHandler:            nodeGroupHandler,
		nodeSubscriptionHandler:     nodeSubscriptionHandler,
		agentHandler:                agentHandler,
		ticketHandler:               ticketHandler,
		notificationHandler:         notificationHandler,
		forwardRuleHandler:          forwardRuleHandler,
		forwardAgentHandler:         forwardAgentHandler,
		forwardChainHandler:         forwardChainHandler,
		forwardAgentAPIHandler:      forwardAgentAPIHandler,
		authMiddleware:              authMiddleware,
		subscriptionOwnerMiddleware: subscriptionOwnerMiddleware,
		nodeTokenMiddleware:         nodeTokenMiddleware,
		forwardAgentTokenMiddleware: forwardAgentTokenMiddleware,
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

	plans := r.engine.Group("/subscription-plans")
	{
		// IMPORTANT: Register specific paths BEFORE parameterized paths to avoid route conflicts
		// e.g., /public must come before /:id, /activate before /:id, etc.

		// Public endpoints (no authentication required)
		plans.GET("/public", r.subscriptionPlanHandler.GetPublicPlans)

		// Protected endpoints
		plansProtected := plans.Group("")
		plansProtected.Use(r.authMiddleware.RequireAuth())
		{
			// Collection operations (no ID parameter)
			plansProtected.POST("", r.subscriptionPlanHandler.CreatePlan)
			plansProtected.GET("", r.subscriptionPlanHandler.ListPlans)

			// Specific action endpoints (must come BEFORE /:id to avoid conflicts)
			// Using PATCH for state changes as per RESTful best practices
			plansProtected.PATCH("/:id/status", r.subscriptionPlanHandler.UpdatePlanStatus)
			plansProtected.GET("/:id/pricings", r.subscriptionPlanHandler.GetPlanPricings)

			// Generic parameterized routes (must come LAST)
			plansProtected.GET("/:id", r.subscriptionPlanHandler.GetPlan)
			plansProtected.PUT("/:id", r.subscriptionPlanHandler.UpdatePlan)
		}
	}

	routes.SetupNodeRoutes(r.engine, &routes.NodeRouteConfig{
		NodeHandler:         r.nodeHandler,
		NodeGroupHandler:    r.nodeGroupHandler,
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
		ForwardChainHandler:         r.forwardChainHandler,
		ForwardAgentAPIHandler:      r.forwardAgentAPIHandler,
		AuthMiddleware:              r.authMiddleware,
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
