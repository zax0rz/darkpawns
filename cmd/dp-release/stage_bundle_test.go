package main

import (
	"os"
	"path/filepath"
	"testing"
)

// Two concurrent stagings of one platform must not interleave: the second
// waits for the first's done marker instead of deleting its output
// mid-copy (the macOS native CI hit exactly that as unlinkat races).
func TestStageBundleLockSerializesConcurrentStaging(t *testing.T) {
	dir := t.TempDir()
	bundle := filepath.Join(dir, "dist", "signed-bundles", "linux_amd64")
	if err := os.MkdirAll(bundle, 0o755); err != nil {
		t.Fatal(err)
	}
	lockPath := bundle + ".lock"
	lock, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()

	done := make(chan error, 1)
	go func() { done <- waitForStaging(bundle, lockPath) }()

	// No done marker yet: the waiter keeps waiting.
	select {
	case err := <-done:
		t.Fatalf("waiter returned early: %v", err)
	default:
	}
	if err := os.WriteFile(filepath.Join(bundle, ".done"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatalf("waiter failed: %v", err)
	}

	// A vanished lock without a marker also releases the waiter.
	os.Remove(filepath.Join(bundle, ".done"))
	os.Remove(lockPath)
	lock2, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	lock2.Close()
	go func() { done <- waitForStaging(bundle, lockPath) }()
	os.Remove(lockPath)
	if err := <-done; err != nil {
		t.Fatalf("waiter failed after lock vanished: %v", err)
	}
}
