package migrate

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/orris-inc/orris/internal/infrastructure/config"
	"github.com/orris-inc/orris/internal/infrastructure/database"
	"github.com/orris-inc/orris/internal/infrastructure/migration"
	"github.com/orris-inc/orris/internal/shared/logger"
)

var (
	env        string
	configPath string
	name       string
	steps      int
	version    int
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Database migration tools",
		Long:  `Manage database migrations including running migrations, checking status, and creating new migration files.`,
	}

	cmd.PersistentFlags().StringVarP(&env, "env", "e", "development", "Environment (development, test, production)")
	cmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "Path to config file (default: ./configs/config.yaml)")

	cmd.AddCommand(
		newUpCommand(),
		newDownCommand(),
		newStatusCommand(),
		newCreateCommand(),
		newGenerateUserCommand(),
	)

	return cmd
}

func newUpCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "up",
		Short: "Run all pending migrations",
		Long:  `Apply all pending database migrations to bring the database schema up to date.`,
		RunE:  runUp,
	}
}

func newDownCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "down",
		Short: "Rollback migrations",
		Long:  `Rollback a specified number of database migrations.`,
		RunE:  runDown,
	}

	cmd.Flags().IntVarP(&steps, "steps", "n", 1, "Number of migrations to rollback")

	return cmd
}

func newStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show migration status",
		Long:  `Display the current migration version and status of the database.`,
		RunE:  runStatus,
	}
}

func newForceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "force",
		Short: "Force set migration version and clear dirty flag",
		Long:  `Force the database migration to a specific version and clear the dirty flag. Use this to fix dirty migration state.`,
		RunE:  runForce,
	}

	cmd.Flags().IntVarP(&version, "version", "v", 1, "Version to force (required)")
	cmd.MarkFlagRequired("version")

	return cmd
}

func newCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new migration",
		Long:  `Create new migration files with the specified name.`,
		RunE:  runCreate,
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Name of the migration (required)")
	cmd.MarkFlagRequired("name")

	return cmd
}

func newGenerateUserCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "generate-user",
		Short: "Generate user table migration",
		Long:  `Generate the initial user table migration files.`,
		RunE:  runGenerateUser,
	}
}

func initEnv() (string, error) {
	cfg, err := config.Load(env, configPath)
	if err != nil {
		return "", fmt.Errorf("failed to load config: %w", err)
	}

	if err := logger.Init(&cfg.Logger); err != nil {
		return "", fmt.Errorf("failed to initialize logger: %w", err)
	}

	if err := database.Init(&cfg.Database); err != nil {
		return "", fmt.Errorf("failed to initialize database: %w", err)
	}

	scriptsPath, err := filepath.Abs("./internal/infrastructure/migration/scripts")
	if err != nil {
		return "", fmt.Errorf("failed to get scripts path: %w", err)
	}

	return scriptsPath, nil
}

func runUp(cmd *cobra.Command, args []string) error {
	scriptsPath, err := initEnv()
	if err != nil {
		return err
	}
	defer logger.Sync()
	defer database.Close()

	logger.Info("running up migrations",
		"environment", env)

	strategy := migration.NewGooseStrategy(scriptsPath)

	if err := strategy.Migrate(database.Get()); err != nil {
		logger.Error("migration failed", "error", err)
		return fmt.Errorf("migration failed: %w", err)
	}

	logger.Info("migrations completed successfully")
	return nil
}

func runDown(cmd *cobra.Command, args []string) error {
	scriptsPath, err := initEnv()
	if err != nil {
		return err
	}
	defer logger.Sync()
	defer database.Close()

	logger.Info("running down migrations",
		"environment", env,
		"steps", steps)

	strategy := migration.NewGooseStrategy(scriptsPath)

	if gooseStrategy, ok := strategy.(*migration.GooseStrategy); ok {
		if err := gooseStrategy.MigrateDown(database.Get(), steps); err != nil {
			logger.Error("down migration failed", "error", err)
			return fmt.Errorf("down migration failed: %w", err)
		}
	} else {
		return fmt.Errorf("down migration is only supported with goose strategy")
	}

	logger.Info("down migration completed successfully")
	return nil
}

func runStatus(cmd *cobra.Command, args []string) error {
	scriptsPath, err := initEnv()
	if err != nil {
		return err
	}
	defer logger.Sync()
	defer database.Close()

	logger.Info("checking migration status",
		"environment", env)

	strategy := migration.NewGooseStrategy(scriptsPath)

	if gooseStrategy, ok := strategy.(*migration.GooseStrategy); ok {
		version, err := gooseStrategy.GetVersion(database.Get())
		if err != nil {
			logger.Error("failed to get migration version", "error", err)
			return fmt.Errorf("failed to get migration version: %w", err)
		}

		fmt.Printf("\nMigration Status:\n")
		fmt.Printf("  Environment:     %s\n", env)
		fmt.Printf("  Current Version: %d\n", version)

		if err := gooseStrategy.Status(database.Get()); err != nil {
			logger.Error("failed to get detailed status", "error", err)
			return fmt.Errorf("failed to get detailed status: %w", err)
		}

		return nil
	}

	return fmt.Errorf("status check is only supported with goose strategy")
}

func runForce(cmd *cobra.Command, args []string) error {
	scriptsPath, err := initEnv()
	if err != nil {
		return err
	}
	defer logger.Sync()
	defer database.Close()

	logger.Info("forcing migration version",
		"environment", env,
		"version", version)

	strategy := migration.NewGolangMigrateStrategy(scriptsPath)

	if golangStrategy, ok := strategy.(*migration.GolangMigrateStrategy); ok {
		if err := golangStrategy.Force(database.Get(), version); err != nil {
			logger.Error("force migration failed", "error", err)
			return fmt.Errorf("force migration failed: %w", err)
		}

		fmt.Printf("✅ Migration version forced to %d\n", version)
		logger.Info("force migration completed successfully", "version", version)
		return nil
	}

	return fmt.Errorf("force is only supported with golang-migrate strategy")
}

func runCreate(cmd *cobra.Command, args []string) error {
	scriptsPath, err := initEnv()
	if err != nil {
		return err
	}
	defer logger.Sync()

	logger.Info("creating new migration",
		"name", name)

	strategy := migration.NewGooseStrategy(scriptsPath)
	if gooseStrategy, ok := strategy.(*migration.GooseStrategy); ok {
		if err := gooseStrategy.Create(name); err != nil {
			logger.Error("failed to create migration", "error", err)
			return fmt.Errorf("failed to create migration: %w", err)
		}
	} else {
		return fmt.Errorf("create is only supported with goose strategy")
	}

	logger.Info("migration created successfully",
		"name", name)
	fmt.Printf("✅ Migration '%s' created successfully\n", name)

	return nil
}

func runGenerateUser(cmd *cobra.Command, args []string) error {
	scriptsPath, err := initEnv()
	if err != nil {
		return err
	}
	defer logger.Sync()

	logger.Info("generating user table migration")

	generator := migration.NewGenerator(scriptsPath)
	if err := generator.CreateUserTableMigration(); err != nil {
		logger.Error("failed to generate user table migration", "error", err)
		return fmt.Errorf("failed to generate user table migration: %w", err)
	}

	logger.Info("user table migration generated successfully")
	fmt.Println("✅ User table migration generated successfully")

	return nil
}
