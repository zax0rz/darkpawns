package game

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeHelpFixture builds a temp help dir with the given index + files and
// returns its path. Each file is the raw .hlp content (CRLF endings up to the
// caller; we join with \n and the loader's getOneLine keeps \r).
func writeHelpFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// TestLoadHelpFilesMultiKeywordAndQuoted verifies one_word splitting: a keyword
// line `AID FIRST "FIRST AID"` yields three keywords — `aid`, `first`, and the
// QUOTED `first aid` (strings.Fields would have split it into `first` and `aid`
// separately, dropping the quoted form). This is the fidelity fix.
func TestLoadHelpFilesMultiKeywordAndQuoted(t *testing.T) {
	dir := writeHelpFixture(t, map[string]string{
		"index": "topics.hlp\n$\n",
		"topics.hlp": "AID FIRST \"FIRST AID\"\r\n" +
			"Body line about first aid.\r\n" +
			"#\r\n" +
			"$\r\n",
	})
	table, err := LoadHelpFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Expect three keywords for the one record.
	want := map[string]bool{"aid": true, "first": true, "first aid": true}
	got := map[string]bool{}
	for _, e := range table {
		got[e.Keyword] = true
	}
	for k := range want {
		if !got[k] {
			t.Errorf("keyword %q missing from table (one_word quoted-split broken); have %v", k, got)
		}
	}
}

// TestLoadHelpFilesKeepsKeywordLineAsFirstLine verifies the entry text retains
// the keyword line as its first line (C load_help stores key\r\n + body; do_help
// skips past the first \n at display). The loader must NOT strip it.
func TestLoadHelpFilesKeepsKeywordLineAsFirstLine(t *testing.T) {
	dir := writeHelpFixture(t, map[string]string{
		"index": "t.hlp\n$\n",
		"t.hlp": "SAY\r\nYou can also use ' for say.\r\n#\r\n$\r\n",
	})
	table, err := LoadHelpFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(table) != 1 {
		t.Fatalf("entries = %d, want 1", len(table))
	}
	if got := table[0].Entry; !strings.HasPrefix(got, "SAY\r\n") {
		t.Errorf("entry first line = %q; want keyword line retained (SAY\\r\\n…)", got)
	}
	// Body after the keyword line is present with CRLF.
	if !strings.Contains(table[0].Entry, "You can also use ' for say.\r\n") {
		t.Errorf("entry body missing or not CRLF: %q", table[0].Entry)
	}
}

// TestLoadHelpFilesHashTerminatorIsFirstChar verifies C's `while (*line != '#')`
// — a body line whose FIRST char is '#' terminates the entry, even if it has
// trailing content. (The prior loader used TrimSpace=="#", a divergence.)
func TestLoadHelpFilesHashTerminatorIsFirstChar(t *testing.T) {
	dir := writeHelpFixture(t, map[string]string{
		"index": "t.hlp\n$\n",
		// A '#'-first line ends the entry; `#foo` would be a terminator in C.
		"t.hlp": "TOPIC\r\nbody\r\n#\r\nNEXT\r\nbody2\r\n#\r\n$\r\n",
	})
	table, err := LoadHelpFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(table) != 2 {
		t.Fatalf("entries = %d, want 2 (each record terminated by a #-first line)", len(table))
	}
}

// TestLoadHelpFilesSortedByKeyword verifies the table is sorted by
// case-insensitive keyword — C qsort/hsort order, which do_help's binary search
// depends on.
func TestLoadHelpFilesSortedByKeyword(t *testing.T) {
	dir := writeHelpFixture(t, map[string]string{
		"index": "t.hlp\n$\n",
		// Deliberately out-of-order keywords; loader must sort.
		"t.hlp": "ZEBRA\r\nz\r\n#\r\nALPHA\r\na\r\n#\r\nmango\r\nm\r\n#\r\n$\r\n",
	})
	table, err := LoadHelpFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i < len(table); i++ {
		if strings.ToLower(table[i-1].Keyword) > strings.ToLower(table[i].Keyword) {
			t.Errorf("table not sorted: %q before %q", table[i-1].Keyword, table[i].Keyword)
		}
	}
	if len(table) >= 3 && table[0].Keyword != "alpha" {
		t.Errorf("first keyword = %q, want alpha (case-insensitive sort)", table[0].Keyword)
	}
}

// --- strnCmp parity (utils.c:125) ------------------------------------------

