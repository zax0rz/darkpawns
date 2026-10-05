package game

import (
	"os"
	"path/filepath"
	"testing"
)

// Alias files are keyed by player name. A name that is not one safe path
// element must never produce a path, or WriteAliases would create files
// outside aliasDir.
func TestAliasFilePathRejectsUnsafeNames(t *testing.T) {
	for _, name := range []string{"", "../x", "guest/../../../tmp/x", "a/b", `a\b`, "..", "Aiko\x1b[31m", "Guest 1", "Gúest"} {
		if got := aliasFilePath(name); got != "" {
			t.Fatalf("aliasFilePath(%q) = %q, want no path", name, got)
		}
	}
	for _, name := range []string{"Aiko", "Guest_12", "zax"} {
		got := aliasFilePath(name)
		if got == "" || filepath.Dir(filepath.Dir(got)) != filepath.Clean(aliasDir) {
			t.Fatalf("aliasFilePath(%q) = %q, want a file directly under %s/<initial>", name, got, aliasDir)
		}
	}
}

func TestWriteAliasesRefusesUnsafeNameWithoutWriting(t *testing.T) {
	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(wd) }()

	err = WriteAliases("guest/../../escaped", []Alias{{Alias: "x", Replacement: " say hi"}})
	if err == nil {
		t.Fatal("WriteAliases accepted an unsafe owner name")
	}
	if _, statErr := os.Stat(filepath.Join(dir, "escaped.alias")); !os.IsNotExist(statErr) {
		t.Fatalf("alias file escaped aliasDir: %v", statErr)
	}
	if aliases, readErr := ReadAliases("guest/../../escaped"); readErr != nil || aliases != nil {
		t.Fatalf("ReadAliases(unsafe) = %v, %v", aliases, readErr)
	}
}
