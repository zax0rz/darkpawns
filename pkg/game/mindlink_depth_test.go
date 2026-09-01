package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newMindlinkGateFixture() (*Player, *MobInstance) {
	ch := NewPlayer(1, "Mindlinker", 1001)
	ch.SetSkill(SkillMindlink, 100)
	ch.SetHP(200)
	ch.SetPosition(combat.PosStanding)
	mob := NewMob(&parser.Mob{VNum: 2003, ShortDesc: "a full-mana dummy"}, 1001)
	mob.SetMana(100)
	mob.SetStatus("standing")
	return ch, mob
}

// TestDoMindlinkDepthGates covers the C order after a valid non-player target
// has been resolved (new_cmds2.c:270-289).
func TestDoMindlinkDepthGates(t *testing.T) {
	t.Run("no skill", func(t *testing.T) {
		ch, mob := newMindlinkGateFixture()
		ch.SetSkill(SkillMindlink, 0)
		if got := DoMindlink(ch, mob).MessageToCh; got != "Yeah, right.\r\n" {
			t.Fatalf("message = %q, want no-skill gate", got)
		}
	})

	t.Run("actor fighting", func(t *testing.T) {
		ch, mob := newMindlinkGateFixture()
		ch.SetFighting(mob.GetName())
		if got := DoMindlink(ch, mob).MessageToCh; got != "There's too much going on to establish a mind link.\r\n" {
			t.Fatalf("message = %q, want actor-fighting gate", got)
		}
	})

	t.Run("target fighting", func(t *testing.T) {
		ch, mob := newMindlinkGateFixture()
		mob.SetFighting(ch.Name)
		if got := DoMindlink(ch, mob).MessageToCh; got != "There's too much going on to establish a mind link.\r\n" {
			t.Fatalf("message = %q, want target-fighting gate", got)
		}
	})

	t.Run("actor low health", func(t *testing.T) {
		ch, mob := newMindlinkGateFixture()
		ch.SetHP(99)
		if got := DoMindlink(ch, mob).MessageToCh; got != "You don't have enough life to spare!\r\n" {
			t.Fatalf("message = %q, want actor-health gate", got)
		}
	})
}
