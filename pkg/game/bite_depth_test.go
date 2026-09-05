package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/dprng"
)

func newBiteDepthPlayers() (*Player, *Player) {
	ch := NewPlayer(1, "Biter", 1001)
	victim := NewPlayer(2, "Victim", 1001)
	ch.Level = 20
	victim.Level = 8
	return ch, victim
}

func TestDoBite_LoveBiteIsLiteralNoDamage(t *testing.T) {
	ch, victim := newBiteDepthPlayers()
	result := DoBite(ch, victim)

	if !result.Success || result.Damage != 0 || result.SkillMsgType != 0 {
		t.Fatalf("love bite result = success %v, damage %d, skill message %d; want true, 0, 0", result.Success, result.Damage, result.SkillMsgType)
	}
	if result.MessageToCh != "You give Victim a love bite." {
		t.Errorf("actor message = %q", result.MessageToCh)
	}
	if result.MessageToVict != "Biter tries to give you a little love bite." {
		t.Errorf("victim message = %q", result.MessageToVict)
	}
	if result.MessageToRoom != "Biter gives Victim a love bite." {
		t.Errorf("room message = %q", result.MessageToRoom)
	}
}

func TestDoBite_SupernaturalEntryAndVictimGates(t *testing.T) {
	ch, victim := newBiteDepthPlayers()
	if result := DoBite(ch, ch); result.MessageToCh != "You bite your tongue and say nothing." {
		t.Errorf("self-target message = %q", result.MessageToCh)
	}

	ch.SetPlrFlag(PlrWerewolf, true)

	if result := DoBite(ch, victim); result.MessageToCh != "You must be transformed to bite!" {
		t.Errorf("untransformed message = %q", result.MessageToCh)
	}

	ch.SetAffect(affWerewolf, true)
	victim.Level = LVL_IMMORT + 1
	ch.Level = LVL_IMMORT
	if result := DoBite(ch, victim); result.MessageToCh != "Yeah, right." {
		t.Errorf("higher immortal message = %q", result.MessageToCh)
	}

	victim.Level = 8
	victim.SetPlrFlag(PlrVampire, true)
	if result := DoBite(ch, victim); result.MessageToCh != "Your victim is already a creature of the night!" {
		t.Errorf("night-creature victim message = %q", result.MessageToCh)
	}
}

func TestDoBite_WerewolfDamageContract(t *testing.T) {
	ch, victim := newBiteDepthPlayers()
	ch.SetPlrFlag(PlrWerewolf, true)
	ch.SetAffect(affWerewolf, true)

	result := DoBite(ch, victim)
	if !result.Success || result.Damage != 15 || result.DamageSkill != SkillBite ||
		result.SkillMsgType != SkillBiteNum || !result.SkillMsgAfterDamage ||
		!result.StartCombat || result.WaitCh != 2 {
		t.Fatalf("werewolf result = %+v", result)
	}
	if result.MessageToCh != "You rip the flesh of Victim, and blood pours over your lips!" {
		t.Errorf("actor message = %q", result.MessageToCh)
	}
}

func TestDoBite_VampireFeedsOnSloppyAndFightingBites(t *testing.T) {
	ch, victim := newBiteDepthPlayers()
	ch.SetPlrFlag(PlrVampire, true)
	ch.SetAffect(affVampire, true)
	ch.SetCondition(CondFull, 20)
	ch.SetCondition(CondThirst, 20)

	// Seed 1 produces a nonzero number(0,10), so this is the sloppy/no-damage
	// arm and still feeds from the victim's level.
	dprng.ResetStream(1)
	sloppy := DoBite(ch, victim)
	if !sloppy.Success || sloppy.Damage != 0 || sloppy.SkillMsgType != 0 || sloppy.WaitCh != 0 {
		t.Fatalf("sloppy result = %+v", sloppy)
	}
	if ch.GetCondition(CondFull) != 28 || ch.GetCondition(CondThirst) != 28 {
		t.Fatalf("sloppy feeding conditions = (%d, %d), want (28, 28)", ch.GetCondition(CondFull), ch.GetCondition(CondThirst))
	}

	// Find a seed whose first vampire bite roll is zero, then verify the
	// fighting bite enters damage() with the C skill-message and wait contract.
	var seed uint32
	for candidate := uint32(1); candidate < 1000; candidate++ {
		dprng.ResetStream(candidate)
		if dprng.Number(0, ch.GetLevel()/2) == 0 {
			seed = candidate
			break
		}
	}
	if seed == 0 {
		t.Fatal("could not find a seed for the vampire fighting-bite arm")
	}
	dprng.ResetStream(seed)
	fighting := DoBite(ch, victim)
	if !fighting.Success || fighting.Damage != 15 || fighting.SkillMsgType != SkillBiteNum ||
		!fighting.SkillMsgAfterDamage || !fighting.StartCombat || fighting.WaitCh != 2 {
		t.Fatalf("fighting result = %+v", fighting)
	}
	if fighting.MessageToRoom == "" || fighting.MessageToVict == "" {
		t.Fatalf("fighting audience messages missing: %+v", fighting)
	}
}
