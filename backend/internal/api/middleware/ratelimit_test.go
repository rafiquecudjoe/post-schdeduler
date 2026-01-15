package middleware

import (
	"testing"
	"time"
)

func TestRateLimiterConfig(t *testing.T) {
	// Test default configurations exist
	if AuthRateLimit.Limit != 5 {
		t.Errorf("Expected AuthRateLimit.Limit to be 5, got %d", AuthRateLimit.Limit)
	}
	if AuthRateLimit.Window != time.Minute {
		t.Errorf("Expected AuthRateLimit.Window to be 1 minute, got %v", AuthRateLimit.Window)
	}

	if RegisterRateLimit.Limit != 3 {
		t.Errorf("Expected RegisterRateLimit.Limit to be 3, got %d", RegisterRateLimit.Limit)
	}

	if APIRateLimit.Limit != 100 {
		t.Errorf("Expected APIRateLimit.Limit to be 100, got %d", APIRateLimit.Limit)
	}

	if CreatePostRateLimit.Limit != 30 {
		t.Errorf("Expected CreatePostRateLimit.Limit to be 30, got %d", CreatePostRateLimit.Limit)
	}
}
