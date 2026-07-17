package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestMobGetHitroll_DerivesCPointsHitrollFromFileTHAC0(t *testing.T) {
	mob := &MobInstance{
		Prototype: &parser.Mob{THAC0: 19},
		Equipment: make(map[int]*ObjectInstance),
	}

	if got := mob.GetHitroll(); got != 1 {
		t.Fatalf("mob hitroll for file THAC0 19 = %d, want C's 20-19 = 1", got)
	}
}
