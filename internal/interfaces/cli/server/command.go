package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"

	userApp "github.com/orris-inc/orris/internal/application/user"
	"github.com/orris-inc/orris/internal/domain/user"
	vo "github.com/orris-inc/orris/internal/domain/user/valueobjects"
	"github.com/orris-inc/orris/internal/infrastructure/auth"
	"github.com/orris-inc/orris/internal/infrastructure/config"
	"github.com/orris-inc/orris/internal/infrastructure/database"
	"github.com/orris-inc/orris/internal/infrastructure/migration"
	"github.com/orris-inc/orris/internal/infrastructure/repository"
	httpRouter "github.com/orris-inc/orris/internal/interfaces/http"
	"github.com/orris-inc/orris/internal/shared/authorization"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/goroutine"
	"github.com/orris-inc/orris/internal/shared/id"
	"github.com/orris-inc/orris/internal/shared/logger"
)

var (
	env                string
	configPath         string
	skipMigrationCheck bool
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start the HTTP server",
		Long:  `Start the Orris HTTP server with specified configuration.`,
		RunE:  run,
	}

	cmd.Flags().StringVarP(&env, "env", "e", "development", "Environment (development, test, production)")
	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to config file (default: ./configs/config.yaml)")
	cmd.Flags().BoolVar(&skipMigrationCheck, "skip-migration-check", false, "Skip migration status check on startup")

	return cmd
}

