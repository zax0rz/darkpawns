package releaseauth

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"os"
	"strings"
	"testing"
)

func TestPublisherSignatureRejectsTamperingAndWrongKey(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	public := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
	data := []byte("darkpawns-release-v1\nversion v0.1.0\ncommit " + strings.Repeat("a", 40) + "\nasset linux amd64 darkpawns_0.1.0_linux_amd64.tar.gz 10 sha256 " + strings.Repeat("b", 64) + "\n")
	hash := sha256.Sum256(data)
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hash[:])
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(data, signature, public); err != nil {
		t.Fatal(err)
	}
	changed := append([]byte(nil), data...)
	changed[0] = 'X'
	if _, err := Verify(changed, signature, public); err == nil {
		t.Fatal("changed manifest authenticated")
	}
	signature[0] ^= 1
	if _, err := Verify(data, signature, public); err == nil {
		t.Fatal("changed signature authenticated")
	}
	signature[0] ^= 1
	if _, err := Verify(data, signature, TrustedKey); err == nil {
		t.Fatal("foreign publisher authenticated")
	}
}

func TestSignedManifestIdentityAndDuplicates(t *testing.T) {
	base := "darkpawns-release-v1\nversion v0.1.0\ncommit " + strings.Repeat("a", 40) + "\n"
	row := "asset windows arm64 darkpawns_0.1.0_windows_arm64.zip 10 sha256 " + strings.Repeat("b", 64) + "\n"
	for _, data := range []string{base + row + row, strings.Replace(base+row, "v0.1.0", "v0.1.0-rc.1", 1), strings.Replace(base+row, " 10 sha256 ", " 99999999999 sha256 ", 1), strings.Replace(base+row, "darkpawns_0.1.0_windows_arm64.zip", "../escape", 1)} {
		if _, err := Parse([]byte(data)); err == nil {
			t.Fatal("invalid metadata accepted", data)
		}
	}
}

// The compiled-in publisher key is the fourth pinned copy (with the PEM
// file, the website stub and the docs one-liner); it must be exactly the
// pinned file's bytes.
func TestTrustedKeyMatchesPinnedFile(t *testing.T) {
	data, err := os.ReadFile("trusted-key.pem")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(TrustedKey, data) {
		t.Fatal("compiled-in TrustedKey differs from trusted-key.pem on disk")
	}
	if _, err := PublicKey(TrustedKeyBytes()); err != nil {
		t.Fatalf("TrustedKeyBytes: %v", err)
	}
}

func TestVerifyBytes(t *testing.T) {
	private, err := rsa.GenerateKey(rand.Reader, 3072)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := x509.MarshalPKIXPublicKey(&private.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	key := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: encoded})
	digest := sha256.Sum256([]byte("bundle manifest"))
	signature, err := rsa.SignPKCS1v15(rand.Reader, private, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyBytes([]byte("bundle manifest"), signature, key); err != nil {
		t.Fatalf("valid signature rejected: %v", err)
	}
	if err := VerifyBytes([]byte("tampered manifest"), signature, key); err == nil {
		t.Fatal("tampered data accepted")
	}
	if err := VerifyBytes([]byte("bundle manifest"), []byte("junk"), key); err == nil {
		t.Fatal("junk signature accepted")
	}
}
