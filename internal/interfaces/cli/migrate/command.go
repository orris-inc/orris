package migrate

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/orris-inc/orris/internal/infrastructure/config"
	"github.com/orris-inc/orris/internal/infrastructure/database"
	"github.com/orris-inc/orris/internal/infrastructure/migration"
	"github.com/orris-inc/orris/internal/shared/biztime"
	"github.com/orris-inc/orris/internal/shared/logger"
)

var (
	env        string
	configPath string
	name       string
	steps      int
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

func initEnv() (string, logger.Interface, error) {
	cfg, err := config.Load(env, configPath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to load config: %w", err)
	}

	if err := logger.Init(&cfg.Logger); err != nil {
		return "", nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	log := logger.NewLogger()

	// Initialize business timezone for date boundary calculations
	if err := biztime.Init(cfg.Server.Timezone); err != nil {
		return "", nil, fmt.Errorf("failed to initialize business timezone: %w", err)
	}

	if err := database.Init(&cfg.Database); err != nil {
		return "", nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	scriptsPath, err := filepath.Abs("./internal/infrastructure/migration/scripts")
	if err != nil {
		return "", nil, fmt.Errorf("failed to get scripts path: %w", err)
	}

	return scriptsPath, log, nil
}

func runUp(cmd *cobra.Command, args []string) error {
	scriptsPath, log, err := initEnv()
	if err != nil {
		return err
	}
	defer logger.Sync()
	defer database.Close()

	log.Infow("running up migrations", "environment", env)

	strategy := migration.NewGooseStrategy(scriptsPath, log)

	if err := strategy.Migrate(database.Get()); err != nil {
		log.Errorw("migration failed", "error", err)
		return fmt.Errorf("migration failed: %w", err)
	}

	log.Infow("migrations completed successfully")
	return nil
}

func runDown(cmd *cobra.Command, args []string) error {
	scriptsPath, log, err := initEnv()
	if err != nil {
		return err
	}
	defer logger.Sync()
	defer database.Close()

	log.Infow("running down migrations", "environment", env, "steps", steps)

	strategy := migration.NewGooseStrategy(scriptsPath, log)

	if gooseStrategy, ok := strategy.(*migration.GooseStrategy); ok {
		if err := gooseStrategy.MigrateDown(database.Get(), steps); err != nil {
			log.Errorw("down migration failed", "error", err)
			return fmt.Errorf("down migration failed: %w", err)
		}
	} else {
		return fmt.Errorf("down migration is only supported with goose strategy")
	}

	log.Infow("down migration completed successfully")
	return nil
}

func runStatus(cmd *cobra.Command, args []string) error {
	scriptsPath, log, err := initEnv()
	if err != nil {
		return err
	}
	defer logger.Sync()
	defer database.Close()

	log.Infow("checking migration status", "environment", env)

	strategy := migration.NewGooseStrategy(scriptsPath, log)

	if gooseStrategy, ok := strategy.(*migration.GooseStrategy); ok {
		version, err := gooseStrategy.GetVersion(database.Get())
		if err != nil {
			log.Errorw("failed to get migration version", "error", err)
			return fmt.Errorf("failed to get migration version: %w", err)
		}

		fmt.Printf("\nMigration Status:\n")
		fmt.Printf("  Environment:     %s\n", env)
		fmt.Printf("  Current Version: %d\n", version)

		if err := gooseStrategy.Status(database.Get()); err != nil {
			log.Errorw("failed to get detailed status", "error", err)
			return fmt.Errorf("failed to get detailed status: %w", err)
		}

		return nil
	}

	return fmt.Errorf("status check is only supported with goose strategy")
}

func runCreate(cmd *cobra.Command, args []string) error {
	scriptsPath, log, err := initEnv()
	if err != nil {
		return err
	}
	defer logger.Sync()

	log.Infow("creating new migration", "name", name)

	strategy := migration.NewGooseStrategy(scriptsPath, log)
	if gooseStrategy, ok := strategy.(*migration.GooseStrategy); ok {
		if err := gooseStrategy.Create(name); err != nil {
			log.Errorw("failed to create migration", "error", err)
			return fmt.Errorf("failed to create migration: %w", err)
		}
	} else {
		return fmt.Errorf("create is only supported with goose strategy")
	}

	log.Infow("migration created successfully", "name", name)
	fmt.Printf("✅ Migration '%s' created successfully\n", name)

	return nil
}

func runGenerateUser(cmd *cobra.Command, args []string) error {
	scriptsPath, log, err := initEnv()
	if err != nil {
		return err
	}
	defer logger.Sync()

	log.Infow("generating user table migration")

	generator := migration.NewGenerator(scriptsPath, log)
	if err := generator.CreateUserTableMigration(); err != nil {
		log.Errorw("failed to generate user table migration", "error", err)
		return fmt.Errorf("failed to generate user table migration: %w", err)
	}

	log.Infow("user table migration generated successfully")
	fmt.Println("✅ User table migration generated successfully")

	return nil
}
