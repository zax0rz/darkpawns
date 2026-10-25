package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestWorldCombatWimpyFleeUsesMobileDirectCommand(t *testing.T) {
	w, _, mob, _ := newNPCCommandWorld(t)
	var gotActor combat.Combatant
	var gotCommand, gotArgument string
	w.CharacterCommandDirectExecFunc = func(actor combat.Combatant, command, argument string) CharacterCommandResult {
		gotActor, gotCommand, gotArgument = actor, command, argument
		return CharacterCommandHandled
	}

	callbacks := w.WireCombatCallbacks()
	if callbacks.DoFlee == nil {
		t.Fatal("combat callbacks have no direct mobile flee route")
	}
	callbacks.DoFlee(mob)
	if gotActor != mob || gotCommand != "flee" || gotArgument != "" {
		t.Fatalf("mobile flee dispatch = (%v, %q, %q), want (%v, flee, empty)", gotActor, gotCommand, gotArgument, mob)
	}
}
