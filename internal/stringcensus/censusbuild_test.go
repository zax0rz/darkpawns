package stringcensus

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCMissingChecksGoContainment guards DP-1357: a C segment is c-missing iff
// no Go segment contains it (goSet.containsSubstring), not the inverse
// (goSet.hasPatternInside).
func TestCMissingChecksGoContainment(t *testing.T) {
	root := t.TempDir()
	goDir := filepath.Join(root, "pkg", "game")
	cDir := filepath.Join(root, "src")
	dataDir := filepath.Join(root, "lib", "world")
	for _, dir := range []string{goDir, cDir, dataDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	goSrc := `package game

func emit(ch interface{ Send(string) }) {
	ch.Send("You cannot\r\n")
	ch.Send("With a groan, The door swings open.\r\n")
}
`
	cSrc := `void send_to_char(const char *messg, void *ch);
void f(void *ch) {
	send_to_char("You cannot cast that spell while sleeping.\r\n", ch);
	send_to_char("The door swings open.\r\n", ch);
}
`
	if err := os.WriteFile(filepath.Join(goDir, "emit.go"), []byte(goSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cDir, "act.c"), []byte(cSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := Run(Options{
		Root:     root,
		GoDirs:   []string{"pkg/game"},
		CDir:     "src",
		DataDirs: []string{"lib/world"},
		OutDir:   "out",
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// "You cannot cast that spell while sleeping." is NOT contained by "You cannot" -> must be c-missing.
	assertCMissing(t, report, "You cannot cast that spell while sleeping.")
	// "The door swings open." IS contained by "With a groan, The door swings open." -> must NOT be c-missing.
	assertNotReported(t, report, "The door swings open.")
}
