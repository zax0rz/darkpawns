package session

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// TestResolveCommandPrefixThreeLaws exercises the three C command_interpreter
// laws (interpreter.c:909-912) directly against the embedded ordered table.
// Row numbers cited below are seq values in command_order.tsv (cmd_info[] line
// order), verified against src/interpreter.c.
func TestResolveCommandPrefixThreeLaws(t *testing.T) {
	// Law 2 — table order wins, not alphabetical. grin (seq 154) precedes
	// grimace (seq 155); gri must resolve to grin.
	t.Run("gri_resolves_by_table_order_to_grin", func(t *testing.T) {
		if got, ok := resolveCommandPrefix("gri", 0); !ok || got != "grin" {
			t.Errorf("resolveCommandPrefix(\"gri\",0) = %q,%v; want \"grin\",true", got, ok)
		}
	})
	t.Run("exact_grin", func(t *testing.T) {
		if got, ok := resolveCommandPrefix("grin", 0); !ok || got != "grin" {
			t.Errorf("resolveCommandPrefix(\"grin\",0) = %q,%v; want \"grin\",true", got, ok)
		}
	})

	// Law 1 — typed-longer-than-entry never matches.
	t.Run("grinx_no_match", func(t *testing.T) {
		if got, ok := resolveCommandPrefix("grinx", 0); ok {
			t.Errorf("resolveCommandPrefix(\"grinx\",0) = %q,true; want \"\",false", got)
		}
	})

	// Law 2 — the qui stub (seq 298) precedes quit (seq 299), so `qui` resolves
	// to the stub, never to quit. This is why the stub exists in C.
	t.Run("qui_resolves_to_stub_not_quit", func(t *testing.T) {
		if got, ok := resolveCommandPrefix("qui", 0); !ok || got != "qui" {
			t.Errorf("resolveCommandPrefix(\"qui\",0) = %q,%v; want \"qui\",true (the stub)", got, ok)
		}
	})
	t.Run("exact_quit", func(t *testing.T) {
		if got, ok := resolveCommandPrefix("quit", 0); !ok || got != "quit" {
			t.Errorf("resolveCommandPrefix(\"quit\",0) = %q,%v; want \"quit\",true", got, ok)
		}
	})

	// Law 3 — REQUIRED: level filters DURING the scan. goto (seq 146, L31)
	// precedes gossip (seq 147, L0). A mortal typing `go` skips goto and
	// resolves to gossip; an immortal resolves to goto. If the level filter
	// were applied post-resolution instead, the mortal would get "Huh" (goto's
	// gate hides it), diverging from C — so this is the load-bearing case.
	t.Run("go_mortal_skips_goto_to_gossip", func(t *testing.T) {
		if got, ok := resolveCommandPrefix("go", 1); !ok || got != "gossip" {
			t.Errorf("resolveCommandPrefix(\"go\",1) = %q,%v; want \"gossip\",true (mortal skips goto)", got, ok)
		}
	})
	t.Run("go_immortal_gets_goto", func(t *testing.T) {
		if got, ok := resolveCommandPrefix("go", 31); !ok || got != "goto" {
			t.Errorf("resolveCommandPrefix(\"go\",31) = %q,%v; want \"goto\",true (immortal)", got, ok)
		}
	})
	// Same prefix, same table — the two levels resolve to DIFFERENT commands.
	t.Run("go_mortal_and_immortal_diverge", func(t *testing.T) {
		mortal, _ := resolveCommandPrefix("go", 1)
		immortal, _ := resolveCommandPrefix("go", 31)
		if mortal == immortal {
			t.Errorf("level-skip broken: mortal and immortal both resolve `go` to %q; want gossip vs goto", mortal)
		}
	})

	// R4 — Go-only commands are not in the C table, so they cannot win by
	// prefix. `ignore` is a Go-only command (absent from cmd_info[]); `ignor`
	// must miss the C scan entirely and fall through to the exact-match
	// registry in the caller.
	t.Run("go_only_prefix_misses_c_table", func(t *testing.T) {
		if got, ok := resolveCommandPrefix("ignor", 0); ok {
			t.Errorf("resolveCommandPrefix(\"ignor\",0) = %q,true; want \"\",false (Go-only names are not C abbreviations)", got)
		}
	})

	// Shared miss path — a junk prefix resolves to nothing.
	t.Run("zz_no_match", func(t *testing.T) {
		if got, ok := resolveCommandPrefix("zz", 0); ok {
			t.Errorf("resolveCommandPrefix(\"zz\",0) = %q,true; want \"\",false", got)
		}
	})

	// Guard: exact match on a registered-but-not-Go command resolves to itself.
	// `murder` IS in the C table (L0) even though the Go port never registered a
	// handler for it, so resolution is faithful even when dispatch later misses.
	t.Run("murd_resolves_to_murder", func(t *testing.T) {
		if got, ok := resolveCommandPrefix("murd", 0); !ok || got != "murder" {
			t.Errorf("resolveCommandPrefix(\"murd\",0) = %q,%v; want \"murder\",true", got, ok)
		}
	})
}

