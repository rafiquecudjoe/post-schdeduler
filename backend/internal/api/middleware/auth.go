package middleware

import (
	"net/http"

	"github.com/scheduler/backend/internal/api/handlers"
	"github.com/scheduler/backend/internal/auth"
	"github.com/scheduler/backend/internal/db"
	"github.com/scheduler/backend/internal/models"
)

// Auth creates an authentication middleware
func Auth(jwtService *auth.JWTService, database *db.DB) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("access_token")
			if err != nil {
				http.Error(w, `{"error":"Unauthorized","message":"No access token"}`, http.StatusUnauthorized)
				return
			}

			claims, err := jwtService.ValidateToken(cookie.Value)
			if err != nil {
				http.Error(w, `{"error":"Unauthorized","message":"Invalid or expired token"}`, http.StatusUnauthorized)
				return
			}

			// Get user from database
			user, err := database.GetUserByID(r.Context(), claims.UserID)
			if err != nil || user == nil {
				http.Error(w, `{"error":"Unauthorized","message":"User not found"}`, http.StatusUnauthorized)
				return
			}

			// Add user to context
			ctx := handlers.SetUserInContext(r.Context(), &models.User{
				ID:        user.ID,
				Email:     user.Email,
				CreatedAt: user.CreatedAt,
				UpdatedAt: user.UpdatedAt,
			})

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
