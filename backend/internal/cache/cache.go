package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/scheduler/backend/internal/models"
)

// Cache provides Redis-based caching for frequently accessed data
type Cache struct {
	redis *redis.Client
}

// NewCache creates a new cache instance
func NewCache(redisClient *redis.Client) *Cache {
	return &Cache{
		redis: redisClient,
	}
}

// TTL configurations
const (
	UpcomingPostsTTL = 30 * time.Second
	HistoryPostsTTL  = 60 * time.Second
)

// Cache key patterns
func upcomingKey(userID uuid.UUID) string {
	return fmt.Sprintf("cache:posts:upcoming:%s", userID.String())
}

func historyKey(userID uuid.UUID) string {
	return fmt.Sprintf("cache:posts:history:%s", userID.String())
}

// GetUpcomingPosts retrieves cached upcoming posts for a user
func (c *Cache) GetUpcomingPosts(ctx context.Context, userID uuid.UUID) ([]*models.Post, bool) {
	data, err := c.redis.Get(ctx, upcomingKey(userID)).Bytes()
	if err != nil {
		return nil, false
	}

	var posts []*models.Post
	if err := json.Unmarshal(data, &posts); err != nil {
		return nil, false
	}

	return posts, true
}

// SetUpcomingPosts caches upcoming posts for a user
func (c *Cache) SetUpcomingPosts(ctx context.Context, userID uuid.UUID, posts []*models.Post) error {
	data, err := json.Marshal(posts)
	if err != nil {
		return err
	}

	return c.redis.Set(ctx, upcomingKey(userID), data, UpcomingPostsTTL).Err()
}

// GetHistoryPosts retrieves cached published posts for a user
func (c *Cache) GetHistoryPosts(ctx context.Context, userID uuid.UUID) ([]*models.Post, bool) {
	data, err := c.redis.Get(ctx, historyKey(userID)).Bytes()
	if err != nil {
		return nil, false
	}

	var posts []*models.Post
	if err := json.Unmarshal(data, &posts); err != nil {
		return nil, false
	}

	return posts, true
}

// SetHistoryPosts caches published posts for a user
func (c *Cache) SetHistoryPosts(ctx context.Context, userID uuid.UUID, posts []*models.Post) error {
	data, err := json.Marshal(posts)
	if err != nil {
		return err
	}

	return c.redis.Set(ctx, historyKey(userID), data, HistoryPostsTTL).Err()
}

// InvalidateUserPosts removes all cached posts for a user
func (c *Cache) InvalidateUserPosts(ctx context.Context, userID uuid.UUID) error {
	keys := []string{
		upcomingKey(userID),
		historyKey(userID),
	}

	return c.redis.Del(ctx, keys...).Err()
}

// InvalidateByPostID finds and invalidates cache for a specific post's user
// This is useful when the worker publishes a post
func (c *Cache) InvalidateByUserID(ctx context.Context, userID uuid.UUID) error {
	return c.InvalidateUserPosts(ctx, userID)
}
