package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// ErrSecretNotFound is returned when a requested secret is neither in the environment nor on disk.
var (
	ErrSecretNotFound = errors.New("secret not found")

	// ErrDecryptionFailed is returned when AES-GCM decryption fails (wrong key or corrupted ciphertext).
	ErrDecryptionFailed = errors.New("decryption failed")
)

// SecretManager handles secure secret storage and retrieval
type SecretManager struct {
	encryptionKey []byte
}

// NewSecretManager creates a new secret manager
func NewSecretManager() (*SecretManager, error) {
	key := os.Getenv("ENCRYPTION_KEY")
	if key == "" {
		return nil, errors.New("ENCRYPTION_KEY environment variable not set")
	}

	keyBytes, err := decodeEncryptionKey(key)
	if err != nil {
		return nil, err
	}

	return &SecretManager{
		encryptionKey: keyBytes,
	}, nil
}

// decodeEncryptionKey turns the ENCRYPTION_KEY env var into 32 raw AES-256 key
// bytes. Operators are documented (k8s/server.yaml) to generate the key via
// `openssl rand -hex 32`, which produces a 64-character hex string. Treating
// that string as raw bytes via []byte(key) uses the ASCII hex characters
// themselves as key material instead of the decoded random bytes, silently
// cutting the real entropy roughly in half (DP-737). Try hex, then base64,
// before falling back to raw bytes for callers that already pass a raw
// 32+ byte secret directly.
func decodeEncryptionKey(key string) ([]byte, error) {
	if decoded, err := hex.DecodeString(key); err == nil && len(decoded) == 32 {
		return decoded, nil
	}
	if decoded, err := base64.StdEncoding.DecodeString(key); err == nil && len(decoded) == 32 {
		return decoded, nil
	}

	keyBytes := []byte(key)
	if len(keyBytes) < 32 {
		return nil, errors.New("ENCRYPTION_KEY must decode to 32 bytes (hex or base64) or be at least 32 raw bytes")
	}
	return keyBytes[:32], nil
}

// GetSecret retrieves and decrypts a secret
func (sm *SecretManager) GetSecret(secretName string) (string, error) {
	// First check environment variable
	envVar := strings.ToUpper(secretName)
	if value := os.Getenv(envVar); value != "" {
		return value, nil
	}

	// Check for encrypted secret file
	// Validate the secret name before building the path so a future refactor
	// can never reach os.Stat/os.ReadFile with a traversal payload. The check
	// rejects ".." and "/" which are the only separators needed to escape the
	// /run/secrets directory (DP-815).
	if strings.Contains(secretName, "..") || strings.Contains(secretName, "/") {
		return "", ErrSecretNotFound
	}
	encryptedFile := fmt.Sprintf("/run/secrets/%s.enc", secretName)
	if _, err := os.Stat(encryptedFile); err == nil {
		encryptedData, err := os.ReadFile(encryptedFile)
		if err != nil {
			return "", err
		}

		return sm.decrypt(string(encryptedData))
	}

	return "", ErrSecretNotFound
}

// Encrypt encrypts a plaintext string
func (sm *SecretManager) Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(sm.encryptionKey)
	if err != nil {
		return "", err
	}

	// Create a GCM cipher
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Create a nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Encrypt the data
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts an encrypted string
func (sm *SecretManager) decrypt(encrypted string) (string, error) {
	// Decode from base64
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(sm.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", ErrDecryptionFailed
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	return string(plaintext), nil
}
