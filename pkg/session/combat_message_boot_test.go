package session

import (
	"errors"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// C's load_messages() cannot continue without the combat corpus: it prints
// "Error reading combat message file %s" and exit(1) (src/fight.c:133-137). The
// port used to substitute an invented skill-message table here, which would have
// replaced every skill's text with prose C never printed (R4). A load failure
// must stop boot instead.
func TestSetCombatMessageFuncStopsStartupOnCorpusFailure(t *testing.T) {
	original := loadCombatMessages
	t.Cleanup(func() { loadCombatMessages = original })

	sentinel := errors.New("corrupt embedded corpus")
	loadCombatMessages = func(*combat.GameCallbacks) error { return sentinel }

	m := makeTestManagerWithVoidRooms(t)
	err := m.SetCombatMessageFunc()
	if err == nil {
		t.Fatal("SetCombatMessageFunc() = nil, want the load failure to stop startup")
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("SetCombatMessageFunc() = %v, want it to wrap %v", err, sentinel)
	}
}

// The healthy path wires C's embedded corpus, so the parsed fight_messages set
// reaches the engine's SkillMessage hook.
func TestSetCombatMessageFuncWiresEmbeddedCorpus(t *testing.T) {
	// CombatEngine.SetCallbacks also assigns the package-level callbacks, so a
	// wired corpus here would otherwise leak into every later test in this
	// package (it did: crash_load_bad_stats_test.go then saw the set-399
	// suffering line it does not wire for itself).
	previous := combat.GetCallbacks()
	t.Cleanup(func() { combat.SetCallbacks(previous) })

	m := makeTestManagerWithVoidRooms(t)
	if err := m.SetCombatMessageFunc(); err != nil {
		t.Fatalf("SetCombatMessageFunc() = %v, want nil", err)
	}
	if m.combatEngine.Callbacks == nil || m.combatEngine.Callbacks.SkillMessage == nil {
		t.Fatal("SetCombatMessageFunc did not wire the corpus SkillMessage hook")
	}
}
