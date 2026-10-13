package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func badStatsWorld(t *testing.T, location int) (*World, *Player, *ObjectInstance, map[string]*strings.Builder) {
	t.Helper()
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001}}, Mobs: []parser.Mob{{VNum: 7907, ShortDesc: "Pestilence"}}, Objs: []parser.Obj{{VNum: 201, Keywords: "helm", ShortDesc: "a cursed helm", TypeFlag: ITEM_ARMOR, WearFlags: [4]int{1 << 4}, Affects: []parser.ObjAffect{{Location: location, Modifier: -10}}, LoadPercent: 100}}})
	if err != nil {
		t.Fatal(err)
	}
	w.StopAITicker()
	t.Cleanup(w.StopAITicker)
	p := NewPlayer(11, "Wearer", 1001)
	p.Stats = CharStats{Str: 10, Int: 10, Wis: 10, Dex: 10, Con: 10, Cha: 10}
	p.CopyBaseAttributes()
	p.SetLevel(15)
	p.SetMaxHP(200)
	p.SetHP(200)
	if err := w.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	obj, err := w.SpawnObject(201, -1)
	if err != nil {
		t.Fatal(err)
	}
	old := combat.GetCallbacks()
	t.Cleanup(func() { combat.SetCallbacks(old) })
	combat.SetCallbacks(w.WireCombatCallbacks())
	return w, p, obj, captureOutput(w)
}

func TestPlayerEquipmentBadStats(t *testing.T) {
	for _, tc := range []struct {
		name     string
		location int
		text     string
		damage   int
	}{
		{"strength", ApplyStr, "You are too weak to fight!\n\r", 0},
		{"intelligence", ApplyInt, "You are too dumb to do much of anything!\n\r", 0},
		{"wisdom", ApplyWis, "You are too dumb to do much of anything!\n\r", 0},
		{"charisma", ApplyCha, "The world hates you!\n\r", 0},
		{"dexterity", ApplyDex, "You trip over your own feet and hit your head!\n\r", 40},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, p, obj, out := badStatsWorld(t, tc.location)
			if err := w.EquipItem(p, obj, 6); err != nil {
				t.Fatal(err)
			}
			if got := outputOf(out, p.Name); got != tc.text {
				t.Fatalf("bad-stat bytes=%q, want %q", got, tc.text)
			}
			if p.GetHP() != 200-tc.damage {
				t.Fatalf("bad-stat HP=%d, want %d", p.GetHP(), 200-tc.damage)
			}
		})
	}
}

func TestPlayerEquipmentBadStatPriority(t *testing.T) {
	w, p, obj, out := badStatsWorld(t, ApplyStr)
	p.Stats = CharStats{}
	p.CopyBaseAttributes()
	if err := w.EquipItem(p, obj, 6); err != nil {
		t.Fatal(err)
	}
	if got := outputOf(out, p.Name); got != "You trip over your own feet and hit your head!\n\r" {
		t.Fatalf("last-zero bytes=%q", got)
	}
	if p.GetHP() != 160 {
		t.Fatalf("last-zero HP=%d", p.GetHP())
	}
}

func TestPlayerEquipmentDexUsesDamageTail(t *testing.T) {
	w, p, obj, out := badStatsWorld(t, ApplyDex)
	p.SetAffect(affSanctuary, true)
	p.SetAffect(affHide, true)
	if err := w.EquipItem(p, obj, 6); err != nil {
		t.Fatal(err)
	}
	if p.GetHP() != 180 || p.IsAffected(affHide) {
		t.Fatalf("DEX bypassed damage(): hp=%d hidden=%v", p.GetHP(), p.IsAffected(affHide))
	}
	if !strings.HasPrefix(outputOf(out, p.Name), "You trip over your own feet and hit your head!\n\r") {
		t.Fatalf("DEX bytes=%q", outputOf(out, p.Name))
	}
}

