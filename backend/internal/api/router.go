package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/redis/go-redis/v9"
	"github.com/scheduler/backend/internal/api/handlers"
	"github.com/scheduler/backend/internal/api/middleware"
	"github.com/scheduler/backend/internal/auth"
	"github.com/scheduler/backend/internal/cache"
	"github.com/scheduler/backend/internal/db"
	"github.com/scheduler/backend/internal/notifier"
	"github.com/scheduler/backend/internal/scheduler"
)

// NewRouter creates and configures the HTTP router
func NewRouter(
	database *db.DB,
	jwtService *auth.JWTService,
	blacklist *auth.Blacklist,
	queue *scheduler.Queue,
	redisClient *redis.Client,
	corsOrigin string,
	secureCookies bool,
) *chi.Mux {
	r := chi.NewRouter()

	// Initialize cache
	postCache := cache.NewCache(redisClient)

	// Initialize notifier for real-time updates (with Redis pub/sub)
	postNotifier := notifier.NewNotifier(redisClient)

	// Global middleware
	r.Use(middleware.Logger)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{corsOrigin},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(database, jwtService, blacklist, secureCookies)
	postHandler := handlers.NewPostHandler(database, queue, postCache, postNotifier)
	sseHandler := handlers.NewSSEHandler(database, postNotifier)

	// Auth middleware
	authMiddleware := middleware.Auth(jwtService, database)

	// Rate limit middleware
	authRateLimit := middleware.RateLimiter(redisClient, middleware.AuthRateLimit)
	registerRateLimit := middleware.RateLimiter(redisClient, middleware.RegisterRateLimit)
	createPostRateLimit := middleware.RateLimiter(redisClient, middleware.CreatePostRateLimit)
	apiRateLimit := middleware.RateLimiter(redisClient, middleware.APIRateLimit)

	// Routes
	r.Route("/api", func(r chi.Router) {
		// Public auth routes with rate limiting
		r.Route("/auth", func(r chi.Router) {
			r.With(registerRateLimit).Post("/register", authHandler.Register)
			r.With(authRateLimit).Post("/login", authHandler.Login)
			r.Post("/logout", authHandler.Logout)
			r.Post("/refresh", authHandler.Refresh)

			// Protected auth route
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware)
				r.Get("/me", authHandler.Me)
			})
		})

		// Protected post routes with rate limiting
		r.Route("/posts", func(r chi.Router) {
			r.Use(authMiddleware)
			r.Use(apiRateLimit)

			r.With(createPostRateLimit).Post("/", postHandler.Create)
			r.Get("/upcoming", postHandler.GetUpcoming)
			r.Get("/history", postHandler.GetHistory)
			r.Get("/stream", sseHandler.StreamPosts) // SSE endpoint for real-time updates
			r.Get("/{id}", postHandler.GetByID)
			r.Put("/{id}", postHandler.Update)
			r.Delete("/{id}", postHandler.Delete)
		})
	})

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	return r
}
