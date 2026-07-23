package session

import (
	"strings"
	"testing"
	"unicode"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// --- next_page parity (the core fidelity tests) -----------------------------

// renderLineCount mirrors C next_page's column/line accounting to count how
// many rendered lines a string occupies (for assertions), independent of the
// production nextPage — a parallel re-implementation as a test oracle.
func renderLineCount(s string) int {
	const pageW = 78
	col, line := 1, 1
	spec := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\x1b' && !spec:
			spec = true
		case c == 'm' && spec:
			spec = false
		case spec:
			// inside ANSI: no column
		case c == '\r':
			col = 1
		case c == '\n':
			col = 1
			line++
		default:
			// Faithful C: col++ > PAGE_WIDTH { col=1; line++ }.
			over := col > pageW
			col++
			if over {
				col = 1
				line++
			}
		}
	}
	return line
}

func TestNextPageSplitsAtPageLength(t *testing.T) {
	// 23 short lines → 2 pages, page 1 boundary lands after line 22.
	var sb strings.Builder
	for i := 1; i <= 23; i++ {
		sb.WriteString("line\r\n")
	}
	text := sb.String()
	pages := paginate(text)
	if len(pages) != 2 {
		t.Fatalf("23-line string → %d pages, want 2", len(pages))
	}
	// Page 1 must contain exactly 22 newlines (lines 1–22).
	if got := strings.Count(string(pages[0]), "\n"); got != 22 {
		t.Errorf("page 1 has %d newlines, want 22 (split at line 22)", got)
	}
	// Page 2 must contain the remaining 1 newline.
	if got := strings.Count(string(pages[1]), "\n"); got != 1 {
		t.Errorf("page 2 has %d newlines, want 1", got)
	}
}

func TestNextPageExactlyPageLengthIsOnePage(t *testing.T) {
	// Exactly 22 lines → one page (whole, no pager mode). This is the boundary
	// that page_string relies on to send short output whole.
	var sb strings.Builder
	for i := 1; i <= 22; i++ {
		sb.WriteString("line\r\n")
	}
	if got := len(paginate(sb.String())); got != 1 {
		t.Errorf("22-line string → %d pages, want 1 (PAGE_LENGTH boundary)", got)
	}
}

func TestNextPageAnsiCodesDoNotCountTowardColumns(t *testing.T) {
	// ANSI escapes must not count toward columns. Compare two lines that differ
	// ONLY in the presence of ANSI codes: with and without color, the same
	// visible width must render the same number of lines.
	visible := strings.Repeat("x", 80) + "\r\n"                     // 80 visible cols
	colored := "\x1b[31m" + strings.Repeat("x", 80) + "\x1b[0m\r\n" // same 80 visible cols, ANSI-wrapped
	plain := renderLineCount(visible)
	withANSI := renderLineCount(colored)
	if plain != withANSI {
		t.Errorf("ANSI changed line count: plain=%d, with ANSI=%d (ANSI must not count toward columns)", plain, withANSI)
	}
	// 80 visible cols wraps at PAGE_WIDTH(78): cols 1..78 on line 1, 79..80 wrap
	// to line 2, then the trailing \r\n adds a 3rd rendered line.
	if plain != 3 {
		t.Errorf("80 visible cols + \\r\\n → %d rendered lines, want 3 (wrap + trailing newline)", plain)
	}
	// A single ANSI-wrapped line that does not cross PAGE_LENGTH stays one page.
	onePage := "\x1b[31m" + strings.Repeat("x", 100) + "\x1b[0m\r\n"
	if got := len(paginate(onePage)); got != 1 {
		t.Errorf("single 100-visible-col ANSI line → %d pages, want 1", got)
	}
}

