package oraclediff

import (
	"regexp"
	"strings"
)

var (
	ansiEscape   = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]`)
	wallClock    = regexp.MustCompile(`\b(?:Sun|Mon|Tue|Wed|Thu|Fri|Sat) (?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec) +\d{1,2} \d{2}:\d{2}:\d{2} \d{4}\b`)
	vitalsPrompt = regexp.MustCompile(`\b\d+H\s+\d+M\s+\d+V\s*>`)
	promptOnly   = regexp.MustCompile(`^\s*(?:<PROMPT>|>)\s*$`)
	promptPrefix = regexp.MustCompile(`^> ?`)
	autoExitLine = regexp.MustCompile(`^\s*\[ Exits:`)
	statusVitals = regexp.MustCompile(`\b(?:HP|Mana|Move):\s*\d+/\d+`)
	statsLine    = regexp.MustCompile(`^\s*(?:Str:.*Dex:.*Int:.*|Wis:.*Con:.*Cha:.*)\s*$`)
	volatileLine = regexp.MustCompile(`(?i)^\s*(?:` +
		`(?:current (?:machine )?(?:time|date)|server (?:time|running since)|uptime)\s*[:=]|` +
		`(?:players online|players\s*:|gods\s*:)\s*[:=]?\s*\d+|` +
		`as of \d{1,4}[-/]\d{1,2}[-/]\d{1,4}\b|` +
		`\*{2,}.*(?:\bversion\b|\bv?\d+\.\d+\b|\bbeta\b)|` +
		`(?:motd|message of the day).*(?:version|updated|date)\s*[:=]|` +
		`version\s*[:=]\s*\S+` +
		`)`)
)

// Normalize applies the Tier-1 rules in this deliberate order. Keep this as
// the single rule list: order matters, and each rule documents why it exists.
func Normalize(raw string) string {
	// 1. Strip ANSI CSI escapes: color capability is transport presentation, not game text.
	raw = ansiEscape.ReplaceAllString(raw, "")

	// 2. Canonicalize CRLF/LFCR/CR and trailing whitespace: both telnet stacks may frame lines differently.
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\n\r", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")
	lines := strings.Split(raw, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
		if autoExitLine.MatchString(lines[i]) {
			lines[i] = strings.TrimLeft(lines[i], " \t")
		}
	}

	// 3. Remove standalone telnet prompts and mask embedded status values. The
	// C and Go transports repaint prompts at different times around identical
	// asynchronous game text, so prompt-only lines are framing, not game output.
	for i := range lines {
		// ctime(3) values in durable-player/admin output are host-clock
		// metadata, not game bytes. The deterministic harness freezes the
		// game clock but intentionally does not freeze process wall time.
		lines[i] = wallClock.ReplaceAllString(lines[i], "<WALL_CLOCK>")
		lines[i] = vitalsPrompt.ReplaceAllString(lines[i], "<PROMPT>")
		lines[i] = promptPrefix.ReplaceAllString(lines[i], "")
		lines[i] = statusVitals.ReplaceAllString(lines[i], "<VITALS>")
	}

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

	// 5. Drop volatile status/MOTD metadata lines: wall-clock, uptime, counts, and release dates vary per boot.
	lines = filtered[:0]
	for _, line := range filtered {
		if !volatileLine.MatchString(line) && !promptOnly.MatchString(line) {
			lines = append(lines, line)
		}
	}

	// Preserve internal blank lines but remove meaningless leading/trailing transcript padding.
	return strings.Trim(strings.Join(lines, "\n"), "\n") + "\n"
}
