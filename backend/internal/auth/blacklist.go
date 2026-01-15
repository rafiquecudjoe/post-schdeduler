package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Blacklist manages token blacklisting in Redis
type Blacklist struct {
	redis *redis.Client
}

// NewBlacklist creates a new token blacklist
func NewBlacklist(redisClient *redis.Client) *Blacklist {
	return &Blacklist{
		redis: redisClient,
	}
}

// Add adds a token JTI to the blacklist with the given TTL
func (b *Blacklist) Add(ctx context.Context, jti string, ttl time.Duration) error {
	key := fmt.Sprintf("blacklist:%s", jti)
	return b.redis.Set(ctx, key, "1", ttl).Err()
}

// IsBlacklisted checks if a token JTI is blacklisted
func (b *Blacklist) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	key := fmt.Sprintf("blacklist:%s", jti)
	result, err := b.redis.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}