func TestPlayerEquipmentBadStatEntryPoints(t *testing.T) {
	for _, path := range []string{"lua", "move", "restore"} {
		t.Run(path, func(t *testing.T) {
			w, p, obj, out := badStatsWorld(t, ApplyDex)
			switch path {
			case "lua":
				if err := w.MoveObject(obj, LocInventoryPlayer(p.Name)); err != nil {
					t.Fatal(err)
				}
				if !w.EquipChar(p.Name, false, obj.VNum) {
					t.Fatal("Lua equip failed")
				}
			case "move":
				if err := w.MoveObject(obj, LocEquippedPlayer(p.Name, SlotHead)); err != nil {
					t.Fatal(err)
				}
			case "restore":
				if !restoreEquippedItem(p, obj, int(SlotHead)) {
					t.Fatal("equip restore failed")
				}
			}
			if p.GetHP() != 160 || outputOf(out, p.Name) != "You trip over your own feet and hit your head!\n\r" {
				t.Fatalf("%s HP=%d bytes=%q", path, p.GetHP(), outputOf(out, p.Name))
			}
		})
	}
}

// replayWorld builds the Crash_load replay fixture: four worn prototypes where
// vnum 201 carries a -10 `location` affect and 202-204 are inert, plus a
// level-15 player at HP 200 whose slots are filled the way restoreSavedItems
// fills them (lowest wear position first).
func replayWorld(t *testing.T, location int) (*World, *Player, map[string]*strings.Builder) {
	t.Helper()
	objs := make([]parser.Obj, 0, 4)
	for vnum := 201; vnum <= 204; vnum++ {
		obj := parser.Obj{VNum: vnum, Keywords: "item", ShortDesc: "a saved item", TypeFlag: ITEM_ARMOR, WearFlags: [4]int{1 << 4}, LoadPercent: 100}
		if vnum == 201 {
			obj.Affects = []parser.ObjAffect{{Location: location, Modifier: -10}}
		}
		objs = append(objs, obj)
	}
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001}},
		Mobs:  []parser.Mob{{VNum: 7907, ShortDesc: "Pestilence"}},
		Objs:  objs,
	})
	if err != nil {
		t.Fatal(err)
	}
	w.StopAITicker()
	t.Cleanup(w.StopAITicker)
	p := NewPlayer(11, "Wearer", 1001)
	p.Stats = CharStats{Str: 10, Int: 10, Wis: 10, Dex: 10, Con: 10, Cha: 10}
	p.CopyBaseAttributes()
	p.SetLevel(15)
	p.SetMaxHP(200)
	p.SetHP(200)
	if err := w.AddPlayer(p); err != nil {
		t.Fatal(err)
	}
	for i, cPos := range []int{0, 5, 6, 7} { // light, body, head, legs
		obj, err := w.SpawnObject(201+i, -1)
		if err != nil {
			t.Fatal(err)
		}
		slot, ok := CWearPosToSlot(cPos)
		if !ok {
			t.Fatalf("C wear position %d has no slot", cPos)
		}
		obj.Location = LocEquippedPlayer(p.Name, slot)
		if err := p.Equipment.SetSlot(slot, obj); err != nil {
			t.Fatal(err)
		}
	}
	old := combat.GetCallbacks()
	t.Cleanup(func() { combat.SetCallbacks(old) })
	combat.SetCallbacks(w.WireCombatCallbacks())
	return w, p, captureOutput(w)
}

// The DEX item in the lowest wear position zeroes DEX on the first equip, and
// every later equip re-runs affect_total over the items equipped so far, so the
// replay fires four times: HP 200 - 4x40 = 40 (src/objsave.c:557;
// src/handler.c:337-372,749). A single final-stats check would fire once.
func TestReplayCrashLoadBadStatsFiresPerEquippedItem(t *testing.T) {
	w, p, _ := replayWorld(t, ApplyDex)
	var lines []string
	w.ReplayCrashLoadBadStats(p, func(text string) { lines = append(lines, text) })
	if len(lines) != 4 {
		t.Fatalf("firings = %d, want 4: %q", len(lines), lines)
	}
	for _, line := range lines {
		if line != "You trip over your own feet and hit your head!\n\r" {
			t.Fatalf("line = %q", line)
		}
	}
	if p.GetHP() != 40 {
		t.Fatalf("HP = %d, want 40 (200 - 4x40)", p.GetHP())
	}
}

