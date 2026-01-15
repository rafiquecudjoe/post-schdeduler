package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/scheduler/backend/internal/models"
)

type contextKey string

const userContextKey contextKey = "user"

// SetUserInContext stores the user in the request context
func SetUserInContext(ctx context.Context, user *models.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// GetUserFromContext retrieves the user from the request context
func GetUserFromContext(ctx context.Context) *models.User {
	user, ok := ctx.Value(userContextKey).(*models.User)
	if !ok {
		return nil
	}
	return user
}

// trimString trims whitespace from a string
func trimString(s string) string {
	return strings.TrimSpace(s)
}

// respondJSON writes a JSON response
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// respondError writes an error response
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, models.ErrorResponse{
		Error:   http.StatusText(status),
		Message: message,
	})
}
