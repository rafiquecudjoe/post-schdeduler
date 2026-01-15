package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const scheduledPostsKey = "posts:scheduled"

// Queue manages the Redis-based scheduling queue
type Queue struct {
	redis *redis.Client
}

// NewQueue creates a new scheduling queue
func NewQueue(redisClient *redis.Client) *Queue {
	return &Queue{
		redis: redisClient,
	}
}

// Enqueue adds a post to the scheduling queue
func (q *Queue) Enqueue(ctx context.Context, postID uuid.UUID, scheduledAt time.Time) error {
	return q.redis.ZAdd(ctx, scheduledPostsKey, redis.Z{
		Score:  float64(scheduledAt.Unix()),
		Member: postID.String(),
	}).Err()
}

// Remove removes a post from the scheduling queue
func (q *Queue) Remove(ctx context.Context, postID uuid.UUID) error {
	return q.redis.ZRem(ctx, scheduledPostsKey, postID.String()).Err()
}

// Update updates a post's scheduled time in the queue
func (q *Queue) Update(ctx context.Context, postID uuid.UUID, scheduledAt time.Time) error {
	// ZADD with XX flag only updates if member exists
	// But for simplicity, we just re-add (ZADD updates score if member exists)
	return q.Enqueue(ctx, postID, scheduledAt)
}

// GetDuePosts retrieves posts that are due for publishing
// Uses atomic ZPOPMIN-like behavior to prevent duplicate processing
func (q *Queue) GetDuePosts(ctx context.Context, maxCount int) ([]uuid.UUID, error) {
	now := time.Now().Unix()

	// Get posts with scores <= now
	results, err := q.redis.ZRangeByScoreWithScores(ctx, scheduledPostsKey, &redis.ZRangeBy{
		Min:   "-inf",
		Max:   fmt.Sprintf("%d", now),
		Count: int64(maxCount),
	}).Result()

	if err != nil {
		return nil, err
	}

	var postIDs []uuid.UUID
	for _, result := range results {
		postIDStr, ok := result.Member.(string)
		if !ok {
			continue
		}

		postID, err := uuid.Parse(postIDStr)
		if err != nil {
			continue
		}

		// Try to remove atomically - if removal fails, another worker got it
		removed, err := q.redis.ZRem(ctx, scheduledPostsKey, postIDStr).Result()
		if err != nil || removed == 0 {
			continue
		}

		postIDs = append(postIDs, postID)
	}

	return postIDs, nil
}

// GetQueueLength returns the number of items in the scheduling queue
func (q *Queue) GetQueueLength(ctx context.Context) (int64, error) {
	return q.redis.ZCard(ctx, scheduledPostsKey).Result()
}
