package game

// spec_ninelives_test.go — ninelives port cover (src/spec_procs2.c:190-237).
// C declares the special without assigning it (src/spec_assign.c), so the
// registered handler is exercised directly; the registration lookup fails
// closed on origin/main.

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestSpecNineLivesRevivesOnPulse(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001, Name: "Room", Zone: 1}}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	spec := GetSpec("ninelives")
	if spec == nil {
		t.Fatal("ninelives is not registered")
	}
	victim := NewPlayer(1, "Ninevictim", 1001)
	if err := w.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	observer := NewPlayer(2, "Nineobserver", 1001)
	if err := w.AddPlayer(observer); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	me := newSpecProcTestMob(t, w, 1001, 10)
	me.SetMovePool(0, 3)
	me.SetHealth(0)
	me.SetFightingBody(victim)
	victim.SetFightingBody(me)
	ce := &testCombatEngine{}
	w.combatEngine = ce

	if !spec(w, nil, me, "", "") {
		t.Fatal("ninelives pulse did not revive")
	}
	// MAX_MOVE counts lives down (src/spec_procs2.c:220-224).
	if got := me.GetMaxMove(); got != 2 {
		t.Fatalf("max move = %d, want 2 after one life spent", got)
	}
	// GET_HIT = GET_MAX_HIT (src/spec_procs2.c:225).
	if got, want := me.GetHP(), me.GetMaxHP(); got != want {
		t.Fatalf("HP = %d, want full %d", got, want)
	}
	// The room hears the rise line (src/spec_procs2.c:228).
	var out strings.Builder
	w.MessageSink = func(_ string, msg []byte) { out.Write(msg) }
	me.SetHealth(0)
	me.SetFightingBody(victim)
	victim.SetFightingBody(me)
	if !spec(w, nil, me, "", "") {
		t.Fatal("second ninelives pulse did not revive")
	}
	if got := out.String(); !strings.Contains(got, "rises from the dead") {
		t.Fatalf("room output = %q, want the rise line", got)
	}
	// Both fights stopped, then hit() re-engaged mob-on-victim.
	if len(ce.starts) == 0 {
		t.Fatal("ninelives revive did not re-engage like C's hit()")
	}
	last := ce.starts[len(ce.starts)-1]
	if last.attacker != me.GetName() || last.defender != victim.GetName() {
		t.Fatalf("re-engage starts = %+v, want mob-on-victim", ce.starts)
	}
}

func TestSpecNineLivesMaxMoveCapsAtEight(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001, Name: "Room", Zone: 1}}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	spec := GetSpec("ninelives")
	if spec == nil {
		t.Fatal("ninelives is not registered")
	}
	victim := NewPlayer(1, "Ninevictim", 1001)
	if err := w.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	me := newSpecProcTestMob(t, w, 1001, 10)
	me.SetMovePool(0, 10)
	me.SetHealth(0)
	me.SetFightingBody(victim)
	w.combatEngine = &testCombatEngine{}
	if !spec(w, nil, me, "", "") {
		t.Fatal("ninelives pulse did not revive")
	}
	// GET_MAX_MOVE > 8 pins to 8 (src/spec_procs2.c:221-222).
	if got := me.GetMaxMove(); got != 8 {
		t.Fatalf("max move = %d, want the 8 cap", got)
	}
}

func TestSpecNineLivesChestGate(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001, Name: "Room", Zone: 1}}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	spec := GetSpec("ninelives")
	if spec == nil {
		t.Fatal("ninelives is not registered")
	}
	ch := NewPlayer(1, "Nineopener", 1001)
	if err := w.AddPlayer(ch); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	me := newSpecProcTestMob(t, w, 1001, 10)
	ce := &testCombatEngine{}
	w.combatEngine = ce

	// src/spec_procs2.c:205: strstr(argument, "chest") gates the arm.
	if spec(w, ch, me, "open", "box") {
		t.Fatal("open box should decline")
	}
	// strstr tolerates a trailing space, exactly like C.
	if !spec(w, ch, me, "open", "old chest ") {
		t.Fatal("open with chest substring should handle")
	}
	if len(ce.starts) != 1 || ce.starts[0].attacker != me.GetName() || ce.starts[0].defender != ch.GetName() {
		t.Fatalf("combat starts = %+v, want one mob-on-opener hit", ce.starts)
	}
	// C returns TRUE even when the hit's extra gates fail (src/spec_procs2.c:208-211).
	ch.SetPlrFlag(PrfNohassle, true)
	ce.starts = nil
	if !spec(w, ch, me, "look", "chest") {
		t.Fatal("look at chest should handle")
	}
	if len(ce.starts) != 0 {
		t.Fatalf("nohassle opener was hit: %+v", ce.starts)
	}
}
