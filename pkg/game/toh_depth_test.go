package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/engine"
)

// C: check_kk_success() / do_kuji_kiri() (src/new_cmds.c:1500-1739).
func TestDoKujiKiri_TohSuccessPreservesTwoCRecords(t *testing.T) {
	p := NewPlayer(1, "Toh", 8162)
	p.Class = ClassNinja
	p.Level = 20
	p.SetSkill(SkillKkToh, 100)
	p.AC = 0
	dprng.ResetStream(1)

	result := DoKujiKiri(p, SkillKkToh, nil)
	if !result.Success {
		t.Fatalf("Toh should succeed at skill 100 with seed 1: %q", result.MessageToCh)
	}
	if result.MessageToCh != "Interlacing your fingers, you focus your inner strength." {
		t.Errorf("actor message = %q", result.MessageToCh)
	}
	if result.MessageToRoom != "Toh interlaces his fingers and meditates deeply." {
		t.Errorf("room message = %q", result.MessageToRoom)
	}
	if got := p.GetDamroll(); got != 1 {
		t.Errorf("GetDamroll = %d, want 1 from successful Toh", got)
	}
	if got := p.GetAC(); got != 10 {
		t.Errorf("GetAC = %d, want 10 from successful Toh", got)
	}

	if len(p.ActiveAffects) != 2 {
		t.Fatalf("active affects = %d, want C's two Toh records", len(p.ActiveAffects))
	}
	want := map[int]int{ApplyDamroll: 1, ApplyAC: 10}
	for _, affect := range p.ActiveAffects {
		if affect == nil {
			t.Fatal("Toh affect is nil")
		}
		wantMagnitude, ok := want[affect.Location]
		if !ok {
			t.Errorf("unexpected Toh affect location %d", affect.Location)
			continue
		}
		if affect.SpellID != skillNumKkToh || affect.Duration != 5 || affect.Magnitude != wantMagnitude || affect.Flags != engine.AFFKujiKiri {
			t.Errorf("Toh affect = %+v, want spell %d duration 5 magnitude %d flags %d", affect, skillNumKkToh, wantMagnitude, engine.AFFKujiKiri)
		}
		delete(want, affect.Location)
	}
	if len(want) != 0 {
		t.Errorf("missing Toh affect locations: %v", want)
	}
}
