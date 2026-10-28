package main

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// Both build hooks of a platform run concurrently with identical inputs.
// Each stages into its own temporary tree and renames it into place; the
// rename loser must treat the existing winner as success (an earlier
// lock-and-wait design deadlocked once on a Windows runner).
func TestConcurrentStagingConvergesWithoutLocks(t *testing.T) {
	root := t.TempDir()
	buildDir := filepath.Join(root, "dist", "launcher_linux_amd64_v1")
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		t.Fatal(err)
	}
	serverDir := filepath.Join(root, "dist", "server_linux_amd64_v1")
	if err := os.MkdirAll(serverDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(buildDir, "darkpawns"), []byte("launcher"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(serverDir, "darkpawns-server"), []byte("server"), 0o755); err != nil {
		t.Fatal(err)
	}
	assets := filepath.Join(root, "release-assets")
	if err := os.MkdirAll(assets, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(assets, "manifest.json"), []byte(`{"format":1}`), 0o644); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	failures := make(chan error, 4)
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			failures <- stageBundleContentsAt(root, "linux", "amd64")
		}()
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatalf("concurrent staging failed: %v", err)
		}
	}

	out := filepath.Join(root, "dist", "signed-bundles", "linux_amd64")
	for _, name := range []string{"darkpawns", "darkpawns-server", "release-assets/manifest.json"} {
		if _, err := os.Lstat(filepath.Join(out, name)); err != nil {
			t.Fatalf("staged tree missing %s: %v", name, err)
		}
	}
	// No staging leftovers: losers delete their temporary trees.
	if leftover, err := os.ReadDir(filepath.Join(root, "dist", "signed-bundles")); err == nil {
		for _, entry := range leftover {
			if strings.Contains(entry.Name(), ".staging") {
				t.Fatalf("staging leftover: %s", entry.Name())
			}
		}
	}
}

// A hook that runs before the platform's second binary exists stands down
// quietly; the archive phase starts only after every build hook returns.
func TestStagingWaitsForMissingBinary(t *testing.T) {
	root := t.TempDir()
	buildDir := filepath.Join(root, "dist", "launcher_linux_amd64_v1")
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(buildDir, "darkpawns"), []byte("launcher"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "release-assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := stageBundleContentsAt(root, "linux", "amd64"); err != nil {
		t.Fatalf("early hook must stand down, not fail: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(root, "dist", "signed-bundles", "linux_amd64")); !os.IsNotExist(err) {
		t.Fatal("nothing should be staged while a binary is missing")
	}
}
