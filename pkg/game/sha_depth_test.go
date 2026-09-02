package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/dprng"
)

func TestDoKujiKiriShaUsesLowLevelHealFloor(t *testing.T) {
	p := NewPlayer(1, "ShaNinja", 3001)
	p.Class = ClassNinja
	p.SetLevel(10)
	p.SetSkill(SkillKkSha, 100)
	p.SetMaxHP(100)
	p.SetHP(1)

	for seed := uint32(1); seed <= 50; seed++ {
		dprng.ResetStream(seed)
		result := DoKujiKiri(p, SkillKkSha, nil)
		if !result.Success {
			p.RemoveAffectBit(affKujiKiri)
			continue
		}
		if got := p.GetHP(); got != 16 {
			t.Errorf("level-10 Sha HP = %d, want 16 (15-point floor)", got)
		}
		return
	}
	t.Fatal("Sha never succeeded at skill 100 across 50 seeds")
}

func TestDoKujiKiriShaCapsAtMaxHP(t *testing.T) {
	p := NewPlayer(1, "ShaNinja", 3001)
	p.Class = ClassNinja
	p.SetLevel(20)
	p.SetSkill(SkillKkSha, 100)
	p.SetMaxHP(100)
	p.SetHP(95)

	for seed := uint32(1); seed <= 50; seed++ {
		dprng.ResetStream(seed)
		result := DoKujiKiri(p, SkillKkSha, nil)
		if !result.Success {
			p.RemoveAffectBit(affKujiKiri)
			continue
		}
		if got := p.GetHP(); got != 100 {
			t.Errorf("capped Sha HP = %d, want 100", got)
		}
		return
	}
	t.Fatal("Sha never succeeded at skill 100 across 50 seeds")
}

func TestDoKujiKiriShaFailureLeavesNoLockout(t *testing.T) {
	p := NewPlayer(1, "ShaNinja", 3001)
	p.Class = ClassNinja
	p.SetLevel(20)
	p.SetSkill(SkillKkSha, 1)
	p.SetMaxHP(100)
	p.SetHP(10)
	dprng.ResetStream(1)

	result := DoKujiKiri(p, SkillKkSha, nil)
	if result.Success {
		t.Fatal("seed 1 should fail at Sha skill 1")
	}
	if result.MessageToCh != "You try the art of kuji-kiri, but can't concentrate!" {
		t.Errorf("failure message = %q", result.MessageToCh)
	}
	if p.GetHP() != 10 {
		t.Errorf("failed Sha changed HP to %d, want 10", p.GetHP())
	}
	if p.IsAffected(affKujiKiri) {
		t.Fatal("failed Sha retained the aggregate kuji-kiri lockout")
	}
}
