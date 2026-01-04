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

	logger.Info("starting server",
		"environment", env,
		"version", "1.0.0")

	gin.SetMode(cfg.Server.Mode)

	gin.DefaultWriter = io.Discard
	gin.DebugPrintRouteFunc = func(httpMethod, absolutePath, handlerName string, nuHandlers int) {
	}

	// Initialize business timezone for date boundary calculations
	biztime.MustInit(cfg.Server.Timezone)
	logger.Info("business timezone initialized", "timezone", biztime.Location().String())

	if err := database.Init(&cfg.Database); err != nil {
		logger.Fatal("failed to initialize database", "error", err)
	}
	defer database.Close()

	if err := handleMigrations(env); err != nil {
		logger.Fatal("migration handling failed", "error", err)
	}

	userRepo := repository.NewUserRepository(database.Get(), logger.NewLogger())
	sessionRepo := repository.NewSessionRepository(database.Get())
	hasher := auth.NewBcryptPasswordHasher(cfg.Auth.Password.BcryptCost)

	userAppService := userApp.NewServiceDDD(userRepo, sessionRepo, hasher, logger.NewLogger())

	// Seed initial admin user if configured
	if err := seedAdminUser(cfg, userRepo, hasher); err != nil {
		logger.Warn("failed to seed admin user", "error", err)
	}

	router := httpRouter.NewRouter(userAppService, database.Get(), cfg, logger.NewLogger())
	router.SetupRoutes(cfg)

	// HTTP Server setup
	srv := &http.Server{
		Addr:         cfg.Server.GetAddr(),
		Handler:      router.GetEngine(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start HTTP Server in background
	go func() {
		logger.Info("server starting",
			"address", cfg.Server.GetAddr(),
			"mode", cfg.Server.Mode)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("failed to start server", "error", err)
		}
	}()

	// Wait for shutdown signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	// Shutdown router first (closes SSE connections, flushes traffic data, etc.)
	// This must happen before HTTP server shutdown to allow connections to close gracefully
	router.Shutdown()

	// Shutdown HTTP Server
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", "error", err)
	}

	logger.Info("server exited gracefully")
	return nil
}

func handleMigrations(environment string) error {
	if skipMigrationCheck {
		logger.Info("skipping migration check")
		return nil
	}

	logger.Info("checking migration status")

	scriptsPath, err := filepath.Abs("./internal/infrastructure/migration/scripts")
	if err != nil {
		logger.Warn("failed to get migration scripts path", "error", err)
		return nil
	}

	strategy := migration.NewGooseStrategy(scriptsPath)
	gooseStrategy, ok := strategy.(*migration.GooseStrategy)
	if !ok {
		logger.Warn("failed to cast to GooseStrategy")
		return nil
	}

	// Check if goose table exists (first startup detection)
	initialized, err := gooseStrategy.IsInitialized(database.Get())
	if err != nil {
		logger.Warn("failed to check goose initialization", "error", err)
		return nil
	}

	if !initialized {
		// First startup: execute all migrations
		logger.Info("first startup detected, running migrations")
		if err := gooseStrategy.Migrate(database.Get()); err != nil {
			return fmt.Errorf("failed to run initial migrations: %w", err)
		}
		logger.Info("initial migrations completed successfully")
	} else {
		// Already initialized: just show current version
		version, err := gooseStrategy.GetVersion(database.Get())
		if err != nil {
			logger.Warn("failed to check migration status", "error", err)
		} else {
			logger.Info("current migration version", "version", version)
		}
	}

	logger.Info("migration check completed")

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
func seedAdminUser(cfg *config.Config, userRepo user.Repository, hasher *auth.BcryptPasswordHasher) error {
	// Check if admin config is provided
	if !cfg.Admin.IsConfigured() {
		logger.Info("admin config not provided, skipping admin user creation")
		return nil
	}

	ctx := context.Background()

	// Check if user with admin email already exists
	existingUser, err := userRepo.GetByEmail(ctx, cfg.Admin.Email)
	if err == nil && existingUser != nil {
		logger.Info("admin user already exists", "email", cfg.Admin.Email)
		return nil
	}

	// Create admin user
	logger.Info("creating initial admin user", "email", cfg.Admin.Email)

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

	logger.Info("admin user created successfully", "email", cfg.Admin.Email, "id", adminUser.ID())
	return nil
}
