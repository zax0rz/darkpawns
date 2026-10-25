package game

// raw_kill strips a dying character before death_cry (src/fight.c:541-572):
// stop fighting, every affect, the tattoo, the nightbreed forms (vampire
// clamps mana), the mount, and every mob's memory of and hunt for the victim.
// Every player death entry point must run that sequence, not only
// combat.RawKill's callback bridge.

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// rawKillAffectsWorld builds a world whose player victim can be armed with
// raw_kill's full cleanup surface, plus a remembering, hunting mob.
func rawKillAffectsWorld(t *testing.T) (*World, *Player, *Player) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Charnel", Zone: 1}},
		Mobs:  []parser.Mob{{VNum: 2001, ShortDesc: "a carrion stalker"}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	victim := NewPlayer(51, "Doomed", 1001)
	victim.SetLevel(10)
	victim.SetMaxHP(100)
	victim.SetHP(-1)
	if err := w.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer victim: %v", err)
	}
	killer := NewPlayer(52, "Executioner", 1001)
	killer.SetLevel(15)
	if err := w.AddPlayer(killer); err != nil {
		t.Fatalf("AddPlayer killer: %v", err)
	}
	return w, victim, killer
}

// armDyingVictim gives the victim everything raw_kill must remove: a timed
// sanctuary record with its affect bit, a spider tattoo with a timer, both
// nightbreed bits with mana above max, and a mob that remembers and hunts the
// victim by name.
func armDyingVictim(t *testing.T, w *World, victim *Player) *MobInstance {
	t.Helper()
	victim.SetAffect(affSanctuary, true)
	victim.AddAffect(&engine.Affect{SpellID: 12, Duration: 10, Flags: engine.AFFSanctuary, Source: "test"})
	victim.Tattoo = TattooSpider
	victim.TatTimer = 5
	victim.SetAffect(affWerewolf, true)
	victim.SetAffect(affVampire, true)
	victim.SetMaxMana(50)
	victim.SetMana(90)

	mob, err := w.SpawnMob(2001, 1001)
	if err != nil {
		t.Fatalf("SpawnMob hunter: %v", err)
	}
	mob.SetMobFlag(MobFlagMemory)
	mob.Remember(victim.GetName())
	mob.SetHunting(victim.GetName())
	return mob
}

// assertRawKillStripped verifies every step of fight.c:541-572 ran.
func assertRawKillStripped(t *testing.T, victim *Player, hunter *MobInstance) {
	t.Helper()
	if victim.IsAffected(affSanctuary) {
		t.Error("sanctuary bit survived death (fight.c:544-545)")
	}
	for _, af := range victim.ActiveAffects {
		if af != nil && af.SpellID == 12 {
			t.Error("sanctuary affect record survived death (fight.c:544-545)")
		}
	}
	if victim.Tattoo != TattooNone {
		t.Errorf("tattoo survived death: %d (fight.c:547-552)", victim.Tattoo)
	}
	if victim.TatTimer != 0 {
		t.Errorf("tattoo timer survived death: %d (fight.c:551)", victim.TatTimer)
	}
	if victim.IsAffected(affWerewolf) {
		t.Error("AFF_WEREWOLF survived death (fight.c:554-555)")
	}
	if victim.IsAffected(affVampire) {
		t.Error("AFF_VAMPIRE survived death (fight.c:556-558)")
	}
	if victim.GetMana() > victim.GetMaxMana() {
		t.Errorf("mana %d above max %d after death (fight.c:559-560)", victim.GetMana(), victim.GetMaxMana())
	}
	if hunter.GetHunting() == victim.GetName() {
		t.Error("mob still hunts the dead victim (fight.c:570-571)")
	}
	if hunter.Forget(victim.GetName()) {
		t.Error("mob still remembered the dead victim (fight.c:568-569)")
	}
}

// TestHandleDeathRunsRawKillSequence is the ordinary combat death path
// (damage → die/die_with_killer → raw_kill): the one Egon reported.
func TestHandleDeathRunsRawKillSequence(t *testing.T) {
	w, victim, killer := rawKillAffectsWorld(t)
	hunter := armDyingVictim(t, w, victim)
	w.HandleDeath(victim, killer, 0)
	assertRawKillStripped(t, victim, hunter)
}

// TestCombatRawKillRunsRawKillSequence guards the combat engine's wire
// (fight_core RawKill → callbacks), which already ran the steps.
func TestCombatRawKillRunsRawKillSequence(t *testing.T) {
	w, victim, _ := rawKillAffectsWorld(t)
	hunter := armDyingVictim(t, w, victim)
	old := combat.GetCallbacks()
	combat.SetCallbacks(w.WireCombatCallbacks())
	t.Cleanup(func() { combat.SetCallbacks(old) })
	combat.RawKill(victim, 0)
	assertRawKillStripped(t, victim, hunter)
}

// TestInstakillRunsRawKillSequence covers do_kill's direct raw_kill call
// (src/act.offensive.c) and petrify's instakill.
func TestInstakillRunsRawKillSequence(t *testing.T) {
	w, victim, killer := rawKillAffectsWorld(t)
	hunter := armDyingVictim(t, w, victim)
	w.Instakill(victim, killer, combat.TYPE_SLASH)
	assertRawKillStripped(t, victim, hunter)
}

// TestRawKillCombatantPlayerRunsSequence covers the command-facing bridge
// (shoot, gates, disintegrate, croak, the Lua world_bridge arm).
func TestRawKillCombatantPlayerRunsSequence(t *testing.T) {
	w, victim, _ := rawKillAffectsWorld(t)
	hunter := armDyingVictim(t, w, victim)
	old := combat.GetCallbacks()
	combat.SetCallbacks(w.WireCombatCallbacks())
	t.Cleanup(func() { combat.SetCallbacks(old) })
	w.RawKillCombatant(victim, 0)
	assertRawKillStripped(t, victim, hunter)
}

// TestHandleMobDeathClearsMemoryAndHunting covers the mob-victim half of the
// class: other mobs must forget and stop hunting a mob that died through the
// HandleDeath path (raw_kill's loop is species-blind, fight.c:566-572).
func TestHandleMobDeathClearsMemoryAndHunting(t *testing.T) {
	w, _, _ := rawKillAffectsWorld(t)
	prey, err := w.SpawnMob(2001, 1001)
	if err != nil {
		t.Fatalf("SpawnMob prey: %v", err)
	}
	hunter, err := w.SpawnMob(2001, 1001)
	if err != nil {
		t.Fatalf("SpawnMob hunter: %v", err)
	}
	hunter.SetMobFlag(MobFlagMemory)
	hunter.Remember(prey.GetName())
	hunter.SetHunting(prey.GetName())

	w.HandleDeath(prey, nil, 0)

	if hunter.GetHunting() == prey.GetName() {
		t.Error("mob still hunts the dead mob victim (fight.c:570-571)")
	}
	if hunter.Forget(prey.GetName()) {
		t.Error("mob still remembered the dead mob victim (fight.c:568-569)")
	}
}
