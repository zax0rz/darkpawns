package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/internal/releaseauth"
)

// L1-006/007/021 build side: a published archive must carry a per-file
// manifest that includes the executables and a publisher signature over it,
// and repacking must preserve entry modes and paths. The outer release
// manifest hashes archives after this step, so its digests cover the result.

type testKeys struct {
	private *rsa.PrivateKey
	public  []byte
}

func generateTestKeys(t *testing.T) testKeys {
	t.Helper()
	private, err := rsa.GenerateKey(rand.Reader, 3072)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := x509.MarshalPKIXPublicKey(&private.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	return testKeys{private: private, public: pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: encoded})}
}

// fixtureBundleDir writes a minimal goreleaser-shaped bundle directory.
func fixtureBundleDir(t *testing.T, root, name string, windows bool) string {
	t.Helper()
	bundle := filepath.Join(root, name)
	assets := filepath.Join(bundle, "release-assets")
	if err := os.MkdirAll(assets, 0o755); err != nil {
		t.Fatal(err)
	}
	suffix := ""
	if windows {
		suffix = ".exe"
	}
	for _, tool := range []string{"darkpawns" + suffix, "darkpawns-server" + suffix} {
		if err := os.WriteFile(filepath.Join(bundle, tool), []byte("fake-binary:"+tool), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(filepath.Join(assets, "web"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assets, "web", "index.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := map[string]any{"format": 1, "update_format": 1, "data_format": 1, "version": "0.0.0", "commit": "0000000000000000000000000000000000000000", "files": map[string]string{"web/index.html": "anything"}}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assets, "manifest.json"), append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	return bundle
}

func tarBundle(t *testing.T, bundle, archive string) {
	t.Helper()
	out, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = out.Close() }()
	gz := gzip.NewWriter(out)
	w := tar.NewWriter(gz)
	base := filepath.Base(bundle)
	_ = filepath.Walk(bundle, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(bundle, path)
		if rel == "." {
			return nil
		}
		header, herr := tar.FileInfoHeader(info, "")
		if herr != nil {
			return herr
		}
		header.Name = base + "/" + filepath.ToSlash(rel)
		if err := w.WriteHeader(header); err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			f, ferr := os.Open(path)
			if ferr != nil {
				return ferr
			}
			_, cerr := io.Copy(w, f)
			_ = f.Close()
			return cerr
		}
		return nil
	})
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestPostProcessArchiveSignsManifestAndPreservesModes(t *testing.T) {
	keys := generateTestKeys(t)
	root := t.TempDir()
	bundle := fixtureBundleDir(t, root, "darkpawns_0.1.4_linux_amd64", false)
	archive := filepath.Join(root, "darkpawns_0.1.4_linux_amd64.tar.gz")
	tarBundle(t, bundle, archive)
	originalManifest, err := os.ReadFile(filepath.Join(bundle, "release-assets", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := postProcessArchive(archive, keys.private, keys.public); err != nil {
		t.Fatalf("postProcessArchive: %v", err)
	}

	// Extract the repacked archive and check the full chain.
	extract := t.TempDir()
	f, err := os.Open(archive)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = gz.Close() }()
	r := tar.NewReader(gz)
	sawExecutableMode := false
	for {
		h, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(extract, filepath.FromSlash(h.Name))
		if h.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(target, 0o755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(r)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, data, os.FileMode(h.Mode)); err != nil {
			t.Fatal(err)
		}
		if strings.HasSuffix(h.Name, "/darkpawns") && h.Mode&0o111 != 0 {
			sawExecutableMode = true
		}
	}
	if !sawExecutableMode {
		t.Fatal("repacked tar lost the executable bit")
	}
	out := filepath.Join(extract, "darkpawns_0.1.4_linux_amd64")
	manifest, err := os.ReadFile(filepath.Join(out, "release-assets", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	// The hop-1 guarantee: older launchers verify this file, so it must be
	// byte-identical to what goreleaser packed.
	if !bytes.Equal(manifest, originalManifest) {
		t.Fatal("post-processing rewrote manifest.json; older launchers could no longer run this release")
	}
	record, err := os.ReadFile(filepath.Join(out, "release-assets", "publisher.json"))
	if err != nil {
		t.Fatal("publisher.json missing from archive")
	}
	signature, err := os.ReadFile(filepath.Join(out, "release-assets", "publisher.json.sig"))
	if err != nil {
		t.Fatal("publisher.json.sig missing from archive")
	}
	if err := releaseauth.VerifyBytes(record, signature, keys.public); err != nil {
		t.Fatalf("publisher signature: %v", err)
	}
	var pub publisherRecord
	if err := json.Unmarshal(record, &pub); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"darkpawns", "darkpawns-server", "release-assets/manifest.json"} {
		if _, ok := pub.Files[name]; !ok {
			t.Fatalf("publisher record missing %s", name)
		}
	}
	// Tamper detection: the signature covers the record bytes.
	if err := releaseauth.VerifyBytes([]byte("{}\n"), signature, keys.public); err == nil {
		t.Fatal("signature accepted over different bytes")
	}
}

func TestPostProcessArchiveZip(t *testing.T) {
	keys := generateTestKeys(t)
	root := t.TempDir()
	bundle := fixtureBundleDir(t, root, "darkpawns_0.1.4_windows_amd64", true)
	archive := filepath.Join(root, "darkpawns_0.1.4_windows_amd64.zip")
	out, err := os.Create(archive)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(out)
	base := filepath.Base(bundle)
	_ = filepath.Walk(bundle, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(bundle, path)
		if rel == "." {
			return nil
		}
		header, herr := zip.FileInfoHeader(info)
		if herr != nil {
			return herr
		}
		header.Name = base + "/" + filepath.ToSlash(rel)
		entry, werr := w.CreateHeader(header)
		if werr != nil {
			return werr
		}
		if info.Mode().IsRegular() {
			data, rerr := os.ReadFile(path)
			if rerr != nil {
				return rerr
			}
			_, werr = entry.Write(data)
			return werr
		}
		return nil
	})
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}
	if err := postProcessArchive(archive, keys.private, keys.public); err != nil {
		t.Fatalf("postProcessArchive zip: %v", err)
	}
	r, err := zip.OpenReader(archive)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = r.Close() }()
	var record, signature []byte
	for _, f := range r.File {
		if !strings.HasSuffix(f.Name, "publisher.json") && !strings.HasSuffix(f.Name, "publisher.json.sig") {
			continue
		}
		in, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(in)
		_ = in.Close()
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasSuffix(f.Name, "publisher.json.sig") {
			signature = data
		} else {
			record = data
		}
	}
	if record == nil || signature == nil {
		t.Fatal("zip archive missing signed publisher record")
	}
	if err := releaseauth.VerifyBytes(record, signature, keys.public); err != nil {
		t.Fatalf("zip publisher signature: %v", err)
	}
	var pub publisherRecord
	if err := json.Unmarshal(record, &pub); err != nil {
		t.Fatal(err)
	}
	if _, ok := pub.Files["darkpawns.exe"]; !ok {
		t.Fatal("windows executable missing from publisher record")
	}
	if _, ok := pub.Files["darkpawns-server.exe"]; !ok {
		t.Fatal("server executable missing from publisher record")
	}
}
