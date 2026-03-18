package auth

import (
	"strings"
	"testing"
)

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("correcthorse")
	if err != nil {
		t.Fatalf("hash error: %v", err)
	}

	// Should be PHC format.
	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Errorf("expected argon2id prefix, got %s", hash[:20])
	}

	parts := strings.Split(hash, "$")
	if len(parts) != 6 {
		t.Errorf("expected 6 parts, got %d", len(parts))
	}
}

func TestHashPassword_Unique(t *testing.T) {
	h1, _ := HashPassword("same-password")
	h2, _ := HashPassword("same-password")

	// Different salts should produce different hashes.
	if h1 == h2 {
		t.Error("two hashes of the same password should differ (different salts)")
	}
}

func TestVerifyPassword_Correct(t *testing.T) {
	hash, _ := HashPassword("mypassword")

	ok, err := VerifyPassword("mypassword", hash)
	if err != nil {
		t.Fatalf("verify error: %v", err)
	}
	if !ok {
		t.Error("expected correct password to verify")
	}
}

func TestVerifyPassword_Wrong(t *testing.T) {
	hash, _ := HashPassword("mypassword")

	ok, err := VerifyPassword("wrongpassword", hash)
	if err != nil {
		t.Fatalf("verify error: %v", err)
	}
	if ok {
		t.Error("expected wrong password to fail verification")
	}
}

func TestVerifyPassword_BadFormat(t *testing.T) {
	_, err := VerifyPassword("pw", "not-a-valid-hash")
	if err == nil {
		t.Error("expected error for invalid hash format")
	}
}