func run(cmd *cobra.Command, args []string) error {
	if envVar := os.Getenv("ENV"); envVar != "" {
		env = envVar
	}

	ginMode := mapEnvToGinMode(env)

	cfg, err := config.Load(env, configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cfg.Server.Mode = ginMode

	if err := logger.Init(&cfg.Logger); err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer logger.Sync()

	log := logger.NewLogger()

	log.Infow("initializing server", "environment", env)

	gin.SetMode(cfg.Server.Mode)

	gin.DefaultWriter = io.Discard
	gin.DebugPrintRouteFunc = func(httpMethod, absolutePath, handlerName string, nuHandlers int) {
	}

	// Initialize business timezone for date boundary calculations
	biztime.MustInit(cfg.Server.Timezone)
	log.Infow("business timezone initialized", "timezone", biztime.Location().String())

	if err := database.Init(&cfg.Database); err != nil {
		log.Fatalw("failed to initialize database", "error", err)
	}
	defer database.Close()

	if err := handleMigrations(log); err != nil {
		log.Fatalw("migration handling failed", "error", err)
	}

	userRepo := repository.NewUserRepository(database.Get(), log)
	sessionRepo := repository.NewSessionRepository(database.Get())
	hasher := auth.NewBcryptPasswordHasher(cfg.Auth.Password.BcryptCost)

	userAppService := userApp.NewServiceDDD(userRepo, sessionRepo, hasher, log)

	// Seed initial admin user if configured
	if err := seedAdminUser(cfg, userRepo, hasher, log); err != nil {
		log.Warnw("failed to seed admin user", "error", err)
	}

	router := httpRouter.NewRouter(userAppService, database.Get(), cfg, log)
	router.SetupRoutes(cfg)

	ctx := context.Background()

	// Start BotServiceManager (handles both polling and webhook modes with hot-reload)
	if err := router.StartTelegramPolling(ctx); err != nil {
		log.Warnw("failed to start telegram bot service", "error", err)
	}

	// Register reminder jobs if telegram service is available
	if telegramService := router.GetTelegramService(); telegramService != nil {
		if schedulerMgr := router.GetSchedulerManager(); schedulerMgr != nil {
			if err := schedulerMgr.RegisterReminderJobs(telegramService.GetProcessReminderUseCase()); err != nil {
				log.Warnw("failed to register reminder jobs", "error", err)
			}
		}
	}

	// Start unified scheduler (all jobs: payment, subscription, usage aggregation, reminder, admin notifications)
	router.StartScheduler()

	// Start USDT payment monitor scheduler (managed separately by USDTServiceManager)
	router.StartUSDTMonitorScheduler(ctx)
	log.Infow("all schedulers started")

	// HTTP Server setup
	srv := &http.Server{
		Addr:         cfg.Server.GetAddr(),
		Handler:      router.GetEngine(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start HTTP Server in background
	goroutine.SafeGo(log, "http-server", func() {
		log.Infow("HTTP server listening",
			"address", cfg.Server.GetAddr(),
			"mode", cfg.Server.Mode)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalw("failed to start server", "error", err)
		}
	})

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Infow("shutting down server")

	// Shutdown router first (stops scheduler, closes SSE connections, flushes traffic data, etc.)
	// This must happen before HTTP server shutdown to allow connections to close gracefully
	router.Shutdown()

	// Shutdown HTTP Server
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Errorw("server forced to shutdown", "error", err)
	}

	log.Infow("server exited gracefully")
	return nil
}

func handleMigrations(log logger.Interface) error {
	if skipMigrationCheck {
		log.Debugw("skipping migration check")
		return nil
	}

	scriptsPath, err := filepath.Abs("./internal/infrastructure/migration/scripts")
	if err != nil {
		log.Warnw("failed to get migration scripts path", "error", err)
		return nil
	}

	// Check if scripts directory exists, skip status check if not (e.g., production deployment)
	if _, err := os.Stat(scriptsPath); os.IsNotExist(err) {
		// Still try to get current version from database
		strategy := migration.NewGooseStrategy(scriptsPath, log)
		if gooseStrategy, ok := strategy.(*migration.GooseStrategy); ok {
			if version, err := gooseStrategy.GetVersion(database.Get()); err == nil {
				log.Infow("current database migration version", "version", version)
			}
		}
		log.Debugw("migration scripts not found, skipping status check")
		return nil
	}

	strategy := migration.NewGooseStrategy(scriptsPath, log)
	gooseStrategy, ok := strategy.(*migration.GooseStrategy)
	if !ok {
		log.Warnw("failed to cast to GooseStrategy")
		return nil
	}

	// Check current migration version
	version, err := gooseStrategy.GetVersion(database.Get())
	if err != nil {
		log.Warnw("failed to get migration version", "error", err)
	} else {
		log.Infow("current database migration version", "version", version)
	}

	return nil
}

func mapEnvToGinMode(environment string) string {
	switch environment {
	case "production", "prod":
		return "release"
	case "development", "dev":
		return "debug"
	case "test", "testing":
		return "test"
	case "debug":
		return "debug"
	case "release":
		return "release"
	default:
		return "debug"
	}
}

// seedAdminUser creates initial admin user if configured via environment variables
func seedAdminUser(cfg *config.Config, userRepo user.Repository, hasher *auth.BcryptPasswordHasher, log logger.Interface) error {
	// Check if admin config is provided
	if !cfg.Admin.IsConfigured() {
		log.Debugw("admin config not provided, skipping admin user creation")
		return nil
	}

	ctx := context.Background()

	// Check if user with admin email already exists
	existingUser, err := userRepo.GetByEmail(ctx, cfg.Admin.Email)
	if err == nil && existingUser != nil {
		log.Debugw("admin user already exists", "email", cfg.Admin.Email)
		return nil
	}

	// Create admin user
	log.Infow("creating initial admin user", "email", cfg.Admin.Email)

	// Create value objects
	email, err := vo.NewEmail(cfg.Admin.Email)
	if err != nil {
		return fmt.Errorf("invalid admin email: %w", err)
	}

	// Set default name if not provided
	adminName := cfg.Admin.Name
	if adminName == "" {
		adminName = "Admin"
	}

	name, err := vo.NewName(adminName)
	if err != nil {
		return fmt.Errorf("invalid admin name: %w", err)
	}

	password, err := vo.NewPassword(cfg.Admin.Password)
	if err != nil {
		return fmt.Errorf("invalid admin password: %w", err)
	}

	// Create user domain object
	adminUser, err := user.NewUser(email, name, id.NewUserID)
	if err != nil {
		return fmt.Errorf("failed to create admin user object: %w", err)
	}

	// Set password
	if err := adminUser.SetPassword(password, hasher); err != nil {
		return fmt.Errorf("failed to set admin password: %w", err)
	}

	// Set admin role
	adminUser.SetRole(authorization.RoleAdmin)

	// Activate user to allow immediate login
	if err := adminUser.Activate(); err != nil {
		return fmt.Errorf("failed to activate admin user: %w", err)
	}

	// Save to database
	if err := userRepo.Create(ctx, adminUser); err != nil {
		return fmt.Errorf("failed to save admin user: %w", err)
	}

	log.Infow("admin user created successfully", "email", cfg.Admin.Email, "id", adminUser.ID())
	return nil
}