func TestNextPageLongUnbrokenLineWraps(t *testing.T) {
	// A long line with no newlines wraps at PAGE_WIDTH and the wraps count as
	// extra rendered lines, eventually crossing PAGE_LENGTH into a second page.
	longLine := strings.Repeat("x", 79*23+50)
	pages := paginate(longLine)
	if len(pages) != 2 {
		t.Fatalf("long unbroken line → %d pages, want 2 (wrap must count as lines)", len(pages))
	}
	// nextPage defines the boundary; the page-1 byte length must equal nextPage's
	// return on the whole string. (We don't assert on a hand-computed line count:
	// the Buselli pager checks `line > PAGE_LENGTH` at the top of the loop, after
	// the wrap increment, so a page can carry up to PAGE_LENGTH+1 line
	// transitions — page 1 of a wrapping doc can render more than 22 lines by
	// design. This is faithful to C, not a bug.)
	if len(pages[0]) != nextPage([]byte(longLine)) {
		t.Errorf("page 1 length %d != nextPage boundary %d", len(pages[0]), nextPage([]byte(longLine)))
	}
	// Sanity: the boundary is well past one screen of wrapped text.
	if len(pages[0]) < 1000 {
		t.Errorf("page 1 length %d implausibly small for a wrapping doc", len(pages[0]))
	}
}

// --- PageString entry point + WS gating -------------------------------------

func TestPageStringEmptySendsWholeNoMode(t *testing.T) {
	m := makeTestManager(t)
	s := makeCommandTestSession(t, m, "pager", 1, 1001)
	PageString(s, "")
	if s.IsPaging() {
		t.Error("empty PageString entered pager mode; want no mode")
	}
}

func TestPageStringShortSendsWholeNoMode(t *testing.T) {
	m := makeTestManager(t)
	s := makeCommandTestSession(t, m, "pager", 1, 1001)
	PageString(s, "just a few lines\r\nsecond\r\n")
	if s.IsPaging() {
		t.Error("short PageString entered pager mode; want no mode (≤1 page)")
	}
	_ = drainSendChannel(t, s) // whole text was sent
}

func TestPageStringLongEntersPagerMode(t *testing.T) {
	m := makeTestManager(t)
	s := makeCommandTestSession(t, m, "pager", 1, 1001)
	var sb strings.Builder
	for i := 1; i <= 30; i++ {
		sb.WriteString("level line\r\n")
	}
	PageString(s, sb.String())
	if !s.IsPaging() {
		t.Fatal("30-line PageString did not enter pager mode; want IsPaging()=true")
	}
	if s.pagerCount != 2 {
		t.Errorf("pagerCount = %d, want 2", s.pagerCount)
	}
	// After displaying page 0, pagerPage is the next-to-display index (1) —
	// matches C's showstr_page, which increments after each display.
	if s.pagerPage != 1 {
		t.Errorf("pagerPage = %d, want 1 (advanced past page 1)", s.pagerPage)
	}
	out := drainSendChannel(t, s)
	// Page 1 + the pager prompt should have been sent. drainSendChannel returns
	// raw JSON bytes (ANSI escapes appear as \u001b and split the prompt's color
	// tokens), so match contiguous visible substrings that have no ANSI between
	// them: "to continue" and "page number".
	if !strings.Contains(out, "to continue") {
		t.Errorf("page 1 output missing pager prompt; got %q", out)
	}
	if !strings.Contains(out, "page number") {
		t.Errorf("page 1 output missing prompt page-number text; got %q", out)
	}
}

func TestPageStringStructuredDataClientGetsWholeTextNoPager(t *testing.T) {
	// The brief's WS-gating decision: structured-data / agent clients receive
	// the whole text as one event and never enter pager mode.
	m := makeTestManager(t)
	s := makeCommandTestSession(t, m, "agent", 1, 1001)
	s.wantsStructuredData = true

	var sb strings.Builder
	for i := 1; i <= 50; i++ { // far more than one page
		sb.WriteString("agent line\r\n")
	}
	text := sb.String()
	PageString(s, text)

	if s.IsPaging() {
		t.Error("structured-data client entered pager mode; want whole text, no pager")
	}
	out := drainSendChannel(t, s)
	// Must contain the FULL text (all 50 lines), not be truncated to a page.
	fullLines := strings.Count(text, "agent line")
	gotLines := strings.Count(out, "agent line")
	if gotLines != fullLines {
		t.Errorf("structured-data client got %d/%d lines; want the whole text", gotLines, fullLines)
	}
}

// --- Navigator (show_string parity) -----------------------------------------

