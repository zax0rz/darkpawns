package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

var (
	ErrSecretNotFound   = errors.New("secret not found")
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
		// In development, generate a temporary key
		if os.Getenv("ENVIRONMENT") == "development" {
			key = generateTempKey()
		} else {
			return nil, errors.New("ENCRYPTION_KEY environment variable not set")
		}
	}
	
	// Ensure key is proper length (32 bytes for AES-256)
	keyBytes := []byte(key)
	if len(keyBytes) < 32 {
		// Pad with zeros (not secure for production!)
		padded := make([]byte, 32)
		copy(padded, keyBytes)
		keyBytes = padded
	} else if len(keyBytes) > 32 {
		keyBytes = keyBytes[:32]
	}
	
	return &SecretManager{
		encryptionKey: keyBytes,
	}, nil
}

// GetSecret retrieves and decrypts a secret
func (sm *SecretManager) GetSecret(secretName string) (string, error) {
	// First check environment variable
	envVar := strings.ToUpper(secretName)
	if value := os.Getenv(envVar); value != "" {
		return value, nil
	}
	
	// Check for encrypted secret file
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
		return "", err
	}
	
	return string(plaintext), nil
}

func generateTempKey() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "dev-temp-key-do-not-use-in-production-12345"
	}
	return base64.StdEncoding.EncodeToString(bytes)
}