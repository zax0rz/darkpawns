package game

// effects_depth_test.go — fidelity cover for the minor, assigned-special
// effect fixes (BRIEF-2026-11-27-ember-special-proc-minor-fixes, PR 2:
// effects). Two rows are reached and verified behaviorally here:
// remorter clears GET_WIMP_LEV (src/spec_procs2.c:757) and pray_for_items
// scans the world-index-next room (src/spec_procs.c:2077/2141).
//
// Tattoo (`give_tat`'s trailing `affect_total(ch)`, src/spec_procs2.c:942)
// and identifier (`spell_identify(LEVEL_IMMORT, ...)`,
// src/spec_procs2.c:1275) are fixed the same one-line-per-site way but
// carry no new unit test: Go's TattooAf already writes the same stat
// cache C's affect_total recomputes (probe: pre/post-affect_total reads of
// GetStr identical on Dragon/Tiger/Fox/Owl/Swords), and Go's
// castIdentifyObject discards `level` exactly like C's spell_identify
// body, so no go-test assertion can move when either line lands. Both are
// regression-covered by the shops tattoo/identifier census scenarios
// (the identifier cast level is invisible to the normalizer and to the
// API; a hook there would test the call, not the game).

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestEffectRemorterClearsWimpy pins C's `GET_WIMP_LEV(ch) = 0` just after
// the remort level/exp reset (src/spec_procs2.c:755-757): the port left
// wimpy as the pre-remort value, so a remorted wimp kept auto-fleeing.
func TestEffectRemorterClearsWimpy(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "X", Zone: 1}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	w.StopAITicker()
	t.Cleanup(w.StopAITicker)
	p := NewPlayer(1, "Remorter", 1001)
	p.WimpLevel = 7
	p.SetGold(70000)
	if err := w.AddPlayer(p); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	dprng.ResetStream(1)
	remortPlayer(p)
	if p.WimpLevel != 0 {
		t.Fatalf("wimpy after remort = %d, C sets GET_WIMP_LEV = 0", p.WimpLevel)
	}
}

// TestEffectPrayForItemsScansNextRoom pins the index-next scan:
// `world[1 + ch->in_room]` (src/spec_procs.c:2077, :2141). The port scanned
// the altar room itself, so the gifts were never found (or were duplicated
// into the wrong stash).
func TestEffectPrayForItemsScansNextRoom(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 8008, Name: "Altar", Zone: 9},
			{VNum: 8009, Name: "Stash", Zone: 9},
		},
		Objs: []parser.Obj{{
			VNum: 3003, Keywords: "gift locket", ShortDesc: "a tiny gift locket",
			LongDesc: "A gift locket lies here.", Cost: 5,
			ExtraDescs: []parser.ExtraDesc{{
				Keywords:    "item_for_Tester",
				Description: "Prayer forgives.",
			}},
		}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	w.StopAITicker()
	t.Cleanup(w.StopAITicker)
	out := map[string]string{}
	w.MessageSink = func(name string, msg []byte) { out[name] += string(msg) }
	ch := NewPlayer(1, "Tester", 8008)
	ch.SetGold(1000)
	if err := w.AddPlayer(ch); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	stash, err := w.SpawnObject(3003, 8009)
	if err != nil {
		t.Fatalf("SpawnObject: %v", err)
	}
	w.AddItemToRoom(stash, 8009)

	if !specPrayForItems(w, ch, nil, "pray", "") {
		t.Fatal("specPrayForItems returned false; the gift is in the index-next room")
	}
	// C mints a fresh read_object copy and obj_to_room()s it to the
	// prayer's room, and deducts gold = 1 + cost.
	gifts := 0
	for _, obj := range w.GetItemsInRoom(8008) {
		if obj.VNum == 3003 {
			gifts++
		}
	}
	if gifts != 1 {
		t.Fatalf("gifts minted into the altar room = %d, want 1", gifts)
	}
	if got, want := ch.GetGold(), 1000-1-5; got != want {
		t.Fatalf("gold = %d, want %d (1 + cost 5)", got, want)
	}
	if got := out["Tester"]; !strings.Contains(got, "faint light") && !strings.Contains(got, "fades into existence") {
		t.Fatalf("output = %q, want the pray acts", got)
	}
}
