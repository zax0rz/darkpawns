package session

// crash_load_bad_stats_test.go — DP-1419. C's Crash_load reads a crash file
// object by object and equips each worn item through auto_equip -> equip_char,
// which calls check_for_bad_stats on every equip (src/objsave.c:557;
// src/handler.c:749). equip_char's affect_total rebuilds aff_abils from the
// slots filled SO FAR (src/handler.c:337-372), so a stat already at zero fires
// again on every later worn item: a zero DEX item costs 40 HP per remaining
// worn item, not once.
//
// Each case below drives the real menu entry (MsgCharInput "1" ->
// handleMenuChoice -> enterReturningPlayer) on a real SQLite store, and asserts
// the exact bytes at the entering descriptor, their order relative to the
// welcome text, and the resulting HP.

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"golang.org/x/crypto/bcrypt"
)

// C wear positions used by the fixtures (0-based; a save's locate is position+1).
const (
	crashWearLight = 0
	crashWearBody  = 5
	crashWearHead  = 6
	crashWearLegs  = 7
)

// savedWornItem is one restored worn item: its obj vnum and the C wear
// position Crash_crashsave would have written it from.
type savedWornItem struct {
	vnum     int
	cWearPos int
}

// crashBadStatsWorn is the four-item set the per-item replay proofs use: the
// zeroing item in the lowest wear position (light), then three items after it
// (body 5, head 6, legs 7).
var crashBadStatsWorn = []savedWornItem{
	{201, crashWearLight}, {202, crashWearBody}, {203, crashWearHead}, {204, crashWearLegs},
}

