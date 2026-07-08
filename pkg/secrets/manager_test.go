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

// TestGetSecret_RejectsPathTraversal is a regression test for DP-815: a secret
// name containing ".." or "/" must be rejected before any path is constructed
// or stat'd, so it can never escape the /run/secrets directory.
func TestGetSecret_RejectsPathTraversal(t *testing.T) {
	os.Setenv("ENCRYPTION_KEY", "test-secret-that-is-at-least-32-characters-long")
	defer os.Unsetenv("ENCRYPTION_KEY")

	sm, err := NewSecretManager()
	if err != nil {
		t.Fatalf("NewSecretManager failed: %v", err)
	}

	for _, name := range []string{"..", "../etc/passwd", "foo/../bar", "sub/dir", "/abs/path"} {
		_, err := sm.GetSecret(name)
		if !errors.Is(err, ErrSecretNotFound) {
			t.Errorf("GetSecret(%q) = %v, want ErrSecretNotFound", name, err)
		}
	}
}
