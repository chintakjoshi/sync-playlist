package database

import "testing"

func TestEncryptDecryptTokenRoundTrip(t *testing.T) {
	t.Setenv("TOKEN_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")

	encrypted, err := encryptToken("test-token")
	if err != nil {
		t.Fatalf("encryptToken() error = %v", err)
	}
	if encrypted == "test-token" {
		t.Fatal("expected encrypted token to differ from plaintext")
	}

	decrypted, err := decryptToken(encrypted)
	if err != nil {
		t.Fatalf("decryptToken() error = %v", err)
	}
	if decrypted != "test-token" {
		t.Fatalf("expected decrypted token to match plaintext, got %q", decrypted)
	}
}

func TestDecryptTokenAllowsLegacyPlaintext(t *testing.T) {
	t.Setenv("TOKEN_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")

	decrypted, err := decryptToken("legacy-plaintext-token")
	if err != nil {
		t.Fatalf("decryptToken() error = %v", err)
	}
	if decrypted != "legacy-plaintext-token" {
		t.Fatalf("expected plaintext token passthrough, got %q", decrypted)
	}
}
