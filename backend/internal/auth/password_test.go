package auth

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	password := "testpassword123"
	
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	
	if hash == "" {
		t.Error("HashPassword returned empty string")
	}
	
	if hash == password {
		t.Error("Hash should not equal plaintext password")
	}
}

func TestCheckPassword(t *testing.T) {
	password := "testpassword123"
	
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	
	// Correct password should match
	if !CheckPassword(password, hash) {
		t.Error("CheckPassword should return true for correct password")
	}
	
	// Wrong password should not match
	if CheckPassword("wrongpassword", hash) {
		t.Error("CheckPassword should return false for wrong password")
	}
}

func TestHashPassword_DifferentHashes(t *testing.T) {
	password := "testpassword123"
	
	hash1, _ := HashPassword(password)
	hash2, _ := HashPassword(password)
	
	// Each hash should be different due to salt
	if hash1 == hash2 {
		t.Error("Two hashes of same password should be different")
	}
	
	// Both should still verify
	if !CheckPassword(password, hash1) || !CheckPassword(password, hash2) {
		t.Error("Both hashes should verify the password")
	}
}
