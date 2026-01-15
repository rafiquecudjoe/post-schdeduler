package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system
type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // Never expose in JSON
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// PostStatus represents the status of a post
type PostStatus string

const (
	PostStatusScheduled PostStatus = "scheduled"
	PostStatusPublished PostStatus = "published"
	PostStatusFailed    PostStatus = "failed"
)

// Channel represents a social media channel
type Channel string

const (
	ChannelTwitter  Channel = "twitter"
	ChannelLinkedIn Channel = "linkedin"
	ChannelFacebook Channel = "facebook"
)

// ValidChannels returns all valid channel values
func ValidChannels() []Channel {
	return []Channel{ChannelTwitter, ChannelLinkedIn, ChannelFacebook}
}

// IsValidChannel checks if a channel value is valid
func IsValidChannel(c string) bool {
	for _, valid := range ValidChannels() {
		if string(valid) == c {
			return true
		}
	}
	return false
}

// Post represents a scheduled or published post
type Post struct {
	ID          uuid.UUID  `json:"id"`
	UserID      uuid.UUID  `json:"user_id"`
	Title       *string    `json:"title,omitempty"`
	Content     string     `json:"content"`
	Channel     Channel    `json:"channel"`
	Status      PostStatus `json:"status"`
	ScheduledAt time.Time  `json:"scheduled_at"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	RetryCount  int        `json:"retry_count,omitempty"`
	LastError   *string    `json:"last_error,omitempty"`
	NextRetryAt *time.Time `json:"next_retry_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// CreatePostRequest represents the request to create a post
type CreatePostRequest struct {
	Title       *string `json:"title"`
	Content     string  `json:"content"`
	Channel     string  `json:"channel"`
	ScheduledAt string  `json:"scheduled_at"`
}

// UpdatePostRequest represents the request to update a post
type UpdatePostRequest struct {
	Title       *string `json:"title"`
	Content     *string `json:"content"`
	Channel     *string `json:"channel"`
	ScheduledAt *string `json:"scheduled_at"`
}

// RegisterRequest represents a user registration request
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest represents a user login request
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse represents the response after successful auth
type AuthResponse struct {
	User *UserResponse `json:"user"`
}

// UserResponse represents the user data returned to clients
type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

// ToResponse converts a User to UserResponse
func (u *User) ToResponse() *UserResponse {
	return &UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		CreatedAt: u.CreatedAt,
	}
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}
