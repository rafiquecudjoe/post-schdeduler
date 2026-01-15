package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimiter configuration
type RateLimiterConfig struct {
	Limit  int           // Max requests
	Window time.Duration // Time window
}

// RateLimiter creates a rate limiting middleware using Redis
func RateLimiter(redisClient *redis.Client, config RateLimiterConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Get client identifier (IP address)
			clientIP := r.RemoteAddr
			if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
				clientIP = forwarded
			}

			// Create rate limit key
			key := fmt.Sprintf("ratelimit:%s:%s", clientIP, r.URL.Path)

			// Check and increment counter
			allowed, remaining, err := checkRateLimit(ctx, redisClient, key, config)
			if err != nil {
				// Fail closed - reject request when Redis unavailable for security
				http.Error(w, `{"error":"Service Unavailable","message":"Rate limiting service unavailable"}`, http.StatusServiceUnavailable)
				return
			}

			// Set rate limit headers
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", config.Limit))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(config.Window).Unix()))

			if !allowed {
				w.Header().Set("Retry-After", fmt.Sprintf("%d", int(config.Window.Seconds())))
				http.Error(w, `{"error":"Too Many Requests","message":"Rate limit exceeded. Please try again later."}`, http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func checkRateLimit(ctx context.Context, client *redis.Client, key string, config RateLimiterConfig) (allowed bool, remaining int, err error) {
	// Use a pipeline for atomic operations
	pipe := client.Pipeline()

	// Increment the counter
	incrCmd := pipe.Incr(ctx, key)

	// Set expiration only if this is a new key
	pipe.Expire(ctx, key, config.Window)

	// Execute pipeline
	_, err = pipe.Exec(ctx)
	if err != nil {
		return false, 0, err
	}

	count := int(incrCmd.Val())
	remaining = config.Limit - count
	if remaining < 0 {
		remaining = 0
	}

	return count <= config.Limit, remaining, nil
}

// Default rate limit configurations
var (
	AuthRateLimit = RateLimiterConfig{
		Limit:  5,
		Window: time.Minute,
	}

	RegisterRateLimit = RateLimiterConfig{
		Limit:  3,
		Window: time.Minute,
	}

	APIRateLimit = RateLimiterConfig{
		Limit:  100,
		Window: time.Minute,
	}

	CreatePostRateLimit = RateLimiterConfig{
		Limit:  30,
		Window: time.Minute,
	}
)
