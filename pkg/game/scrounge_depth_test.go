package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newScroungeDepthWorld(t *testing.T, sector int) (*World, *Player) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 8105, Name: "Wilderness", Zone: 1, Sector: sector}},
		Objs: []parser.Obj{
			{VNum: 27, Keywords: "fish", ShortDesc: "some foreign breed of fish"},
			{VNum: 28, Keywords: "game", ShortDesc: "some small game"},
			{VNum: 29, Keywords: "mouse", ShortDesc: "a field mouse"},
			{VNum: 30, Keywords: "sandworm", ShortDesc: "a sandworm"},
			{VNum: 31, Keywords: "berries", ShortDesc: "some edible berries"},
		},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	ch := NewPlayer(1, "Scrounger", 8105)
	ch.SetPosition(combat.PosStanding)
	ch.SetSkill(SkillScrounge, 100)
	return w, ch
}

func TestDoScroungeDepthSectorMapAndMessages(t *testing.T) {
	tests := []struct {
		name   string
		sector int
		vnum   int
		actor  string
		room   string
		find   bool
	}{
		{name: "forest", sector: SECT_FOREST, vnum: 28, actor: "You capture and kill some small game.\r\n", room: "Scrounger searches for something to eat."},
		{name: "field", sector: SECT_FIELD, vnum: 29, actor: "You capture and kill a field mouse.\r\n", room: "Scrounger searches for something to eat."},
		{name: "hills", sector: SECT_HILLS, vnum: 29, actor: "You capture and kill a field mouse.\r\n", room: "Scrounger searches for something to eat."},
		{name: "desert", sector: SECT_DESERT, vnum: 30, actor: "You capture and kill a sandworm.\r\n", room: "Scrounger searches for something to eat."},
		{name: "mountain", sector: SECT_MOUNTAIN, vnum: 31, actor: "You find some edible berries.\r\n", room: "Scrounger searches for something to eat.", find: true},
		{name: "water swim", sector: SECT_WATER_SWIM, vnum: 27, actor: "You capture and kill some foreign breed of fish.\r\n", room: "Scrounger searches for something to eat."},
		{name: "water no-swim", sector: SECT_WATER_NOSWIM, vnum: 27, actor: "You capture and kill some foreign breed of fish.\r\n", room: "Scrounger searches for something to eat."},
		{name: "underwater", sector: SECT_UNDERWATER, vnum: 27, actor: "You capture and kill some foreign breed of fish.\r\n", room: "Scrounger searches for something to eat."},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w, ch := newScroungeDepthWorld(t, test.sector)
			dprng.ResetStream(1)
			result := DoScrounge(ch, w)
			if !result.Success {
				t.Fatal("skill 100 should reach the seed-1 success arm")
			}
			if got := result.MessageToCh; got != test.actor {
				t.Fatalf("actor message = %q, want %q", got, test.actor)
			}
			if got := result.MessageToRoom; got != test.room {
				t.Fatalf("room message = %q, want %q", got, test.room)
			}
			if result.WaitCh != 2 {
				t.Fatalf("wait rounds = %d, want 2", result.WaitCh)
			}
			if len(result.DeferredImprove) != 1 || result.DeferredImprove[0] != SkillScrounge {
				t.Fatalf("deferred improvement = %v, want [%q]", result.DeferredImprove, SkillScrounge)
			}
			if len(ch.Inventory.Items) != 1 || ch.Inventory.Items[0].VNum != test.vnum {
				t.Fatalf("inventory = %#v, want object vnum %d", ch.Inventory.Items, test.vnum)
			}
			if test.find && result.MessageToCh != "You find some edible berries.\r\n" {
				t.Fatal("mountain must use the find wording")
			}
		})
	}
}

func TestDoScroungeDepthGatesAndDefault(t *testing.T) {
	t.Run("no skill waits without a room act", func(t *testing.T) {
		w, ch := newScroungeDepthWorld(t, SECT_FOREST)
		ch.SetSkill(SkillScrounge, 0)
		result := DoScrounge(ch, w)
		if result.Success || result.MessageToCh != "You can't seem to find anything edible.\r\n" {
			t.Fatalf("no-skill result = %#v", result)
		}
		if result.MessageToRoom != "" || result.WaitCh != 2 {
			t.Fatalf("no-skill room/wait = %q/%d, want empty/2", result.MessageToRoom, result.WaitCh)
		}
	})

	t.Run("mounted gate precedes the roll", func(t *testing.T) {
		w, ch := newScroungeDepthWorld(t, SECT_FOREST)
		ch.SetAffect(affMount, true)
		dprng.ResetStream(1)
		result := DoScrounge(ch, w)
		if result.Success || result.MessageToCh != "Dismount first!\r\n" || result.WaitCh != 0 {
			t.Fatalf("mounted result = %#v", result)
		}
		gotNext := dprng.Number(1, 101)
		wantNext := func() int {
			dprng.ResetStream(1)
			return dprng.Number(1, 101)
		}()
		if gotNext != wantNext {
			t.Fatalf("mounted gate consumed a roll: next=%d want=%d", gotNext, wantNext)
		}
	})

	t.Run("default consumes the roll and keeps the room act", func(t *testing.T) {
		w, ch := newScroungeDepthWorld(t, SECT_CITY)
		dprng.ResetStream(1)
		result := DoScrounge(ch, w)
		if result.Success || result.MessageToCh != "You need to be in the wilderness to scrounge!\r\n" {
			t.Fatalf("default result = %#v", result)
		}
		if result.MessageToRoom != "Scrounger searches for something to eat." || result.WaitCh != 0 {
			t.Fatalf("default room/wait = %q/%d", result.MessageToRoom, result.WaitCh)
		}
		gotNext := dprng.Number(1, 101)
		wantNext := func() int {
			dprng.ResetStream(1)
			dprng.Number(1, 101)
			return dprng.Number(1, 101)
		}()
		if gotNext != wantNext {
			t.Fatalf("default branch consumed the wrong draws: next=%d want=%d", gotNext, wantNext)
		}
	})
}

func TestDoScroungeDepthDeferredImprovePreservesDrawOrder(t *testing.T) {
	w, ch := newScroungeDepthWorld(t, SECT_FOREST)
	ch.Stats.Wis = 100
	ch.Stats.Int = 100
	dprng.ResetStream(1)
	result := DoScrounge(ch, w)
	if !result.Success || len(result.DeferredImprove) != 1 || result.DeferredImprove[0] != SkillScrounge {
		t.Fatalf("success result = %#v, want one deferred scrounge improvement", result)
	}
	for _, skill := range result.DeferredImprove {
		ImproveSkill(ch, skill)
	}
	gotNext := dprng.Number(1, 101)

	dprng.ResetStream(1)
	dprng.Number(1, 101)
	dprng.Number(1, 200)
	wantNext := dprng.Number(1, 101)
	if gotNext != wantNext {
		t.Fatalf("deferred improvement consumed the wrong draws: next=%d want=%d", gotNext, wantNext)
	}
}
