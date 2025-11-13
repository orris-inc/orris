package http

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"

	nodeUsecases "orris/internal/application/node/usecases"
	notificationApp "orris/internal/application/notification"
	paymentGateway "orris/internal/application/payment/payment_gateway"
	paymentUsecases "orris/internal/application/payment/usecases"
	subscriptionUsecases "orris/internal/application/subscription/usecases"
	"orris/internal/application/user"
	"orris/internal/application/user/helpers"
	"orris/internal/application/user/usecases"
	"orris/internal/infrastructure/adapters"
	"orris/internal/infrastructure/auth"
	"orris/internal/infrastructure/cache"
	"orris/internal/infrastructure/config"
	"orris/internal/infrastructure/email"
	"orris/internal/infrastructure/repository"
	"orris/internal/infrastructure/token"
	"orris/internal/interfaces/http/handlers"
	nodeHandlers "orris/internal/interfaces/http/handlers/node"
	ticketHandlers "orris/internal/interfaces/http/handlers/ticket"
	"orris/internal/interfaces/http/middleware"
	"orris/internal/interfaces/http/routes"
	"orris/internal/shared/authorization"
	"orris/internal/shared/logger"
	"orris/internal/shared/services/markdown"

	_ "orris/docs"
)

// Router represents the HTTP router configuration
type Router struct {
	engine                   *gin.Engine
	userHandler              *handlers.UserHandler
	authHandler              *handlers.AuthHandler
	profileHandler           *handlers.ProfileHandler
	subscriptionHandler      *handlers.SubscriptionHandler
	subscriptionPlanHandler  *handlers.SubscriptionPlanHandler
	subscriptionTokenHandler *handlers.SubscriptionTokenHandler
	paymentHandler           *handlers.PaymentHandler
	nodeHandler              *handlers.NodeHandler
	nodeGroupHandler         *handlers.NodeGroupHandler
	nodeSubscriptionHandler  *handlers.NodeSubscriptionHandler
	nodeReportHandler        *handlers.NodeReportHandler
	agentHandler             *nodeHandlers.AgentHandler
	ticketHandler            *ticketHandlers.TicketHandler
	notificationHandler      *handlers.NotificationHandler
	authMiddleware           *middleware.AuthMiddleware
	nodeTokenMiddleware      *middleware.NodeTokenMiddleware
	rateLimiter              *middleware.RateLimiter
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
		RedirectURL:  cfg.OAuth.Google.RedirectURL,
	})
	googleClient := &oauthClientAdapter{googleBase}

	githubBase := auth.NewGitHubOAuthClient(auth.GitHubOAuthConfig{
		ClientID:     cfg.OAuth.GitHub.ClientID,
		ClientSecret: cfg.OAuth.GitHub.ClientSecret,
		RedirectURL:  cfg.OAuth.GitHub.RedirectURL,
	})
	githubClient := &oauthClientAdapter{githubBase}

	// Initialize Redis client for OAuth state storage
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
	planPricingRepo := repository.NewPlanPricingRepository(db, log)
	paymentRepo := repository.NewPaymentRepository(db)

	tokenGenerator := token.NewTokenGenerator()

	createSubscriptionUC := subscriptionUsecases.NewCreateSubscriptionUseCase(
		subscriptionRepo, subscriptionPlanRepo, subscriptionTokenRepo, planPricingRepo, tokenGenerator, log,
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
		cancelSubscriptionUC, renewSubscriptionUC, changePlanUC,
		activateSubscriptionUC, log,
	)
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

	// Initialize node use cases
	createNodeUC := nodeUsecases.NewCreateNodeUseCase(nodeRepoImpl, log)
	getNodeUC := nodeUsecases.NewGetNodeUseCase(nodeRepoImpl, log)
	updateNodeUC := nodeUsecases.NewUpdateNodeUseCase(log, nodeRepoImpl)
	deleteNodeUC := nodeUsecases.NewDeleteNodeUseCase(nodeRepoImpl, nodeGroupRepoImpl, log)
	listNodesUC := nodeUsecases.NewListNodesUseCase(nodeRepoImpl, log)
	generateNodeTokenUC := nodeUsecases.NewGenerateNodeTokenUseCase(nodeRepoImpl, log)

	// Initialize Redis traffic cache
	trafficCache := cache.NewRedisTrafficCache(redisClient, nodeRepoImpl, log)

	// Initialize node traffic repository (used by report adapter)
	nodeTrafficRepo := repository.NewNodeTrafficRepository(db, log)

	// Initialize user traffic repository
	userTrafficRepo := repository.NewUserTrafficRepository(db, log)

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
	listGroupNodesUC := nodeUsecases.NewListGroupNodesUseCase(nodeRepoImpl, nodeGroupRepoImpl, log)
	associateGroupWithPlanUC := nodeUsecases.NewAssociateGroupWithPlanUseCase(nodeGroupRepoImpl, subscriptionPlanRepo, log)
	disassociateGroupFromPlanUC := nodeUsecases.NewDisassociateGroupFromPlanUseCase(nodeGroupRepoImpl, log)

	// Initialize NodeReport use case with adapters (using Redis traffic cache)
	trafficRecorder := adapters.NewNodeTrafficRecorderAdapter(trafficCache, nodeTrafficRepo, log)
	statusUpdater := adapters.NewNodeStatusUpdaterAdapter(nodeRepoImpl, log)
	limitChecker := adapters.NewNodeLimitCheckerAdapter(log)
	reportNodeDataUC := nodeUsecases.NewReportNodeDataUseCase(trafficRecorder, statusUpdater, limitChecker, log)

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
		listGroupNodesUC,
		associateGroupWithPlanUC,
		disassociateGroupFromPlanUC,
	)
	nodeSubscriptionHandler := handlers.NewNodeSubscriptionHandler(generateSubscriptionUC)
	nodeReportHandler := handlers.NewNodeReportHandler(reportNodeDataUC, validateNodeTokenUC)

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
	// Currently mock gateway is removed as per CLAUDE.md rule: "不允许mock数据"
	var gateway paymentGateway.PaymentGateway = nil // Temporary placeholder until real implementation
	paymentConfig := paymentUsecases.PaymentConfig{
		NotifyURL: cfg.Server.BaseURL + "/payments/callback",
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

	// XrayR API handlers - v2raysocks compatible backend
	getNodeConfigUC := nodeUsecases.NewGetNodeConfigUseCase(nodeRepoImpl, log)
	getNodeUsersUC := nodeUsecases.NewGetNodeUsersUseCase(subscriptionRepo, log)

	// Initialize XrayR report use cases with adapters
	userTrafficRecorder := adapters.NewUserTrafficRecorderAdapter(userTrafficRepo, log)
	systemStatusUpdater := adapters.NewNodeSystemStatusUpdaterAdapter(log)
	onlineUserTracker := adapters.NewOnlineUserTrackerAdapter(log)
	reportUserTrafficUC := nodeUsecases.NewReportUserTrafficUseCase(userTrafficRecorder, log)
	reportNodeStatusUC := nodeUsecases.NewReportNodeStatusUseCase(systemStatusUpdater, log)
	reportOnlineUsersUC := nodeUsecases.NewReportOnlineUsersUseCase(onlineUserTracker, log)

	// Initialize RESTful Agent Handler
	agentHandler := nodeHandlers.NewAgentHandler(
		getNodeConfigUC,
		getNodeUsersUC,
		reportUserTrafficUC,
		reportNodeStatusUC,
		reportOnlineUsersUC,
		log,
	)

	return &Router{
		engine:                   engine,
		userHandler:              userHandler,
		authHandler:              authHandler,
		profileHandler:           profileHandler,
		subscriptionHandler:      subscriptionHandler,
		subscriptionPlanHandler:  subscriptionPlanHandler,
		subscriptionTokenHandler: subscriptionTokenHandler,
		paymentHandler:           paymentHandler,
		nodeHandler:              nodeHandler,
		nodeGroupHandler:         nodeGroupHandler,
		nodeSubscriptionHandler:  nodeSubscriptionHandler,
		nodeReportHandler:        nodeReportHandler,
		agentHandler:             agentHandler,
		ticketHandler:            ticketHandler,
		notificationHandler:      notificationHandler,
		authMiddleware:           authMiddleware,
		nodeTokenMiddleware:      nodeTokenMiddleware,
		rateLimiter:              rateLimiter,
	}
}

