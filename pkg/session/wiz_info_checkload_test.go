package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/dprng"
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
				{Command: "R", Arg1: 1002, Arg2: 0, Arg3: 200}, // C form: kind 0 (mobile), vnum
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

// C do_checkload reads a throwaway instance before it reports (read_mobile at
// src/act.wizard.c:3841-3844, read_object at :3853-3855), and check_load reads
// one per zone command (:3729-3732, :3750-3753). Those reads roll the hit dice,
// the gold variance and the ITEM_RARE variance, so the command must consume them
// even though only the short description reaches the player (R3a). Reverting the
// reads leaves the stream short by the read's draws and this test fails.
func TestCheckloadCommandReadsThrowawayInstances(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "First Room", Zone: 1}},
		Mobs: []parser.Mob{{
			VNum:      200,
			ShortDesc: "a test mob",
			HP:        parser.DiceRoll{Num: 1, Sides: 5, Plus: 20},
			Gold:      100,
		}},
		Objs: []parser.Obj{{
			VNum:       300,
			ShortDesc:  "a rare sword",
			ExtraFlags: [4]int{game.FlagItemRare},
			Affects:    []parser.ObjAffect{{Location: 19, Modifier: 1}},
		}},
		Zones: []parser.Zone{{
			Number: 1,
			Commands: []parser.ZoneCommand{
				{Command: "M", Arg1: 200, Arg2: 3, Arg3: 1001},
				{Command: "O", Arg1: 300, Arg2: 1, Arg3: 1001},
			},
		}},
	}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	m := newTestManager(t, w, nil)
	s := makeCommandTestSession(t, m, "Loadgod", game.LVL_IMMORT, 1001)

	const seed = 11
	// 'mob': read_mobile's hit dice + gold sign + gold percentage, and nothing
	// more — C's check_load type-'m' arm has no per-command reads.
	dprng.ResetStream(seed)
	if err := cmdCheckload(s, []string{"mob", "200"}); err != nil {
		t.Fatal(err)
	}
	mobRef := dprng.New(seed)
	readMobile := func(ref *dprng.Generator) {
		ref.Dice(1, 5)
		ref.Number(0, 1)
		ref.Number(1, 20)
	}
	readRare := func(ref *dprng.Generator) {
		if ref.Number(1, 100) <= 20 {
			ref.Number(0, 1)
		}
	}
	readMobile(mobRef) // do_checkload's read_mobile
	if got, want := dprng.Next(), mobRef.Next(); got != want {
		t.Errorf("stream after `checkload mob 200` = %d, want %d (one mobile read)", got, want)
	}

	// 'obj': read_object's rare variance, then check_load's per-command reads in
	// zone order — the 'M' command reads a mobile, the 'O' command reads an
	// object (src/act.wizard.c:3729-3732, :3750-3753).
	dprng.ResetStream(seed)
	if err := cmdCheckload(s, []string{"obj", "300"}); err != nil {
		t.Fatal(err)
	}
	objRef := dprng.New(seed)
	readRare(objRef)   // do_checkload's read_object
	readMobile(objRef) // check_load's M command
	readRare(objRef)   // check_load's O command
	if got, want := dprng.Next(), objRef.Next(); got != want {
		t.Errorf("stream after `checkload obj 300` = %d, want %d (three reads)", got, want)
	}
}
