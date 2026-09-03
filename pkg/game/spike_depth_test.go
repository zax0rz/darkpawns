package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newSpikeMobWorld(t *testing.T) (*World, *Player, *MobInstance) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Test Room", Zone: 1}},
		Mobs: []parser.Mob{{
			VNum:       2001,
			Keywords:   "werewolf dummy",
			ShortDesc:  "a werewolf dummy",
			Level:      5,
			Position:   combat.PosStanding,
			DefaultPos: combat.PosStanding,
		}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	ch := NewPlayer(1, "Hunter", 1001)
	ch.Level = 10
	ch.Class = ClassWarrior
	ch.Race = RaceHuman
	ch.Inventory = NewInventory()
	ch.Equipment = NewEquipment()
	if err := w.AddPlayer(ch); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	mob, err := w.SpawnMobQuiet(2001, 1001)
	if err != nil {
		t.Fatalf("SpawnMobQuiet: %v", err)
	}
	mob.SetAffected(affWerewolf)
	return w, ch, mob
}

func spikePlayerTarget(t *testing.T, w *World, level int) *Player {
	t.Helper()
	victim := NewPlayer(2, "Wolf", 1001)
	victim.Level = level
	victim.SetAffect(affWerewolf, true)
	if err := w.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer victim: %v", err)
	}
	return victim
}

func TestDoSpike_ImmortalTargetGate(t *testing.T) {
	w, ch := newSpikeTestWorld(t)
	victim := spikePlayerTarget(t, w, LVL_IMMORT)
	equipWeapon(t, ch, makeSpikeWeapon("spike"))

	result := DoSpike(ch, victim, 0, w)
	if result.Success {
		t.Fatal("mortal spike attacker should not destroy an immortal target")
	}
	if result.MessageToCh != "Yeah, right.\r\n" {
		t.Errorf("immortal-target message = %q, want exact C text", result.MessageToCh)
	}
}

func TestDoSpike_UsesObjectKeywords(t *testing.T) {
	w, ch := newSpikeTestWorld(t)
	victim := spikePlayerTarget(t, w, 5)
	weapon := makeSpikeWeapon("spike")
	weapon.Prototype.ShortDesc = "a plain carving"
	equipWeapon(t, ch, weapon)

	result := DoSpike(ch, victim, 0, w)
	if !result.Success {
		t.Fatalf("keyword-matching spike should succeed despite short description, got %q", result.MessageToCh)
	}
}

func TestDoSpike_MobTarget(t *testing.T) {
	w, ch, victim := newSpikeMobWorld(t)
	equipWeapon(t, ch, makeSpikeWeapon("spike"))

	result := DoSpike(ch, victim, 0, w)
	if !result.Success {
		t.Fatalf("expected spike against affected NPC to succeed, got %q", result.MessageToCh)
	}
	if !result.RawKill {
		t.Fatal("successful spike must request the C raw_kill tail")
	}
	if !strings.Contains(result.MessageToCh, "drive") {
		t.Errorf("success message = %q, want authored drive act", result.MessageToCh)
	}
	if got := len(w.GetMobsInRoom(victim.GetRoom())); got != 1 {
		t.Fatalf("DoSpike extracted NPC before authored acts were sent: active mobs = %d, want 1", got)
	}

	w.RawKillCombatant(victim, combat.TYPE_UNDEFINED)
	if victim.IsAlive() {
		t.Fatal("RawKillCombatant left the NPC alive")
	}
	if got := len(w.GetMobsInRoom(1001)); got != 0 {
		t.Fatalf("RawKillCombatant left %d NPCs in room, want 0", got)
	}
}

func TestDoSpike_MissWaitState(t *testing.T) {
	w, ch := newSpikeTestWorld(t)
	ch.Level = 0
	victim := spikePlayerTarget(t, w, 30)
	equipWeapon(t, ch, makeSpikeWeapon("spike"))

	var seed uint32
	for candidate := uint32(1); candidate < 100000; candidate++ {
		if dprng.New(candidate).Number(0, LVL_IMMORT) == 0 {
			seed = candidate
			break
		}
	}
	if seed == 0 {
		t.Fatal("could not find a deterministic spike miss seed")
	}
	dprng.ResetStream(seed)
	result := DoSpike(ch, victim, 0, w)
	if result.Success {
		t.Fatalf("seed %d unexpectedly succeeded: %q", seed, result.MessageToCh)
	}
	if result.WaitCh != 2 {
		t.Errorf("miss WaitCh = %d, want 2 violence pulses", result.WaitCh)
	}
}

func TestDoSpike_SuccessProbability(t *testing.T) {
	w, ch := newSpikeTestWorld(t)
	ch.Level = 0
	victim := spikePlayerTarget(t, w, 30)
	equipWeapon(t, ch, makeSpikeWeapon("spike"))

	// C calls number(0, LEVEL_IMMORT), inclusive. With a level gap of exactly
	// 30, only the inclusive upper draw can pass this arm.
	var seed uint32
	for candidate := uint32(1); candidate < 100000; candidate++ {
		if dprng.New(candidate).Number(0, LVL_IMMORT) == LVL_IMMORT {
			seed = candidate
			break
		}
	}
	if seed == 0 {
		t.Fatal("could not find a deterministic inclusive-upper-bound seed")
	}
	dprng.ResetStream(seed)
	result := DoSpike(ch, victim, 0, w)
	if !result.Success {
		t.Fatalf("seed %d did not use inclusive number(0,LEVEL_IMMORT): %q", seed, result.MessageToCh)
	}
}
