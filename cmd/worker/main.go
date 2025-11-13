package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"orris/internal/infrastructure/cache"
	"orris/internal/infrastructure/config"
	"orris/internal/infrastructure/database"
	"orris/internal/infrastructure/repository"
	"orris/internal/shared/logger"
)

func main() {
	// Parse environment from command line or env variable
	env := "development"
	if len(os.Args) > 1 {
		env = os.Args[1]
	}
	if envVar := os.Getenv("ENV"); envVar != "" {
		env = envVar
	}

	// Load configuration
	cfg, err := config.Load(env)
	if err != nil {
		fmt.Printf("failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	if err := logger.Init(&cfg.Logger); err != nil {
		fmt.Printf("failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	log := logger.NewLogger()
	log.Infow("starting traffic sync worker", "environment", env)

	// Initialize database
	if err := database.Init(&cfg.Database); err != nil {
		log.Fatalw("failed to initialize database", "error", err)
	}
	defer database.Close()

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.GetAddr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer redisClient.Close()

	// Test Redis connection
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalw("failed to connect to redis", "error", err)
	}
	log.Infow("redis connection established", "address", cfg.Redis.GetAddr())

	// Initialize repositories
	nodeRepo := repository.NewNodeRepository(database.Get(), log)

	// Initialize traffic cache
	trafficCache := cache.NewRedisTrafficCache(redisClient, nodeRepo, log)

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start periodic sync (every 5 minutes)
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	// Run initial sync
	log.Infow("running initial traffic sync")
	if err := trafficCache.FlushToDatabase(ctx); err != nil {
		log.Errorw("initial traffic flush failed", "error", err)
	}

	log.Infow("traffic sync worker started, syncing every 5 minutes")

	for {
		select {
		case <-ticker.C:
			log.Infow("running scheduled traffic sync")
			if err := trafficCache.FlushToDatabase(ctx); err != nil {
				log.Errorw("traffic flush failed", "error", err)
			}

		case sig := <-sigChan:
			log.Infow("received signal, shutting down", "signal", sig)

			// Perform final flush before shutdown
			log.Infow("performing final traffic flush")
			flushCtx, flushCancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := trafficCache.FlushToDatabase(flushCtx); err != nil {
				log.Errorw("final traffic flush failed", "error", err)
			}
			flushCancel()

			log.Infow("traffic sync worker stopped")
			return
		}
	}
}
