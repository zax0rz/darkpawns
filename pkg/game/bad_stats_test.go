package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
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
//
// Each firing also runs damage()'s position switch (src/fight.c:1562-1600).
// POS_STANDING takes the default arm: 40 damage is not over max_hit/4 (50), so
// no "That really did HURT!" line, but the fourth firing leaves 40 HP, which is
// under max_hit/4, so C appends "You wish that your wounds would stop BLEEDING
// so much!". The replay delivers those bytes to the entering descriptor too.
func TestReplayCrashLoadBadStatsFiresPerEquippedItem(t *testing.T) {
	w, p, _ := replayWorld(t, ApplyDex)
	var lines []string
	w.ReplayCrashLoadBadStats(p, func(text string) { lines = append(lines, text) })
	want := []string{
		"You trip over your own feet and hit your head!\n\r",
		"You trip over your own feet and hit your head!\n\r",
		"You trip over your own feet and hit your head!\n\r",
		"You trip over your own feet and hit your head!\n\r",
		"You wish that your wounds would stop BLEEDING so much!\r\n",
	}
	if len(lines) != len(want) {
		t.Fatalf("lines = %d, want %d: %q", len(lines), len(want), lines)
	}
	for i, line := range lines {
		if line != want[i] {
			t.Fatalf("line %d = %q, want %q", i, line, want[i])
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

// The lethal trip at NOWHERE is C's full death pipeline minus raw_kill's body.
// raw_kill returns at its first statement when in_room == NOWHERE
// (src/fight.c:538-539), so death_cry, make_corpse and extract_char never run
// and the character stays POS_DEAD where it stands. Everything before that
// guard is defined and ported: damage()'s HP drop and update_pos (:1483-1489),
// its POS_DEAD bytes (:1586-1588), the death counters and mudlog (:1670-1691),
// die_with_killer's gain_exp(-exp/37) (:590) and its RNG-gated CON check
// (:600-607).
func TestReplayCrashLoadBadStatsLethalStopsAtRawKill(t *testing.T) {
	w, p, _ := replayWorld(t, ApplyDex)
	p.SetHP(10) // the first 40-point trip is lethal
	p.SetExp(100000)
	objects := len(w.GetAllObjects())
	var lines []string
	w.ReplayCrashLoadBadStats(p, func(text string) { lines = append(lines, text) })

	// damage(): 10 - 40 = -30, floored to -11 by DP-1021's clamp (C keeps -30;
	// recorded in the handoff), update_pos -> POS_DEAD for HP <= -11
	// (src/fight.c:192-193).
	if p.GetHP() != -11 || p.GetPosition() != combat.PosDead {
		t.Fatalf("hp=%d pos=%d, want -11/POS_DEAD", p.GetHP(), p.GetPosition())
	}
	// fight.c:1689-1691 counters, then die_with_killer's gain_exp(-GET_EXP/37).
	if p.Deaths != 1 || p.Kills != 1 || p.LastDeath == 0 {
		t.Fatalf("deaths=%d kills=%d lastDeath=%d, want 1/1/nonzero", p.Deaths, p.Kills, p.LastDeath)
	}
	if want := 100000 - 100000/37; p.GetExp() != want {
		t.Fatalf("exp=%d, want %d", p.GetExp(), want)
	}
	// raw_kill returned at NOWHERE: no corpse and no extraction.
	if got := len(w.GetAllObjects()); got != objects {
		t.Fatalf("objects=%d, want %d (raw_kill makes no corpse at NOWHERE)", got, objects)
	}
	if w.HasPendingExtractions() {
		t.Fatal("the character was queued for extraction; raw_kill returns at NOWHERE")
	}
	// The bytes: the first trip's bad-stat line, then damage()'s POS_DEAD
	// pair, then the three later trips — refused at the corpse gate, so no
	// further HP loss and no further death bytes.
	want := []string{
		"You trip over your own feet and hit your head!\n\r",
		"You are dead!  Sorry...\r\n",
		"You trip over your own feet and hit your head!\n\r",
		"You trip over your own feet and hit your head!\n\r",
		"You trip over your own feet and hit your head!\n\r",
	}
	if len(lines) != len(want) {
		t.Fatalf("lines = %d, want %d: %q", len(lines), len(want), lines)
	}
	for i, line := range lines {
		if line != want[i] {
			t.Fatalf("line %d = %q, want %q", i, line, want[i])
		}
	}
	if p.GetRoom() != 1001 {
		t.Fatalf("room = %d, want the record's room unchanged", p.GetRoom())
	}
}

// R3: the CON loss is C's draw sequence — one number(0,3) at level 6+
// (src/fight.c:600), then one number(0,5) only if the first came back 0 at
// level 21+ (:602-603). Each case proves the CON decrement follows that first
// draw and that the replay leaves the stream exactly where one such draw would.
func TestReplayCrashLoadBadStatsLethalConLossDraw(t *testing.T) {
	for _, seed := range []uint32{1, 2, 3, 5, 8} {
		w, p, _ := replayWorld(t, ApplyDex)
		p.SetHP(10)

		// Reference stream position after exactly one number(0,3).
		dprng.ResetStream(seed)
		first := dprng.Number(0, 3)
		next := dprng.Number(0, 99)

		dprng.ResetStream(seed)
		w.ReplayCrashLoadBadStats(p, nil)
		gotNext := dprng.Number(0, 99)

		wantCon := 10
		if first == 0 {
			wantCon = 9
		}
		if p.Stats.Con != wantCon {
			t.Fatalf("seed %d: con=%d, want %d (first draw %d)", seed, p.Stats.Con, wantCon, first)
		}
		if gotNext != next {
			t.Fatalf("seed %d: next draw %d, want %d — the replay did not consume exactly one number(0,3)", seed, gotNext, next)
		}
	}
}

// C's gain_exp clamps a death loss at -max_exp_loss (src/limits.c:319;
// src/config.c:82 sets max_exp_loss = 500000), which the old inline
// GET_EXP - GET_EXP/37 computation missed.
func TestReplayCrashLoadBadStatsLethalExpLossClamp(t *testing.T) {
	w, p, _ := replayWorld(t, ApplyDex)
	p.SetHP(10)
	p.SetExp(40_000_000) // 40M/37 is well past the 500000 clamp
	w.ReplayCrashLoadBadStats(p, nil)
	if want := 40_000_000 - 500_000; p.GetExp() != want {
		t.Fatalf("exp=%d, want %d (the -max_exp_loss clamp)", p.GetExp(), want)
	}
}

// A level-21+ death makes the second CON check when the first passes, so the
// replay consumes two draws there and the CON total follows both.
func TestReplayCrashLoadBadStatsLethalConLossSecondDraw(t *testing.T) {
	for _, seed := range []uint32{1, 2, 3, 5, 8} {
		w, p, _ := replayWorld(t, ApplyDex)
		p.SetHP(10)
		p.SetLevel(21)

		dprng.ResetStream(seed)
		first := dprng.Number(0, 3)
		wantCon := 10
		if first == 0 {
			wantCon = 9
			if dprng.Number(0, 5) == 0 {
				wantCon = 8
			}
		}
		next := dprng.Number(0, 99)

		dprng.ResetStream(seed)
		w.ReplayCrashLoadBadStats(p, nil)
		gotNext := dprng.Number(0, 99)

		if p.Stats.Con != wantCon {
			t.Fatalf("seed %d: con=%d, want %d (first draw %d)", seed, p.Stats.Con, wantCon, first)
		}
		if gotNext != next {
			t.Fatalf("seed %d: next draw %d, want %d — the second-check gate differs", seed, gotNext, next)
		}
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
