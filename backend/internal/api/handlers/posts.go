package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/scheduler/backend/internal/cache"
	"github.com/scheduler/backend/internal/db"
	"github.com/scheduler/backend/internal/models"
	"github.com/scheduler/backend/internal/notifier"
	"github.com/scheduler/backend/internal/scheduler"
)

// PostHandler handles post endpoints
type PostHandler struct {
	db       *db.DB
	queue    *scheduler.Queue
	cache    *cache.Cache
	notifier *notifier.Notifier
}

// NewPostHandler creates a new post handler
func NewPostHandler(database *db.DB, queue *scheduler.Queue, postCache *cache.Cache, n *notifier.Notifier) *PostHandler {
	return &PostHandler{
		db:       database,
		queue:    queue,
		cache:    postCache,
		notifier: n,
	}
}

// Create handles creating a new scheduled post
func (h *PostHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	var req models.CreatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Trim and validate content
	req.Content = trimString(req.Content)
	if req.Content == "" {
		respondError(w, http.StatusBadRequest, "Content is required")
		return
	}
	if len(req.Content) < 3 {
		respondError(w, http.StatusBadRequest, "Content must be at least 3 characters")
		return
	}
	if len(req.Content) > 5000 {
		respondError(w, http.StatusBadRequest, "Content must not exceed 5000 characters")
		return
	}

	// Trim and validate title if provided
	if req.Title != nil {
		trimmed := trimString(*req.Title)
		if trimmed == "" {
			req.Title = nil // Treat empty title as nil
		} else {
			if len(trimmed) > 200 {
				respondError(w, http.StatusBadRequest, "Title must not exceed 200 characters")
				return
			}
			req.Title = &trimmed
		}
	}

	// Validate channel
	if !models.IsValidChannel(req.Channel) {
		respondError(w, http.StatusBadRequest, "Invalid channel. Must be one of: twitter, linkedin, facebook")
		return
	}

	// Parse and validate scheduled_at
	scheduledAt, err := time.Parse(time.RFC3339, req.ScheduledAt)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid scheduled_at format. Use RFC3339 (e.g., 2024-01-15T14:00:00Z)")
		return
	}

	if scheduledAt.Before(time.Now()) {
		respondError(w, http.StatusBadRequest, "scheduled_at must be in the future")
		return
	}

	// Validate scheduled_at is not too far in the future (max 1 year)
	maxFutureDate := time.Now().AddDate(1, 0, 0)
	if scheduledAt.After(maxFutureDate) {
		respondError(w, http.StatusBadRequest, "scheduled_at cannot be more than 1 year in the future")
		return
	}

	// Create post in database
	post, err := h.db.CreatePost(r.Context(), user.ID, req.Title, req.Content, models.Channel(req.Channel), scheduledAt)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create post")
		return
	}

	// Add to scheduling queue (async, don't block response)
	go func() {
		if err := h.queue.Enqueue(context.Background(), post.ID, scheduledAt); err != nil {
			log.Printf("⚠️ Failed to enqueue post %s: %v", post.ID, err)
		}
	}()

	// Invalidate cache (async)
	go func() {
		if h.cache != nil {
			_ = h.cache.InvalidateUserPosts(context.Background(), user.ID)
		}
	}()

	// Notify SSE clients of the new post (async for Redis pub, sync for local)
	log.Printf("📢 [POST CREATE] Sending notification for user %s, post %s", user.ID, post.ID)
	h.notifier.Notify(user.ID, notifier.UpdateTypeCreate)
	log.Printf("✅ [POST CREATE] Notification sent for user %s", user.ID)

	respondJSON(w, http.StatusCreated, post)
}

// GetUpcoming returns all scheduled posts for the user
func (h *PostHandler) GetUpcoming(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	// Try cache first
	if h.cache != nil {
		if posts, found := h.cache.GetUpcomingPosts(r.Context(), user.ID); found {
			respondJSON(w, http.StatusOK, posts)
			return
		}
	}

	posts, err := h.db.GetUpcomingPosts(r.Context(), user.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch posts")
		return
	}

	if posts == nil {
		posts = []*models.Post{}
	}

	// Cache the result
	if h.cache != nil {
		_ = h.cache.SetUpcomingPosts(r.Context(), user.ID, posts)
	}

	respondJSON(w, http.StatusOK, posts)
}

// GetHistory returns all published posts for the user
func (h *PostHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	// Try cache first
	if h.cache != nil {
		if posts, found := h.cache.GetHistoryPosts(r.Context(), user.ID); found {
			respondJSON(w, http.StatusOK, posts)
			return
		}
	}

	posts, err := h.db.GetPublishedPosts(r.Context(), user.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch posts")
		return
	}

	if posts == nil {
		posts = []*models.Post{}
	}

	// Cache the result
	if h.cache != nil {
		_ = h.cache.SetHistoryPosts(r.Context(), user.ID, posts)
	}

	respondJSON(w, http.StatusOK, posts)
}

