package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRunProfilingSessionReturnsErrorOnInvalidDir(t *testing.T) {
	tmp := t.TempDir()
	// Use an existing regular file as the profile directory so MkdirAll fails.
	profileDir := filepath.Join(tmp, "notadir")
	if err := os.WriteFile(profileDir, []byte("x"), 0o644); err != nil {
		t.Fatalf("create file: %v", err)
	}

	err := RunProfilingSession(profileDir, 1*time.Millisecond)
	if err == nil {
		t.Fatal("expected error when profile dir is invalid")
	}
}

func TestRunProfilingSessionReturnsErrorWhenWritesFail(t *testing.T) {
	profileDir := t.TempDir()

	done := make(chan error, 1)
	go func() {
		done <- RunProfilingSession(profileDir, 500*time.Millisecond)
	}()

	// Wait for the CPU profile to start, then make the directory read-only
	// so the heap/block/mutex/goroutine writes fail.
	time.Sleep(100 * time.Millisecond)
	if err := os.Chmod(profileDir, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	defer os.Chmod(profileDir, 0o755)

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected error when profile writes fail")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("RunProfilingSession did not return")
	}
}

func TestProfilerWriteHeapProfileReturnsErrorOnReadOnlyDir(t *testing.T) {
	profileDir := t.TempDir()
	if err := os.Chmod(profileDir, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	defer os.Chmod(profileDir, 0o755)

	p := NewProfiler(profileDir)
	err := p.WriteHeapProfile()
	if err == nil {
		t.Fatal("expected error writing heap profile to read-only dir")
	}
}
