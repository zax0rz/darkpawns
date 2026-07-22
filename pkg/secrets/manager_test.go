package secrets

import (
	"encoding/base64"
	"encoding/hex"
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

// TestNewSecretManager_HexKeyUsesDecodedBytes is a regression test for DP-737:
// a hex-encoded ENCRYPTION_KEY (as produced by the documented `openssl rand
// -hex 32` setup in k8s/server.yaml) must be decoded to its 32 raw bytes, not
// used as the 64 ASCII hex characters. We confirm this by encrypting under
// the raw-byte key directly and decrypting via NewSecretManager fed the hex
// string — if the manager used the ASCII bytes instead, decryption would fail.
func TestNewSecretManager_HexKeyUsesDecodedBytes(t *testing.T) {
	rawKey := []byte("0123456789abcdef0123456789abcdef")[:32]
	hexKey := hex.EncodeToString(rawKey)
	if len(hexKey) != 64 {
		t.Fatalf("test setup: expected 64-char hex key, got %d chars", len(hexKey))
	}

	sm := &SecretManager{encryptionKey: rawKey}
	ciphertext, err := sm.Encrypt("super secret data")
	if err != nil {
		t.Fatalf("Encrypt with raw key failed: %v", err)
	}

	os.Setenv("ENCRYPTION_KEY", hexKey)
	defer os.Unsetenv("ENCRYPTION_KEY")

	smFromHex, err := NewSecretManager()
	if err != nil {
		t.Fatalf("NewSecretManager(hexKey) failed: %v", err)
	}

	plaintext, err := smFromHex.decrypt(ciphertext)
	if err != nil {
		t.Fatalf("decrypt with hex-derived key failed (key was not decoded): %v", err)
	}
	if plaintext != "super secret data" {
		t.Errorf("plaintext = %q, want %q", plaintext, "super secret data")
	}
}

// TestNewSecretManager_Base64KeyUsesDecodedBytes mirrors the hex case for a
// base64-encoded 32-byte key (DP-737).
func TestNewSecretManager_Base64KeyUsesDecodedBytes(t *testing.T) {
	rawKey := []byte("0123456789abcdef0123456789abcdef")[:32]
	b64Key := base64.StdEncoding.EncodeToString(rawKey)

	sm := &SecretManager{encryptionKey: rawKey}
	ciphertext, err := sm.Encrypt("super secret data")
	if err != nil {
		t.Fatalf("Encrypt with raw key failed: %v", err)
	}

	os.Setenv("ENCRYPTION_KEY", b64Key)
	defer os.Unsetenv("ENCRYPTION_KEY")

	smFromB64, err := NewSecretManager()
	if err != nil {
		t.Fatalf("NewSecretManager(b64Key) failed: %v", err)
	}

	plaintext, err := smFromB64.decrypt(ciphertext)
	if err != nil {
		t.Fatalf("decrypt with base64-derived key failed (key was not decoded): %v", err)
	}
	if plaintext != "super secret data" {
		t.Errorf("plaintext = %q, want %q", plaintext, "super secret data")
	}
}

// TestNewSecretManager_RawKeyFallback confirms a raw (non-hex, non-base64)
// 32+ byte string is still accepted as before, for backward compatibility.
func TestNewSecretManager_RawKeyFallback(t *testing.T) {
	os.Setenv("ENCRYPTION_KEY", "test-secret-that-is-at-least-32-characters-long")
	defer os.Unsetenv("ENCRYPTION_KEY")

	sm, err := NewSecretManager()
	if err != nil {
		t.Fatalf("NewSecretManager failed: %v", err)
	}
	if len(sm.encryptionKey) != 32 {
		t.Errorf("expected 32-byte key, got %d bytes", len(sm.encryptionKey))
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

// TestGetSecret_DoesNotLeakEncryptionKey is a regression test for DP-???:
// calling GetSecret("encryption_key") must not return the ENCRYPTION_KEY
// env var value, which is the AES-256 master key.
func TestGetSecret_DoesNotLeakEncryptionKey(t *testing.T) {
	os.Setenv("ENCRYPTION_KEY", "test-secret-that-is-at-least-32-characters-long")
	defer os.Unsetenv("ENCRYPTION_KEY")

	sm, err := NewSecretManager()
	if err != nil {
		t.Fatalf("NewSecretManager failed: %v", err)
	}

	for _, name := range []string{"encryption_key", "ENCRYPTION_KEY", "Encryption_Key"} {
		_, err := sm.GetSecret(name)
		if !errors.Is(err, ErrSecretNotFound) {
			t.Errorf("GetSecret(%q) = %v, want ErrSecretNotFound", name, err)
		}
	}
}
