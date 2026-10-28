package portable

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// L1-006/007/021: the strict bundle gate. A bundle executes only when its
// publisher record is signed by the pinned key and every hash it lists —
// the executables and the data manifest — matches. These are the refusal
// halves of the L1 exploit tests: planted pointers, journals and manifests
// must all fail here.

type publisherKey struct {
	private *rsa.PrivateKey
	public  []byte
}

func generatePublisherKey(t *testing.T) publisherKey {
	t.Helper()
	private, err := rsa.GenerateKey(rand.Reader, 3072)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := x509.MarshalPKIXPublicKey(&private.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	return publisherKey{private: private, public: pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: encoded})}
}

func (k publisherKey) sign(t *testing.T, data []byte) []byte {
	t.Helper()
	digest := sha256.Sum256(data)
	signature, err := rsa.SignPKCS1v15(rand.Reader, k.private, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return signature
}

func hashPath(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

// writeSignedBundle lays down a minimal signed bundle: the data manifest
// with the keys ReadManifest requires and correct hashes, binaries at the
// root, and a signed publisher record covering both.
func writeSignedBundle(t *testing.T, dir string, key publisherKey) string {
	t.Helper()
	return writeSignedBundleWithServer(t, dir, key, "#!/bin/sh\nexit 0\n")
}

// writeSignedBundleWithServer is writeSignedBundle with an explicit
// darkpawns-server body, so lifecycle tests can run a stand-in for the bundled
// game binary and still pass the real publisher gate.
func writeSignedBundleWithServer(t *testing.T, dir string, key publisherKey, server string) string {
	t.Helper()
	assets := filepath.Join(dir, "release-assets")
	dataFiles := map[string]string{
		"lib/world/wld/index":      "index\n",
		"web/index.html":           "<html></html>\n",
		"admin-ui/index.html":      "<html></html>\n",
		"lib/world/wld/darkpawn.w": "#1\n",
	}
	for name, content := range dataFiles {
		path := filepath.Join(assets, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for name, body := range map[string]string{"darkpawns": "#!/bin/sh\nexit 0\n", "darkpawns-server": server} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string]string{}
	for name := range dataFiles {
		files[name] = hashPath(t, filepath.Join(assets, filepath.FromSlash(name)))
	}
	manifest := map[string]any{"format": 1, "update_format": 1, "data_format": 1, "version": "9.9.9", "commit": "0000000000000000000000000000000000000000", "files": files}
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	manifestData = append(manifestData, '\n')
	if err := os.WriteFile(filepath.Join(assets, "manifest.json"), manifestData, 0o644); err != nil {
		t.Fatal(err)
	}
	record := map[string]any{"format": 1, "version": "9.9.9", "files": map[string]string{
		"release-assets/manifest.json": string(hex.EncodeToString(sumBytes(manifestData))),
		"darkpawns":                    hashPath(t, filepath.Join(dir, "darkpawns")),
		"darkpawns-server":             hashPath(t, filepath.Join(dir, "darkpawns-server")),
	}}
	recordData, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	recordData = append(recordData, '\n')
	if err := os.WriteFile(filepath.Join(assets, "publisher.json"), recordData, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assets, "publisher.json.sig"), key.sign(t, recordData), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func sumBytes(data []byte) []byte {
	digest := sha256.Sum256(data)
	return digest[:]
}

func TestVerifyBundlePublisherAcceptsSignedBundle(t *testing.T) {
	key := generatePublisherKey(t)
	bundle := writeSignedBundle(t, t.TempDir(), key)
	if err := verifyBundlePublisherWithKey(bundle, key.public, nil); err != nil {
		t.Fatalf("signed bundle rejected: %v", err)
	}
}

func TestVerifyBundlePublisherRefusesUnsignedBundle(t *testing.T) {
	key := generatePublisherKey(t)
	bundle := writeSignedBundle(t, t.TempDir(), key)
	if err := os.Remove(filepath.Join(bundle, "release-assets", "publisher.json.sig")); err != nil {
		t.Fatal(err)
	}
	if err := verifyBundlePublisherWithKey(bundle, key.public, nil); !errors.Is(err, ErrUnsignedBundle) {
		t.Fatalf("unsigned bundle = %v, want ErrUnsignedBundle", err)
	}
}

func TestVerifyBundlePublisherRefusesForeignSignature(t *testing.T) {
	key := generatePublisherKey(t)
	bundle := writeSignedBundle(t, t.TempDir(), key)
	other := generatePublisherKey(t)
	if err := verifyBundlePublisherWithKey(bundle, other.public, nil); err == nil {
		t.Fatal("bundle signed by a foreign key accepted")
	}
}

func TestVerifyBundlePublisherRefusesTamperedExecutable(t *testing.T) {
	key := generatePublisherKey(t)
	bundle := writeSignedBundle(t, t.TempDir(), key)
	if err := os.WriteFile(filepath.Join(bundle, "darkpawns-server"), []byte("#!/bin/sh\nrm -rf /\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := verifyBundlePublisherWithKey(bundle, key.public, nil); err == nil {
		t.Fatal("tampered executable accepted")
	}
}

func TestVerifyBundlePublisherRefusesGenuineRecordWithSwappedFiles(t *testing.T) {
	key := generatePublisherKey(t)
	bundle := writeSignedBundle(t, t.TempDir(), key)
	// A genuine signed record beside swapped data: the signature verifies,
	// the data-manifest hash must not.
	if err := os.WriteFile(filepath.Join(bundle, "release-assets", "web", "index.html"), []byte("<html>swapped</html>\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := verifyBundlePublisherWithKey(bundle, key.public, nil); err == nil {
		t.Fatal("genuine record accepted with swapped data files")
	}
}

// Defence in depth: the executables an exec path runs must be covered by the
// publisher record, even when the bundle as a whole verifies.
func TestVerifyBundlePublisherRequiresListedExecutables(t *testing.T) {
	key := generatePublisherKey(t)
	bundle := writeSignedBundle(t, t.TempDir(), key)
	if err := verifyBundlePublisherWithKey(bundle, key.public, []string{"darkpawns", "darkpawns-server"}); err != nil {
		t.Fatalf("fully covered bundle rejected: %v", err)
	}
	// Strip the server from the record and re-sign: the plain gate passes,
	// but an exec path requiring the server must refuse.
	recordPath := filepath.Join(bundle, "release-assets", "publisher.json")
	var record map[string]any
	data, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &record); err != nil {
		t.Fatal(err)
	}
	delete(record["files"].(map[string]any), "darkpawns-server")
	data, err = json.MarshalIndent(record, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(recordPath, append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(recordPath+".sig", key.sign(t, append(data, '\n')), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := verifyBundlePublisherWithKey(bundle, key.public, nil); err != nil {
		t.Fatalf("record without the server should still pass the plain gate: %v", err)
	}
	if err := verifyBundlePublisherWithKey(bundle, key.public, []string{"darkpawns-server"}); err == nil {
		t.Fatal("exec path accepted a bundle whose record does not cover the server executable")
	}
}
