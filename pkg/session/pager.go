package session

import (
	"encoding/json"
	"strings"
)

// Output pager — port of the Buselli/CircleMUD pager (src/modify.c:346-527,
// src/comm.c:617-618,1028-1056). Cite: rules R1, R4.
//
// While a session is paging, every input line routes to handlePagerInput
// instead of the command interpreter (C: comm.c:617 `else if (d->showstr_count)
// show_string(d, comm)`). Navigation: RETURN = next page, q/Q = quit, r/R =
// refresh (re-send current), b/B = back one page, a number = jump to that page.

// PAGE_LENGTH and PAGE_WIDTH mirror src/comm.h:63-64.
const (
	pageLength = 22
	pageWidth  = 78
)

// nextPage returns the byte index into s at which the NEXT page begins, or
// len(s) if this is the last page. It is a byte-faithful port of next_page
// (modify.c:355-397): col/line count from 1; ANSI color escapes (\x1B…m) count
// zero columns; '\r' resets col to 1; '\n' resets col to 1 and advances line;
// any other visible byte post-increments col and, when col exceeds pageWidth,
// wraps to col=1 and advances line. A new page starts once line > pageLength.
func nextPage(s []byte) int {
	col, line := 1, 1
	specCode := false

	for i := 0; i < len(s); i++ {
		// If we're at the start of the next page, return this index.
		if line > pageLength {
			return i
		}
		c := s[i]
		switch {
		case c == '\x1b' && !specCode:
			// Beginning of an ANSI color code block.
			specCode = true
		case c == 'm' && specCode:
			// End of an ANSI color code block.
			specCode = false
		case specCode:
			// Inside an ANSI escape: counts zero columns (no-op).
		case c == '\r':
			col = 1
		case c == '\n':
			col = 1
			line++
		default:
			// Faithful port of C's `else if (col++ > PAGE_WIDTH) { col=1; line++; }`.
			// The post-increment is a side effect of the condition test: it runs
			// whether or not the branch is taken. When the branch IS taken it
			// sets col=1, overwriting the increment. Net: a line holds visible
			// columns 1..78 (78 chars); the 79th visible char triggers the wrap.
			over := col > pageWidth // evaluate with current col
			col++                   // post-increment always happens
			if over {
				col = 1
				line++
			}
		}
	}
	// End of string: this is the last page.
	return len(s)
}

// paginate splits text into page byte-slices, mirroring C's showstr_vector
// construction (modify.c paginate_string + count_pages). Each returned slice is
// one page's worth of bytes; the page boundaries are exactly where next_page
// lands them.
func paginate(text string) [][]byte {
	s := []byte(text)
	var pages [][]byte
	for start := 0; start < len(s); {
		end := nextPage(s[start:])
		pages = append(pages, s[start:start+end])
		// next_page returns the offset into the sub-slice; a zero-length page
		// would loop forever, so guard against it (cannot happen for non-empty
		// input because the first byte always advances past page 0, but be
		// safe).
		if end == 0 {
			break
		}
		start += end
	}
	// A non-empty string always yields at least one page (count_pages starts at
	// 1). An empty string yields zero pages here; callers handle that case.
	if len(pages) == 0 {
		pages = [][]byte{{}}
	}
	return pages
}

// PageString begins paging text through s, mirroring C's page_string
// (modify.c:430-449).
//
//   - Empty text is sent as "" (C:435-438).
//   - Structured-data clients (agents / wantsStructuredData) receive the whole
//     text as one event and never enter pager mode — the pager is a
//     telnet/plain-text surface behavior; machines consume full JSON, not a
//     22-line terminal.
//   - Otherwise, if the text is one page or less, it is sent whole with no
//     prompt and the session does not enter pager mode (C: even for 1 page,
//     show_string sees page+1>=count and frees without prompting).
//   - Otherwise the pages are stored, pager mode is entered, page 0 is sent,
//     and the pager prompt is printed.
func PageString(s *Session, text string) {
	if text == "" {
		s.Send("")
		return
	}
	// Structured-data / agent clients: whole text, no pager. (C has no analog —
	// every descriptor is a terminal — but the brief requires gating these
	// clients out, and they need the full payload.)
	if s.wantsStructuredData || s.isAgent {
		s.Send(text)
		return
	}

	pages := paginate(text)
	if len(pages) <= 1 {
		// At most one page: send whole, no prompt, no pager mode.
		s.Send(text)
		return
	}

	s.pagerPages = pages
	s.pagerCount = len(pages)
	s.pagerPage = 0
	// C page_string → show_string(d, ""): display page 0 (RETURN/empty falls
	// through to the display path).
	s.displayPage()
}

// pagerPrompt returns the C pager prompt (comm.c:1042-1056), with current page
// (1-based) and total. ANSI: cyan brackets/labels, red emphasis, per the C
// CCCYN/CCRED/CCNRM constants.
func pagerPrompt(current, total int) string {
	const (
		cyan = "\x1b[36m"
		red  = "\x1b[31m"
		norm = "\x1b[0m"
	)
	// C: "\r%s[ %sReturn%s to continue, (%sq%s)uit, (%sr%s)efresh, (%sb%s)ack,
	//     or page number (%s%d%s/%s%d%s) ]%s"
	// where the %d's are showstr_page / showstr_count. show_string increments
	// showstr_page AFTER sending, so make_prompt (which runs after) shows the
	// page just displayed, 1-based.
	// C make_prompt (comm.c:1044-1046): leading \r only, NO trailing newline —
	// the player types on the prompt line. The blank line before the prompt
	// comes from the output-flush cycle, which the caller supplies.
	return cyan + "[ " + red + "Return" + cyan +
		" to continue, (" + red + "q" + cyan + ")uit, (" +
		red + "r" + cyan + ")efresh, (" + red + "b" + cyan +
		")ack, or page number (" + red + pageItoa(current) + cyan + "/" +
		red + pageItoa(total) + cyan + ") ]" + norm
}

func pageItoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

// displayPage shows the page at s.pagerPage (0-based "next to display" index),
// matching C show_string's display tail (modify.c:506-526). If it is the last
// page, the pager auto-exits (C frees the vector after the last page — no q
// required). Otherwise pagerPage is advanced and the prompt is printed for the
// page just shown (1-based = the old pagerPage + 1).
func (s *Session) displayPage() {
	idx := s.pagerPage
	shown := idx + 1 // 1-based number of the page being displayed
	s.sendText(string(s.pagerPages[idx]))
	if idx+1 >= s.pagerCount {
		// Last page: auto-exit the pager (C:506-516).
		s.exitPager()
		return
	}
	s.pagerPage = idx + 1
	// Line separation before the prompt comes from the message framing; the
	// prompt carries neither C's leading \r (a same-line cursor return the Go
	// write path doesn't need — oracle-verified parity without it) nor any
	// trailing newline (the player types on the prompt line).
	s.sendText(pagerPrompt(shown, s.pagerCount))
}

// exitPager clears all pager state (C: FREE(showstr_vector); showstr_count = 0).
func (s *Session) exitPager() {
	s.pagerPages = nil
	s.pagerCount = 0
	s.pagerPage = 0
}

// IsPaging reports whether the session is in pager mode. Mirrors C's
// showstr_count != 0 check (comm.c:617).
func (s *Session) IsPaging() bool {
	return s.pagerCount > 0
}

// handlePagerInputMsg processes one input line while paging — a direct port of
// show_string (modify.c:454-527). RETURN (empty) advances; q quits; r refreshes
// (re-sends current); b backs up; a number jumps. Anything else prints the
// exact "Valid commands while paging" line and does not advance. Routed from
// both transports via the pager_input message type.
func (s *Session) handlePagerInputMsg(data json.RawMessage) error {
	var input CharInputData
	if err := json.Unmarshal(data, &input); err != nil {
		return err
	}
	s.navigatePager(input.Choice)
	return nil
}

// navigatePager applies one pager navigation command (the raw input line). It is
// split from handlePagerInputMsg so unit tests can drive it with a plain string.
func (s *Session) navigatePager(line string) {
	// C: any_one_arg(input, buf) — take the first whitespace-delimited word.
	// For paging the first word is all that matters (q/r/b or a number).
	buf := strings.TrimSpace(line)
	if sp := strings.IndexAny(buf, " \t"); sp >= 0 {
		buf = buf[:sp]
	}
	first := byte(0)
	if len(buf) > 0 {
		first = buf[0]
	}
	firstLower := first
	if firstLower >= 'A' && firstLower <= 'Z' {
		firstLower += 'a' - 'A'
	}

	switch {
	case firstLower == 'q':
		// Q is for quit — free and exit (C:462-472).
		s.exitPager()
		return
	case firstLower == 'r':
		// R is for refresh — back up one so displayPage re-shows the page just
		// displayed. pagerPage is the next-to-display index, so the page just
		// shown is pagerPage-1. C: showstr_page = MAX(0, showstr_page-1).
		if s.pagerPage > 0 {
			s.pagerPage--
		}
	case firstLower == 'b':
		// B is for back — display two pages back. C: showstr_page = MAX(0,
		// showstr_page-2). (pagerPage already points past the last-shown page,
		// so -2 lands on the page before the one just shown.)
		s.pagerPage -= 2
		if s.pagerPage < 0 {
			s.pagerPage = 0
		}
	case first >= '0' && first <= '9':
		// Jump to a page number (1-based input). C: showstr_page = MAX(0,
		// MIN(atoi-1, count-1)). Set pagerPage to the index to display.
		n := parsePageNum(buf) - 1
		if n < 0 {
			n = 0
		}
		if n > s.pagerCount-1 {
			n = s.pagerCount - 1
		}
		s.pagerPage = n
	case first != 0:
		// Non-empty, unrecognized input — print the exact line and do NOT
		// advance or display (C:497-502 returns early). C's prompt cycle
		// (make_prompt, comm.c:647) still reprints the pager prompt after the
		// message: emit it without redisplaying the page. pagerPage is the
		// next-to-display index, so the page on screen is pagerPage (1-based).
		s.sendText("Valid commands while paging are RETURN, Q, R, B, or a numeric value.\r\n")
		s.sendText(pagerPrompt(s.pagerPage, s.pagerCount))
		return
	}
	// Empty input (RETURN) and R/B/number all fall through to display the page
	// at pagerPage (the next-to-display index), matching C's display tail.
	s.displayPage()
}

// parsePageNum parses a leading integer (C: atoi). Returns 0 on no digits.
func parsePageNum(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	return n
}