// TestResolveCommandPrefixEmptyGuard ensures the resolver never matches on an
// empty input (which would satisfy HasPrefix for every entry).
func TestResolveCommandPrefixEmptyGuard(t *testing.T) {
	if got, ok := resolveCommandPrefix("", 0); ok {
		t.Errorf("resolveCommandPrefix(\"\",0) = %q,true; want \"\",false", got)
	}
}

// TestCommandOrderGoldenMatchesCOracle is the drift test, mirroring
// TestCommandGateGoldenCoversCGoRegistrationsAndSocials: it pins the row count
// and a sha256 over the seq/name/level triples so any change to the C table or
// the generator is caught here with a "regenerate" hint.
func TestCommandOrderGoldenMatchesCOracle(t *testing.T) {
	if len(commandOrder) == 0 {
		t.Fatal("commandOrder is empty; command_order.tsv did not parse")
	}

	// Reconstruct seq (1-based source order) alongside each entry for hashing.
	hash := sha256.New()
	for i, entry := range commandOrder {
		fmt.Fprintf(hash, "%d\t%s\t%d\n", i+1, entry.name, entry.level)
	}
	// Pin: update both constants together after regenerating from the reviewed
	// oracle via `go run ./cmd/dp-command-gates -oracle <interpreter.c>`.
	const wantRows = 508
	const wantHash = "1a2ed144984de30ff2bbb57a3b66aeb957bdb4b5de0c324825c39434521c0c95"
	if len(commandOrder) != wantRows {
		t.Errorf("command_order.tsv rows = %d, want %d; regenerate via cmd/dp-command-gates", len(commandOrder), wantRows)
	}
	if got := fmt.Sprintf("%x", hash.Sum(nil)); got != wantHash {
		t.Errorf("command_order.tsv hash = %s, want %s; regenerate from the reviewed oracle", got, wantHash)
	}

	// Names must be unique in source order (matches the gates-table invariant).
	seen := make(map[string]int, len(commandOrder))
	for i, entry := range commandOrder {
		if prev, dup := seen[entry.name]; dup {
			t.Errorf("command_order.tsv: duplicate name %q at seq %d (first at %d)", entry.name, i+1, prev+1)
		}
		seen[entry.name] = i
	}
}

// TestResolveCommandPrefixExamplesFromBrief sanity-checks the brief's named
// probes against the real table, so the PR description's probe table stays
// honest. These pin the behaviour the oracle scenario depends on.
func TestResolveCommandPrefixExamplesFromBrief(t *testing.T) {
	type tc struct {
		typed string
		level int
		want  string // empty string means no match
	}
	cases := []tc{
		// guard probes
		{"l", 0, "look"},
		{"in", 0, "inventory"},
		{"sa", 0, "say"},
		// gos → gossip (a channel command; the prefix lands on gossip, L0)
		{"gos", 0, "gossip"},
		// gri → grin (social, table-order proof)
		{"gri", 0, "grin"},
		// qui → the stub
		{"qui", 0, "qui"},
		// junk
		{"zz", 0, ""},
	}
	for _, c := range cases {
		got, ok := resolveCommandPrefix(c.typed, c.level)
		if c.want == "" {
			if ok {
				t.Errorf("resolveCommandPrefix(%q,%d) = %q; want no match", c.typed, c.level, got)
			}
			continue
		}
		if !ok || got != c.want {
			t.Errorf("resolveCommandPrefix(%q,%d) = %q,%v; want %q,true", c.typed, c.level, got, ok, c.want)
		}
	}
}

