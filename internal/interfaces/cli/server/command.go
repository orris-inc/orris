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

	userApp "orris/internal/application/user"
	"orris/internal/domain/shared/events"
	"orris/internal/infrastructure/config"
	"orris/internal/infrastructure/database"
	"orris/internal/infrastructure/migration"
	"orris/internal/infrastructure/repository"
	httpRouter "orris/internal/interfaces/http"
	"orris/internal/shared/logger"
)

var (
	env                string
	autoMigrate        bool
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
	cmd.Flags().BoolVar(&autoMigrate, "auto-migrate", false, "Automatically run database migrations on startup (not recommended for production)")
	cmd.Flags().BoolVar(&skipMigrationCheck, "skip-migration-check", false, "Skip migration status check on startup")

	return cmd
}

func run(cmd *cobra.Command, args []string) error {
	if envVar := os.Getenv("ENV"); envVar != "" {
		env = envVar
	}

	ginMode := mapEnvToGinMode(env)

	cfg, err := config.Load(env)
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
		"version", "1.0.0",
		"auto-migrate", autoMigrate)

	gin.SetMode(cfg.Server.Mode)

	gin.DefaultWriter = io.Discard
	gin.DebugPrintRouteFunc = func(httpMethod, absolutePath, handlerName string, nuHandlers int) {
	}

	if err := database.Init(&cfg.Database); err != nil {
		logger.Fatal("failed to initialize database", "error", err)
	}
	defer database.Close()

	if err := handleMigrations(env); err != nil {
		logger.Fatal("migration handling failed", "error", err)
	}

	eventDispatcher := events.NewInMemoryEventDispatcher(100)
	if err := eventDispatcher.Start(); err != nil {
		logger.Fatal("failed to start event dispatcher", "error", err)
	}
	defer func() {
		if err := eventDispatcher.Stop(); err != nil {
			logger.Error("failed to stop event dispatcher", "error", err)
		}
	}()
	logger.Info("event dispatcher started")

	userRepo := repository.NewUserRepositoryDDD(database.Get(), eventDispatcher, logger.NewLogger())

	userAppService := userApp.NewServiceDDD(userRepo, eventDispatcher, logger.NewLogger())

	router := httpRouter.NewRouter(userAppService, database.Get(), cfg, logger.NewLogger())
	router.SetupRoutes()

	srv := &http.Server{
		Addr:         cfg.Server.GetAddr(),
		Handler:      router.GetEngine(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("server starting",
			"address", cfg.Server.GetAddr(),
			"mode", cfg.Server.Mode)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("failed to start server", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", "error", err)
		return err
	}

	logger.Info("server exited gracefully")
	return nil
}

func handleMigrations(environment string) error {
	if skipMigrationCheck {
		logger.Info("skipping migration check")
		return nil
	}

	if autoMigrate {
		if environment == "production" {
			logger.Warn("auto-migration is enabled in production environment - this is not recommended!")
		}

		logger.Warn("auto-migration is deprecated, please use 'orris migrate up' instead")
		return fmt.Errorf("auto-migration is no longer supported, use 'orris migrate up'")
	}

	logger.Info("checking migration status")

	scriptsPath, err := filepath.Abs("./internal/infrastructure/migration/scripts")
	if err != nil {
		logger.Warn("failed to get migration scripts path", "error", err)
		return nil
	}

	strategy := migration.NewGooseStrategy(scriptsPath)
	if gooseStrategy, ok := strategy.(*migration.GooseStrategy); ok {
		version, err := gooseStrategy.GetVersion(database.Get())
		if err != nil {
			logger.Warn("failed to check migration status", "error", err)
		} else {
			logger.Info("current migration version",
				"version", version)
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
