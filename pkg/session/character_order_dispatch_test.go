package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestNPCOrderUsesSharedExecutorAndInterpreterDraws pins C do_order's mob lane
// (src/act.offensive.c:332,348-351): the ordered command reaches the same
// command_interpreter dispatch as a forced command — its leading number(0,3)
// draw, its gate ladder, and its handler — rather than a mobile-only adapter
// that skips the draw (R1/R3/R5e).
func TestNPCOrderUsesSharedExecutorAndInterpreterDraws(t *testing.T) {
	f := newActorPositionFixture(t, combat.PosStanding)
	caster := makeCommandTestSession(t, f.manager, "Ordergod", LVL_IMMORT, 1001)
	if err := f.world.AddPlayer(caster.player); err != nil {
		t.Fatal(err)
	}
	// The order gate requires a charmed follower whose master is the orderer
	// (src/act.offensive.c:326-330).
	f.actor.SetFollowing(caster.player.Name)
	f.actor.SetAffected(game.AffCharm)

	var draws int
	previous := characterCommandNumber
	characterCommandNumber = func(min, max int) int {
		draws++
		return 1
	}
	t.Cleanup(func() { characterCommandNumber = previous })

	if err := cmdOrder(caster, []string{"guard", "sit"}); err != nil {
		t.Fatalf("order returned error: %v", err)
	}
	if got := readMsgText(t, caster); got != "Okay.\r\n" {
		t.Fatalf("orderer bytes = %q, want acknowledgement", got)
	}
	if got := f.actor.GetPosition(); got != combat.PosSitting {
		t.Fatalf("ordered NPC position = %d, want sitting", got)
	}
	if draws != 1 {
		t.Fatalf("command interpreter draws = %d, want one per ordered command", draws)
	}
}

// TestNPCForcedMovementRunsPerformMove proves the force lane reaches do_move's
// perform_move leg for the six direction rows (src/interpreter.c:314-319;
// act.movement.c:361-366, 311-358): the room sees the C leave line and the
// body lands in the destination room. (do_order cannot prove this leg: C's
// charm gate refuses a charmed follower's move while its master shares the
// room, and an ordered follower is by definition charmed and co-located,
// src/act.movement.c:121-125.)
func TestNPCForcedMovementRunsPerformMove(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Hall"},
			{VNum: 1002, Name: "Yard", Exits: map[string]parser.Exit{}},
		},
		Mobs: []parser.Mob{{
			VNum: 99041, Keywords: "guard trainee", ShortDesc: "a guard trainee",
			Position: combat.PosStanding, DefaultPos: combat.PosStanding,
		}},
	}
	parsed.Rooms[0].Exits = map[string]parser.Exit{"north": {ToRoom: 1002}}
	parsed.Rooms[1].Exits["south"] = parser.Exit{ToRoom: 1001}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	manager := newTestManager(t, w, nil)
	manager.SetCommandExecFunc()
	observer := game.NewPlayer(1, "Observer", 1001)
	observer.SetPosition(combat.PosStanding)
	if err := w.AddPlayer(observer); err != nil {
		t.Fatal(err)
	}
	var roomOut strings.Builder
	w.MessageSink = func(name string, message []byte) {
		if name == observer.GetName() {
			roomOut.Write(message)
		}
	}
	actor, err := w.SpawnMob(99041, 1001)
	if err != nil {
		t.Fatal(err)
	}
	actor.SetPosition(combat.PosStanding)
	caster := makeCommandTestSession(t, manager, "Forcegod", LVL_GOD, 1001)
	if err := w.AddPlayer(caster.player); err != nil {
		t.Fatal(err)
	}

	if err := cmdForce(caster, []string{"guard", "north"}); err != nil {
		t.Fatalf("force north returned error: %v", err)
	}
	if got := readMsgText(t, caster); got != "Okay.\r\n" {
		t.Fatalf("caster bytes = %q, want acknowledgement", got)
	}
	if got := actor.GetRoom(); got != 1002 {
		t.Fatalf("forced NPC room = %d, want 1002", got)
	}
	if got, want := roomOut.String(), "A guard trainee leaves north.\r\n"; got != want {
		t.Fatalf("room bytes = %q, want %q", got, want)
	}
}
