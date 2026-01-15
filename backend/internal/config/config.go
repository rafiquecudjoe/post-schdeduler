package config

import (
	"log"
	"os"
	"time"
)

type Config struct {
	DatabaseURL     string
	RedisURL        string
	JWTSecret       string
	CORSOrigin      string
	ServerPort      string
	SecureCookies   bool
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	WorkerInterval  time.Duration
}

func Load() *Config {
	cfg := &Config{
		DatabaseURL:     getEnvRequired("DATABASE_URL"),
		RedisURL:        getEnvRequired("REDIS_URL"),
		JWTSecret:       getEnvRequired("JWT_SECRET"),
		CORSOrigin:      getEnv("CORS_ORIGIN", "http://localhost:3000"),
		ServerPort:      getEnv("SERVER_PORT", "8080"),
		SecureCookies:   getEnv("SECURE_COOKIES", "false") == "true",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
		WorkerInterval:  2 * time.Second, // Reduced to 2 seconds for faster publishing
	}

	// Validate JWT secret strength
	if len(cfg.JWTSecret) < 32 {
		log.Fatal("JWT_SECRET must be at least 32 characters for security")
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvRequired(key string) string {
	value, exists := os.LookupEnv(key)
	if !exists || value == "" {
		log.Fatalf("%s environment variable is required but not set", key)
	}
	return value
}
