package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestJWTService_GenerateTokenPair(t *testing.T) {
	service := NewJWTService("test-secret-key", 15*time.Minute, 7*24*time.Hour)
	
	userID := uuid.New()
	email := "test@example.com"
	
	tokens, err := service.GenerateTokenPair(userID, email)
	if err != nil {
		t.Fatalf("GenerateTokenPair failed: %v", err)
	}
	
	if tokens.AccessToken == "" {
		t.Error("AccessToken should not be empty")
	}
	
	if tokens.RefreshToken == "" {
		t.Error("RefreshToken should not be empty")
	}
	
	if tokens.AccessJTI == "" {
		t.Error("AccessJTI should not be empty")
	}
	
	if tokens.RefreshJTI == "" {
		t.Error("RefreshJTI should not be empty")
	}
}

func TestJWTService_ValidateToken(t *testing.T) {
	service := NewJWTService("test-secret-key", 15*time.Minute, 7*24*time.Hour)
	
	userID := uuid.New()
	email := "test@example.com"
	
	tokens, _ := service.GenerateTokenPair(userID, email)
	
	// Validate access token
	claims, err := service.ValidateToken(tokens.AccessToken)
	if err != nil {
		t.Fatalf("ValidateToken failed: %v", err)
	}
	
	if claims.UserID != userID {
		t.Errorf("UserID mismatch: got %v, want %v", claims.UserID, userID)
	}
	
	if claims.Email != email {
		t.Errorf("Email mismatch: got %v, want %v", claims.Email, email)
	}
}

func TestJWTService_ValidateToken_InvalidToken(t *testing.T) {
	service := NewJWTService("test-secret-key", 15*time.Minute, 7*24*time.Hour)
	
	_, err := service.ValidateToken("invalid-token")
	if err == nil {
		t.Error("ValidateToken should fail for invalid token")
	}
}

func TestJWTService_ValidateToken_WrongSecret(t *testing.T) {
	service1 := NewJWTService("secret-1", 15*time.Minute, 7*24*time.Hour)
	service2 := NewJWTService("secret-2", 15*time.Minute, 7*24*time.Hour)
	
	tokens, _ := service1.GenerateTokenPair(uuid.New(), "test@example.com")
	
	_, err := service2.ValidateToken(tokens.AccessToken)
	if err == nil {
		t.Error("ValidateToken should fail when using different secret")
	}
}

func TestJWTService_ExpiredToken(t *testing.T) {
	// Create service with very short TTL
	service := NewJWTService("test-secret-key", 1*time.Millisecond, 1*time.Millisecond)
	
	tokens, _ := service.GenerateTokenPair(uuid.New(), "test@example.com")
	
	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)
	
	_, err := service.ValidateToken(tokens.AccessToken)
	if err != ErrExpiredToken {
		t.Errorf("Expected ErrExpiredToken, got: %v", err)
	}
}