// crashBadStatsEnv builds the stack a returning character's menu entry runs on:
// a real SQLite store holding the record and its crash file, a world with the
// four worn item prototypes (vnum 201 carries the -10 `zeroLocation` affect;
// 202-204 are inert), and a fresh menu-stage session whose player is the stored
// record. `worn` selects which items the file restored and where.
func crashBadStatsEnv(t *testing.T, name string, hp, zeroLocation int, worn []savedWornItem) (*Manager, *db.DB, *Session) {
	t.Helper()
	t.Setenv("JWT_SECRET", "crash-load-bad-stats-secret-at-least-32-bytes")
	store := entryDatabase(t)

	objs := make([]parser.Obj, 0, 4)
	for vnum := 201; vnum <= 204; vnum++ {
		obj := parser.Obj{VNum: vnum, Keywords: "item", ShortDesc: "a saved item", TypeFlag: game.ITEM_ARMOR, WearFlags: [4]int{1 << 4}, LoadPercent: 100}
		if vnum == 201 {
			obj.Affects = []parser.ObjAffect{{Location: zeroLocation, Modifier: -10}}
		}
		objs = append(objs, obj)
	}
	w, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Watch Room", Zone: 1}, {VNum: game.MortalStartRoom, Name: "The Temple", Zone: 80}},
		Mobs:  []parser.Mob{{VNum: 7907, ShortDesc: "Pestilence"}},
		Objs:  objs,
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	equipment := make([]game.SaveItemData, 0, len(worn))
	for _, item := range worn {
		equipment = append(equipment, game.SaveItemData{VNum: item.vnum, Count: 1, Locate: item.cWearPos + 1})
	}
	eqJSON, err := json.Marshal(equipment)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("oraclepass"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	record := &db.PlayerRecord{
		Name: name, Password: string(hash), RoomVNum: game.MortalStartRoom,
		Level: 15, Health: hp, MaxHealth: hp, Mana: 100, MaxMana: 100, Move: 100, MaxMove: 100,
		Class: game.ClassWarrior, Race: game.RaceHuman,
		StatStr: 10, StatInt: 10, StatWis: 10, StatDex: 10, StatCon: 10, StatCha: 10,
		Inventory: []byte("[]"), Equipment: eqJSON,
	}
	if err := store.CreatePlayer(record); err != nil {
		t.Fatalf("CreatePlayer: %v", err)
	}
	// The crash file exists: C's Crash_load fopen succeeds and equips the file.
	if err := db.SaveObjectSnapshot(store, name, rentCrash, []byte("[]"), eqJSON); err != nil {
		t.Fatalf("SaveObjectSnapshot: %v", err)
	}

	m := newTestManager(t, w, store)
	m.WirePlayerSaver(w)
	p, err := db.RecordToPlayer(record, w)
	if err != nil {
		t.Fatalf("RecordToPlayer: %v", err)
	}
	s := makeCharSession(t, m)
	s.player = p
	s.playerName = name
	s.authenticated = true
	s.menuActive = true
	s.menuStage = "menu"
	return m, store, s
}

// C's save order is ascending wear position (Crash_crashsave's
// for j = 0..NUM_WEARS loop, src/objsave.c:807-812), so the light (position 0)
// equips first, then body (5), head (6) and legs (7). The -10 DEX light drives
// DEX to 0 on the FIRST equip, and every later equip re-runs affect_total over
// the items equipped so far, so check_for_bad_stats fires FOUR times:
//
//	firing 1: light alone          -> DEX 10-10 = 0 -> 40 HP
//	firing 2: light+body           -> DEX 0         -> 40 HP
//	firing 3: light+body+head      -> DEX 0         -> 40 HP
//	firing 4: light+body+head+legs -> DEX 0         -> 40 HP
//
// HP 200 - 4x40 = 40. A single check on the final stats would subtract 40 once
// and print one line; that is the revert control.
func TestCrashLoadBadStatsDexFiresPerRestoredItem(t *testing.T) {
	_, _, s := crashBadStatsEnv(t, "Weary", 200, game.ApplyDex, crashBadStatsWorn)
	if err := entryInput(s, "1"); err != nil {
		t.Fatalf("menu entry: %v", err)
	}
	got := strings.Join(drainSessionText(t, s), "")
	want := strings.Repeat("You trip over your own feet and hit your head!\n\r", 4)
	if !strings.HasPrefix(got, want) {
		t.Fatalf("bad-stat bytes = %q, want prefix %q", got, want)
	}
	if s.player.GetHP() != 40 {
		t.Fatalf("HP = %d, want 40 (200 - 4x40)", s.player.GetHP())
	}
	// C sends these from Crash_load, before save_char and WELC_MESSG
	// (src/interpreter.c:2184-2189); the welcome text must follow them.
	if rest := got[len(want):]; !strings.Contains(rest, "Welcome to Dark Pawns!") {
		t.Fatalf("bad-stat lines did not precede the welcome text: %q", got)
	}
}

// The zeroing item in the highest wear position (legs, 7) equips last: the
// three inert items before it leave DEX at 10, so only the final equip sees
// DEX 0 and only one check fires. HP 200 - 40 = 160.
func TestCrashLoadBadStatsZeroingItemLastFiresOnce(t *testing.T) {
	worn := []savedWornItem{{201, crashWearLegs}, {202, crashWearLight}, {203, crashWearBody}, {204, crashWearHead}}
	_, _, s := crashBadStatsEnv(t, "Latezero", 200, game.ApplyDex, worn)
	if err := entryInput(s, "1"); err != nil {
		t.Fatalf("menu entry: %v", err)
	}
	got := strings.Join(drainSessionText(t, s), "")
	if want := "You trip over your own feet and hit your head!\n\r"; !strings.HasPrefix(got, want) {
		t.Fatalf("bad-stat bytes = %q, want prefix %q", got, want)
	}
	if firings := strings.Count(got, "You trip over your own feet"); firings != 1 {
		t.Fatalf("firings = %d, want 1: %q", firings, got)
	}
	if s.player.GetHP() != 160 {
		t.Fatalf("HP = %d, want 160 (200 - 40)", s.player.GetHP())
	}
}

// One firing each for the non-DEX arms: C's check_for_bad_stats selects the
// LAST zero in STR/INT/WIS/CHA/DEX order and sends one line per equip
// (src/handler.c:640-675). STR, INT and CHA take no HP.
func TestCrashLoadBadStatsSingleItemArms(t *testing.T) {
	for _, tc := range []struct {
		name     string
		location int
		line     string
	}{
		{"strength", game.ApplyStr, "You are too weak to fight!\n\r"},
		{"intelligence", game.ApplyInt, "You are too dumb to do much of anything!\n\r"},
		{"charisma", game.ApplyCha, "The world hates you!\n\r"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, s := crashBadStatsEnv(t, "Zerostat", 200, tc.location, []savedWornItem{{201, crashWearHead}})
			if err := entryInput(s, "1"); err != nil {
				t.Fatalf("menu entry: %v", err)
			}
			got := strings.Join(drainSessionText(t, s), "")
			if !strings.HasPrefix(got, tc.line) {
				t.Fatalf("bad-stat bytes = %q, want prefix %q", got, tc.line)
			}
			if s.player.GetHP() != 200 {
				t.Fatalf("HP = %d, want 200 (no self-damage arm)", s.player.GetHP())
			}
		})
	}
}

