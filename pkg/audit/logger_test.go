package audit

import (
	"os"
	"path/filepath"
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
