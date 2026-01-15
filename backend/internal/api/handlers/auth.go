package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/mail"
	"time"
	"unicode"

	"github.com/scheduler/backend/internal/auth"
	"github.com/scheduler/backend/internal/db"
	"github.com/scheduler/backend/internal/models"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	db            *db.DB
	jwtService    *auth.JWTService
	blacklist     *auth.Blacklist
	secureCookies bool
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(database *db.DB, jwtService *auth.JWTService, blacklist *auth.Blacklist, secureCookies bool) *AuthHandler {
	return &AuthHandler{
		db:            database,
		jwtService:    jwtService,
		blacklist:     blacklist,
		secureCookies: secureCookies,
	}
}

// Register handles user registration
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate input
	if req.Email == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "Email and password are required")
		return
	}

	if !isValidEmail(req.Email) {
		respondError(w, http.StatusBadRequest, "Invalid email format")
		return
	}

	if err := validatePassword(req.Password); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Check if user already exists
	existing, err := h.db.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if existing != nil {
		respondError(w, http.StatusConflict, "Email already registered")
		return
	}

	// Hash password
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to hash password")
		return
	}

	// Create user
	user, err := h.db.CreateUser(r.Context(), req.Email, passwordHash)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	// Generate tokens
	tokens, err := h.jwtService.GenerateTokenPair(user.ID, user.Email)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate tokens")
		return
	}

	// Set cookies
	h.setAuthCookies(w, tokens)

	respondJSON(w, http.StatusCreated, models.AuthResponse{
		User: user.ToResponse(),
	})
}

// Login handles user login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate input
	if req.Email == "" || req.Password == "" {
		respondError(w, http.StatusBadRequest, "Email and password are required")
		return
	}

	// Get user
	user, err := h.db.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Database error")
		return
	}
	if user == nil {
		respondError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	// Check password
	if !auth.CheckPassword(req.Password, user.PasswordHash) {
		respondError(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	// Generate tokens
	tokens, err := h.jwtService.GenerateTokenPair(user.ID, user.Email)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate tokens")
		return
	}

	// Set cookies
	h.setAuthCookies(w, tokens)

	respondJSON(w, http.StatusOK, models.AuthResponse{
		User: user.ToResponse(),
	})
}

// Logout handles user logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Get refresh token from cookie and blacklist it
	if cookie, err := r.Cookie("refresh_token"); err == nil {
		claims, err := h.jwtService.ValidateToken(cookie.Value)
		if err == nil && claims.ID != "" {
			// Blacklist the refresh token
			ttl := time.Until(claims.ExpiresAt.Time)
			if ttl > 0 {
				_ = h.blacklist.Add(r.Context(), claims.ID, ttl)
			}
		}
	}

	// Clear cookies
	h.clearAuthCookies(w)

	respondJSON(w, http.StatusOK, map[string]string{"message": "Logged out successfully"})
}

// Refresh handles token refresh
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		respondError(w, http.StatusUnauthorized, "No refresh token")
		return
	}

	// Validate refresh token
	claims, err := h.jwtService.ValidateToken(cookie.Value)
	if err != nil {
		h.clearAuthCookies(w)
		respondError(w, http.StatusUnauthorized, "Invalid refresh token")
		return
	}

	// Check if blacklisted
	isBlacklisted, err := h.blacklist.IsBlacklisted(r.Context(), claims.ID)
	if err != nil || isBlacklisted {
		h.clearAuthCookies(w)
		respondError(w, http.StatusUnauthorized, "Token revoked")
		return
	}

	// Blacklist the old refresh token
	ttl := time.Until(claims.ExpiresAt.Time)
	if ttl > 0 {
		_ = h.blacklist.Add(r.Context(), claims.ID, ttl)
	}

	// Get user to ensure they still exist
	user, err := h.db.GetUserByID(r.Context(), claims.UserID)
	if err != nil || user == nil {
		h.clearAuthCookies(w)
		respondError(w, http.StatusUnauthorized, "User not found")
		return
	}

	// Generate new tokens
	tokens, err := h.jwtService.GenerateTokenPair(user.ID, user.Email)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to generate tokens")
		return
	}

	// Set new cookies
	h.setAuthCookies(w, tokens)

	respondJSON(w, http.StatusOK, models.AuthResponse{
		User: user.ToResponse(),
	})
}

// Me returns the current user
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	user := GetUserFromContext(r.Context())
	if user == nil {
		respondError(w, http.StatusUnauthorized, "Not authenticated")
		return
	}

	respondJSON(w, http.StatusOK, models.AuthResponse{
		User: user.ToResponse(),
	})
}

// setAuthCookies sets the authentication cookies
func (h *AuthHandler) setAuthCookies(w http.ResponseWriter, tokens *auth.TokenPair) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    tokens.AccessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(h.jwtService.GetAccessTokenTTL().Seconds()),
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    tokens.RefreshToken,
		Path:     "/api/auth/refresh",
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(h.jwtService.GetRefreshTokenTTL().Seconds()),
	})
}

// clearAuthCookies clears the authentication cookies
func (h *AuthHandler) clearAuthCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/api/auth/refresh",
		HttpOnly: true,
		MaxAge:   -1,
	})
}

// isValidEmail performs RFC 5322 compliant email validation
func isValidEmail(email string) bool {
	if len(email) > 254 { // RFC 5321 max length
		return false
	}
	
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}
	
	// Ensure the parsed email matches the input (no display name)
	if addr.Address != email {
		return false
	}
	
	return true
}

// validatePassword enforces strong password policy
func validatePassword(password string) error {
	if len(password) < 12 {
		return errors.New("password must be at least 12 characters")
	}
	
	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}
	
	if !hasUpper {
		return errors.New("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return errors.New("password must contain at least one lowercase letter")
	}
	if !hasDigit {
		return errors.New("password must contain at least one digit")
	}
	if !hasSpecial {
		return errors.New("password must contain at least one special character")
	}
	
	return nil
}