// GetByID returns a single post by ID
func (h *PostHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	postID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid post ID")
		return
	}

	post, err := h.db.GetPostByID(r.Context(), postID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch post")
		return
	}

	if post == nil {
		respondError(w, http.StatusNotFound, "Post not found")
		return
	}

	// Check ownership
	if post.UserID != user.ID {
		respondError(w, http.StatusForbidden, "Access denied")
		return
	}

	respondJSON(w, http.StatusOK, post)
}

// Update updates a scheduled post
func (h *PostHandler) Update(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	postID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid post ID")
		return
	}

	// Check if post exists and is owned by user
	existingPost, err := h.db.GetPostByID(r.Context(), postID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch post")
		return
	}
	if existingPost == nil {
		respondError(w, http.StatusNotFound, "Post not found")
		return
	}
	if existingPost.UserID != user.ID {
		respondError(w, http.StatusForbidden, "Access denied")
		return
	}
	if existingPost.Status != models.PostStatusScheduled {
		respondError(w, http.StatusBadRequest, "Cannot update a post that is not scheduled")
		return
	}

	var req models.UpdatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate channel if provided
	var channel *models.Channel
	if req.Channel != nil {
		if !models.IsValidChannel(*req.Channel) {
			respondError(w, http.StatusBadRequest, "Invalid channel. Must be one of: twitter, linkedin, facebook")
			return
		}
		ch := models.Channel(*req.Channel)
		channel = &ch
	}

	// Parse and validate scheduled_at if provided
	var scheduledAt *time.Time
	if req.ScheduledAt != nil {
		parsed, err := time.Parse(time.RFC3339, *req.ScheduledAt)
		if err != nil {
			respondError(w, http.StatusBadRequest, "Invalid scheduled_at format. Use RFC3339")
			return
		}
		if parsed.Before(time.Now()) {
			respondError(w, http.StatusBadRequest, "scheduled_at must be in the future")
			return
		}
		scheduledAt = &parsed
	}

	// Update post
	post, err := h.db.UpdatePost(r.Context(), postID, user.ID, req.Title, req.Content, channel, scheduledAt)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to update post")
		return
	}

	if post == nil {
		respondError(w, http.StatusNotFound, "Post not found or cannot be updated")
		return
	}

	// Update queue if scheduled_at changed (async)
	if scheduledAt != nil {
		go func() {
			if err := h.queue.Update(context.Background(), post.ID, *scheduledAt); err != nil {
				log.Printf("⚠️ Failed to update queue for post %s: %v", post.ID, err)
			}
		}()
	}

	// Invalidate cache (async)
	go func() {
		if h.cache != nil {
			_ = h.cache.InvalidateUserPosts(context.Background(), user.ID)
		}
	}()

	// Notify SSE clients of the update
	log.Printf("📢 [POST UPDATE] Sending notification for user %s, post %s", user.ID, post.ID)
	h.notifier.Notify(user.ID, notifier.UpdateTypeUpdate)
	log.Printf("✅ [POST UPDATE] Notification sent for user %s", user.ID)

	respondJSON(w, http.StatusOK, post)
}

// Delete deletes a scheduled post
func (h *PostHandler) Delete(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	postID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid post ID")
		return
	}

	// Check if post exists and is owned by user
	existingPost, err := h.db.GetPostByID(r.Context(), postID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch post")
		return
	}
	if existingPost == nil {
		respondError(w, http.StatusNotFound, "Post not found")
		return
	}
	if existingPost.UserID != user.ID {
		respondError(w, http.StatusForbidden, "Access denied")
		return
	}
	if existingPost.Status != models.PostStatusScheduled {
		respondError(w, http.StatusBadRequest, "Cannot delete a post that is not scheduled")
		return
	}

	// Delete from database
	deleted, err := h.db.DeletePost(r.Context(), postID, user.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to delete post")
		return
	}

	if !deleted {
		respondError(w, http.StatusNotFound, "Post not found or cannot be deleted")
		return
	}

	// Remove from queue (async)
	go func() {
		_ = h.queue.Remove(context.Background(), postID)
	}()

	// Invalidate cache (async)
	go func() {
		if h.cache != nil {
			_ = h.cache.InvalidateUserPosts(context.Background(), user.ID)
		}
	}()

	// Notify SSE clients of the deletion
	log.Printf("📢 [POST DELETE] Sending notification for user %s, post %s", user.ID, postID)
	h.notifier.Notify(user.ID, notifier.UpdateTypeDelete)
	log.Printf("✅ [POST DELETE] Notification sent for user %s", user.ID)

	w.WriteHeader(http.StatusNoContent)
}
