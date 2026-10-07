package session

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// R4 hygiene tests for the agent variable feeds. Each pins a fact the mortal
// game already enforces: no integer enemy HP, sight-gated rooms, no closed
// doors or prototype VNums, stable instance IDs.

func r4World(t *testing.T) (*game.World, *parser.World) {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Lit Room", Zone: 1},
			{VNum: 1002, Name: "Dark Room", Zone: 1, Flags: []string{"1", "0", "0", "0"}},
		},
		Mobs: []parser.Mob{{VNum: 5001, ShortDesc: "a goblin", Level: 1}},
		Objs: []parser.Obj{{VNum: 6001, Keywords: "sword", ShortDesc: "a sword", LongDesc: "A sword."}},
	}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	return w, parsed
}

// Bug 1: FIGHTING must carry do_diagnose's qualitative condition and a bucket
// index — never integer hp/max_hp a mortal cannot see (R4).
func TestVarFightingNoIntegerEnemyHP(t *testing.T) {
	w, _ := r4World(t)
	m := newTestManager(t, w, nil)
	s := makeTestSession(t, m, "Fighter", 1001, true)

	// Establish a real fight through the combat engine.
	mob, err := w.SpawnMob(5001, 1001)
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}
	mob.CurrentHP = 50
	if err := m.combatEngine.StartCombat(s.player, mob); err != nil {
		t.Fatalf("StartCombat: %v", err)
	}

	payload := s.buildVarValue(VarFighting)
	if payload == nil {
		t.Fatal("buildVarValue(FIGHTING) = nil with a live fight")
	}
	encoded, _ := json.Marshal(payload)
	if strings.Contains(string(encoded), `"hp"`) || strings.Contains(string(encoded), `"max_hp"`) {
		t.Fatalf("FIGHTING payload leaks integer HP: %s", encoded)
	}
	obj := payload.(map[string]interface{})
	cond, _ := obj["condition"].(string)
	if !strings.Contains(cond, "condition.") && !strings.Contains(cond, "scratches") && !strings.Contains(cond, "wounds") && !strings.Contains(cond, "hurt") {
		t.Fatalf("condition = %q, want one of diag_char_to_char's eight sentences", cond)
	}
	if b, ok := obj["health_bucket"].(int); !ok || b < 0 || b > 7 {
		t.Fatalf("health_bucket = %v, want 0-7", obj["health_bucket"])
	}
}

// Bug 2: a blind player (or one in a dark room without infravision) gets no
// room name, exits, mobs, or items — the feeds render "Darkness" on screen,
// so they render nothing here.
func TestRoomVarsGatedBySight(t *testing.T) {
	w, _ := r4World(t)
	m := newTestManager(t, w, nil)
	s := makeTestSession(t, m, "Blind", 1001, true)

	if name, _ := s.buildVarValue(VarRoomName).(string); name == "" {
		t.Fatal("precondition: room name visible in a lit room")
	}

	s.player.SetAffectBit(uint64(game.AffBlind), true)
	if name, _ := s.buildVarValue(VarRoomName).(string); name != "" {
		t.Fatalf("blind room_name = %q, want empty", name)
	}
	if exits := s.buildVarValue(VarRoomExits); exits != nil {
		if list, ok := exits.([]string); ok && len(list) != 0 {
			t.Fatalf("blind room_exits = %v, want empty", exits)
		}
	}
	if mobs := s.buildRoomMobs(); len(mobs) != 0 {
		t.Fatalf("blind room_mobs = %d entries, want 0", len(mobs))
	}
	if items := s.buildRoomItems(); len(items) != 0 {
		t.Fatalf("blind room_items = %d entries, want 0", len(items))
	}

	s.player.SetAffectBit(uint64(game.AffBlind), false)
	s.player.SetRoom(1002) // dark room, no infravision
	if name, _ := s.buildVarValue(VarRoomName).(string); name != "" {
		t.Fatalf("dark room_name = %q, want empty (no infravision)", name)
	}
}

// Bug 3: closed doors are omitted from room_exits exactly as do_auto_exits
// omits them for mortals; immortal sees them parenthesized; no VNums.
func TestRoomExitsOmitClosedDoors(t *testing.T) {
	w, parsed := r4World(t)
	_ = parsed
	m := newTestManager(t, w, nil)
	s := makeTestSession(t, m, "Walker", 1001, true)

	room, ok := w.GetRoom(1001)
	if !ok {
		t.Fatal("room 1001 missing")
	}
	room.Exits = map[string]parser.Exit{
		"north": {ToRoom: 1002},
		"south": {ToRoom: 1002, ExitInfo: parser.ExitClosed},
	}

	exits, _ := s.buildVarValue(VarRoomExits).([]string)
	joined := strings.Join(exits, " ")
	if !strings.Contains(joined, "north") {
		t.Fatalf("open exit missing: %v", exits)
	}
	if strings.Contains(joined, "south") {
		t.Fatalf("closed door leaked to mortal: %v", exits)
	}
}

