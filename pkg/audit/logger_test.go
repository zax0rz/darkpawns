package audit

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestAuditLogger_CloseReturnsError(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "audit.log")

	logger, err := NewAuditLogger(path)
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}

	if err := logger.Close(); err != nil {
		t.Fatalf("Close returned unexpected error: %v", err)
	}

	// Second close should return an error because the file is already closed.
	if err := logger.Close(); err == nil {
		t.Error("expected error on second Close, got nil")
	}
}

func TestAuditInit_RepeatedCalls(t *testing.T) {
	tmpDir := t.TempDir()
	first := filepath.Join(tmpDir, "audit1.log")
	second := filepath.Join(tmpDir, "audit2.log")

	if err := Init(first); err != nil {
		t.Fatalf("first Init failed: %v", err)
	}

	LogEvent(AuditEvent{EventType: "test", Action: "first", Success: true})

	if err := Init(second); err != nil {
		t.Fatalf("second Init failed: %v", err)
	}

	LogEvent(AuditEvent{EventType: "test", Action: "second", Success: true})

	// Verify the second file received its event.
	data, err := os.ReadFile(second)
	if err != nil {
		t.Fatalf("read second audit log: %v", err)
	}
	if !bytes.Contains(data, []byte(`"action":"second"`)) {
		t.Errorf("second audit log missing expected event: %s", data)
	}
}

func TestAuditInit_ConcurrentLogEvent(t *testing.T) {
	tmpDir := t.TempDir()
	first := filepath.Join(tmpDir, "audit1.log")
	second := filepath.Join(tmpDir, "audit2.log")

	if err := Init(first); err != nil {
		t.Fatalf("first Init failed: %v", err)
	}

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Goroutine continuously logging events.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				LogEvent(AuditEvent{EventType: "test", Action: "concurrent", Success: true})
			}
		}
	}()

	// Re-initialize from the main goroutine while logging is active.
	if err := Init(second); err != nil {
		close(stop)
		wg.Wait()
		t.Fatalf("second Init failed: %v", err)
	}

	close(stop)
	wg.Wait()
}

func TestAuditLogger_ConcurrentWriteProducesValidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "audit.log")

	logger, err := NewAuditLogger(path)
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				logger.Log(AuditEvent{
					EventType: "test",
					Action:    "concurrent_write",
					Details:   strings.Repeat("x", 1000),
					Success:   true,
				})
			}
		}()
	}
	wg.Wait()

	if err := logger.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open audit log: %v", err)
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var e AuditEvent
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			t.Fatalf("invalid JSON line: %v", err)
		}
		count++
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan audit log: %v", err)
	}
	if count != 500 {
		t.Errorf("expected 500 events, got %d", count)
	}
}

func TestNewAuditLogger_PathTraversalRejected(t *testing.T) {
	for _, p := range []string{"/foo/../../etc/passwd", "../../etc/passwd", "a/../../../etc/hosts"} {
		_, err := NewAuditLogger(p)
		if err == nil {
			t.Errorf("expected error for path %q", p)
		}
	}
}

func TestLogEvent_SilentDropWhenUninitialized(t *testing.T) {
	var buf bytes.Buffer
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))

	// Reset global logger — prior tests may have called Init.
	loggerMu.Lock()
	globalLogger = nil
	loggerMu.Unlock()

	LogEvent(AuditEvent{EventType: "test", Action: "dropped", Success: true})

	if buf.Len() == 0 {
		t.Error("expected a log warning when globalLogger is nil")
	}
}

func TestAuditLogger_LogAndClose(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "audit.log")

	logger, err := NewAuditLogger(path)
	if err != nil {
		t.Fatalf("NewAuditLogger failed: %v", err)
	}

	logger.Log(AuditEvent{
		EventType: "test",
		Action:    "unit_test",
		Success:   true,
	})

	if err := logger.Close(); err != nil {
		t.Fatalf("Close returned unexpected error: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat audit log: %v", err)
	}
	if info.Size() == 0 {
		t.Error("expected audit log to contain data after Log and Close")
	}
}
