package main

import (
	"context"
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

func TestStartPProfServerMissingAuthReturnsError(t *testing.T) {
	t.Setenv("PPROF_USER", "")
	t.Setenv("PPROF_PASS", "")

	server, err := StartPProfServer(":0")
	if err == nil {
		t.Fatal("expected error when auth env vars are missing")
	}
	if server != nil {
		t.Fatal("expected nil server when auth env vars are missing")
	}
}

func TestStartPProfServerStartsWithAuth(t *testing.T) {
	t.Setenv("PPROF_USER", "user")
	t.Setenv("PPROF_PASS", "pass")

	server, err := StartPProfServer("127.0.0.1:0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if server == nil {
		t.Fatal("expected non-nil server")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}

func TestPProfBindAddr(t *testing.T) {
	tests := []struct {
		name string
		args []string
		env  map[string]string
		want string
	}{
		{
			name: "default localhost",
			args: []string{"profiler", "pprof"},
			env:  map[string]string{},
			want: "127.0.0.1:6060",
		},
		{
			name: "env override",
			args: []string{"profiler", "pprof"},
			env:  map[string]string{"PPROF_BIND_ADDR": "0.0.0.0:6060"},
			want: "0.0.0.0:6060",
		},
		{
			name: "cli arg beats env",
			args: []string{"profiler", "pprof", ":7070"},
			env:  map[string]string{"PPROF_BIND_ADDR": "0.0.0.0:6060"},
			want: ":7070",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getenv := func(key string) string {
				return tt.env[key]
			}
			if got := pprofBindAddr(tt.args, getenv); got != tt.want {
				t.Fatalf("pprofBindAddr() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProfileFilename_Uniqueness(t *testing.T) {
	profileDir := t.TempDir()
	p := NewProfiler(profileDir)

	// Generate two heap profiles in rapid succession.
	if err := p.WriteHeapProfile(); err != nil {
		t.Fatalf("first WriteHeapProfile failed: %v", err)
	}
	if err := p.WriteHeapProfile(); err != nil {
		t.Fatalf("second WriteHeapProfile failed: %v", err)
	}

	entries, err := os.ReadDir(profileDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}

	heapFiles := make(map[string]struct{})
	for _, e := range entries {
		if !e.Type().IsRegular() {
			continue
		}
		name := e.Name()
		if len(name) > 5 && name[:5] == "heap-" {
			heapFiles[name] = struct{}{}
		}
	}
	if len(heapFiles) != 2 {
		t.Fatalf("expected 2 distinct heap profile files, got %v", heapFiles)
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
