package session

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// Approved divergence (2026-10-22): C has no alias limit; the port refuses
// new aliases past game.MaxAliasCount entries or MaxAliasFileBytes persisted
// bytes, with the exact refusal bytes below. Redefinitions stay C-identical.
func TestCmdAliasCapRefusal(t *testing.T) {
	t.Chdir(t.TempDir())
	m := makeTestManager(t)
	s := makeTestSession(t, m, "AliasCapTest", 1001, true)

	add := func(name, repl string) string {
		t.Helper()
		if err := cmdAlias(s, []string{name, repl}); err != nil {
			t.Fatalf("cmdAlias(%s): %v", name, err)
		}
		return readSessionText(t, s)
	}

	for i := 0; i < game.MaxAliasCount; i++ {
		if got := add(fmt.Sprintf("a%d", i), "look"); got != "Alias added." {
			t.Fatalf("fill %d = %q, want \"Alias added.\"", i, got)
		}
	}
	if got := len(s.player.Aliases); got != game.MaxAliasCount {
		t.Fatalf("filled to %d aliases, want %d", got, game.MaxAliasCount)
	}

	// The next NEW alias is refused with the approved bytes; state is unchanged.
	if got, want := add("overflow", "look"), "You have too many aliases. Delete one first.\r\n"; got != want {
		t.Fatalf("count-cap refusal = %q, want %q", got, want)
	}
	if got := len(s.player.Aliases); got != game.MaxAliasCount {
		t.Fatalf("refused add still grew state: %d", got)
	}

	// Redefining an existing alias at the cap is count-neutral and succeeds.
	if got, want := add("a0", "score"), "Alias added."; got != want {
		t.Fatalf("redefine at cap = %q, want %q", got, want)
	}
	if got := len(s.player.Aliases); got != game.MaxAliasCount {
		t.Fatalf("redefine changed count: %d", got)
	}
}

// Long names (uncapped in C and the port) with full-length replacements trip
// the 64 KiB size cap before the count cap, and the persisted table never
// exceeds the cap. Replacements stay at or below 120 characters so each add
// emits exactly one message and the session channel never backs up.
func TestCmdAliasCapFileBytes(t *testing.T) {
	t.Chdir(t.TempDir())
	m := makeTestManager(t)
	s := makeTestSession(t, m, "AliasBytesTest", 1001, true)

	repl := strings.Repeat("x", 119) // " " + 119 = 120: no truncation notice
	added := 0
	for i := 0; i < game.MaxAliasCount; i++ {
		name := fmt.Sprintf("b%029d", i) // 30-char trigger, ~163 serialized bytes/entry
		if err := cmdAlias(s, []string{name, repl}); err != nil {
			t.Fatalf("cmdAlias(%d): %v", i, err)
		}
		if strings.Contains(readSessionText(t, s), "You have too many aliases. Delete one first.\r\n") {
			break
		}
		added++
	}
	if added == 0 || added >= game.MaxAliasCount {
		t.Fatalf("size cap tripped at %d entries, want 0 < n < %d", added, game.MaxAliasCount)
	}
	if got := len(s.player.Aliases); got != added {
		t.Fatalf("state = %d aliases, want %d", got, added)
	}
	if got := game.AliasFileBytes(s.player.Aliases); got > game.MaxAliasFileBytes {
		t.Fatalf("persisted alias bytes %d exceed cap %d", got, game.MaxAliasFileBytes)
	}
}
