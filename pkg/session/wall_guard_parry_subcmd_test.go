package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestCmdParrySubcmdRefusalProceeds pins do_parry's subcmd=1 arm
// (C-ERRATA E-034; src/spec_procs.c:526): the no-skill refusal prints and the
// body proceeds (src/new_cmds.c:2344-2349), with prob drawn as
// number(50,100) (src/new_cmds.c:2369).
func TestCmdParrySubcmdRefusalProceeds(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Parry Room", Zone: 1}},
		Mobs: []parser.Mob{{
			VNum:      5000,
			Keywords:  "target dummy",
			ShortDesc: "a test target",
			LongDesc:  "A test target stands here.",
			Level:     15,
			HP:        parser.DiceRoll{Num: 1, Sides: 1, Plus: 100},
			Race:      1,
		}},
		Objs: []parser.Obj{{
			VNum:      6001,
			Keywords:  "sword",
			ShortDesc: "a short sword",
			WearFlags: [4]int{8193},
			Weight:    3,
		}},
	}
	newMage := func(t *testing.T) (*Session, *game.MobInstance) {
		t.Helper()
		w, err := game.NewWorld(parsed)
		if err != nil {
			t.Fatalf("NewWorld: %v", err)
		}
		t.Cleanup(func() { w.StopAITicker() })
		m := newTestManager(t, w, nil)
		mob, err := m.world.SpawnMob(5000, 1001)
		if err != nil {
			t.Fatalf("SpawnMob: %v", err)
		}
		s := makeGateSession(t, m, 1, "Wallmage", 20)
		s.player.Stats = game.CharStats{Str: 10, Int: 10, Wis: 10, Dex: 10, Con: 10, Cha: 10}
		s.player.CopyBaseAttributes()
		weapon, err := m.world.SpawnObject(6001, -1)
		if err != nil {
			t.Fatalf("SpawnObject: %v", err)
		}
		if err := s.player.Inventory.AddItem(weapon); err != nil {
			t.Fatalf("AddItem: %v", err)
		}
		if err := m.world.EquipItem(s.player, weapon, cWearWield); err != nil {
			t.Fatalf("EquipItem: %v", err)
		}
		s.player.SetFightingBody(mob)
		mob.SetFightingBody(s.player)
		return s, mob
	}

	t.Run("subcmd 1 proceeds past the refusal", func(t *testing.T) {
		s, _ := newMage(t)
		if err := cmdParrySubcmd(s, 1); err != nil {
			t.Fatalf("cmdParrySubcmd: %v", err)
		}
		got := readSendText(t, s) + readSendText(t, s)
		if !strings.HasPrefix(got, "You're not good enough at swordplay to parry!\r\n") {
			t.Fatalf("first line = %q, want the refusal (src/new_cmds.c:2346)", got)
		}
		rest := strings.TrimPrefix(got, "You're not good enough at swordplay to parry!\r\n")
		if !strings.Contains(rest, "dazzling show of swordplay") {
			t.Fatalf("subcmd 1 parry stopped at the refusal; want a dazzle line (src/new_cmds.c:2371-2382), got %q", rest)
		}
		if s.player.GetWaitState() == 0 {
			t.Fatal("subcmd 1 parry set no wait; both C arms wait (src/new_cmds.c:2377,2388)")
		}
	})

	t.Run("subcmd 0 returns at the refusal", func(t *testing.T) {
		s, _ := newMage(t)
		if err := cmdParrySubcmd(s, 0); err != nil {
			t.Fatalf("cmdParrySubcmd control: %v", err)
		}
		if got := readSendText(t, s); got != "You're not good enough at swordplay to parry!\r\n" {
			t.Fatalf("subcmd 0 output = %q, want the refusal alone", got)
		}
		if s.player.GetWaitState() != 0 {
			t.Fatal("subcmd 0 refusal proceeded; C returns (src/new_cmds.c:2347-2348)")
		}
	})
}
