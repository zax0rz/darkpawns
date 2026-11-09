package oraclediff

import (
	"fmt"
	"regexp"
	"strings"
)

// setupRefusal is one whole-line refusal string: text C prints when a command
// is not executed — not recognized, below its level gate, or recognized but
// refused by the handler. Warmup steps answered with one of these lines never
// configured what they claim to configure, so a scenario whose warmup contains
// one is hollow: both engines refuse identically, the probe compares two
// unconfigured worlds, and the pass proves nothing (DP-1456, second recurrence
// of the refused-setup class after the September peer logins, RO-012).
type setupRefusal struct {
	line string
	cite string
}

// setupRefusalLines is the original DP-1456 exact inventory (2026-11-05).
// Keep these entries intact; DP-1511's additional exact strings and compiled
// templates live in setuprefusal_inventory.go, with a source audit and actual
// C witnesses in docs/fidelity/evidence/dp-1511/. Matching uses whole normalized
// lines so prose merely containing a refusal phrase does not trip the check.
//
// Deliberately excluded: the interpreter's position refusals
// (src/interpreter.c:923ff — "You need to stand up!" and kin). No incident
// has shown a position-refused warmup step passing hollow; add them with an
// incident if one appears.
var setupRefusalLines = []setupRefusal{
	// The interpreter's universal refusals. "Huh?!?" covers every
	// below-level command in the cmd_info[] table, which is how a demoted
	// setup actor loses oedit/load/transfer without any handler running.
	{"Huh?!?", "src/interpreter.c:916"},
	{"You try, but the mind-numbing cold prevents you...", "src/interpreter.c:918"},
	{"Sorry, that command hasn't been implemented yet.", "src/interpreter.c:920"},
	{"You can't use immortal commands while switched.", "src/interpreter.c:922"},

	// set (do_set). "level" needs LVL_GRGOD (src/act.wizard.c:2577), so a
	// warmup actor demoted below 38 refuses every later "set ... level".
	{"Usage: set <victim> <field> <value>", "src/act.wizard.c:2626"},
	{"There is no such player.", "src/act.wizard.c:2635"},
	{"There is no such creature.", "src/act.wizard.c:2643"},
	{"Sorry, you can't do that.", "src/act.wizard.c:2661"},
	{"Maybe that's not such a great idea...", "src/act.wizard.c:2678"},
	{"You are not godly enough for that!", "src/act.wizard.c:2688"},

	// goto / find_target_room.
	{"You must supply a room number or name.", "src/act.wizard.c:195"},
	{"No room exists with that number.", "src/act.wizard.c:201"},
	{"That object is not available.", "src/act.wizard.c:210"},
	{"No such creature or object around.", "src/act.wizard.c:214"},
	{"You are not godly enough to use that room!", "src/act.wizard.c:221"},
	{"There's a private conversation going on in that room.", "src/act.wizard.c:226"},

	// teleport (do_teleport).
	{"Whom do you wish to teleport?", "src/act.wizard.c:373"},
	{"No-one by that name here.", "src/config.c:93 (NOPERSON, via src/act.wizard.c:375)"},
	{"Use 'goto' to teleport yourself.", "src/act.wizard.c:377"},
	{"Maybe you shouldn't do that.", "src/act.wizard.c:379"},
	{"Where do you wish to send this person?", "src/act.wizard.c:381"},

	// transfer (do_trans). LVL_IMMORT+1 (src/interpreter.c:784).
	{"Whom do you wish to transfer?", "src/act.wizard.c:318"},
	// NOPERSON again, via src/act.wizard.c:321.
	{"That doesn't make much sense, does it?", "src/act.wizard.c:323"},
	{"Go transfer someone your own size.", "src/act.wizard.c:326"},
	{"I think not.", "src/act.wizard.c:343"},

	// load (do_load). LVL_IMMORT (src/interpreter.c:544).
	{"Mob name not found.", "src/act.wizard.c:1274"},
	{"Object name not found.", "src/act.wizard.c:1285"},
	{"Usage: load { obj | mob } <number or name>", "src/act.wizard.c:1289"},
	{"A NEGATIVE number??", "src/act.wizard.c:1295"},
	{"There is no monster with that number.", "src/act.wizard.c:1300"},
	{"There is no object with that number.", "src/act.wizard.c:1324"},

	// force (do_force).
	{"Whom do you wish to force do what?", "src/act.wizard.c:1867"},
	// NOPERSON again, via src/act.wizard.c:1870.
	{"No, no, no!", "src/act.wizard.c:1872"},

	// cast (do_cast). "Cast what?!?" is the no-argument refusal named in
	// DP-1456; the know/unfamiliar pair is a spell that never fires.
	{"Psionics 'will' things, not 'cast' them!", "src/spell_parser.c:928"},
	{"Cast what where?", "src/spell_parser.c:937"},
	{"Cast what?!?", "src/spell_parser.c:958"},
	{"Will what?!?", "src/spell_parser.c:960"},
	{"You do not know that spell!", "src/spell_parser.c:965"},
	{"You are not learned in that power!", "src/spell_parser.c:967"},
	{"You are unfamiliar with that spell.", "src/spell_parser.c:972"},
	{"You are unfamiliar with that power.", "src/spell_parser.c:974"},

	// skillset (do_skillset).
	{"Syntax: skillset <name> '<skill>' <value>", "src/modify.c:265"},
	// NOPERSON again, via src/modify.c:283.
	{"Skill name expected.", "src/modify.c:290"},
	{"Skill must be enclosed in: ''", "src/modify.c:294"},

	// advance (do_advance).
	{"That player is not here.", "src/act.wizard.c:1494"},
	{"Advance who?", "src/act.wizard.c:1498"},
	{"Maybe that's not such a great idea.", "src/act.wizard.c:1504"},
	{"NO!  Not on NPC's.", "src/act.wizard.c:1510"},
	{"That's not a level!", "src/act.wizard.c:1514"},

	// The OLC editors (oedit/medit/redit/zedit/sedit through do_olc; the
	// below-level cases print "Huh?!?" at src/interpreter.c:916).
	{"Save which zone?", "src/olc.c:125"},
	{"Specify a new zone number.", "src/olc.c:138"},
	{"Yikes!  Stop that, someone will get hurt!", "src/olc.c:143"},
	{"Sorry, there is no zone for that number!", "src/olc.c:172"},
	{"You do not have permission to edit this zone.", "src/olc.c:181"},

	// hcontrol (do_hcontrol): the first line of HCONTROL_FORMAT.
	{"Usage: hcontrol build <house vnum> <exit direction> <player name>", "src/house.c:324 (HCONTROL_FORMAT first line, sent via src/house.c:401)"},

	// sethunt (do_sethunt).
	{"Who do you wish to hunt?", "src/act.wizard.c:3455"},
	{"No-one by that name around.", "src/act.wizard.c:3457"},
	{"Who shall be the hunter?", "src/act.wizard.c:3459"},
	{"Yeah right.", "src/act.wizard.c:3461"},
	{"PCs can't be made to hunt.", "src/act.wizard.c:3463"},
	{"Cant hunt higher than your level.", "src/act.wizard.c:3466"},

	// switch (do_switch).
	{"You're already switched.", "src/act.wizard.c:1182"},
	{"Switch with who?", "src/act.wizard.c:1184"},
	{"No such character.", "src/act.wizard.c:1186"},
	{"Hee hee... we are jolly funny today, eh?", "src/act.wizard.c:1188"},
	{"You can't do that, the body is already in use!", "src/act.wizard.c:1190"},
	{"You aren't holy enough to use a mortal's body.", "src/act.wizard.c:1192"},
}

