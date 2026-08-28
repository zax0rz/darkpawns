package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/engine"
)

func TestDoFirstAidDepthSuccessState(t *testing.T) {
	ch := NewPlayer(1, "Aidder", 1001)
	ch.Level = LVL_IMMORT + 1
	ch.SetPosition(combat.PosStanding)
	ch.SetSkill(SkillFirstAid, 1)

	target := NewPlayer(2, "Downed", 1001)
	target.Level = 1
	target.SetHP(0)
	target.SetPosition(combat.PosStunned)

	result := DoFirstAid(ch, target)
	if !result.Success {
		t.Fatal("immortal first aid should succeed regardless of the roll")
	}
	if got := target.GetHP(); got != 1 {
		t.Fatalf("target HP = %d, want 1", got)
	}
	if got := target.GetPosition(); got != combat.PosStanding {
		t.Fatalf("target position = %d, want standing", got)
	}
	if got := result.WaitTarget; got != 1 {
		t.Fatalf("target wait rounds = %d, want 1", got)
	}
	if len(result.DeferredImprove) != 1 || result.DeferredImprove[0] != SkillFirstAid {
		t.Fatalf("deferred improvement = %#v, want first aid once", result.DeferredImprove)
	}
}

func TestDoFirstAidDepthFailureState(t *testing.T) {
	ch := NewPlayer(1, "Aidder", 1001)
	ch.Level = 1
	ch.SetPosition(combat.PosStanding)
	ch.SetSkill(SkillFirstAid, 1)

	target := NewPlayer(2, "Downed", 1001)
	target.Level = 40
	target.SetHP(0)
	target.SetPosition(combat.PosStunned)

	result := DoFirstAid(ch, target)
	if result.Success {
		t.Fatal("level-1 aid at skill 1 should fail against a level-40 target")
	}
	if got := target.GetHP(); got != 0 {
		t.Fatalf("target HP = %d after failure, want 0", got)
	}
	if got := result.WaitChPulses; got != engine.PULSE_VIOLENCE+3 {
		t.Fatalf("actor wait pulses = %d, want %d", got, engine.PULSE_VIOLENCE+3)
	}
	if !result.RoomIncludesTarget {
		t.Fatal("failure room message must include the target (C TO_ROOM)")
	}
}
