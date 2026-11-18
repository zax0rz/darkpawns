package session

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestOLCOpenParentObstructionClassifier(t *testing.T) {
	root := t.TempDir()
	blocked := filepath.Join(root, "blocked")
	if err := os.WriteFile(blocked, []byte("obstruction"), 0o600); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(root, "missing")
	for _, test := range []struct {
		name, op, parent string
		cause            error
		want             bool
	}{
		{"open non-directory", "open", blocked, syscall.ENOTDIR, true},
		{"mkdir non-directory", "mkdir", blocked, syscall.ENOTDIR, true},
		{"open missing", "open", missing, os.ErrNotExist, true},
		{"open descendant of file", "open", filepath.Join(blocked, "child"), syscall.ENOTDIR, true},
		{"valid parent", "open", root, os.ErrNotExist, false},
		{"permission", "open", blocked, os.ErrPermission, false},
		{"mkdir permission", "mkdir", blocked, os.ErrPermission, false},
		{"write", "write", blocked, syscall.ENOTDIR, false},
		{"sync", "sync", blocked, syscall.ENOTDIR, false},
		{"close", "close", blocked, syscall.ENOTDIR, false},
		{"chmod", "chmod", blocked, syscall.ENOTDIR, false},
		{"stat", "stat", blocked, syscall.ENOTDIR, false},
		{"rename", "rename", blocked, syscall.ENOTDIR, false},
		{"no root", "open", "", os.ErrNotExist, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := fmt.Errorf("wrapped: %w", &os.PathError{Op: test.op, Path: filepath.Join(test.parent, ".tmp-probe"), Err: test.cause})
			if got := olcOpenParentObstruction(err, test.parent); got != test.want {
				t.Fatalf("classified=%v want=%v", got, test.want)
			}
		})
	}
	if olcOpenParentObstruction(fmt.Errorf("plain failure"), blocked) || olcOpenParentObstruction(nil, blocked) {
		t.Fatal("non-path failure classified")
	}
}

func TestOLCSaveTargetDirectoryFailureClassifier(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "30.wld")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	tmp := filepath.Join(root, ".tmp-save")
	if err := os.WriteFile(tmp, []byte("save"), 0o600); err != nil {
		t.Fatal(err)
	}
	actualErr := os.Rename(tmp, target)
	if actualErr == nil {
		t.Fatal("rename to directory unexpectedly succeeded")
	}
	for _, test := range []struct {
		name   string
		err    error
		target string
		want   bool
	}{
		{"exact target directory rename", actualErr, target, true},
		{"different target", actualErr, target + ".other", false},
		{"other rename error", &os.LinkError{Op: "rename", Old: ".tmp-save", New: target, Err: syscall.EACCES}, target, false},
		{"rename to nonexistent target", &os.LinkError{Op: "rename", Old: ".tmp-save", New: target + ".missing", Err: syscall.EEXIST}, target + ".missing", false},
		{"other operation", &os.LinkError{Op: "open", Old: ".tmp-save", New: target, Err: syscall.EISDIR}, target, false},
		{"plain failure", fmt.Errorf("plain failure"), target, false},
		{"no target", &os.LinkError{Op: "rename", Old: ".tmp-save", New: target, Err: syscall.EISDIR}, "", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := olcSaveTargetDirectoryFailure(test.err, test.target); got != test.want {
				t.Fatalf("classified=%v want=%v", got, test.want)
			}
		})
	}
}
