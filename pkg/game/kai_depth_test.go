package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/engine"
)

func TestSkillStorageName_AllKujiSealsUseCommandSkillKeys(t *testing.T) {
	tests := []struct {
		num  int
		name string
		want string
	}{
		{skillNumKkRin, "kuji-kiri rin", SkillKkRin},
		{skillNumKkKyo, "kuji-kiri kyo", SkillKkKyo},
		{skillNumKkToh, "kuji-kiri toh", SkillKkToh},
		{skillNumKkKai, "kuji-kiri kai", SkillKkKai},
		{skillNumKkJin, "kuji-kiri jin", SkillKkJin},
		{skillNumKkRetsu, "kuji-kiri retsu", SkillKkRetsu},
		{skillNumKkZai, "kuji-kiri zai", SkillKkZai},
		{skillNumKkZhen, "kuji-kiri zhen", SkillKkZhen},
		{skillNumKkSha, "kuji-kiri sha", SkillKkSha},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SkillStorageName(tt.num); got != tt.want {
				t.Fatalf("SkillStorageName(%d) = %q, want %q", tt.num, got, tt.want)
			}
		})
	}
}

func TestDoKujiKiri_KaiSuccessPreservesTwoCRecords(t *testing.T) {
	p := NewPlayer(1, "Kai", 8162)
	p.Class = ClassNinja
	p.Level = 20
	p.SetSkill(SkillKkKai, 100)
	dprng.ResetStream(1)

	result := DoKujiKiri(p, SkillKkKai, nil)
	if !result.Success {
		t.Fatalf("Kai should succeed at skill 100 with seed 1: %q", result.MessageToCh)
	}
	if result.MessageToCh != "Interlacing your fingers, your body becomes your fortress." {
		t.Errorf("actor message = %q", result.MessageToCh)
	}
	if result.MessageToRoom != "Kai interlaces his fingers and meditates deeply." {
		t.Errorf("room message = %q", result.MessageToRoom)
	}
	if len(p.ActiveAffects) != 2 {
		t.Fatalf("active affects = %d, want C's two Kai records", len(p.ActiveAffects))
	}
	want := map[int]int{ApplyDamroll: -1, ApplyAC: -10}
	for _, affect := range p.ActiveAffects {
		if affect == nil {
			t.Fatal("Kai affect is nil")
		}
		wantMagnitude, ok := want[affect.Location]
		if !ok {
			t.Errorf("unexpected Kai affect location %d", affect.Location)
			continue
		}
		if affect.SpellID != skillNumKkKai || affect.Duration != 5 || affect.Magnitude != wantMagnitude || affect.Flags != engine.AFFKujiKiri {
			t.Errorf("Kai affect = %+v, want spell %d duration 5 magnitude %d flags %d", affect, skillNumKkKai, wantMagnitude, engine.AFFKujiKiri)
		}
		delete(want, affect.Location)
	}
	if len(want) != 0 {
		t.Errorf("missing Kai affect locations: %v", want)
	}
}

func TestDoKujiKiri_KaiFailureKeepsPrimaryLockout(t *testing.T) {
	p := NewPlayer(1, "Kai", 8162)
	p.Class = ClassNinja
	p.Level = 20
	p.SetSkill(SkillKkKai, 1)
	dprng.ResetStream(1) // The first C number(1,101) is above skill 1.

	result := DoKujiKiri(p, SkillKkKai, nil)
	if result.Success {
		t.Fatal("Kai should fail at skill 1 with seed 1")
	}
	if len(p.ActiveAffects) != 2 {
		t.Fatalf("active affects = %d, want C's separate failed Kai records", len(p.ActiveAffects))
	}
	var sawDamroll, sawAC bool
	for _, affect := range p.ActiveAffects {
		if affect == nil || affect.SpellID != skillNumKkKai || affect.Duration != 5 || affect.Magnitude != 0 {
			t.Errorf("failure affect = %+v, want zeroed Kai record", affect)
			continue
		}
		switch affect.Location {
		case ApplyDamroll:
			if affect.Flags != engine.AFFKujiKiri {
				t.Errorf("failed Kai damroll flags = %d, want %d", affect.Flags, engine.AFFKujiKiri)
			}
			sawDamroll = true
		case ApplyAC:
			if affect.Flags != engine.AFFNothing {
				t.Errorf("failed Kai AC flags = %d, want %d", affect.Flags, engine.AFFNothing)
			}
			sawAC = true
		default:
			t.Errorf("unexpected failed Kai location %d", affect.Location)
		}
	}
	if !sawDamroll || !sawAC {
		t.Fatalf("failed Kai records = %+v, want damroll and AC", p.ActiveAffects)
	}
	if !p.IsAffected(affKujiKiri) {
		t.Fatal("failed Kai should retain the C aggregate lockout through af[0]")
	}
}
