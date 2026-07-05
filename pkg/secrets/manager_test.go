package secrets

import (
	"errors"
	"os"
	"testing"
)

func TestDecrypt_WrongKeyReturnsSentinel(t *testing.T) {
	key1 := "test-secret-that-is-at-least-32-characters-long"
	key2 := "different-secret-that-is-32-characters"

	os.Setenv("ENCRYPTION_KEY", key1)
	defer os.Unsetenv("ENCRYPTION_KEY")

	sm1, err := NewSecretManager()
	if err != nil {
		t.Fatalf("NewSecretManager(key1) failed: %v", err)
	}

	ciphertext, err := sm1.Encrypt("super secret data")
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	os.Setenv("ENCRYPTION_KEY", key2)
	sm2, err := NewSecretManager()
	if err != nil {
		t.Fatalf("NewSecretManager(key2) failed: %v", err)
	}

	_, err = sm2.decrypt(ciphertext)
	if err == nil {
		t.Fatal("decrypt with wrong key expected error, got nil")
	}
	if !errors.Is(err, ErrDecryptionFailed) {
		t.Errorf("expected errors.Is(err, ErrDecryptionFailed), got: %v", err)
	}
}
