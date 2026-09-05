package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// makeHelpTestManager returns a Manager whose world has a small, hand-built
// help table + screen, so cmdHelp tests don't depend on the on-disk lib/text.
// The table is built pre-sorted (as LoadHelpFiles would leave it).
func makeHelpTestManager(t *testing.T) *Manager {
	t.Helper()
	m := makeTestManager(t)
	m.world.HelpTable = []game.HelpEntry{
		{Keyword: "move", Entry: "MOVE\r\nHow to move.\r\n"},
		{Keyword: "sanctuary", Entry: "SANCTUARY\r\nA protective spell.\r\n"},
		{
			Keyword: "say",
			// Keyword line retained as first line; body includes the ' shorthand.
			Entry: "SAY\r\n" +
				"Usage: say <message>\r\n" +
				"You can also use ' for say.\r\n",
		},
		{
			Keyword: "secret",
			// wizonly entry: hidden from mortals (existence hidden, like a miss).
			Entry: "SECRET\r\nImmortal-only detail. wizonly\r\n",
		},
		{Keyword: "sleep", Entry: "SLEEP\r\nGo to sleep.\r\n"},
	}
	m.world.HelpScreen = "Help screen line 1\r\n"
	for i := 2; i <= 30; i++ { // >22 lines → paginated on a plain-text client
		m.world.HelpScreen += "screen line\r\n"
	}
	return m
}

func TestHelpRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["help"]
	if !ok {
		t.Fatal("help command has no C gate")
	}
	if gate.MinLevel != 0 || gate.MinPosition != combat.PosDead {
		t.Fatalf("help gate = level %d position %d, want level 0 position %d", gate.MinLevel, gate.MinPosition, combat.PosDead)
	}
	for _, name := range []string{"help", "?"} {
		entry, ok := cmdRegistry.Lookup(name)
		if !ok {
			t.Fatalf("%s command is not registered", name)
		}
		if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
			t.Fatalf("%s registry gate = level %d position %d, want level %d position %d", name, entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
		}
	}
}

// --- no-arg → screen, paginated --------------------------------------------

func TestCmdHelpNoArgPagesScreen(t *testing.T) {
	m := makeHelpTestManager(t)
	s := makeCommandTestSession(t, m, "helpee", 1, 1001)
	s.player.SetPosition(combat.PosStanding)
	if err := ExecuteCommand(s, "help", nil); err != nil {
		t.Fatalf("ExecuteCommand(help): %v", err)
	}
	// >22 lines → enters pager mode (the screen goes through PageString).
	if !s.IsPaging() {
		t.Error("bare help did not enter pager mode; want screen page_string'd")
	}
	out := drainSendChannel(t, s)
	if !strings.Contains(out, "Help screen line 1") {
		t.Errorf("help no-arg missing screen content; got %q", out)
	}
}

// --- hit: [ TOPIC ] header + 75-dash separator + body via pager ------------

func TestCmdHelpHitDisplayByteExact(t *testing.T) {
	m := makeHelpTestManager(t)
	s := makeCommandTestSession(t, m, "helpee", 1, 1001)
	s.player.SetPosition(combat.PosStanding)
	if err := ExecuteCommand(s, "help", []string{"say"}); err != nil {
		t.Fatalf("ExecuteCommand(help say): %v", err)
	}
	out := drainSendChannel(t, s)
	// Topic header, uppercased, with the C color bracketing. drainSendChannel
	// returns raw JSON; ANSI escapes appear as \u001b and split the color
	// tokens around the brackets/word, so match the contiguous pieces:
	// "[ ", "SAY", and " ]" all appear (ANSI sits between them).
	for _, want := range []string{"[ ", "SAY", " ]"} {
		if !strings.Contains(out, want) {
			t.Errorf("help say header missing %q; got %q", want, out)
		}
	}
	// Body shows the entry AFTER the keyword line (the keyword line SAY is
	// skipped; the ' shorthand body line is present).
	if !strings.Contains(out, "You can also use ' for say.") {
		t.Errorf("help say missing body text after keyword line; got %q", out)
	}
	// The separator: exactly 75 dashes with a leading space, in red. Count the
	// longest run of dashes in the output.
	if n := longestDashRun(out); n != 75 {
		t.Errorf("separator dash run = %d, want 75; got %q", n, out)
	}
}

// longestDashRun returns the length of the longest consecutive '-' run in s.
func longestDashRun(s string) int {
	best, cur := 0, 0
	for i := 0; i < len(s); i++ {
		if s[i] == '-' {
			cur++
			if cur > best {
				best = cur
			}
		} else {
			cur = 0
		}
	}
	return best
}

// --- prefix + first-match --------------------------------------------------

func TestCmdHelpPrefixFirstMatch(t *testing.T) {
	m := makeHelpTestManager(t)
	s := makeCommandTestSession(t, m, "helpee", 1, 1001)
	s.player.SetPosition(combat.PosStanding)
	if err := ExecuteCommand(s, "help", []string{"s"}); err != nil {
		t.Fatalf("ExecuteCommand(help s): %v", err)
	}
	out := drainSendChannel(t, s)
	// First match for "s" in sorted order is "sanctuary" (the test table is
	// move, sanctuary, say, secret, sleep). ANSI splits the header, so match
	// the uppercased topic word directly.
	if !strings.Contains(out, "SANCTUARY") {
		t.Errorf("help s should resolve to first match sanctuary; got %q", out)
	}
}