// pageTag returns a marker string identifying page p (1-based) in the pager
// document built by newPagerTestSession, so tests can assert which page was
// displayed by scanning the drained output.
const pageMarkerPrefix = "PAGEMARK-"

func pageTag(p int) string { return pageMarkerPrefix + pageItoa(p) }

// newPagerTestSession sets up a session mid-pager on a known 4-page document.
// Each page's first line carries a unique marker (PAGEMARK-1 … PAGEMARK-4) so
// navigator tests can assert which page was displayed by scanning output. The
// page-1 display + prompt from PageString are drained before returning, so the
// caller starts "just after page 1 was shown" (pagerPage = next-to-display = 1).
func newPagerTestSession(t *testing.T) *Session {
	t.Helper()
	m := makeTestManager(t)
	s := makeCommandTestSession(t, m, "pager", 1, 1001)
	// 4 pages of 22 lines each (88 lines total → pages 1..4). Each page begins
	// with a unique marker so we can tell which page was displayed.
	var sb strings.Builder
	for page := 1; page <= 4; page++ {
		for line := 1; line <= 22; line++ {
			if line == 1 {
				sb.WriteString(pageTag(page) + "\r\n")
				continue
			}
			sb.WriteString("filler\r\n")
		}
	}
	PageString(s, sb.String())
	if !s.IsPaging() {
		t.Fatal("setup failed: did not enter pager mode")
	}
	if s.pagerCount != 4 {
		t.Fatalf("setup: pagerCount = %d, want 4", s.pagerCount)
	}
	_ = drainSendChannel(t, s) // discard page 1 + prompt
	return s
}

func TestPagerReturnAdvances(t *testing.T) {
	s := newPagerTestSession(t) // page 1 was shown; next-to-display = 1 (page 2)
	s.navigatePager("")         // RETURN → display page 2
	out := drainSendChannel(t, s)
	if !strings.Contains(out, pageTag(2)) {
		t.Errorf("RETURN displayed page %q; want page 2 (advance)", lastDisplayedPage(out))
	}
}

func TestPagerBackFromPage2ReturnsToPage1(t *testing.T) {
	s := newPagerTestSession(t)
	s.navigatePager("") // → page 2
	_ = drainSendChannel(t, s)
	s.navigatePager("b") // back → page 1
	out := drainSendChannel(t, s)
	if !strings.Contains(out, pageTag(1)) {
		t.Errorf("B from page 2 displayed %q; want page 1 (back)", lastDisplayedPage(out))
	}
}

func TestPagerBackFromPage1ClampsToPage1(t *testing.T) {
	s := newPagerTestSession(t) // showing page 1
	s.navigatePager("b")        // back from page 1 → clamp, re-show page 1
	out := drainSendChannel(t, s)
	if !strings.Contains(out, pageTag(1)) {
		t.Errorf("B from page 1 displayed %q; want page 1 (clamp, no earlier page)", lastDisplayedPage(out))
	}
}

func TestPagerRefreshResendsSamePage(t *testing.T) {
	s := newPagerTestSession(t)
	s.navigatePager("") // → page 2
	_ = drainSendChannel(t, s)
	s.navigatePager("r") // refresh → re-show page 2
	out := drainSendChannel(t, s)
	if !strings.Contains(out, pageTag(2)) {
		t.Errorf("R displayed %q; want page 2 (refresh re-sends current)", lastDisplayedPage(out))
	}
}

func TestPagerNumberJump(t *testing.T) {
	s := newPagerTestSession(t)
	s.navigatePager("3") // jump to page 3 (1-based)
	out := drainSendChannel(t, s)
	if !strings.Contains(out, pageTag(3)) {
		t.Errorf("'3' displayed %q; want page 3", lastDisplayedPage(out))
	}
}

func TestPagerNumberJumpOutOfRangeClampsToLast(t *testing.T) {
	s := newPagerTestSession(t) // 4 pages
	s.navigatePager("99")       // far over range
	out := drainSendChannel(t, s)
	if !strings.Contains(out, pageTag(4)) {
		t.Errorf("'99' displayed %q; want page 4 (clamp to last)", lastDisplayedPage(out))
	}
}

