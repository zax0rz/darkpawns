package fileedit

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func writeFixture(t *testing.T, path, content string, perm fs.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), perm); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, perm); err != nil {
		t.Fatal(err)
	}
}

func TestWriteUsesAtomicRenameAndStrongETag(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "script.lua")
	writeFixture(t, path, "return 1\n", 0o640)
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	old := ETag([]byte("return 1\n"))
	got, err := Write(root, "script.lua", []byte("return 2\n"), old)
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	// A new inode proves temp-file + rename: an in-place truncate keeps the
	// inode and opens the torn-read window the brief closes.
	if os.SameFile(before, after) {
		t.Fatal("save reused the old inode; expected temp-file + rename")
	}
	if after.Mode().Perm() != 0o640 {
		t.Fatalf("mode = %v, want the existing file's 0640 preserved", after.Mode().Perm())
	}
	if got.ETag != ETag([]byte("return 2\n")) {
		t.Fatalf("etag = %q", got.ETag)
	}
	if _, err := Write(root, "script.lua", []byte("stale"), old); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale write error = %v, want ErrConflict", err)
	}
	content, _ := os.ReadFile(path)
	if string(content) != "return 2\n" {
		t.Fatalf("stale write changed file: %q", content)
	}
	leftovers, _ := filepath.Glob(filepath.Join(root, ".fileedit-*"))
	if len(leftovers) != 0 {
		t.Fatalf("temp files left behind: %v", leftovers)
	}
}

func TestConcurrentWritesWithOneETagHaveOneWinner(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, filepath.Join(root, "race.lua"), "base\n", 0o600)
	etag := ETag([]byte("base\n"))
	var wg sync.WaitGroup
	var mu sync.Mutex
	wins := 0
	for i := range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := Write(root, "race.lua", []byte{byte('a' + i)}, etag); err == nil {
				mu.Lock()
				wins++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if wins != 1 {
		t.Fatalf("%d writers won with the same If-Match; want exactly 1", wins)
	}
}

func TestCreateRefusesExistingFile(t *testing.T) {
	root := t.TempDir()
	if _, err := Create(root, "new.lua", []byte("return 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Create(root, "new.lua", []byte("return 2\n"), 0o644); !errors.Is(err, ErrExists) {
		t.Fatalf("second create = %v, want ErrExists", err)
	}
	if _, err := Write(root, "missing.lua", []byte("x"), ETag(nil)); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("write to a missing file = %v, want ErrNotExist (creation is Create)", err)
	}
}

func TestPathsCannotLeaveTheRoot(t *testing.T) {
	outside := t.TempDir()
	root := t.TempDir()
	secret := filepath.Join(outside, "secret")
	writeFixture(t, secret, "keep\n", 0o600)
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(root, "link.lua")); err != nil {
		t.Fatal(err)
	}
	etag := ETag([]byte("keep\n"))
	for _, name := range []string{"../outside", "/etc/passwd", "escape/secret", "link.lua"} {
		if _, err := Read(root, name); err == nil {
			t.Errorf("Read(%q) succeeded; want refusal", name)
		}
		if _, err := Write(root, name, []byte("pwned"), etag); err == nil {
			t.Errorf("Write(%q) succeeded; want refusal", name)
		}
		if _, err := Create(root, name, []byte("pwned"), 0o644); err == nil {
			t.Errorf("Create(%q) succeeded; want refusal", name)
		}
		if err := Delete(root, name, etag); err == nil {
			t.Errorf("Delete(%q) succeeded; want refusal", name)
		}
	}
	if _, err := List(root, "escape", nil); err == nil {
		t.Error("List through a symlinked directory succeeded; want refusal")
	}
	content, err := os.ReadFile(secret)
	if err != nil || string(content) != "keep\n" {
		t.Fatalf("file outside the root changed: %q %v", content, err)
	}
}

func TestAtomicWriteKeepsSymlinkAndModeForTelnetSave(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "news.real")
	writeFixture(t, real, "old\n", 0o640)
	link := filepath.Join(dir, "news")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	if err := AtomicWrite(link, []byte("new\n"), 0o666); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Lstat(link); err != nil || info.Mode()&fs.ModeSymlink == 0 {
		t.Fatalf("telnet save replaced the symlink: %v %v", info, err)
	}
	info, err := os.Stat(real)
	if err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(real)
	if string(content) != "new\n" || info.Mode().Perm() != 0o640 {
		t.Fatalf("target = %q mode %v; want new content, 0640 kept", content, info.Mode().Perm())
	}
}

func TestListAppliesLuaFilterAndHidesTempFiles(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"mob", "144", "junk"} {
		if err := os.Mkdir(filepath.Join(root, dir), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	writeFixture(t, filepath.Join(root, "globals.lua"), "", 0o600)
	writeFixture(t, filepath.Join(root, "README"), "", 0o600)
	writeFixture(t, filepath.Join(root, ".fileedit-abc.lua"), "", 0o600)
	entries, err := List(root, "", func(e fs.DirEntry) bool { return LuaFilter(e.Name()) })
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, e := range entries {
		got[e.Name] = true
	}
	for _, want := range []string{"mob", "144", "globals.lua"} {
		if !got[want] {
			t.Errorf("listing missing %q: %v", want, got)
		}
	}
	for _, hidden := range []string{"junk", "README", ".fileedit-abc.lua"} {
		if got[hidden] {
			t.Errorf("listing shows %q; luafilter should drop it", hidden)
		}
	}
}
