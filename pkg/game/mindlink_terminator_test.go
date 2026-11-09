package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestDoMindlinkDrainedLineIsBare pins R1: C's do_mindlink sends
// "You feel a little drained..." with NO trailing CRLF (src/new_cmds2.c:314).
// The reachable command arm returns it via MessageToChAfterRoom; the port used
// to hard-code \r\n in the message itself, which the terminator-normalizing
// oracle cannot see.
func TestDoMindlinkDrainedLineIsBare(t *testing.T) {
	world, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Room", Zone: 1}},
		Mobs: []parser.Mob{{
			VNum: 2001, Keywords: "target", ShortDesc: "a target", Level: 5,
			Position: combat.PosStanding, DefaultPos: combat.PosStanding,
		}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(world.StopAITicker)

	ch := NewPlayer(1, "Linker", 1001)
	ch.Level = 20
	ch.SetHP(500)
	ch.SetSkill(SkillMindlink, 80)
	if err := world.AddPlayer(ch); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	mob, err := world.SpawnMobQuiet(2001, 1001)
	if err != nil {
		t.Fatalf("SpawnMobQuiet: %v", err)
	}
	mob.SetMana(200)

	result := DoMindlink(ch, mob)
	if result.MessageToCh != "You feel a little drained..." {
		t.Fatalf("MessageToCh = %q, want the bare drained line", result.MessageToCh)
	}
	if !result.NoMessageToChTerminator {
		t.Fatal("DoMindlink did not mark the drained line as terminator-less")
	}
}
