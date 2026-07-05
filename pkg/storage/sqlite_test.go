package storage

import (
	"path/filepath"
	"testing"
)

func TestNewSQLiteBackend_DSNWithSpecialChars(t *testing.T) {
	tmpDir := t.TempDir()

	// Spaces in the path should be URI-escaped and the database should still open.
	dbPath := filepath.Join(tmpDir, "db with spaces", "player data.db")

	backend, err := NewSQLiteBackend(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteBackend with spaces in path failed: %v", err)
	}
	defer func() { _ = backend.Close() }()

	// Sanity check that we can actually read/write.
	_, err = backend.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
}