func TestRoomVarsCarryNoVNums(t *testing.T) {
	w, _ := r4World(t)
	m := newTestManager(t, w, nil)
	s := makeTestSession(t, m, "Peek", 1001, true)

	if _, err := w.SpawnMob(5001, 1001); err != nil {
		t.Fatalf("spawn mob: %v", err)
	}
	mobs := s.buildRoomMobs()
	if len(mobs) == 0 {
		t.Fatal("precondition: mob visible")
	}
	encoded, _ := json.Marshal(mobs)
	if strings.Contains(string(encoded), "vnum") {
		t.Fatalf("room_mobs leaks prototype vnum: %s", encoded)
	}
	item := game.NewObjectInstance(&parser.Obj{VNum: 6001, Keywords: "sword", ShortDesc: "a sword", LongDesc: "A sword."}, -1)
	if err := w.MoveObjectToRoom(item, 1001); err != nil {
		t.Fatalf("drop item: %v", err)
	}
	items := s.buildRoomItems()
	if len(items) == 0 {
		t.Fatal("precondition: item visible")
	}
	encodedI, _ := json.Marshal(items)
	if strings.Contains(string(encodedI), "vnum") {
		t.Fatalf("room_items leaks prototype vnum: %s", encodedI)
	}
}

// Bug 4 (objects): raw-constructed objects share registry ID 0 — the feed
// must still hand out distinct, stable identifiers (session-local when the
// world has not registered the object). Two dropped-but-unregistered objects
// and a rebuilt list must show two different, unchanged IDs.
func TestUnregisteredObjectFeedIDsDistinctAndStable(t *testing.T) {
	w, _ := r4World(t)
	m := newTestManager(t, w, nil)
	s := makeTestSession(t, m, "Dropper", 1001, true)

	a := game.NewObjectInstance(&parser.Obj{VNum: 6001, Keywords: "sword", ShortDesc: "a sword", LongDesc: "A sword."}, -1)
	b := game.NewObjectInstance(&parser.Obj{VNum: 6001, Keywords: "sword", ShortDesc: "a sword", LongDesc: "A sword."}, -1)
	if err := w.MoveObjectToRoom(a, 1001); err != nil {
		t.Fatalf("drop a: %v", err)
	}
	if err := w.MoveObjectToRoom(b, 1001); err != nil {
		t.Fatalf("drop b: %v", err)
	}

	first := s.buildRoomItems()
	if len(first) != 2 {
		t.Fatalf("precondition: 2 items, got %d", len(first))
	}
	if first[0].InstanceID == first[1].InstanceID {
		t.Fatalf("unregistered objects share feed ID %q", first[0].InstanceID)
	}
	if !strings.HasPrefix(first[0].InstanceID, "obj_") && !strings.HasPrefix(first[0].InstanceID, "obj_local_") {
		t.Fatalf("feed ID %q has unexpected shape", first[0].InstanceID)
	}

	second := s.buildRoomItems()
	if second[0].InstanceID != first[0].InstanceID || second[1].InstanceID != first[1].InstanceID {
		t.Fatalf("feed IDs unstable across rebuilds: %q/%q -> %q/%q", first[0].InstanceID, first[1].InstanceID, second[0].InstanceID, second[1].InstanceID)
	}
}

// Bug 4: InstanceIDs derive from world-assigned runtime IDs — killing one mob
// must not shift any survivor's identifier.
func TestInstanceIDsStableAcrossDeaths(t *testing.T) {
	w, _ := r4World(t)
	m := newTestManager(t, w, nil)
	s := makeTestSession(t, m, "Killer", 1001, true)

	m1, err := w.SpawnMob(5001, 1001)
	if err != nil {
		t.Fatalf("spawn 1: %v", err)
	}
	m2, err := w.SpawnMob(5001, 1001)
	if err != nil {
		t.Fatalf("spawn 2: %v", err)
	}

	before := s.buildRoomMobs()
	if len(before) != 2 {
		t.Fatalf("precondition: 2 mobs, got %d", len(before))
	}
	// GetMobsInRoom yields newest arrival first (char_to_room prepend order),
	// so list position is not spawn order — identify by runtime ID.
	m1ID, m2ID := "mob_"+itoa(m1.GetID()), "mob_"+itoa(m2.GetID())
	seen := map[string]bool{}
	for _, mv := range before {
		seen[mv.InstanceID] = true
	}
	if !seen[m1ID] || !seen[m2ID] {
		t.Fatalf("precondition: instance IDs %q/%q not both present in %v", m1ID, m2ID, seen)
	}

	// Kill one mob; the survivor's InstanceID must be unchanged.
	m1.SetAlive(false)
	w.ExtractMob(m1)

	after := s.buildRoomMobs()
	if len(after) != 1 {
		t.Fatalf("after death: %d mobs, want 1", len(after))
	}
	if after[0].InstanceID != m2ID {
		t.Fatalf("survivor InstanceID shifted: want %q, got %q", m2ID, after[0].InstanceID)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
