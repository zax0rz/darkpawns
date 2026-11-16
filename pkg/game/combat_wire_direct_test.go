package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestWorldCombatWimpyFleeUsesMobileDirectCommand(t *testing.T) {
	w, _, mob, _ := newNPCCommandWorld(t)
	var gotActor combat.Combatant
	var gotCommand, gotArgument string
	w.CharacterCommandDirectExecFunc = func(actor combat.Combatant, command, argument string, _ int) CharacterCommandResult {
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

func TestCombatWeaponDescriptionIncludesMobileWeapon(t *testing.T) {
	w, _, mob, _ := newNPCCommandWorld(t)
	weapon := NewObjectInstance(&parser.Obj{
		VNum:      2002,
		Keywords:  "dagger",
		ShortDesc: "a narrow dagger",
	}, -1)
	weapon.Location = LocEquippedMob(mob.GetID(), EquipmentSlot(mobWearWield))
	mob.mu.Lock()
	mob.Equipment = map[int]*ObjectInstance{mobWearWield: weapon}
	mob.mu.Unlock()

	description := w.WireCombatCallbacks().GetWeaponDescription(mob)
	if description != "a narrow dagger" {
		t.Fatalf("mobile weapon description = %q, want %q", description, "a narrow dagger")
	}
}
