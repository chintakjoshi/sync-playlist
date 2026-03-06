package auth

import (
	"os"
	"testing"
)

func TestOAuthStateRoundTrip(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")

	state, nonce, err := GenerateOAuthState(StatePurposeLink, "spotify", 42)
	if err != nil {
		t.Fatalf("GenerateOAuthState() error = %v", err)
	}

	claims, err := ValidateOAuthState(state, StatePurposeLink, "spotify", nonce)
	if err != nil {
		t.Fatalf("ValidateOAuthState() error = %v", err)
	}

	if claims.UserID != 42 {
		t.Fatalf("expected user ID 42, got %d", claims.UserID)
	}
}

func TestOAuthStateRejectsNonceMismatch(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")

	state, _, err := GenerateOAuthState(StatePurposeLogin, "", 0)
	if err != nil {
		t.Fatalf("GenerateOAuthState() error = %v", err)
	}

	if _, err := ValidateOAuthState(state, StatePurposeLogin, "", "wrong"); err == nil {
		t.Fatal("expected nonce mismatch error")
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
