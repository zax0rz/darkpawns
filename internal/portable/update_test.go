package portable

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestUpdateArchiveRefusesUnsafeEntries(t *testing.T) {
	for _, name := range []string{"../escape", "/absolute", "root/../escape", "root/a\\b", "root/C:escape", "root/CON.txt", "root/end.", "root/end "} {
		t.Run(name, func(t *testing.T) {
			var data bytes.Buffer
			gz := gzip.NewWriter(&data)
			tw := tar.NewWriter(gz)
			if err := tw.WriteHeader(&tar.Header{Name: name, Size: 1, Mode: 0o600}); err != nil {
				t.Fatal(err)
			}
			if _, err := tw.Write([]byte("x")); err != nil {
				t.Fatal(err)
			}
			if err := tw.Close(); err != nil {
				t.Fatal(err)
			}
			if err := gz.Close(); err != nil {
				t.Fatal(err)
			}
			archive := filepath.Join(t.TempDir(), "archive")
			if err := os.WriteFile(archive, data.Bytes(), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := extractUpdate(archive, t.TempDir()); err == nil {
				t.Fatal("unsafe path accepted")
			}
		})
	}
	for _, kind := range []byte{tar.TypeLink, tar.TypeSymlink, tar.TypeFifo} {
		var data bytes.Buffer
		gz := gzip.NewWriter(&data)
		tw := tar.NewWriter(gz)
		if err := tw.WriteHeader(&tar.Header{Name: "root/link", Typeflag: kind, Linkname: "elsewhere", Mode: 0o600}); err != nil {
			t.Fatal(err)
		}
		if err := tw.Close(); err != nil {
			t.Fatal(err)
		}
		if err := gz.Close(); err != nil {
			t.Fatal(err)
		}
		archive := filepath.Join(t.TempDir(), "archive")
		if err := os.WriteFile(archive, data.Bytes(), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := extractUpdate(archive, t.TempDir()); err == nil {
			t.Fatalf("special type %v accepted", kind)
		}
	}
}

func TestUpdateZipAndDuplicateRefusal(t *testing.T) {
	for _, names := range [][]string{{"release/darkpawns.exe", "release/release-assets/manifest.json"}, {"release/A", "release/a"}, {"one/a", "two/b"}} {
		var data bytes.Buffer
		zw := zip.NewWriter(&data)
		for _, name := range names {
			w, err := zw.Create(name)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := w.Write([]byte("test")); err != nil {
				t.Fatal(err)
			}
		}
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}
		archive := filepath.Join(t.TempDir(), "archive")
		if err := os.WriteFile(archive, data.Bytes(), 0o600); err != nil {
			t.Fatal(err)
		}
		root, err := extractUpdate(archive, t.TempDir())
		if names[0] == "release/darkpawns.exe" {
			if err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(filepath.Join(root, "darkpawns.exe")); err != nil {
				t.Fatal(err)
			}
		} else if err == nil {
			t.Fatal("duplicate/multiple root accepted")
		}
	}
}

func TestUpdateDigestAndOfficialReleaseSelection(t *testing.T) {
	source := filepath.Join(t.TempDir(), "archive")
	if err := os.WriteFile(source, []byte("archive"), 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte("archive"))
	digest := hex.EncodeToString(sum[:])
	for _, want := range []string{"", strings.Repeat("0", 64), digest} {
		_, err := fetchUpdate(context.Background(), source, want, filepath.Join(t.TempDir(), "download"))
		if (want == digest) != (err == nil) {
			t.Fatalf("digest %q: %v", want, err)
		}
	}
	extension := ".tar.gz"
	if runtime.GOOS == "windows" {
		extension = ".zip"
	}
	name := "darkpawns_1.0.0_" + runtime.GOOS + "_" + runtime.GOARCH + extension
	asset := releaseAsset{Name: name, URL: "https://github.com/zax0rz/darkpawns/releases/download/v1.0.0/" + name, Digest: "sha256:" + digest, Size: 10}
	release := githubRelease{Tag: "v1.0.0", Assets: []releaseAsset{asset}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if err := json.NewEncoder(w).Encode(release); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	if _, err := latestAsset(context.Background(), server.Client(), server.URL); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"draft", "prerelease", "missing digest", "foreign host", "duplicate", "wrong platform", "too large"} {
		bad := release
		bad.Assets = append([]releaseAsset(nil), release.Assets...)
		switch kind {
		case "draft":
			bad.Draft = true
		case "prerelease":
			bad.Prerelease = true
		case "missing digest":
			bad.Assets[0].Digest = ""
		case "foreign host":
			bad.Assets[0].URL = "https://evil.invalid/" + name
		case "duplicate":
			bad.Assets = append(bad.Assets, asset)
		case "wrong platform":
			bad.Assets[0].Name = "wrong.zip"
		case "too large":
			bad.Assets[0].Size = maxArchive + 1
		}
		if _, err := selectAsset(bad); err == nil {
			t.Fatal("accepted", kind)
		}
	}
}

func TestUpdateRecoveryRestoresWholeMutableTreeAndIsRetryable(t *testing.T) {
	bundle := fixture(t)
	instance := filepath.Join(t.TempDir(), "world")
	if err := Init(bundle, instance, 4350); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(filepath.Join(instance, "instance.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"lib/data/notifications.json", "lib/data/notifications.json.key", "lib/world/etc/board.mort"} {
		if err := os.WriteFile(filepath.Join(instance, name), []byte("private original"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	record := updateRecord{ID: strings.Repeat("a", 32), Phase: "testing", Previous: bundle, Candidate: filepath.Join(instance, ".updates", strings.Repeat("a", 32), "release", "candidate")}
	work := filepath.Join(updatePath(instance), record.ID)
	if err := os.MkdirAll(work, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := snapshotInstance(instance, filepath.Join(work, "backup")); err != nil {
		t.Fatal(err)
	}
	if err := verifySnapshot(work); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(instance, "lib/data/notifications.json"), []byte("candidate writes"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(instance, "lib/data/extra"), []byte("candidate only"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := saveUpdate(instance, record); err != nil {
		t.Fatal(err)
	}
	if err := pendingUpdate(instance); err == nil {
		t.Fatal("interrupted update did not block normal start")
	}
	if err := updateRunAllowed(instance, record.Candidate, "wrong"); err == nil {
		t.Fatal("wrong candidate token accepted")
	}
	if err := updateRunAllowed(instance, record.Candidate, record.ID); err != nil {
		t.Fatal(err)
	}
	// Model interruption immediately after the failed tree was moved aside.
	record.Phase = "recovering"
	if err := os.Rename(filepath.Join(instance, "lib"), filepath.Join(work, "failed-lib")); err != nil {
		t.Fatal(err)
	}
	if err := saveUpdate(instance, record); err != nil {
		t.Fatal(err)
	}
	if err := RecoverUpdate(context.Background(), instance); err != nil {
		t.Fatal(err)
	}
	if err := pendingUpdate(instance); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(filepath.Join(instance, "instance.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(original, after) {
		t.Fatal("signing configuration changed")
	}
	for _, name := range []string{"lib/data/notifications.json", "lib/data/notifications.json.key", "lib/world/etc/board.mort"} {
		data, err := os.ReadFile(filepath.Join(instance, name))
		if err != nil || string(data) != "private original" {
			t.Fatalf("state not restored: %s %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(instance, "lib/data/extra")); !os.IsNotExist(err) {
		t.Fatal("candidate state leaked into restore")
	}
	if _, err := os.Stat(filepath.Join(work, "failed-lib/data/extra")); err != nil {
		t.Fatal("attempted state not retained", err)
	}
	active, err := ActiveBundle("unused", instance)
	if err != nil || active != bundle {
		t.Fatal(active, err)
	}
}

func TestDamagedBackupAndRunLockBlockRecovery(t *testing.T) {
	for _, kind := range []string{"checksum", "lock"} {
		t.Run(kind, func(t *testing.T) {
			bundle := fixture(t)
			instance := filepath.Join(t.TempDir(), "world")
			if err := Init(bundle, instance, 4350); err != nil {
				t.Fatal(err)
			}
			record := updateRecord{ID: strings.Repeat("b", 32), Phase: "testing", Previous: bundle, Candidate: filepath.Join(instance, ".updates", strings.Repeat("b", 32), "release", "candidate")}
			work := filepath.Join(updatePath(instance), record.ID)
			if err := os.MkdirAll(work, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := snapshotInstance(instance, filepath.Join(work, "backup")); err != nil {
				t.Fatal(err)
			}
			if err := saveUpdate(instance, record); err != nil {
				t.Fatal(err)
			}
			if kind == "checksum" {
				if err := os.WriteFile(filepath.Join(work, "backup/instance.json"), []byte("bad"), 0o600); err != nil {
					t.Fatal(err)
				}
			} else {
				unlock, err := stoppedLock(instance)
				if err != nil {
					t.Fatal(err)
				}
				defer unlock()
			}
			if err := RecoverUpdate(context.Background(), instance); err == nil {
				t.Fatal("unsafe restore accepted")
			}
			if _, err := Load(instance); err != nil {
				t.Fatal("live config damaged", err)
			}
		})
	}
}

func TestAppliedPlanCloseRetainsHistoryAfterAnotherTransaction(t *testing.T) {
	root := t.TempDir()
	work := filepath.Join(root, "keep")
	if err := os.Mkdir(work, 0o700); err != nil {
		t.Fatal(err)
	}
	p := UpdatePlan{instance: root, work: work, retained: true}
	if err := p.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(work); err != nil {
		t.Fatal("applied history was deleted", err)
	}
}
