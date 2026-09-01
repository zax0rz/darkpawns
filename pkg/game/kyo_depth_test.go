package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/engine"
)

func TestDoKujiKiri_KyoSuccessPreservesTwoCRecords(t *testing.T) {
	p := NewPlayer(1, "Kyo", 8162)
	p.Class = ClassNinja
	p.Level = 20
	p.SetSkill(SkillKkKyo, 100)
	dprng.ResetStream(1)

	result := DoKujiKiri(p, SkillKkKyo, nil)
	if !result.Success {
		t.Fatalf("Kyo should succeed at skill 100 with seed 1: %q", result.MessageToCh)
	}
	if result.MessageToCh != "Interlacing your fingers, you focus your battle rage." {
		t.Errorf("actor message = %q", result.MessageToCh)
	}
	if result.MessageToRoom != "Kyo interlaces his fingers and meditates deeply." {
		t.Errorf("room message = %q", result.MessageToRoom)
	}
	if got := p.GetHitroll(); got != 1 {
		t.Errorf("GetHitroll = %d, want 1 from successful Kyo", got)
	}

	if len(p.ActiveAffects) != 2 {
		t.Fatalf("active affects = %d, want C's two Kyo records", len(p.ActiveAffects))
	}
	want := map[int]int{ApplyHitroll: 1, ApplySpell: 0}
	for _, affect := range p.ActiveAffects {
		if affect == nil {
			t.Fatal("Kyo affect is nil")
		}
		wantMagnitude, ok := want[affect.Location]
		if !ok {
			t.Errorf("unexpected Kyo affect location %d", affect.Location)
			continue
		}
		if affect.SpellID != skillNumKkKyo || affect.Duration != 5 || affect.Magnitude != wantMagnitude || affect.Flags != engine.AFFKujiKiri {
			t.Errorf("Kyo affect = %+v, want spell %d duration 5 magnitude %d flags %d", affect, skillNumKkKyo, wantMagnitude, engine.AFFKujiKiri)
		}
		delete(want, affect.Location)
	}
	if len(want) != 0 {
		t.Errorf("missing Kyo affect locations: %v", want)
	}
}

func TestDoKujiKiri_KyoFailureKeepsPrimaryLockout(t *testing.T) {
	p := NewPlayer(1, "Kyo", 8162)
	p.Class = ClassNinja
	p.Level = 20
	p.SetSkill(SkillKkKyo, 1)
	dprng.ResetStream(1) // The first C number(1,101) is above skill 1.

	result := DoKujiKiri(p, SkillKkKyo, nil)
	if result.Success {
		t.Fatal("Kyo should fail at skill 1 with seed 1")
	}
	if result.MessageToCh != "You try the art of kuji-kiri, but can't concentrate!" {
		t.Errorf("failure message = %q", result.MessageToCh)
	}
	if result.MessageToRoom != "Kyo interlaces his fingers and meditates deeply." {
		t.Errorf("failure room message = %q", result.MessageToRoom)
	}
	if len(p.ActiveAffects) != 2 {
		t.Fatalf("active affects = %d, want C's separate failed Kyo records", len(p.ActiveAffects))
	}
	var sawHitroll, sawSpell bool
	for _, affect := range p.ActiveAffects {
		if affect == nil || affect.SpellID != skillNumKkKyo || affect.Duration != 5 || affect.Magnitude != 0 {
			t.Errorf("failure affect = %+v, want zeroed Kyo record", affect)
			continue
		}
		switch affect.Location {
		case ApplyHitroll:
			if affect.Flags != engine.AFFKujiKiri {
				t.Errorf("failed Kyo hitroll flags = %d, want %d", affect.Flags, engine.AFFKujiKiri)
			}
			sawHitroll = true
		case ApplySpell:
			if affect.Flags != engine.AFFNothing {
				t.Errorf("failed Kyo spell flags = %d, want %d", affect.Flags, engine.AFFNothing)
			}
			sawSpell = true
		default:
			t.Errorf("unexpected failed Kyo location %d", affect.Location)
		}
	}
	if !sawHitroll || !sawSpell {
		t.Fatalf("failed Kyo records = %+v, want hitroll and spell", p.ActiveAffects)
	}
	if !p.IsAffected(affKujiKiri) {
		t.Fatal("failed Kyo should retain the C aggregate lockout through af[0]")
	}
}