// matchingSetupRefusal reports the first refusal line that appears as a whole
// normalized line in output, with its src/ citation.
func matchingSetupRefusal(output string) (line string, cite string) {
	for _, norm := range strings.Split(Normalize(output), "\n") {
		norm = strings.TrimSpace(norm)
		for _, r := range setupRefusalLines {
			if norm == r.line {
				return r.line, r.cite
			}
		}
		for _, r := range setupRefusalAdditionalLines {
			if norm == r.line {
				return norm, r.cite
			}
		}
		for _, r := range setupRefusalPatterns {
			if r.pattern.MatchString(norm) {
				return norm, r.cite
			}
		}
	}
	return "", ""
}

// checkWarmupRefusals scans the sender block of every warmup step and returns
// a setup-refused error naming the first one a refusal line answered. The
// sender's block is the first block of each step's group: runAudienceProbe
// appends the step's sender block before that step's audience blocks, and the
// peer set is constant for the whole warmup (peer-drop runs after warmup), so
// each step's group has the same stride.
func checkWarmupRefusals(steps []string, blocks []AudienceProbeBlock, peers int, engine string) error {
	stride := peers + 1
	if len(blocks) != stride*len(steps) {
		return fmt.Errorf("setup-refused check: %d captured warmup blocks for %d steps with %d peers: unexpected block alignment", len(blocks), len(steps), peers)
	}
	for i, step := range steps {
		if line, _ := matchingSetupRefusal(blocks[i*stride].Output); line != "" {
			return fmt.Errorf("setup-refused: warmup step %d %q engine=%s: %s", i+1, step, engine, line)
		}
	}
	return nil
}

