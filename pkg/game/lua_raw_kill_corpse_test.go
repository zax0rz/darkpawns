package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/scripting"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

// TestLuaRawKillLeavesPlainCorpse proves C-ERRATA E-001: C's lua_raw_kill
// declares raw_kill with three parameters (src/scripts.c:1231) but the real
// function takes two (src/fight.c:534), so C reads the killer pointer as the
// attack type — 0 with no killer, an unmatched pointer value otherwise — and
// every Lua raw_kill death lands on make_corpse's default arm
// (src/fight.c:369-370): the plain corpse, whatever type the script named.
// The port passed the script's type through and showed wording C never did.
func TestLuaRawKillLeavesPlainCorpse(t *testing.T) {
	for _, tc := range []struct {
		name       string
		withKiller bool
	}{
		{"with killer", true},
		{"without killer", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, _ := newZoneResetTestSpawner(t)
			old := combat.GetCallbacks()
			t.Cleanup(func() { combat.SetCallbacks(old) })
			combat.SetCallbacks(w.WireCombatCallbacks())
			victim := NewPlayer(4100, "Rawvictim", 100)
			if err := w.AddPlayer(victim); err != nil {
				t.Fatalf("AddPlayer(victim): %v", err)
			}
			killer := NewPlayer(4101, "Rawkiller", 100)
			if err := w.AddPlayer(killer); err != nil {
				t.Fatalf("AddPlayer(killer): %v", err)
			}

			adapter := NewWorldScriptableAdapter(w)
			victimRef := scripting.CharRef{ID: victim.GetID()}
			var killerRef *scripting.CharRef
			if tc.withKiller {
				ref := scripting.CharRef{ID: killer.GetID()}
				killerRef = &ref
			}
			// The archived scripts' call shape: raw_kill(vict, killer,
			// SPELL_HELLFIRE) (daemonic_focus.lua:193).
			adapter.RawKill(victimRef, killerRef, spells.SpellHellfire)

			objects := w.GetItemsInRoom(100)
			if len(objects) != 1 {
				t.Fatalf("raw_kill left %d room objects, want one corpse", len(objects))
			}
			if !objects[0].IsCorpse {
				t.Fatalf("room object is not a corpse: vnum %d", objects[0].GetVNum())
			}
			if got, want := objects[0].GetLongDesc(), "The corpse of Rawvictim is lying here."; got != want {
				t.Fatalf("corpse long desc = %q, want %q (E-001: C's raw_kill never sees the script's type)", got, want)
			}
		})
	}
}
