package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/scheduler/backend/internal/api"
	"github.com/scheduler/backend/internal/auth"
	"github.com/scheduler/backend/internal/cache"
	"github.com/scheduler/backend/internal/config"
	"github.com/scheduler/backend/internal/db"
	"github.com/scheduler/backend/internal/scheduler"
)

func main() {
	// Parse flags
	workerMode := flag.Bool("worker", false, "Run in worker mode")
	flag.Parse()

	// Load configuration
	cfg := config.Load()

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Shutdown signal received")
		cancel()
	}()

	// Connect to database
	database, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()
	log.Println("✅ Connected to PostgreSQL")

	// Run migrations only in API server mode (not in worker mode)
	if !*workerMode {
		if err := database.RunMigrations(ctx); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}
		log.Println("✅ Database migrations complete")
	}

	// Connect to Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.RedisURL,
	})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()
	log.Println("✅ Connected to Redis")

	// Initialize services
	jwtService := auth.NewJWTService(cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	blacklist := auth.NewBlacklist(redisClient)
	queue := scheduler.NewQueue(redisClient)

	if *workerMode {
		// Run as worker
		log.Println("🔧 Starting in WORKER mode")
		postCache := cache.NewCache(redisClient)
		worker := scheduler.NewWorker(database, queue, postCache, cfg.WorkerInterval)
		worker.Run(ctx)
	} else {
		// Run as API server
		log.Println("🌐 Starting in API SERVER mode")

		router := api.NewRouter(database, jwtService, blacklist, queue, redisClient, cfg.CORSOrigin, cfg.SecureCookies)

		server := &http.Server{
			Addr:         ":" + cfg.ServerPort,
			Handler:      router,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 0, // No write timeout for SSE streaming
			IdleTimeout:  120 * time.Second,
		}

		// Start server in goroutine
		go func() {
			log.Printf("🚀 Server listening on port %s", cfg.ServerPort)
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Server error: %v", err)
			}
		}()

		// Wait for shutdown
		<-ctx.Done()

		// Graceful shutdown
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
		log.Println("Server stopped")
	}

	fmt.Println("Goodbye!")
}
