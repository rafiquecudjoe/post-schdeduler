package scheduler

import (
	"context"
	"log"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/scheduler/backend/internal/cache"
	"github.com/scheduler/backend/internal/db"
	"github.com/scheduler/backend/internal/notifier"
)

const (
	// MaxRetries is the maximum number of retry attempts
	MaxRetries = 3
)

// Worker handles background post publishing
type Worker struct {
	db       *db.DB
	queue    *Queue
	cache    *cache.Cache
	notifier *notifier.Notifier
	interval time.Duration
}

// NewWorker creates a new background worker
func NewWorker(database *db.DB, queue *Queue, postCache *cache.Cache, n *notifier.Notifier, interval time.Duration) *Worker {
	return &Worker{
		db:       database,
		queue:    queue,
		cache:    postCache,
		notifier: n,
		interval: interval,
	}
}

// Run starts the worker loop
func (w *Worker) Run(ctx context.Context) {
	log.Printf("🔄 Worker started, polling every %v", w.interval)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Process immediately on start
	w.processDuePosts(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("⏹️ Worker stopped")
			return
		case <-ticker.C:
			w.processDuePosts(ctx)
		}
	}
}

// processDuePosts processes all posts that are due for publishing
func (w *Worker) processDuePosts(ctx context.Context) {
	// Get due posts from Redis queue
	postIDs, err := w.queue.GetDuePosts(ctx, 100)
	if err != nil {
		log.Printf("❌ Error getting due posts from queue: %v", err)
		return
	}

	if len(postIDs) == 0 {
		return
	}

	log.Printf("📋 Found %d posts to publish", len(postIDs))

	for _, postID := range postIDs {
		if err := w.publishPost(ctx, postID); err != nil {
			log.Printf("❌ Failed to publish post %s: %v", postID, err)
		}
	}
}

// publishPost publishes a single post with retry logic
func (w *Worker) publishPost(ctx context.Context, postID uuid.UUID) error {
	// Get post with retry info
	post, err := w.db.GetPostForRetry(ctx, postID)
	if err != nil {
		return err
	}

	if post == nil {
		log.Printf("⚠️ Post %s not found", postID)
		return nil
	}

	if post.Status != "scheduled" {
		log.Printf("⚠️ Post %s is not scheduled (status: %s)", postID, post.Status)
		return nil
	}

	// Attempt to publish (mock publishing - in real app, this would call social media APIs)
	publishErr := w.mockPublish(post)

	if publishErr != nil {
		// Handle failure with retry logic
		return w.handlePublishError(ctx, post, publishErr)
	}

	// Success - mark as published
	publishedPost, err := w.db.PublishPost(ctx, postID)
	if err != nil {
		return err
	}

	if publishedPost == nil {
		log.Printf("⚠️ Post %s not found or already published", postID)
		return nil
	}

	// Invalidate cache for this user
	if w.cache != nil {
		_ = w.cache.InvalidateUserPosts(ctx, post.UserID)
	}

	// Notify SSE clients via Redis pub/sub
	if w.notifier != nil {
		w.notifier.Notify(post.UserID, notifier.UpdateTypePublish)
	}

	log.Printf("📤 Published post %s to %s: %s", post.ID, post.Channel, truncate(post.Content, 50))
	return nil
}

// mockPublish simulates publishing to a social media platform
// In a real application, this would make API calls to Twitter, LinkedIn, etc.
func (w *Worker) mockPublish(post *db.PostWithRetry) error {
	// Simulate occasional failures for testing (1 in 10 chance)
	// In production, remove this and implement real API calls
	// if rand.Intn(10) == 0 {
	// 	return fmt.Errorf("simulated API failure")
	// }
	return nil
}

// handlePublishError handles a failed publish attempt with exponential backoff
func (w *Worker) handlePublishError(ctx context.Context, post *db.PostWithRetry, publishErr error) error {
	retryCount := post.RetryCount + 1
	errorMsg := publishErr.Error()

	if retryCount >= MaxRetries {
		// Max retries exceeded, mark as failed
		log.Printf("❌ Post %s failed after %d retries: %s", post.ID, retryCount, errorMsg)
		return w.db.MarkPostFailed(ctx, post.ID, errorMsg)
	}

	// Calculate next retry with exponential backoff: 2, 4, 8 minutes
	backoffMinutes := math.Pow(2, float64(retryCount))
	nextRetryAt := time.Now().Add(time.Duration(backoffMinutes) * time.Minute)

	// Schedule retry
	log.Printf("🔄 Scheduling retry %d/%d for post %s at %s", retryCount, MaxRetries, post.ID, nextRetryAt.Format(time.RFC3339))

	if err := w.db.ScheduleRetry(ctx, post.ID, nextRetryAt, errorMsg); err != nil {
		return err
	}

	// Re-enqueue in Redis for the next retry time
	return w.queue.Enqueue(ctx, post.ID, nextRetryAt)
}

// truncate truncates a string to maxLen and adds ellipsis
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