// SetupRoutes configures all HTTP routes
func (r *Router) SetupRoutes(cfg *config.Config) {
	r.engine.Use(middleware.Logger())
	r.engine.Use(middleware.Recovery())
	r.engine.Use(middleware.CORS(cfg.Server.AllowedOrigins))

	r.engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

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
		users.POST("", authorization.RequireAdmin(), r.userHandler.CreateUser)
		users.GET("", authorization.RequireAdmin(), r.userHandler.ListUsers)
		users.GET("/:id", authorization.RequireAdmin(), r.userHandler.GetUser)
		users.PATCH("/:id", authorization.RequireAdmin(), r.userHandler.UpdateUser)
		users.DELETE("/:id", authorization.RequireAdmin(), r.userHandler.DeleteUser)
		users.GET("/email/:email", authorization.RequireAdmin(), r.userHandler.GetUserByEmail)

		// Profile management routes
		users.PATCH("/me", r.profileHandler.UpdateProfile)
		users.PUT("/me/password", r.profileHandler.ChangePassword)
	}

	subscriptions := r.engine.Group("/subscriptions")
	subscriptions.Use(r.authMiddleware.RequireAuth())
	{
		subscriptions.POST("", r.subscriptionHandler.CreateSubscription)
		subscriptions.GET("", r.subscriptionHandler.ListUserSubscriptions)
		subscriptions.GET("/:id", r.subscriptionHandler.GetSubscription)
		subscriptions.POST("/:id/cancel", r.subscriptionHandler.CancelSubscription)
		subscriptions.POST("/:id/renew", r.subscriptionHandler.RenewSubscription)
		subscriptions.POST("/:id/change-plan", r.subscriptionHandler.ChangePlan)

		subscriptions.POST("/:id/tokens", r.subscriptionTokenHandler.GenerateToken)
		subscriptions.GET("/:id/tokens", r.subscriptionTokenHandler.ListTokens)
		subscriptions.DELETE("/:id/tokens/:token_id", r.subscriptionTokenHandler.RevokeToken)
		subscriptions.POST("/:id/tokens/:token_id/refresh", r.subscriptionTokenHandler.RefreshToken)
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
		plans.GET("/public", r.subscriptionPlanHandler.GetPublicPlans)
		plans.GET("/:id/pricings", r.subscriptionPlanHandler.GetPlanPricings)

		plansProtected := plans.Group("")
		plansProtected.Use(r.authMiddleware.RequireAuth())
		{
			plansProtected.POST("", r.subscriptionPlanHandler.CreatePlan)
			plansProtected.PUT("/:id", r.subscriptionPlanHandler.UpdatePlan)
			plansProtected.GET("/:id", r.subscriptionPlanHandler.GetPlan)
			plansProtected.GET("", r.subscriptionPlanHandler.ListPlans)
			plansProtected.POST("/:id/activate", r.subscriptionPlanHandler.ActivatePlan)
			plansProtected.POST("/:id/deactivate", r.subscriptionPlanHandler.DeactivatePlan)
		}
	}

	routes.SetupNodeRoutes(r.engine, &routes.NodeRouteConfig{
		NodeHandler:         r.nodeHandler,
		NodeGroupHandler:    r.nodeGroupHandler,
		SubscriptionHandler: r.nodeSubscriptionHandler,
		NodeReportHandler:   r.nodeReportHandler,
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

		// GET /agents/:id/users - Get authorized users for node
		agentAPI.GET("/:id/users", r.agentHandler.GetUsers)

		// POST /agents/:id/traffic - Report user traffic data
		agentAPI.POST("/:id/traffic", r.agentHandler.ReportTraffic)

		// PUT /agents/:id/status - Update node system status
		agentAPI.PUT("/:id/status", r.agentHandler.UpdateStatus)

		// PUT /agents/:id/online-users - Update online users list
		agentAPI.PUT("/:id/online-users", r.agentHandler.UpdateOnlineUsers)
	}

	routes.SetupTicketRoutes(r.engine, &routes.TicketRouteConfig{
		TicketHandler:  r.ticketHandler,
		AuthMiddleware: r.authMiddleware,
	})

	routes.SetupNotificationRoutes(r.engine, &routes.NotificationRouteConfig{
		NotificationHandler: r.notificationHandler,
		AuthMiddleware:      r.authMiddleware,
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
