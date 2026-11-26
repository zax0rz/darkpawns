package game

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDoGenWriteOpenFileFailureUnitProof pins Round 2 row 11 under the
// 2026-11-11 unit-proof exception (Zach's 2026-11-25 ruling 6): when fopen()
// fails in C do_gen_write (src/act.other.c:1122-1125), C sends
// "Could not open the file.  Sorry.\r\n" and returns without writing a report.
func TestDoGenWriteOpenFileFailureUnitProof(t *testing.T) {
	tmp := t.TempDir()
	// Creating misc/bugs as a directory lets os.MkdirAll("misc") succeed while
	// forcing os.OpenFile("misc/bugs", ...) to fail with EISDIR.
	if err := os.MkdirAll(filepath.Join(tmp, "misc", "bugs"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(tmp)

	w := &World{}
	ch := NewPlayer(1, "Reporter", 3001)
	var got []string
	ch.entryTextSink = func(msg string) { got = append(got, msg) }

	w.doGenWrite(ch, nil, "bug", "something broke")

	const want = "Could not open the file.  Sorry.\r\n"
	if len(got) != 1 || got[0] != want {
		t.Fatalf("messages = %q, want [%q]", got, want)
	}
}
