package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestOrderRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["order"]
	if !ok {
		t.Fatal("order command has no C gate")
	}
	if gate.MinLevel != 1 || gate.MinPosition != combat.PosResting {
		t.Fatalf("order gate = level %d position %d, want level 1 position %d", gate.MinLevel, gate.MinPosition, combat.PosResting)
	}

	entry, ok := cmdRegistry.Lookup("order")
	if !ok {
		t.Fatal("order command is not registered")
	}
	if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
		t.Fatalf("order registry gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
	}
}

func TestOrderCharmedPlayerExecutesExactCommandPath(t *testing.T) {
	m := makeTestManager(t)
	leader := makeTestSession(t, m, "Orderleader", 1001, true)
	target := makeTestSession(t, m, "Ordertarget", 1001, true)
	observer := makeTestSession(t, m, "Orderobserver", 1001, true)
	registerInWorld(t, leader)
	registerInWorld(t, target)
	registerInWorld(t, observer)

	target.player.SetFollowing(leader.player.Name)
	target.player.SetAffect(game.AffCharm, true)
	var calledPlayer *game.Player
	var calledCommand string
	m.world.CommandExecFunc = func(player *game.Player, command string) bool {
		calledPlayer = player
		calledCommand = command
		return true
	}

	if err := cmdOrder(leader, []string{"Ordertarget", "say", "hello"}); err != nil {
		t.Fatalf("cmdOrder: %v", err)
	}
	if calledPlayer != target.player || calledCommand != "say hello" {
		t.Fatalf("forced command = (%v, %q), want (%v, %q)", calledPlayer, calledCommand, target.player, "say hello")
	}
	if got := readSessionText(t, leader); !strings.Contains(got, "Okay.\r\n") {
		t.Fatalf("leader output = %q, want C OK response", got)
	}
	if got := readSessionText(t, target); !strings.Contains(got, "Orderleader orders you to 'say hello'\r\n") {
		t.Fatalf("target output = %q, want C order text", got)
	}
	if got := readSessionText(t, observer); !strings.Contains(got, "Orderleader gives Ordertarget an order.\r\n") {
		t.Fatalf("observer output = %q, want C room announcement", got)
	}

	if err := cmdOrder(leader, []string{"followers", "say", "hello"}); err != nil {
		t.Fatalf("cmdOrder followers: %v", err)
	}
	if calledPlayer != target.player || calledCommand != "say hello" {
		t.Fatalf("followers forced command = (%v, %q), want (%v, %q)", calledPlayer, calledCommand, target.player, "say hello")
	}
	if got := readSessionText(t, leader); !strings.Contains(got, "Okay.\r\n") {
		t.Fatalf("followers leader output = %q, want C OK response", got)
	}
	if got := readSessionText(t, observer); !strings.Contains(got, "Orderleader issues the order 'say hello'.\r\n") {
		t.Fatalf("followers observer output = %q, want C room announcement", got)
	}
}

func TestOrderRejectsCharmedActorWithCMessage(t *testing.T) {
	m := makeTestManager(t)
	actor := makeTestSession(t, m, "Orderactor", 1001, true)
	target := makeTestSession(t, m, "Ordertarget", 1001, true)
	registerInWorld(t, actor)
	registerInWorld(t, target)
	actor.player.SetAffect(game.AffCharm, true)

	if err := cmdOrder(actor, []string{"Ordertarget", "say", "hello"}); err != nil {
		t.Fatalf("cmdOrder: %v", err)
	}
	if got := readSessionText(t, actor); got != "Your superior would not aprove of you giving orders.\r\n" {
		t.Fatalf("charmed actor output = %q, want C message", got)
	}
	select {
	case got := <-target.send:
		t.Fatalf("target received output despite actor charm gate: %q", got)
	default:
	}
}
