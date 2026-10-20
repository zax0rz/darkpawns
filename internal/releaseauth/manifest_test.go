package releaseauth

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
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
