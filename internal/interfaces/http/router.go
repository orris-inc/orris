package http

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"

	"orris/internal/application/user"
	"orris/internal/application/user/usecases"
	"orris/internal/infrastructure/auth"
	"orris/internal/infrastructure/config"
	"orris/internal/infrastructure/email"
	"orris/internal/infrastructure/repository"
	"orris/internal/interfaces/http/handlers"
	"orris/internal/interfaces/http/middleware"
	"orris/internal/shared/logger"

	_ "orris/docs"
)

// Router represents the HTTP router configuration
type Router struct {
	engine         *gin.Engine
	userHandler    *handlers.UserHandler
	authHandler    *handlers.AuthHandler
	authMiddleware *middleware.AuthMiddleware
	rateLimiter    *middleware.RateLimiter
}

type jwtServiceAdapter struct {
	*auth.JWTService
}

func (a *jwtServiceAdapter) Generate(userID uint, sessionID string) (*usecases.TokenPair, error) {
	pair, err := a.JWTService.Generate(userID, sessionID)
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
		GetAuthURL(state string) string
		ExchangeCode(ctx context.Context, code string) (string, error)
		GetUserInfo(ctx context.Context, accessToken string) (*auth.OAuthUserInfo, error)
	}
}

func (a *oauthClientAdapter) GetAuthURL(state string) string {
	return a.client.GetAuthURL(state)
}

func (a *oauthClientAdapter) ExchangeCode(ctx context.Context, code string) (string, error) {
	return a.client.ExchangeCode(ctx, code)
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

	userRepo := repository.NewUserRepository(db)
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

	registerUC := usecases.NewRegisterWithPasswordUseCase(userRepo, hasher, emailService, log)
	loginUC := usecases.NewLoginWithPasswordUseCase(userRepo, sessionRepo, hasher, jwtService, log)
	verifyEmailUC := usecases.NewVerifyEmailUseCase(userRepo, log)
	requestResetUC := usecases.NewRequestPasswordResetUseCase(userRepo, emailService, log)
	resetPasswordUC := usecases.NewResetPasswordUseCase(userRepo, sessionRepo, hasher, emailService, log)
	initiateOAuthUC := usecases.NewInitiateOAuthLoginUseCase(googleClient, githubClient, log)
	handleOAuthUC := usecases.NewHandleOAuthCallbackUseCase(userRepo, oauthRepo, sessionRepo, googleClient, githubClient, jwtService, initiateOAuthUC, log)
	refreshTokenUC := usecases.NewRefreshTokenUseCase(sessionRepo, jwtService, log)
	logoutUC := usecases.NewLogoutUseCase(sessionRepo, log)

	authHandler := handlers.NewAuthHandler(
		registerUC, loginUC, verifyEmailUC, requestResetUC, resetPasswordUC,
		initiateOAuthUC, handleOAuthUC, refreshTokenUC, logoutUC, userRepo, log,
	)

	authMiddleware := middleware.NewAuthMiddleware(jwtSvc, log)
	rateLimiter := middleware.NewRateLimiter(100, 1*time.Minute)

	return &Router{
		engine:         engine,
		userHandler:    userHandler,
		authHandler:    authHandler,
		authMiddleware: authMiddleware,
		rateLimiter:    rateLimiter,
	}
}

// SetupRoutes configures all HTTP routes
func (r *Router) SetupRoutes() {
	r.engine.Use(middleware.Logger())
	r.engine.Use(middleware.Recovery())
	r.engine.Use(middleware.CORS())

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
		users.POST("", r.userHandler.CreateUser)
		users.GET("", r.userHandler.ListUsers)
		users.GET("/:id", r.userHandler.GetUser)
		users.PUT("/:id", r.userHandler.UpdateUser)
		users.DELETE("/:id", r.userHandler.DeleteUser)
		users.GET("/email/:email", r.userHandler.GetUserByEmail)
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
