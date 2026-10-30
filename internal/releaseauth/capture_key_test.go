package releaseauth

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

func TestDocsCaptureKeyIsSeparateFromPublisher(t *testing.T) {
	fixture := func(name string) []byte {
		t.Helper()
		data, err := os.ReadFile(filepath.Join("..", "..", "scripts", "fixtures", name))
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	capture, err := PublicKey(fixture("docs-capture-test-public.pem"))
	if err != nil {
		t.Fatal(err)
	}
	publisher, err := PublicKey(TrustedKey)
	if err != nil {
		t.Fatal(err)
	}
	if capture.Equal(publisher) {
		t.Fatal("documentation capture key must never be the production publisher key")
	}
	block, rest := pem.Decode(fixture("docs-capture-test-private.pem"))
	if block == nil || len(rest) != 0 || block.Type != "PRIVATE KEY" {
		t.Fatal("invalid documentation capture private-key fixture")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	private, ok := parsed.(*rsa.PrivateKey)
	if !ok || !capture.Equal(&private.PublicKey) {
		t.Fatal("documentation capture public and private fixtures don't match")
	}
}
