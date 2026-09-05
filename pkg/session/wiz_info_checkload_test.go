package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestCheckloadAtoiUsesCDecimalPrefix(t *testing.T) {
	if got := checkloadAtoi("16303x"); got != 16303 {
		t.Fatalf("checkloadAtoi(16303x) = %d, want 16303", got)
	}
	if got := checkloadAtoi("0"); got != 0 {
		t.Fatalf("checkloadAtoi(0) = %d, want 0", got)
	}
}

func TestCheckloadReportsMobAndObjectResetBranches(t *testing.T) {
	w, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "First Room"},
			{VNum: 1002, Name: "Second Room"},
		},
		Mobs: []parser.Mob{{VNum: 200, ShortDesc: "a test mob"}},
		Objs: []parser.Obj{
			{VNum: 300, ShortDesc: "a box", LoadPercent: 25},
			{VNum: 301, ShortDesc: "a sword", LoadPercent: 50},
			{VNum: 302, ShortDesc: "a shield", LoadPercent: 75},
		},
		Zones: []parser.Zone{{
			Number: 1,
			Commands: []parser.ZoneCommand{
				{Command: "M", Arg1: 200, Arg2: 3, Arg3: 1001},
				{Command: "E", Arg1: 301, Arg2: 4, Arg3: 16},
				{Command: "G", Arg1: 302, Arg2: 5},
				{Command: "O", Arg1: 300, Arg2: 1, Arg3: 1002},
				{Command: "P", Arg1: 301, Arg2: 2, Arg3: 300},
				{Command: "R", Arg1: 1002, Arg2: 301, Arg3: -1},
				{Command: "R", Arg1: 1002, Arg2: 200, Arg3: 0},
			},
		}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	wantMob := "Checking load info for a test mob...\r\n" +
		" [ 1001] First Room\r\n         3 Max\r\n" +
		" [ 1002] Second Room\r\n         Removed from room\r\n"
	if got := checkloadMobReport(w, 200, "a test mob"); got != wantMob {
		t.Errorf("mob report = %q, want %q", got, wantMob)
	}

	wantObj := "Checking load info for a sword...\r\n" +
		" [ 1001] First Room\r\n         Equipped to a test mob [200]\r\n         50.00% Load, 4 Max\r\n" +
		" [ 1002] Second Room\r\n         Put in a box [300]\r\n         25.00% Load, 2 Max\r\n" +
		" [ 1002] Second Room\r\n         Removed from room\r\n"
	if got := checkloadObjectReport(w, 301, "a sword"); got != wantObj {
		t.Errorf("object report = %q, want %q", got, wantObj)
	}

	gotShield := checkloadObjectReport(w, 302, "a shield")
	if !strings.Contains(gotShield, "Given to a test mob [200]") || !strings.Contains(gotShield, "75.00% Load, 5 Max") {
		t.Errorf("give report = %q, want the C G branch", gotShield)
	}
}
