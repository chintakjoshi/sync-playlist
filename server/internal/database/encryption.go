package database

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"
)

const encryptedTokenPrefix = "enc:v1:"

func ValidateEncryptionConfig() error {
	_, err := getTokenEncryptionKey()
	return err
}

func encryptToken(plaintext string) (string, error) {
	if plaintext == "" || isEncryptedToken(plaintext) {
		return plaintext, nil
	}

	key, err := getTokenEncryptionKey()
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return encryptedTokenPrefix + base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

func decryptToken(value string) (string, error) {
	if value == "" || !isEncryptedToken(value) {
		return value, nil
	}

	key, err := getTokenEncryptionKey()
	if err != nil {
		return "", err
	}

	rawCiphertext, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(value, encryptedTokenPrefix))
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(rawCiphertext) < nonceSize {
		return "", fmt.Errorf("encrypted token payload too short")
	}

	nonce, ciphertext := rawCiphertext[:nonceSize], rawCiphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func getTokenEncryptionKey() ([]byte, error) {
	keyValue := strings.TrimSpace(os.Getenv("TOKEN_ENCRYPTION_KEY"))
	if keyValue == "" {
		return nil, fmt.Errorf("TOKEN_ENCRYPTION_KEY is required")
	}

	if decoded, err := base64.StdEncoding.DecodeString(keyValue); err == nil && len(decoded) == 32 {
		return decoded, nil
	}
	if decoded, err := base64.RawStdEncoding.DecodeString(keyValue); err == nil && len(decoded) == 32 {
		return decoded, nil
	}
	if len(keyValue) == 32 {
		return []byte(keyValue), nil
	}

	return nil, fmt.Errorf("TOKEN_ENCRYPTION_KEY must be 32 raw bytes or base64 for 32 bytes")
}

func isEncryptedToken(value string) bool {
	return strings.HasPrefix(value, encryptedTokenPrefix)
}
