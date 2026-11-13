package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestRunSpecialsSwitchedActorIsMobBody pins the switched-actor dispatch
// (DP-1474): C's special() receives the descriptor's character — the switched
// mobile (src/interpreter.c:1407-1481 walks ch = d->character) — so the
// transport walk must offer every special that body as ch, never the owning
// player. The probe registers both lanes: the legacy lane must not run at all.
func TestRunSpecialsSwitchedActorIsMobBody(t *testing.T) {
	w, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1000, Name: "Void"}, {VNum: 1001, Name: "Hall"}},
		Mobs:  []parser.Mob{{VNum: 99001, Keywords: "generic", ShortDesc: "a generic", Level: 20}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	manager := newTestManager(t, w, nil)
	owner := game.NewPlayer(1, "Switchgod", 1000)
	owner.SetPosition(combat.PosStanding)
	if err := w.AddPlayer(owner); err != nil {
		t.Fatal(err)
	}
	mob, err := w.SpawnMob(99001, 1001)
	if err != nil {
		t.Fatal(err)
	}

	var actorSaw combat.Combatant
	var legacySaw *game.Player
	game.RegisterActorSpec("dp1474-switched-probe", func(_ *game.World, ctx game.CharacterSpecialContext, _ *game.MobInstance, _, _ string) game.SpecialResult {
		actorSaw = ctx.Actor
		return game.SpecialHandled
	})
	game.RegisterSpec("dp1474-switched-probe", func(_ *game.World, ch *game.Player, _ *game.MobInstance, _, _ string) bool {
		legacySaw = ch
		return true
	})
	game.RoomSpecAssign[1001] = "dp1474-switched-probe"
	t.Cleanup(func() {
		delete(game.RoomSpecAssign, 1001)
		delete(game.SpecRegistry, "dp1474-switched-probe")
	})

	s := &Session{manager: manager, player: owner, isSwitched: true, switchedMob: mob}
	if !runSpecials(characterCommandContextForSession(s), "south", nil, "") {
		t.Fatal("the probe special did not consume the command")
	}
	if legacySaw != nil {
		t.Fatalf("the legacy lane ran with %T %v; a switched command must not reach it", legacySaw, legacySaw)
	}
	if actorSaw != mob {
		t.Fatalf("the special received %T %v, want the switched mob %p", actorSaw, actorSaw, mob)
	}
}

// TestRunSpecialsSwitchedUnsupportedDoesNotTrap pins the SpecialUnsupported
// policy (DP-1474 follow-up): an unported NPC lane must not swallow a switched
// session's commands — including `return`, which would trap the immortal in
// the foreign body. Unsupported is treated as a decline (R4: the block was
// Go-invented player-visible behavior) and logged once per spec name.
func TestRunSpecialsSwitchedUnsupportedDoesNotTrap(t *testing.T) {
	w, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1000, Name: "Void"}, {VNum: 1001, Name: "Hall"}},
		Mobs:  []parser.Mob{{VNum: 99001, Keywords: "generic", ShortDesc: "a generic", Level: 20}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	manager := newTestManager(t, w, nil)
	owner := game.NewPlayer(1, "Switchgod", 1000)
	owner.SetPosition(combat.PosStanding)
	if err := w.AddPlayer(owner); err != nil {
		t.Fatal(err)
	}
	mob, err := w.SpawnMob(99001, 1001)
	if err != nil {
		t.Fatal(err)
	}

	// A legacy-only special whose name is absent from the NPC decline tables:
	// the dispatch reports SpecialUnsupported for every NPC command.
	game.RegisterSpec("dp1474-unsupported-probe", func(_ *game.World, _ *game.Player, _ *game.MobInstance, _, _ string) bool {
		return false
	})
	game.RoomSpecAssign[1001] = "dp1474-unsupported-probe"
	t.Cleanup(func() {
		delete(game.RoomSpecAssign, 1001)
		delete(game.SpecRegistry, "dp1474-unsupported-probe")
	})

	s := &Session{manager: manager, player: owner, isSwitched: true, switchedMob: mob}
	if runSpecials(characterCommandContextForSession(s), "return", nil, "") {
		t.Fatal("an unported NPC lane consumed `return`; a switched immortal is trapped in the body")
	}
	if runSpecials(characterCommandContextForSession(s), "south", nil, "") {
		t.Fatal("an unported NPC lane consumed `south`; the command must reach its handler")
	}
}