// --- wizonly hidden for mortals, visible for immortals ---------------------

func TestCmdHelpWizonlyHiddenFromMortal(t *testing.T) {
	m := makeHelpTestManager(t)
	s := makeCommandTestSession(t, m, "mortal", 1, 1001) // level 1 < LVL_IMMORT
	s.player.SetPosition(combat.PosStanding)
	if err := ExecuteCommand(s, "help", []string{"secret"}); err != nil {
		t.Fatalf("ExecuteCommand(help secret): %v", err)
	}
	out := drainSendChannel(t, s)
	// Existence hidden — the miss message, not the entry body.
	if !strings.Contains(out, "There is no help on: secret") {
		t.Errorf("mortal help secret = %q; want the miss line (wizonly hidden)", out)
	}
	if strings.Contains(out, "Immortal-only detail") {
		t.Errorf("mortal saw wizonly body; existence should be hidden: %q", out)
	}
}

func TestCmdHelpWizonlyVisibleToImmortal(t *testing.T) {
	m := makeHelpTestManager(t)
	s := makeCommandTestSession(t, m, "imm", LVL_IMMORT, 1001) // level 31
	s.player.SetPosition(combat.PosStanding)
	if err := ExecuteCommand(s, "help", []string{"secret"}); err != nil {
		t.Fatalf("ExecuteCommand(help secret): %v", err)
	}
	out := drainSendChannel(t, s)
	// Header present (ANSI splits the bracket tokens) and body visible.
	if !strings.Contains(out, "SECRET") {
		t.Errorf("immortal help secret missing header topic; got %q", out)
	}
	if !strings.Contains(out, "Immortal-only detail") {
		t.Errorf("immortal help secret missing body; got %q", out)
	}
}

// --- miss → exact message --------------------------------------------------

func TestCmdHelpMissExactMessage(t *testing.T) {
	m := makeHelpTestManager(t)
	s := makeCommandTestSession(t, m, "helpee", 1, 1001)
	s.player.SetPosition(combat.PosStanding)
	if err := ExecuteCommand(s, "help", []string{"zzqx"}); err != nil {
		t.Fatalf("ExecuteCommand(help zzqx): %v", err)
	}
	out := drainSendChannel(t, s)
	const want = "There is no help on: zzqx"
	if !strings.Contains(out, want) {
		t.Errorf("help zzqx = %q; want it to contain %q", out, want)
	}
}

// --- invented surface is gone (R4) -----------------------------------------

func TestCmdHelpNoRegistryFallback(t *testing.T) {
	m := makeHelpTestManager(t)
	s := makeCommandTestSession(t, m, "helpee", 1, 1001)
	s.player.SetPosition(combat.PosStanding)
	// "look" is a registered command but has NO help-table entry in the test
	// table. The old cmdHelp fell back to the registry description; the faithful
	// version must report the C miss line instead.
	if err := ExecuteCommand(s, "help", []string{"look"}); err != nil {
		t.Fatalf("ExecuteCommand(help look): %v", err)
	}
	out := drainSendChannel(t, s)
	if !strings.Contains(out, "There is no help on: look") {
		t.Errorf("help look = %q; want miss line (registry fallback must be gone)", out)
	}
	if strings.Contains(out, "Did you mean") {
		t.Errorf("help look printed invented 'Did you mean' text; got %q", out)
	}
}

// --- ? dispatches to the same handler --------------------------------------

func TestCmdHelpQuestionMarkAlias(t *testing.T) {
	m := makeHelpTestManager(t)
	s := makeCommandTestSession(t, m, "helpee", 1, 1001)
	s.player.SetPosition(combat.PosStanding)
	// `?` is registered to the same handler as `help` (kept from #420).
	if err := ExecuteCommand(s, "?", []string{"say"}); err != nil {
		t.Fatalf("ExecuteCommand(? say): %v", err)
	}
	out := drainSendChannel(t, s)
	// `?` dispatched to cmdHelp (ANSI splits the header; match the topic word).
	if !strings.Contains(out, "SAY") {
		t.Errorf("? say did not dispatch to help handler; got %q", out)
	}
}

func TestCmdHelpRawArgumentSpacingMatchesC(t *testing.T) {
	m := makeHelpTestManager(t)
	s := makeCommandTestSession(t, m, "helpee", 1, 1001)
	s.player.SetPosition(combat.PosStanding)
	if err := executeCommandRaw(s, "help", []string{"zzqx", "tail"}, true, "zzqx  tail"); err != nil {
		t.Fatalf("executeCommandRaw(help): %v", err)
	}
	if got := drainSendChannel(t, s); !strings.Contains(got, "There is no help on: zzqx  tail") {
		t.Fatalf("raw help miss = %q, want preserved internal spacing", got)
	}

	if err := executeCommandRaw(s, "?", []string{"cure", "light"}, true, "cure  light"); err != nil {
		t.Fatalf("executeCommandRaw(?): %v", err)
	}
	if got := drainSendChannel(t, s); !strings.Contains(got, "There is no help on: cure  light") {
		t.Fatalf("raw ? miss = %q, want C's exact miss rather than normalized topic", got)
	}
}