func TestStrnCmpParity(t *testing.T) {
	cases := []struct {
		name string
		a, b string
		n    int
		want int // -1, 0, +1; we only care about sign and zero
	}{
		{"prefix match", "s", "say", 1, 0},
		{"prefix match 2", "sa", "say", 2, 0},
		{"equal", "say", "say", 3, 0},
		{"equal n over", "say", "say", 10, 0},
		{"longer-than-keyword rejects", "sayings", "say", 7, 1}, // C: arg1 longer → nonzero
		{"differ at byte", "say", "save", 3, 1},                 // 'y' > 'v'
		{"differ at byte neg", "save", "say", 3, -1},
		{"case-insensitive", "SAY", "say", 3, 0},
		{"case-insensitive prefix", "S", "say", 1, 0},
		{"empty arg matches", "", "say", 0, 0}, // n=0 → loop never runs → 0
	}
	for _, c := range cases {
		got := strnCmp(c.a, c.b, c.n)
		// Normalize to sign.
		sign := func(x int) int {
			if x == 0 {
				return 0
			}
			if x < 0 {
				return -1
			}
			return 1
		}
		if sign(got) != c.want {
			t.Errorf("strnCmp(%q,%q,%d) = %d, want sign %d", c.a, c.b, c.n, got, c.want)
		}
	}
}

// --- SearchHelp prefix + first-match ---------------------------------------

func TestSearchHelpPrefixAndFirstMatch(t *testing.T) {
	// A hand-built sorted table where two keywords share the prefix "s".
	// In a sorted table, "sanctuary" precedes "say" precedes "sleep". C's
	// first-match backtrack must return the EARLIEST one for prefix "s".
	table := []HelpEntry{
		{Keyword: "move", Entry: "m\r\n"},
		{Keyword: "sanctuary", Entry: "sanc\r\n"},
		{Keyword: "say", Entry: "say\r\n"},
		{Keyword: "sleep", Entry: "sleep\r\n"},
	}
	if got := SearchHelp(table, "s"); got == nil || got.Keyword != "sanctuary" {
		t.Errorf("SearchHelp(s) = %v; want sanctuary (first match in sorted order)", got)
	}
	if got := SearchHelp(table, "sa"); got == nil || got.Keyword != "sanctuary" {
		t.Errorf("SearchHelp(sa) = %v; want sanctuary", got)
	}
	if got := SearchHelp(table, "say"); got == nil || got.Keyword != "say" {
		t.Errorf("SearchHelp(say) = %v; want say (exact)", got)
	}
	if got := SearchHelp(table, "sl"); got == nil || got.Keyword != "sleep" {
		t.Errorf("SearchHelp(sl) = %v; want sleep", got)
	}
	if got := SearchHelp(table, "zzzx"); got != nil {
		t.Errorf("SearchHelp(zzzx) = %v; want nil (miss)", got)
	}
}

// TestSearchHelpRequiresSortedTable documents the precondition: binary search
// assumes a sorted table (LoadHelpFiles / sortHelpTable enforce it). This is a
// contract reminder, not a behavior we defend against at runtime.
func TestSearchHelpEmptyAndSingle(t *testing.T) {
	if got := SearchHelp(nil, "x"); got != nil {
		t.Error("SearchHelp(nil) should be nil")
	}
	if got := SearchHelp([]HelpEntry{}, "x"); got != nil {
		t.Error("SearchHelp(empty) should be nil")
	}
	if got := SearchHelp([]HelpEntry{{Keyword: "say", Entry: "s\r\n"}}, ""); got != nil {
		t.Error("SearchHelp(empty-arg) should be nil")
	}
}

// --- Real-data parity (the in-repo lib/text/help files) --------------------

// TestLoadHelpFilesRealDataParity loads the actual in-repo help files (the 2010
// data) and checks structural invariants that must hold for do_help to behave
// like C. Run from the repo root (the server's CWD convention).
func TestLoadHelpFilesRealDataParity(t *testing.T) {
	table, err := LoadHelpFiles("lib/text/help")
	if err != nil {
		t.Skipf("lib/text/help not available in this test CWD: %v", err)
	}
	if len(table) == 0 {
		t.Fatal("loaded help table is empty")
	}
	// Sorted invariant.
	for i := 1; i < len(table); i++ {
		if strings.ToLower(table[i-1].Keyword) > strings.ToLower(table[i].Keyword) {
			t.Fatalf("real help table not sorted at %d: %q > %q", i, table[i-1].Keyword, table[i].Keyword)
		}
	}
	// The multi-keyword quoted record resolves under the quoted form.
	if SearchHelp(table, "first aid") == nil {
		t.Error(`real table missing "first aid" keyword — one_word quoted split regressed`)
	}
	// say is present and its entry retains the keyword line as first line.
	say := SearchHelp(table, "say")
	if say == nil {
		t.Fatal(`real table missing "say" keyword`)
	}
	if !strings.HasPrefix(say.Entry, "SAY") {
		t.Errorf("say entry first line = %q; want keyword line retained", say.Entry)
	}
}
