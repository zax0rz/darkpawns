package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// huntingFixture registers a LVL_GRGOD wizard, a victim and the three-way
// audience the CMP / LVL_IMMORT / file-FALSE hunting producer must pass.
func huntingFixture(t *testing.T) (m *Manager, wizard, victim, watch, below, brief *Session) {
	t.Helper()
	m = makeTestManagerWithMobs(t)
	wizard = makeTestSession(t, m, "Wizard", 1001, true)
	wizard.player.SetLevel(LVL_GRGOD)
	registerTestSession(t, m, wizard, "Wizard")

	victim = makeTestSession(t, m, "Victim", 1001, true)
	victim.player.SetLevel(1)
	registerTestSession(t, m, victim, "Victim")

	watch, below, brief = autowizObservers(t, m)
	return m, wizard, victim, watch, below, brief
}

// setHunterKeyword spawns mob vnum 2001 in room 1001 and returns it with a
// keyword cmdSethunt can resolve.
func spawnHunterKeyword(t *testing.T, m *Manager, vnum int) (*game.MobInstance, string) {
	t.Helper()
	mob, err := m.world.SpawnMob(vnum, 1001)
	if err != nil {
		t.Fatalf("SpawnMob(%d): %v", vnum, err)
	}
	keywords := strings.Fields(mob.Proto().Keywords)
	if len(keywords) == 0 {
		t.Fatalf("mob %d has no keyword", vnum)
	}
	return mob, keywords[0]
}

// src/utils.c:708-729: set_hunting logs "%s started hunting %s" at
// CMP / LVL_IMMORT / file FALSE only when the prey is not a mobile and has an
// id number, and it logs before the hunting target is stored. Driven through
// the real sethunt command (src/act.wizard.c:3443-3472).
func TestHuntingMudlogProducer(t *testing.T) {
	t.Run("player victim", func(t *testing.T) {
		m, wizard, _, watch, below, brief := huntingFixture(t)
		file := captureMudlogFile(t)
		hunter, keyword := spawnHunterKeyword(t, m, 2001)

		if err := cmdSethunt(wizard, []string{"Victim", keyword}); err != nil {
			t.Fatal(err)
		}

		want := "[ " + hunter.GetName() + " started hunting Victim ]\r\n"
		if got := strings.Join(drainSessionText(t, watch), ""); got != want {
			t.Fatalf("observer = %q, want %q", got, want)
		}
		if got := strings.Join(drainSessionText(t, below), ""); got != "" {
			t.Fatalf("a level-%d observer saw %q", LVL_IMMORT-1, got)
		}
		// CMP is 3; the brief bit alone is 1.
		if got := strings.Join(drainSessionText(t, brief), ""); got != "" {
			t.Fatalf("brief-syslog observer saw %q", got)
		}
		// File FALSE leaves no alog line.
		if got := file.String(); got != "" {
			t.Fatalf("the file flag was TRUE: %q", got)
		}
	})

	t.Run("mobile victim is silent", func(t *testing.T) {
		m, wizard, _, watch, _, _ := huntingFixture(t)
		hunter, keyword := spawnHunterKeyword(t, m, 2001)
		// 2001 and 2002 share the "goblin guard" keyword, so the prey is the
		// distinctively keyworded rat.
		prey, preyKeyword := spawnHunterKeyword(t, m, 2004)

		if err := cmdSethunt(wizard, []string{preyKeyword, keyword}); err != nil {
			t.Fatal(err)
		}
		if got := hunter.GetHunting(); got != prey.GetName() {
			t.Fatalf("hunting target = %q, want %q", got, prey.GetName())
		}
		if got := strings.Join(drainSessionText(t, watch), ""); got != "" {
			t.Fatalf("C's IS_MOB arm logged: %q", got)
		}
	})

	t.Run("victim without an id is silent", func(t *testing.T) {
		m, wizard, victim, watch, _, _ := huntingFixture(t)
		victim.player.ID = 0 // C's GET_IDNUM(vict) > 0 gate
		_, keyword := spawnHunterKeyword(t, m, 2001)

		if err := cmdSethunt(wizard, []string{"Victim", keyword}); err != nil {
			t.Fatal(err)
		}
		if got := strings.Join(drainSessionText(t, watch), ""); got != "" {
			t.Fatalf("an id-less prey logged: %q", got)
		}
	})
}