// Substitutions are non-empty and never cross a line. C's MAX_STRING_LENGTH
// (src/structs.h:643) bounds the whole line through refusalMatcher.maxLen
// rather than counted regex repeats: chunked {0,1000} repeats compiled each
// pattern into a very large RE2 program, costing ~120 MB and 0.24 s at every
// process start (under -race, enough to break the WS first-byte helper test).
const setupRefusalField = `[^\r\n]+`

// setupRefusalMaxString is C's MAX_STRING_LENGTH (src/structs.h:643).
const setupRefusalMaxString = 8192

// refusalMatcher is a compiled refusal template plus the longest line C could
// have produced from it: MAX_STRING_LENGTH of substituted text plus the
// template's fixed bytes.
type refusalMatcher struct {
	re     *regexp.Regexp
	maxLen int
}

func (m refusalMatcher) MatchString(s string) bool {
	return len(s) <= m.maxLen && m.re.MatchString(s)
}

type setupRefusalPattern struct {
	template string
	cite     string
	pattern  refusalMatcher
}

func refusalPattern(template, cite string) setupRefusalPattern {
	return setupRefusalPattern{template, cite, compileRefusalTemplate(template)}
}

// compileRefusalTemplate accepts C printf conversions and act() substitutions
// (src/comm.c:2408-2457). Everything else remains escaped literal text. Invalid
// templates panic at initialization instead of silently weakening the gate.
func compileRefusalTemplate(template string) refusalMatcher {
	var out strings.Builder
	fixed := 0
	out.WriteString("^")
	for i := 0; i < len(template); {
		switch template[i] {
		case '%':
			i++
			if i < len(template) && template[i] == '%' {
				out.WriteString("%")
				fixed++
				i++
				continue
			}
			start := i
			for i < len(template) && strings.ContainsRune("-+ #0.123456789hlz", rune(template[i])) {
				i++
			}
			if i == len(template) || !strings.ContainsRune("sdiuoxXfFeEgGc", rune(template[i])) {
				panic("invalid printf refusal template: " + template[start-1:])
			}
			if strings.ContainsRune("diu", rune(template[i])) {
				out.WriteString(`[+-]?[0-9]{1,20}`)
				fixed += 21
			} else {
				out.WriteString(setupRefusalField)
			}
			i++
		case '$':
			i++
			if i == len(template) || !strings.ContainsRune("nNmMsSeEoOpPaATF$", rune(template[i])) {
				panic("invalid act refusal template: " + template)
			}
			if template[i] == '$' {
				out.WriteString(`\$`)
				fixed++
			} else {
				switch template[i] {
				case 'm', 'M':
					fixed += 3 // longest C pronoun or article
					out.WriteString(`(?:[Hh]im|[Hh]er|[Ii]t)`)
				case 's', 'S':
					fixed += 3 // longest C pronoun or article
					out.WriteString(`(?:[Hh]is|[Hh]er|[Ii]ts)`)
				case 'e', 'E':
					fixed += 3 // longest C pronoun or article
					out.WriteString(`(?:[Hh]e|[Ss]he|[Ii]t)`)
				case 'a', 'A':
					fixed += 3 // longest C pronoun or article
					out.WriteString(`(?:[Aa]|[Aa]n)`)
				default:
					out.WriteString(setupRefusalField)
				}
			}
			i++
		default:
			out.WriteString(regexp.QuoteMeta(template[i : i+1]))
			fixed++
			i++
		}
	}
	out.WriteString("$")
	return refusalMatcher{re: regexp.MustCompile(out.String()), maxLen: setupRefusalMaxString + fixed}
}
