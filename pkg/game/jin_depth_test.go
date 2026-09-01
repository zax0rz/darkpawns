package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/engine"
)

func TestSkillStorageName_JinUsesCommandSkillKey(t *testing.T) {
	if got := SkillStorageName(skillNumKkJin); got != SkillKkJin {
		t.Fatalf("SkillStorageName(%d) = %q, want %q", skillNumKkJin, got, SkillKkJin)
	}
}

func TestDoKujiKiri_JinSuccessPreservesCDefaultSecondAffect(t *testing.T) {
	p := NewPlayer(1, "Jin", 8162)
	p.Class = ClassNinja
	p.Level = 20
	p.SetSkill(SkillKkJin, 100)
	dprng.ResetStream(1)

	result := DoKujiKiri(p, SkillKkJin, nil)
	if !result.Success {
		t.Fatalf("Jin should succeed at skill 100 with seed 1: %q", result.MessageToCh)
	}
	if result.MessageToCh != "Interlacing your fingers, you focus on recooperation." {
		t.Errorf("actor message = %q", result.MessageToCh)
	}
	if result.MessageToRoom != "Jin interlaces his fingers and meditates deeply." {
		t.Errorf("room message = %q", result.MessageToRoom)
	}
	if len(p.ActiveAffects) != 1 {
		t.Fatalf("active affects = %d, want C's joined Jin lockout record", len(p.ActiveAffects))
	}
	for i, affect := range p.ActiveAffects {
		if affect == nil {
			t.Fatalf("affect %d is nil", i)
		}
		if affect.SpellID != skillNumKkJin || affect.Location != ApplySpell || affect.Duration != 5 || affect.Magnitude != 0 || affect.Flags != engine.AFFKujiKiri {
			t.Errorf("affect %d = (spell %d, location %d, duration %d, magnitude %d, flags %d), want (%d, %d, 5, 0, %d)",
				i, affect.SpellID, affect.Location, affect.Duration, affect.Magnitude, affect.Flags,
				skillNumKkJin, ApplySpell, engine.AFFKujiKiri)
		}
	}
}

func TestDoKujiKiri_JinFailureJoinsAsInertRecord(t *testing.T) {
	p := NewPlayer(1, "Jin", 8162)
	p.Class = ClassNinja
	p.Level = 20
	p.SetSkill(SkillKkJin, 1)
	dprng.ResetStream(1) // The first C number(1,101) is above skill 1.

	result := DoKujiKiri(p, SkillKkJin, nil)
	if result.Success {
		t.Fatal("Jin should fail at skill 1 with seed 1")
	}
	if result.MessageToCh != "You try the art of kuji-kiri, but can't concentrate!" {
		t.Errorf("failure message = %q", result.MessageToCh)
	}
	if len(p.ActiveAffects) != 1 {
		t.Fatalf("active affects = %d, want C's joined inert Jin record after failure", len(p.ActiveAffects))
	}
	affect := p.ActiveAffects[0]
	if affect.SpellID != skillNumKkJin || affect.Location != ApplySpell || affect.Duration != 5 || affect.Magnitude != 0 || affect.Flags != engine.AFFNothing {
		t.Errorf("failure affect = %+v, want joined AFF_NOTHING record", affect)
	}
	if p.IsAffected(affKujiKiri) {
		t.Error("failed Jin must not set the C aggregate AFF_KUJI_KIRI flag")
	}
}

func TestDoKujiKiri_JinUnmasteredConsumesCSuccessDraw(t *testing.T) {
	const seed uint32 = 1

	wantStream := dprng.New(seed)
	wantStream.Number(1, 101) // C check_kk_success() runs before the mastery gate.
	wantNext := wantStream.Number(0, 999)

	p := NewPlayer(1, "Jin", 8162)
	p.Class = ClassNinja
	p.Level = 20
	dprng.ResetStream(seed)
	result := DoKujiKiri(p, SkillKkJin, nil)
	if result.Success || result.MessageToCh != "You have not mastered this art yet!" {
		t.Fatalf("unmastered Jin result = %+v", result)
	}
	if got := dprng.Number(0, 999); got != wantNext {
		t.Errorf("next RNG draw = %d, want %d after C's pre-gate success draw", got, wantNext)
	}
}

func TestDoKujiKiri_JinGuardMessages(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*Player)
		want  string
	}{
		{
			name: "class",
			setup: func(p *Player) {
				p.Class = ClassWarrior
				p.Level = 20
				p.SetSkill(SkillKkJin, 100)
			},
			want: "You know nothing of kuji-kiri!",
		},
		{
			name: "fighting",
			setup: func(p *Player) {
				p.Class = ClassNinja
				p.Level = 20
				p.SetSkill(SkillKkJin, 100)
				p.Fighting = "a training dummy"
			},
			want: "You are too busy fighting to practice kuji-kiri!",
		},
		{
			name: "lockout",
			setup: func(p *Player) {
				p.Class = ClassNinja
				p.Level = 20
				p.SetSkill(SkillKkJin, 100)
				p.SetAffect(affKujiKiri, true)
			},
			want: "You can not practice kuji-kiri again right now!",
		},
		{
			name: "mounted",
			setup: func(p *Player) {
				p.Class = ClassNinja
				p.Level = 20
				p.SetSkill(SkillKkJin, 100)
				p.MountName = "a warhorse"
			},
			want: "Dismount first!",
		},
		{
			name: "unmastered",
			setup: func(p *Player) {
				p.Class = ClassNinja
				p.Level = 20
			},
			want: "You have not mastered this art yet!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPlayer(1, "Jin", 8162)
			tt.setup(p)
			result := DoKujiKiri(p, SkillKkJin, nil)
			if result.Success || result.MessageToCh != tt.want {
				t.Fatalf("result = %+v, want failure %q", result, tt.want)
			}
		})
	}
}
