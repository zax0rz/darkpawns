package oraclediff

import (
	"regexp"
	"strings"
)

var (
	// C prints "%sStarted: %-20.16s  Last: %-20.16s  Played: %3dh %2dm\r\n"
	// (src/act.wizard.c:2342-2346): the %s carries whatever precedes the line —
	// a color code on colored surfaces, an indent — so the anchor admits ANSI
	// escapes and whitespace before the label but no other text, keeping
	// ordinary prose that merely contains "Started:" unnormalized. Each
	// %-20.16s pads 16 ctime chars with 4 spaces, followed by 2 literal spaces
	// = exactly 6 spaces before "Last:" and before "Played:".
	playerSavedDates = regexp.MustCompile(`^((?:\x1b\[[0-?]*[ -/]*[@-~])*[\t ]*Started: )(?:Sun|Mon|Tue|Wed|Thu|Fri|Sat) (?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec) (?: [1-9]|[12][0-9]|3[01]) (?:[01][0-9]|2[0-3]):[0-5][0-9]( {6}Last: )(?:Sun|Mon|Tue|Wed|Thu|Fri|Sat) (?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec) (?: [1-9]|[12][0-9]|3[01]) (?:[01][0-9]|2[0-3]):[0-5][0-9]( {6}Played:.*)$`)
	// playerSavedDatesMultiline is playerSavedDates for the keep-prompts
	// pipeline, which compares raw captured text without splitting lines:
	// (?m) re-anchors ^ and $ per line and the trailing .* absorbs the \r the
	// mode deliberately preserves ahead of the \n. It exists because
	// show-player-stored runs keep-prompts and diverged on the Started minute
	// whenever the two engines' setups straddled a wall-clock minute
	// (dp-1404-rent-code-claims 2026-10-09; pr1904-dead-code-d 2026-10-10
	// seed 5).
	playerSavedDatesMultiline = regexp.MustCompile(`(?m)^((?:\x1b\[[0-?]*[ -/]*[@-~])*[\t ]*Started: )(?:Sun|Mon|Tue|Wed|Thu|Fri|Sat) (?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec) (?: [1-9]|[12][0-9]|3[01]) (?:[01][0-9]|2[0-3]):[0-5][0-9]( {6}Last: )(?:Sun|Mon|Tue|Wed|Thu|Fri|Sat) (?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec) (?: [1-9]|[12][0-9]|3[01]) (?:[01][0-9]|2[0-3]):[0-5][0-9]( {6}Played:.*)$`)
	ansiEscape                = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]`)
	// C asctime(3)/ctime(3) (src/act.wizard.c:1814-1826, src/mail.c:429, etc.)
	// formats "%.3s %.3s%3d %.2d:%.2d:%.2d %d": single-digit days are
	// space-padded ("Nov  5"), never single-spaced ("Nov 5", PF-151) or
	// zero-padded ("Nov 05").
	wallClock = regexp.MustCompile(`\b(?:Sun|Mon|Tue|Wed|Thu|Fri|Sat) (?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec) (?: [1-9]|[12][0-9]|3[01]) (?:[01][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9] \d{4}\b`)
	// The short ctime form (first 16 chars of asctime, src/act.wizard.c:781-784)
	// stat prints inside the Last Death bracket: a wall-clock minute boundary
	// between the two engines' deaths reddens otherwise-identical stat blocks.
	// Anchored to the stat line so ordinary text quoting a date stays raw, like
	// the Started/Last report anchors above.
	lastDeathStamp = regexp.MustCompile(`(Last Death: \[)(?:Sun|Mon|Tue|Wed|Thu|Fri|Sat) (?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec) (?: [1-9]|[12][0-9]|3[01]) (?:[01][0-9]|2[0-3]):[0-5][0-9](\])`)
	// SCMD_VERSION's immortal line (src/act.informative.c:2165-2175) carries
	// load_compile_time()'s stamp (src/version.c:12-65), which is each engine's
	// own process wall clock at boot — "Month D, YYYY   HH:MM", minute
	// granularity, unpadded day (%d, 1..31). The harness freezes the game
	// calendar (DP_FIXED_TIME) but not process wall time, and the port
	// deliberately reads the real boot clock because C's compile_time is
	// time(0) (R1), so two boots that straddle a minute differ only in this
	// stamp. Masked on the exact C shape only: an abbreviated month, a padded
	// day or a wrong separator is a real format divergence and must still fail.
	compileTimeStamp = regexp.MustCompile(`^Compile Time: (?:January|February|March|April|May|June|July|August|September|October|November|December) (?:[1-9]|[12][0-9]|3[01]), \d{4}   (?:[01][0-9]|2[0-3]):[0-5][0-9]$`)
	vitalsPrompt     = regexp.MustCompile(`\b\d+H\s+\d+M\s+\d+V\s*>`)
	promptOnly       = regexp.MustCompile(`^\s*(?:<PROMPT>|>)\s*$`)
	promptPrefix     = regexp.MustCompile(`^> ?`)
	statsLine        = regexp.MustCompile(`^\s*(?:Str:.*Dex:.*Int:.*|Wis:.*Con:.*Cha:.*)\s*$`)
)

// Normalize applies the Tier-1 rules in this deliberate order. Keep this as
// the single rule list: order matters, and each rule documents why it exists.
func Normalize(raw string) string {
	return normalize(raw, false)
}

// NormalizeKeepANSI runs the same Tier-1 pipeline with rule 1 disabled, so the
// comparison certifies raw ANSI bytes. This is the proof mode for surfaces
// whose C colors are player-facing law (OLC's get_char_cols menus): the
// ordinary mode strips ANSI and therefore cannot see a missing-color port.
// Colored prompts and vitals masks expect rule-1-stripped text, so scenarios
// using this mode must stay inside the colored surface they certify.
func NormalizeKeepANSI(raw string) string {
	return normalize(raw, true)
}

// NormalizeKeepPrompts preserves framing and colors, masking only host-clock
// metadata approved for identical comparison in both engines: the users
// Login@ field and the durable-player Started/Last date line printed by
// show player (act.wizard.c:2344). TCPConn has already removed telnet IAC
// negotiation. Focused prompt scenarios use this mode because ordinary
// normalization erases prompt framing.
func NormalizeKeepPrompts(raw string) string {
	return normalizeUsersLoginAt(raw)
}

// TrailingSimplePromptFrame isolates the line breaks and default "> " prompt
// after an entry look. The no-settle fixture captures the first CON_PLAYING
// flush before a later clock pulse can hide a missing entry prompt.
func TrailingSimplePromptFrame(raw string) string {
	i := len(raw)
	if strings.HasSuffix(raw, "> ") {
		i -= 2
	}
	for i > 0 && (raw[i-1] == '\r' || raw[i-1] == '\n') {
		i--
	}
	return raw[i:]
}

func normalize(raw string, keepANSI bool) string {
	// 1. Strip ANSI CSI escapes: color capability is transport presentation, not game text.
	if !keepANSI {
		raw = ansiEscape.ReplaceAllString(raw, "")
	}

	// 2. Canonicalize CRLF/LFCR/CR: both telnet stacks may frame lines differently.
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\n\r", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")
	lines := strings.Split(raw, "\n")

	// 3. Mask approved host-clock metadata in C's exact shapes and normalize
	// standalone telnet prompts. The C and Go transports repaint prompts at
	// different times around identical asynchronous game text, so prompt-only
	// lines are framing, not game output.
	for i := range lines {
		// ctime(3) values in durable-player/admin output are host-clock
		// metadata, not game bytes. The deterministic harness freezes the
		// game clock but intentionally does not freeze process wall time.
		lines[i] = wallClock.ReplaceAllString(lines[i], "<WALL_CLOCK>")
		lines[i] = lastDeathStamp.ReplaceAllString(lines[i], "${1}<WALL_CLOCK>${2}")
		lines[i] = compileTimeStamp.ReplaceAllString(lines[i], "Compile Time: <COMPILE_TIME>")
		lines[i] = playerSavedDates.ReplaceAllString(lines[i], "${1}<WALL_CLOCK>${2}<WALL_CLOCK>${3}")
		lines[i] = vitalsPrompt.ReplaceAllString(lines[i], "<PROMPT>")
		lines[i] = promptPrefix.ReplaceAllString(lines[i], "")
	}

	lines = strings.Split(normalizeUsersLoginAt(strings.Join(lines, "\n")), "\n")

	// 4. Mask the rolled stat block: adjective values derive from unmatched Tier-1 PRNG streams.
	maskedStats := false
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		if statsLine.MatchString(line) {
			if !maskedStats {
				filtered = append(filtered, "<ROLLED_STATS>")
				maskedStats = true
			}
			continue
		}
		maskedStats = false
		filtered = append(filtered, line)
	}

	// 5. Canonicalize prompt framing. Ordinary comparisons keep prompt-only
	// lines as <PROMPT> tokens: each framing run is canonicalized so
	// repaint-count differences collapse while the blank lines before a run's
	// first prompt — where the real divergences live (#1880, PF-063) — stay
	// comparable; ReconcilePromptFrames then pairs the two sides' block-edge
	// frames in the harness. Keep-ANSI keeps its prior contract (PR #1927
	// review): that mode certifies raw color bytes inside surfaces whose own
	// menus own the prompt, so prompt-only lines stay dropped there exactly as
	// before adoption.
	lines = filtered[:0]
	if keepANSI {
		for _, line := range filtered {
			if !promptOnly.MatchString(line) {
				lines = append(lines, line)
			}
		}
	} else {
		lines = canonicalizePromptRuns(filtered)
	}

	// Preserve internal blank lines but remove meaningless leading/trailing transcript padding.
	return strings.Trim(strings.Join(lines, "\n"), "\n") + "\n"
}

// promptFramingLine reports whether a normalized line is blank or a standalone
// prompt token — the two shapes repaint framing is made of.
func promptFramingLine(line string) bool {
	return line == "" || promptOnly.MatchString(line)
}

// canonicalizePromptRuns collapses each maximal run of blank/prompt lines that
// contains a prompt: the blank lines before the run's first prompt are kept
// (their count is the certified message-terminator bytes), and everything from
// that first prompt to the run's end — later prompts and the blanks between
// them, i.e. repaint framing — becomes a single <PROMPT> token. Blank runs
// with no prompt are message-internal bytes and pass through unchanged.
func canonicalizePromptRuns(lines []string) []string {
	out := make([]string, 0, len(lines))
	for i := 0; i < len(lines); {
		if !promptFramingLine(lines[i]) {
			out = append(out, lines[i])
			i++
			continue
		}
		j := i
		for j < len(lines) && promptFramingLine(lines[j]) {
			j++
		}
		run := lines[i:j]
		firstPrompt := -1
		for k, line := range run {
			if promptOnly.MatchString(line) {
				firstPrompt = k
				break
			}
		}
		if firstPrompt < 0 {
			out = append(out, run...)
		} else {
			out = append(out, run[:firstPrompt]...)
			out = append(out, "<PROMPT>")
		}
		i = j
	}
	return out
}

const (
	usersLoginHeader  = "Num Class   Name         State          Idl Login@   Site"
	usersLoginDivider = "--- ------- ------------ -------------- --- -------- ------------------------"
)

// src/act.informative.c:2061-2096 renders descriptor login_time as a host-clock
// HH:MM:SS (8 bytes) followed by a single space and "[host]", independent of
// DP_CLOCK. Only a valid clock token in a users table is masked; all other
// bytes (including malformed clocks) remain comparable.
var usersLoginRowClock = regexp.MustCompile(`^(?:\r|\x1b\[[0-?]*[ -/]*[@-~])*[ \t]*[0-9]+[ \t]+(?:\[[^\]\r\n]+\]|-)[ \t]+[^\r\n]*[ \t]+((?:[01][0-9]|2[0-3]):[0-5][0-9]:[0-5][0-9]) \[[^\r\n]*\](?:\x1b\[[0-?]*[ -/]*[@-~])*\r?\n?$`)

func normalizeUsersLoginAt(raw string) string {
	// The keep-prompts mode masks host-clock metadata without touching
	// framing: the durable-player Started/Last dates are wall clock, not
	// game bytes — the same ruling as the Login@ field masked below.
	raw = playerSavedDatesMultiline.ReplaceAllString(raw, "${1}<WALL_CLOCK>${2}<WALL_CLOCK>${3}")
	lines := strings.SplitAfter(raw, "\n")
	divider, rows := false, false
	for i, line := range lines {
		// Inspect control-free headers without changing their captured bytes.
		heading := strings.Trim(ansiEscape.ReplaceAllString(line, ""), "\r\n")
		if heading == usersLoginHeader {
			divider, rows = true, false
			continue
		}
		if divider {
			divider, rows = false, heading == usersLoginDivider
			continue
		}
		if !rows {
			continue
		}
		match := usersLoginRowClock.FindStringSubmatchIndex(line)
		if match == nil {
			rows = false
			continue
		}
		// Same eight-byte width as HH:MM:SS; no adjacent whitespace is erased.
		lines[i] = line[:match[2]] + "<LOGIN@>" + line[match[3]:]
	}
	return strings.Join(lines, "")
}