// The CHA arm also starts Pestilence hunting the character when it is not
// already hunting someone (src/handler.c:660-663), with the existing
// LogHuntingStart producer (src/utils.c:715-724).
func TestCrashLoadBadStatsCharismaStartsHunting(t *testing.T) {
	m, _, s := crashBadStatsEnv(t, "Hated", 200, game.ApplyCha, []savedWornItem{{201, crashWearHead}})
	pest, err := m.world.SpawnMobQuiet(7907, 1001)
	if err != nil {
		t.Fatalf("spawn Pestilence: %v", err)
	}
	observer := mudlogObserver(t, m, "ChaWatch", game.LVL_GOD, game.PrfLog1, game.PrfLog2)
	if err := entryInput(s, "1"); err != nil {
		t.Fatalf("menu entry: %v", err)
	}
	got := strings.Join(drainSessionText(t, s), "")
	if !strings.HasPrefix(got, "The world hates you!\n\r") {
		t.Fatalf("bad-stat bytes = %q", got)
	}
	if pest.GetHunting() != "Hated" || pest.GetHuntingID() != strconv.Itoa(s.player.GetID()) {
		t.Fatalf("Pestilence target=%q id=%q, want Hated/%d", pest.GetHunting(), pest.GetHuntingID(), s.player.GetID())
	}
	// C's order: the rent-code producer fires from the file header read
	// (src/objsave.c:519) before the items load and their checks run.
	if line := strings.Join(drainSessionText(t, observer), ""); line != "[ Hated retrieving crash-saved items and entering game. ]\r\n[ Pestilence started hunting Hated ]\r\n" {
		t.Fatalf("observer stream = %q", line)
	}
}

// A rented re-entry takes the other path through enterReturningPlayer: the
// character is rebuilt from the store by reloadCharacterFromStore and then runs
// the same menu entry. The checks must fire once for the entry, never twice.
func TestCrashLoadBadStatsRentedReentryFiresOnce(t *testing.T) {
	_, _, s := crashBadStatsEnv(t, "Renter", 200, game.ApplyDex, crashBadStatsWorn)
	s.player.RentedOut = true
	if err := entryInput(s, "1"); err != nil {
		t.Fatalf("menu entry: %v", err)
	}
	got := strings.Join(drainSessionText(t, s), "")
	if firings := strings.Count(got, "You trip over your own feet"); firings != 4 {
		t.Fatalf("firings = %d, want 4 (one entry, not two): %q", firings, got)
	}
	if s.player.GetHP() != 40 {
		t.Fatalf("HP = %d, want 40 (200 - 4x40)", s.player.GetHP())
	}
}

// C equips nothing when the crash file is absent — Crash_load's fopen failure
// arm returns 1 with the "entering game with no equipment" mudlog and never
// reaches auto_equip (src/objsave.c:474-489). The Go record still carries the
// equipment, so the gate is the stored object save: no file, no checks.
func TestCrashLoadBadStatsNoFileFiresNothing(t *testing.T) {
	_, store, s := crashBadStatsEnv(t, "NoFile", 200, game.ApplyDex, crashBadStatsWorn)
	// Remove the crash file the env wrote, leaving the record's equipment.
	if err := db.DeleteCrashObjectSave(store, "NoFile"); err != nil {
		t.Fatalf("delete crash file: %v", err)
	}
	if err := entryInput(s, "1"); err != nil {
		t.Fatalf("menu entry: %v", err)
	}
	got := strings.Join(drainSessionText(t, s), "")
	if strings.Contains(got, "You trip over your own feet") {
		t.Fatalf("a missing crash file fired the check: %q", got)
	}
	if s.player.GetHP() != 200 {
		t.Fatalf("HP = %d, want 200", s.player.GetHP())
	}
}