// TestExecuteCommandPrefixResolutionEndToEnd drives ExecuteCommand with
// abbreviated input and asserts the resolver canonicalizes before dispatch.
// Uses drainSendChannel so assertions are robust to state-vs-text ordering and
// to socials that may emit zero actor-facing messages.
func TestExecuteCommandPrefixResolutionEndToEnd(t *testing.T) {
	t.Run("l_abbreviates_look", func(t *testing.T) {
		m := makeTestManager(t)
		s := makeCommandTestSession(t, m, "abbrev", 1, 1001)
		s.player.SetPosition(combat.PosStanding)
		// `l` resolves to `look`; look is registered and prints the room rather
		// than the "Huh?!?" an exact-match-only port would give.
		if err := ExecuteCommand(s, "l", nil); err != nil {
			t.Fatalf("ExecuteCommand(l): %v", err)
		}
		if got := drainSendChannel(t, s); strings.Contains(got, "Huh?!?") {
			t.Errorf("abbreviated `l` got %q; want look output (resolution failed)", got)
		}
	})

	t.Run("zz_junk_is_huh", func(t *testing.T) {
		m := makeTestManager(t)
		s := makeCommandTestSession(t, m, "abbrev", 1, 1001)
		s.player.SetPosition(combat.PosStanding)
		if err := ExecuteCommand(s, "zz", nil); err != nil {
			t.Fatalf("ExecuteCommand(zz): %v", err)
		}
		if got := drainSendChannel(t, s); !strings.Contains(got, "Huh?!?") {
			t.Errorf("junk `zz` got %q; want \"Huh?!?\"", got)
		}
	})

	t.Run("go_only_prefix_ignor_is_huh", func(t *testing.T) {
		m := makeTestManager(t)
		s := makeCommandTestSession(t, m, "abbrev", 1, 1001)
		s.player.SetPosition(combat.PosStanding)
		// `ignor` is a prefix of the Go-only `ignore` but not in the C table, so
		// it must NOT resolve by prefix and must miss the exact registry too.
		if err := ExecuteCommand(s, "ignor", nil); err != nil {
			t.Fatalf("ExecuteCommand(ignor): %v", err)
		}
		if got := drainSendChannel(t, s); !strings.Contains(got, "Huh?!?") {
			t.Errorf("Go-only prefix `ignor` got %q; want \"Huh?!?\" (R4: Go-only is exact-only)", got)
		}
	})

	t.Run("go_mortal_abbrev_to_gossip_not_goto", func(t *testing.T) {
		// Law 3 end-to-end: a mortal typing `go` resolves to gossip (not the
		// earlier, immortal-gated goto). gossip is registered, so dispatch is
		// reached rather than "Huh". The gate on goto is MinLevel 31; if the
		// resolver failed to level-skip, the mortal would resolve to goto and
		// be hidden ("Huh").
		m := makeTestManager(t)
		s := makeCommandTestSession(t, m, "abbrev", 1, 1001)
		s.player.SetPosition(combat.PosStanding)
		if err := ExecuteCommand(s, "go", nil); err != nil {
			t.Fatalf("ExecuteCommand(go): %v", err)
		}
		if got := drainSendChannel(t, s); strings.Contains(got, "Huh?!?") {
			t.Errorf("mortal `go` got %q; want gossip dispatch (level-skip to gossip failed)", got)
		}
	})
}

// keep combat import in scope for the position constants used above.
var _ = combat.PosStanding