func TestPagerNumberJumpZeroClampsToFirst(t *testing.T) {
	s := newPagerTestSession(t)
	s.navigatePager("0") // below range → clamp to first
	out := drainSendChannel(t, s)
	if !strings.Contains(out, pageTag(1)) {
		t.Errorf("'0' displayed %q; want page 1 (clamp to first)", lastDisplayedPage(out))
	}
}

func TestPagerQuitExits(t *testing.T) {
	s := newPagerTestSession(t)
	s.navigatePager("q")
	if s.IsPaging() {
		t.Error("Q did not exit pager mode")
	}
}

func TestPagerCaseInsensitiveQuit(t *testing.T) {
	// C uses LOWER(*buf); uppercase Q must also quit.
	s := newPagerTestSession(t)
	s.navigatePager("Q")
	if s.IsPaging() {
		t.Error("uppercase Q did not exit pager mode")
	}
}

func TestPagerJunkInputPrintsValidLineAndDoesNotAdvance(t *testing.T) {
	s := newPagerTestSession(t) // next-to-display = page 2
	s.navigatePager("xyzzy")
	out := drainSendChannel(t, s)
	// drainSendChannel returns raw JSON bytes, so match the visible message text
	// without the trailing CRLF (which is JSON-escaped).
	const want = "Valid commands while paging are RETURN, Q, R, B, or a numeric value."
	if !strings.Contains(out, want) {
		t.Errorf("junk input output = %q; want it to contain %q", out, want)
	}
	// Junk input must NOT have advanced — the next RETURN should still show page 2.
	s.navigatePager("")
	out = drainSendChannel(t, s)
	if !strings.Contains(out, pageTag(2)) {
		t.Errorf("after junk input, RETURN displayed %q; want page 2 (junk must not advance)", lastDisplayedPage(out))
	}
}

func TestPagerAutoExitsAfterLastPage(t *testing.T) {
	// C: displaying the last page frees the vector and exits — no q required.
	s := newPagerTestSession(t)
	s.navigatePager("4") // jump to the last page
	_ = drainSendChannel(t, s)
	if s.IsPaging() {
		t.Error("displaying the last page did not auto-exit pager; C auto-exits after the last page")
	}
}

func TestPagerInputIsNotACommand(t *testing.T) {
	// While paging, a command-shaped word ("look") is pager input, not a
	// command. It must not be dispatched; it falls into the junk branch and
	// prints the Valid-commands line.
	s := newPagerTestSession(t)
	s.navigatePager("look")
	out := drainSendChannel(t, s)
	if !strings.Contains(out, "Valid commands while paging") {
		t.Errorf("'look' while paging → %q; want the Valid-commands line (not command dispatch)", out)
	}
}

// lastDisplayedPage extracts the highest-numbered PAGEMARK-N present in out, to
// report which page was displayed in a failure message.
func lastDisplayedPage(out string) string {
	best := "(none)"
	for p := 1; p <= 9; p++ {
		if strings.Contains(out, pageTag(p)) {
			best = pageTag(p)
		}
	}
	return best
}

// --- levels end-to-end ------------------------------------------------------

func TestLevelsEntersPagerForLongOutput(t *testing.T) {
	m := makeTestManager(t)
	s := makeCommandTestSession(t, m, "lvl", 1, 1001)
	s.player.SetPosition(combat.PosStanding)
	if err := ExecuteCommand(s, "levels", nil); err != nil {
		t.Fatalf("ExecuteCommand(levels): %v", err)
	}
	if !s.IsPaging() {
		t.Error("levels (>22 lines) did not enter pager mode; want IsPaging()=true")
	}
	out := drainSendChannel(t, s)
	// drainSendChannel returns raw JSON; ANSI escapes split the prompt's color
	// tokens, so match a contiguous visible substring ("to continue").
	if !strings.Contains(out, "to continue") {
		t.Errorf("levels output missing pager prompt; got %q", out)
	}
}

// unicode import guard: keep the linter happy if a future assertion needs it.
var _ = unicode.IsLetter

// keep combat import meaningful for the position constant above.
var _ = combat.PosStanding
