package oraclediff

import (
	"regexp"
	"strings"
)

var (
	ansiEscape   = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]`)
	vitalsPrompt = regexp.MustCompile(`\b\d+H\s+\d+M\s+\d+V\s*>`)
	statusVitals = regexp.MustCompile(`\b(?:HP|Mana|Move):\s*\d+/\d+`)
	statsLine    = regexp.MustCompile(`^\s*(?:Str:.*Dex:.*Int:.*|Wis:.*Con:.*Cha:.*)\s*$`)
	volatileLine = regexp.MustCompile(`(?i)^\s*(?:` +
		`(?:current (?:time|date)|server (?:time|running since)|uptime)\s*[:=]|` +
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

	// 2. Canonicalize CRLF/CR and trailing whitespace: both telnet stacks may frame lines differently.
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")
	lines := strings.Split(raw, "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t")
	}

	// 3. Mask vitals prompts: H/M/V values are RNG-derived, while prompt placement is structural.
	for i := range lines {
		lines[i] = vitalsPrompt.ReplaceAllString(lines[i], "<PROMPT>")
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
		if !volatileLine.MatchString(line) {
			lines = append(lines, line)
		}
	}

	// Preserve internal blank lines but remove meaningless leading/trailing transcript padding.
	return strings.Trim(strings.Join(lines, "\n"), "\n") + "\n"
}
