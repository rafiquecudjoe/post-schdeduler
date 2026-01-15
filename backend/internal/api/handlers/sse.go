package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/scheduler/backend/internal/db"
	"github.com/scheduler/backend/internal/models"
)

// SSEHandler handles Server-Sent Events for real-time updates
type SSEHandler struct {
	db *db.DB
}

// NewSSEHandler creates a new SSE handler
func NewSSEHandler(database *db.DB) *SSEHandler {
	return &SSEHandler{
		db: database,
	}
}

// StreamPosts sends real-time post updates to the client
func (h *SSEHandler) StreamPosts(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		log.Printf("SSE: ERROR - No user in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Set SSE headers - must be set before writing
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering
	// CORS headers are already set by middleware - don't override

	// Chi middleware wraps the ResponseWriter, breaking http.Flusher interface
	// We need to unwrap ALL layers to access the original ResponseWriter
	origWriter := w
	unwrapCount := 0
	for {
		if unwrapper, ok := origWriter.(interface{ Unwrap() http.ResponseWriter }); ok {
			origWriter = unwrapper.Unwrap()
			unwrapCount++
		} else {
			break
		}
	}

	// Get flusher - required for SSE
	flusher, ok := origWriter.(http.Flusher)
	if !ok {
		log.Printf("SSE: ERROR - Streaming not supported even after unwrapping %d layers", unwrapCount)
		log.Printf("SSE: ResponseWriter type: %T", origWriter)
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Send initial connection event
	if _, err := fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"connected\"}\n\n"); err != nil {
		log.Printf("SSE: ERROR - Failed to send connected event: %v", err)
		return
	}
	flusher.Flush()

	// Create ticker for periodic updates (every 5 seconds for faster updates)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Create keepalive ticker to prevent timeout (every 10 seconds, more frequent)
	keepaliveTicker := time.NewTicker(10 * time.Second)
	defer keepaliveTicker.Stop()

	// Keep track of last sent data to avoid duplicate updates
	var lastUpcomingHash string
	var lastHistoryHash string

	// Send initial data immediately
	upcoming, _ := h.db.GetUpcomingPosts(r.Context(), user.ID)
	history, _ := h.db.GetPublishedPosts(r.Context(), user.ID)
	if upcoming == nil {
		upcoming = []*models.Post{}
	}
	if history == nil {
		history = []*models.Post{}
	}

	lastUpcomingHash = hashPosts(upcoming)
	lastHistoryHash = hashPosts(history)

	data := map[string]interface{}{
		"upcoming": upcoming,
		"history":  history,
	}
	jsonData, _ := json.Marshal(data)
	fmt.Fprintf(w, "event: update\ndata: %s\n\n", jsonData)
	flusher.Flush()

	// Send updates until client disconnects
	for {
		select {
		case <-r.Context().Done():
			return
		case <-keepaliveTicker.C:
			// Send keepalive comment to prevent timeout
			if _, err := fmt.Fprintf(w, ": keepalive\n\n"); err != nil {
				log.Printf("SSE: ERROR - Failed to send keepalive, client may have disconnected: %v", err)
				return
			}
			flusher.Flush()
		case <-ticker.C:
			// Fetch current data
			upcoming, err := h.db.GetUpcomingPosts(r.Context(), user.ID)
			if err != nil {
				continue
			}
			if upcoming == nil {
				upcoming = []*models.Post{}
			}

			history, err := h.db.GetPublishedPosts(r.Context(), user.ID)
			if err != nil {
				continue
			}
			if history == nil {
				history = []*models.Post{}
			}

			// Create hashes to detect changes
			upcomingHash := hashPosts(upcoming)
			historyHash := hashPosts(history)

			// Send update if data changed
			if upcomingHash != lastUpcomingHash || historyHash != lastHistoryHash {
				lastUpcomingHash = upcomingHash
				lastHistoryHash = historyHash

				data := map[string]interface{}{
					"upcoming": upcoming,
					"history":  history,
				}

				jsonData, err := json.Marshal(data)
				if err != nil {
					log.Printf("SSE: ERROR - Failed to marshal data: %v", err)
					continue
				}

				if _, err := fmt.Fprintf(w, "event: update\ndata: %s\n\n", jsonData); err != nil {
					log.Printf("SSE: ERROR - Failed to write update, client disconnected: %v", err)
					return
				}
				flusher.Flush()
			}
		}
	}
}

// hashPosts creates a simple hash to detect changes
func hashPosts(posts []*models.Post) string {
	if posts == nil || len(posts) == 0 {
		return "empty"
	}

	// Create a simple hash based on post IDs and statuses
	result := ""
	for _, p := range posts {
		result += fmt.Sprintf("%s:%s:%d;", p.ID, p.Status, p.UpdatedAt.Unix())
	}
	return result
}