// C's reset_char leaves ch->in_room NOWHERE for the whole of Crash_load
// (src/db.c:2945), so the DEX arm's hide-fade and room act() have no audience
// and log C's SYSERR instead (src/comm.c:2519-2526). The record's room must not
// become an audience, and must be restored for the entry that follows.
func TestReplayCrashLoadBadStatsHoldsRoomAtNowhere(t *testing.T) {
	w, p, out := replayWorld(t, ApplyDex)
	peer := NewPlayer(12, "Peer", 1001)
	if err := w.AddPlayer(peer); err != nil {
		t.Fatal(err)
	}
	p.SetAffect(affHide, true)
	w.ReplayCrashLoadBadStats(p, nil)
	if got := outputOf(out, peer.Name); got != "" {
		t.Fatalf("room peer saw %q, want nothing", got)
	}
	if p.GetRoom() != 1001 {
		t.Fatalf("room = %d, want the record's room restored", p.GetRoom())
	}
}

// The HP <= 0 sub-case is recorded as blocked (R1a): C's damage() at NOWHERE
// reaches die()/death_cry, which explicitly char_to_room(ch, 0) — the first room
// of the world array (src/fight.c:507-514) — and makes a corpse there. The port
// does not reproduce that branch. This guard only proves the replay survives a
// lethal zero-DEX set and leaves the character's room where the entry left it.
func TestReplayCrashLoadBadStatsLethalLeavesRoomAlone(t *testing.T) {
	w, p, _ := replayWorld(t, ApplyDex)
	p.SetHP(10)
	w.ReplayCrashLoadBadStats(p, nil)
	if p.GetHP() > 0 {
		t.Fatalf("HP = %d, want the four 40-point firings to be lethal", p.GetHP())
	}
	if p.GetRoom() != 1001 {
		t.Fatalf("room = %d, want the record's room unchanged", p.GetRoom())
	}
}

func TestPlayerEquipmentCharismaHunting(t *testing.T) {
	w, p, obj, out := badStatsWorld(t, ApplyCha)
	watcher := NewPlayer(12, "Watcher", 1001)
	watcher.SetLevel(LVL_IMMORT)
	watcher.SetPlrFlag(PrfLog1, true)
	watcher.SetPlrFlag(PrfLog2, true)
	if err := w.AddPlayer(watcher); err != nil {
		t.Fatal(err)
	}
	old := getImmortalSessionProvider()
	SetImmortalSessionProvider(&testSessions{watcher})
	t.Cleanup(func() { SetImmortalSessionProvider(old) })
	older, err := w.SpawnMobQuiet(7907, 1001)
	if err != nil {
		t.Fatal(err)
	}
	pest, err := w.SpawnMobQuiet(7907, 1001)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.EquipItem(p, obj, 6); err != nil {
		t.Fatal(err)
	}
	if older.GetHunting() != "" || pest.GetHunting() != p.Name {
		t.Fatalf("Pestilence target=%q, older=%q", pest.GetHunting(), older.GetHunting())
	}
	if pest.GetHuntingID() != "11" {
		t.Fatalf("Pestilence player id=%q, want 11", pest.GetHuntingID())
	}
	if got := outputOf(out, watcher.Name); got != "[ Pestilence started hunting Wearer ]\r\n" {
		t.Fatalf("hunting mudlog=%q", got)
	}
	pest.SetHunting("Already")
	before := outputOf(out, watcher.Name)
	w.applyBadStat(p, 'c')
	if pest.GetHunting() != "Already" || outputOf(out, watcher.Name) != before {
		t.Fatal("busy Pestilence was retargeted or logged")
	}
}
