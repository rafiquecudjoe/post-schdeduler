package models

import (
	"testing"
)

func TestIsValidChannel(t *testing.T) {
	tests := []struct {
		channel string
		valid   bool
	}{
		{"twitter", true},
		{"linkedin", true},
		{"facebook", true},
		{"instagram", false},
		{"tiktok", false},
		{"", false},
		{"Twitter", false}, // case sensitive
	}
	
	for _, tt := range tests {
		t.Run(tt.channel, func(t *testing.T) {
			if got := IsValidChannel(tt.channel); got != tt.valid {
				t.Errorf("IsValidChannel(%q) = %v, want %v", tt.channel, got, tt.valid)
			}
		})
	}
}

func TestValidChannels(t *testing.T) {
	channels := ValidChannels()
	
	if len(channels) != 3 {
		t.Errorf("Expected 3 channels, got %d", len(channels))
	}
	
	expected := map[Channel]bool{
		ChannelTwitter:  true,
		ChannelLinkedIn: true,
		ChannelFacebook: true,
	}
	
	for _, ch := range channels {
		if !expected[ch] {
			t.Errorf("Unexpected channel: %v", ch)
		}
	}
}

func TestUserToResponse(t *testing.T) {
	user := &User{
		Email:        "test@example.com",
		PasswordHash: "secret-hash",
	}
	
	response := user.ToResponse()
	
	if response.Email != user.Email {
		t.Errorf("Email mismatch: got %v, want %v", response.Email, user.Email)
	}
	
	// PasswordHash should not be exposed
	// This is verified by the struct not having PasswordHash field
}
