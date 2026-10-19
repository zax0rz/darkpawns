package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPortableWindowsEditorDelete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "editor.txt")
	if err := killEditorFile(path); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("text"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := killEditorFile(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("readable file was not removed")
	}
}
