package session

import (
	"strings"
	"testing"
)

// TestCTruncateInputLine pins C's process_input over-long line rule
// (src/comm.c:1969-1999, DP-1528): a 255-kept-byte budget ('$' costs 2,
// backspace restores), and when the budget runs out with input remaining C
// first sends "Line too long.  Truncated to:\r\n<prefix>\r\n" -- prefix with
// '$' doubled, exactly as C's buffer holds it -- then runs the prefix.
// Fails if the cap reverts to a silent cut (R5h).
func TestCTruncateInputLine(t *testing.T) {
	t.Run("300 chars cut to 255 with notice", func(t *testing.T) {
		run, notice := cTruncateInputLine(strings.Repeat("x", 300))
		if run != strings.Repeat("x", 255) {
			t.Fatalf("run length = %d, want 255", len(run))
		}
		want := "Line too long.  Truncated to:\r\n" + strings.Repeat("x", 255) + "\r\n"
		if notice != want {
			t.Fatalf("notice = %q, want %q", notice, want)
		}
	})
	t.Run("exactly 255 fits", func(t *testing.T) {
		run, notice := cTruncateInputLine(strings.Repeat("x", 255))
		if run != strings.Repeat("x", 255) || notice != "" {
			t.Fatalf("255-char line = (%q len %d, %q), want no notice", run, len(run), notice)
		}
	})
	t.Run("256th char triggers", func(t *testing.T) {
		run, notice := cTruncateInputLine(strings.Repeat("x", 256))
		if len(run) != 255 || notice == "" {
			t.Fatalf("256-char line = (len %d, notice %q), want cut + notice", len(run), notice)
		}
	})
	t.Run("dollar costs two", func(t *testing.T) {
		// 254 x's + '$' + 'y': the '$' exhausts the budget, 'y' remains.
		line := strings.Repeat("x", 254) + "$y"
		run, notice := cTruncateInputLine(line)
		if run != strings.Repeat("x", 254)+"$" {
			t.Fatalf("run = %q, want 254 x's and the dollar", run)
		}
		want := "Line too long.  Truncated to:\r\n" + strings.Repeat("x", 254) + "$$\r\n"
		if notice != want {
			t.Fatalf("notice = %q, want the doubled dollar form %q", notice, want)
		}
	})
	t.Run("backspace restores budget", func(t *testing.T) {
		// 254 x's, backspace, 2 x's: net 255 kept, fits with no notice.
		run, notice := cTruncateInputLine(strings.Repeat("x", 254) + "\bx" + "x")
		if run != strings.Repeat("x", 255) || notice != "" {
			t.Fatalf("backspace line = (len %d, %q), want 255 x's and no notice", len(run), notice)
		}
	})
	t.Run("non printables do not count", func(t *testing.T) {
		// Control bytes inside the line are skipped without spending budget.
		run, notice := cTruncateInputLine("\x01" + strings.Repeat("x", 255))
		if run != strings.Repeat("x", 255) || notice != "" {
			t.Fatalf("wrapped line = (len %d, %q), want 255 x's and no notice", len(run), notice)
		}
		// C's notice condition is (space_left <= 0 && ptr < nl_pos): bytes
		// remaining AFTER the budget empties trigger it even when they are
		// control bytes C would skip.
		_, notice = cTruncateInputLine(strings.Repeat("x", 255) + "\x7f")
		if notice == "" {
			t.Fatal("255 x's plus a trailing byte: want the notice, C fires with input remaining")
		}
	})
}
