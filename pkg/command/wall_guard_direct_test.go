package command

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// C-ERRATA E-034: wall_guard_ns's fighter arm calls the typing player's
// headbutt/parry/bash/berserk directly with subcmd=1
// (src/spec_procs.c:522-531). These tests pin the Direct lanes' half of that
// contract: the no-skill refusal prints and the body proceeds anyway.

func wallGuardDirectSession(t *testing.T) (*skillCommandSession, *game.MobInstance) {
	t.Helper()
	world, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Wall Room", Zone: 1}},
		Mobs: []parser.Mob{{
			VNum:      1,
			Keywords:  "rat",
			ShortDesc: "a rat",
			LongDesc:  "A rat is here.",
			Level:     1,
			HP:        parser.DiceRoll{Num: 1, Sides: 1, Plus: 50},
		}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(world.StopAITicker)

	mage := game.NewPlayer(1, "Wallmage", 1001)
	mage.Class = game.ClassMageUser
	mage.SetLevel(10)
	mage.SetPosition(combat.PosFighting)
	mage.SetMove(100)
	mage.SetHP(100)
	if err := world.AddPlayer(mage); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	mob, err := world.SpawnMob(1, 1001)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	mob.SetPosition(combat.PosFighting)
	mage.SetFightingBody(mob)
	return &skillCommandSession{player: mage, world: world}, mob
}

func TestCmdBashDirectSubcmdRefusalProceeds(t *testing.T) {
	session, _ := wallGuardDirectSession(t)
	if err := CmdBashDirect(session, "rat", 1); err != nil {
		t.Fatalf("CmdBashDirect: %v", err)
	}
	if len(session.messages) == 0 || session.messages[0] != "You'd better leave all the martial arts to fighters.\r\n" {
		t.Fatalf("first line = %q, want the refusal (src/act.offensive.c:429-430)", session.messages)
	}
	if got := session.player.GetWaitState(); got == 0 {
		t.Fatal("subcmd 1 bash returned at the refusal; C proceeds to the WAIT_STATE tail (src/act.offensive.c:431-432,496-497)")
	}
	if got := session.player.GetMove(); got != 90 {
		t.Fatalf("move = %d, want 90 — the body spent its 10 (src/act.offensive.c:467-473)", got)
	}

	control, _ := wallGuardDirectSession(t)
	if err := CmdBashDirect(control, "rat", 0); err != nil {
		t.Fatalf("CmdBashDirect control: %v", err)
	}
	if got := strings.Join(control.messages, ""); got != "You'd better leave all the martial arts to fighters.\r\n" {
		t.Fatalf("subcmd 0 output = %q, want the refusal alone", got)
	}
	if control.player.GetWaitState() != 0 || control.player.GetMove() != 100 {
		t.Fatal("subcmd 0 refusal proceeded; C returns (src/act.offensive.c:431-432)")
	}
}

func TestCmdHeadbuttDirectSubcmdRefusalProceeds(t *testing.T) {
	session, _ := wallGuardDirectSession(t)
	if err := CmdHeadbuttDirect(session, "rat", 1); err != nil {
		t.Fatalf("CmdHeadbuttDirect: %v", err)
	}
	if len(session.messages) == 0 || session.messages[0] != "You aren't qualified to headbutt anyone!\r\n" {
		t.Fatalf("first line = %q, want the refusal (src/new_cmds.c:384)", session.messages)
	}
	if got := session.player.GetWaitState(); got == 0 {
		t.Fatal("subcmd 1 headbutt returned at the refusal; C proceeds to WAIT_STATE (src/new_cmds.c:385-386,459)")
	}

	control, _ := wallGuardDirectSession(t)
	if err := CmdHeadbuttDirect(control, "rat", 0); err != nil {
		t.Fatalf("CmdHeadbuttDirect control: %v", err)
	}
	if got := strings.Join(control.messages, ""); got != "You aren't qualified to headbutt anyone!\r\n" {
		t.Fatalf("subcmd 0 output = %q, want the refusal alone", got)
	}
	if control.player.GetWaitState() != 0 {
		t.Fatal("subcmd 0 refusal proceeded; C returns (src/new_cmds.c:385-386)")
	}
}

func TestCmdBerserkDirectSubcmdRefusalProceeds(t *testing.T) {
	session, _ := wallGuardDirectSession(t)
	if err := CmdBerserkDirect(session, 1); err != nil {
		t.Fatalf("CmdBerserkDirect: %v", err)
	}
	if len(session.messages) == 0 || session.messages[0] != "You're about as berserk as you're gonna get.\r\n" {
		t.Fatalf("first line = %q, want the refusal (src/new_cmds.c:2142)", session.messages)
	}
	joined := strings.Join(session.messages[1:], "")
	if !strings.Contains(joined, "battle rage") {
		t.Fatalf("subcmd 1 berserk stopped at the refusal; want the battle-rage line (src/new_cmds.c:2143-2144,2161-2168), got %q", joined)
	}
	if got := session.player.GetWaitState(); got == 0 {
		t.Fatal("subcmd 1 berserk set no wait; C's tail is unconditional (src/new_cmds.c:2186)")
	}

	control, _ := wallGuardDirectSession(t)
	if err := CmdBerserkDirect(control, 0); err != nil {
		t.Fatalf("CmdBerserkDirect control: %v", err)
	}
	if got := strings.Join(control.messages, ""); got != "You're about as berserk as you're gonna get.\r\n" {
		t.Fatalf("subcmd 0 output = %q, want the refusal alone", got)
	}
	if control.player.GetWaitState() != 0 {
		t.Fatal("subcmd 0 refusal proceeded; C returns (src/new_cmds.c:2143-2144)")
	}
}
