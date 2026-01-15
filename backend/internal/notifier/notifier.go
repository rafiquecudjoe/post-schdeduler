package notifier

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	// Redis channel for post updates
	postUpdateChannel = "post_updates"
)

// PostUpdate represents a notification about a post change
type PostUpdate struct {
	UserID uuid.UUID  `json:"user_id"`
	Type   UpdateType `json:"type"`
}

// UpdateType represents the type of update
type UpdateType string

const (
	UpdateTypeCreate  UpdateType = "create"
	UpdateTypeUpdate  UpdateType = "update"
	UpdateTypeDelete  UpdateType = "delete"
	UpdateTypePublish UpdateType = "publish"
)

// Notifier broadcasts post updates to SSE clients
type Notifier struct {
	mu          sync.RWMutex
	subscribers map[uuid.UUID][]chan PostUpdate
	redis       *redis.Client
	pubsub      *redis.PubSub
}

// NewNotifier creates a new notifier
func NewNotifier(redisClient *redis.Client) *Notifier {
	n := &Notifier{
		subscribers: make(map[uuid.UUID][]chan PostUpdate),
		redis:       redisClient,
	}

	// Start listening to Redis pub/sub for updates from worker
	if redisClient != nil {
		n.pubsub = redisClient.Subscribe(context.Background(), postUpdateChannel)
		go n.listenRedis()
	}

	return n
}

// listenRedis listens for updates from Redis pub/sub (from worker process)
func (n *Notifier) listenRedis() {
	log.Println("🔊 [NOTIFIER] Started listening to Redis pub/sub for cross-process notifications")
	ch := n.pubsub.Channel()
	for msg := range ch {
		var update PostUpdate
		if err := json.Unmarshal([]byte(msg.Payload), &update); err != nil {
			log.Printf("❌ [NOTIFIER] Failed to unmarshal Redis update: %v", err)
			continue
		}

		log.Printf("📨 [NOTIFIER] Received Redis update for user %s (type: %s)", update.UserID, update.Type)
		// Broadcast to local subscribers
		subscriberCount := n.notifyLocal(update.UserID, update.Type)
		log.Printf("📬 [NOTIFIER] Forwarded to %d local subscribers", subscriberCount)
	}
	log.Println("🔇 [NOTIFIER] Stopped listening to Redis pub/sub")
}

// Subscribe creates a new channel for receiving updates for a specific user
func (n *Notifier) Subscribe(userID uuid.UUID) chan PostUpdate {
	n.mu.Lock()
	defer n.mu.Unlock()

	ch := make(chan PostUpdate, 10) // Buffered channel to prevent blocking
	n.subscribers[userID] = append(n.subscribers[userID], ch)
	return ch
}

// Unsubscribe removes a channel from receiving updates
func (n *Notifier) Unsubscribe(userID uuid.UUID, ch chan PostUpdate) {
	n.mu.Lock()
	defer n.mu.Unlock()

	subscribers := n.subscribers[userID]
	for i, sub := range subscribers {
		if sub == ch {
			// Remove channel from slice
			n.subscribers[userID] = append(subscribers[:i], subscribers[i+1:]...)
			close(ch)
			break
		}
	}

	// Clean up empty subscriber lists
	if len(n.subscribers[userID]) == 0 {
		delete(n.subscribers, userID)
	}
}

// Notify sends an update to all subscribers for a specific user
// This also publishes to Redis so worker instances can notify
func (n *Notifier) Notify(userID uuid.UUID, updateType UpdateType) {
	update := PostUpdate{
		UserID: userID,
		Type:   updateType,
	}

	// Notify local subscribers
	subscriberCount := n.notifyLocal(userID, updateType)
	log.Printf("📤 [NOTIFIER] Notified %d local subscribers for user %s (type: %s)", subscriberCount, userID, updateType)

	// Publish to Redis for cross-process communication
	if n.redis != nil {
		data, err := json.Marshal(update)
		if err == nil {
			result := n.redis.Publish(context.Background(), postUpdateChannel, data)
			receivers, _ := result.Result()
			log.Printf("📡 [NOTIFIER] Published to Redis, %d receivers (user: %s, type: %s)", receivers, userID, updateType)
		} else {
			log.Printf("❌ [NOTIFIER] Failed to marshal update: %v", err)
		}
	}
}

// notifyLocal sends updates to local subscribers only
func (n *Notifier) notifyLocal(userID uuid.UUID, updateType UpdateType) int {
	n.mu.RLock()
	defer n.mu.RUnlock()

	subscribers := n.subscribers[userID]
	if len(subscribers) == 0 {
		return 0
	}

	update := PostUpdate{
		UserID: userID,
		Type:   updateType,
	}

	sent := 0
	// Send to all subscribers (non-blocking)
	for _, ch := range subscribers {
		select {
		case ch <- update:
			// Successfully sent
			sent++
		default:
			// Channel is full, skip this subscriber
			log.Printf("⚠️ [NOTIFIER] Channel full for user %s, skipping subscriber", userID)
		}
	}
	return sent
}

// Close closes the notifier and cleans up resources
func (n *Notifier) Close() {
	if n.pubsub != nil {
		_ = n.pubsub.Close()
	}
}

// SubscriberCount returns the number of active subscribers for a user
func (n *Notifier) SubscriberCount(userID uuid.UUID) int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return len(n.subscribers[userID])
}

// TotalSubscribers returns the total number of active subscribers across all users
func (n *Notifier) TotalSubscribers() int {
	n.mu.RLock()
	defer n.mu.RUnlock()

	total := 0
	for _, subs := range n.subscribers {
		total += len(subs)
	}
	return total
}
