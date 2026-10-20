package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// newActorRoomTestWorld builds a disposable world with the given rooms.
func newActorRoomTestWorld(t *testing.T, rooms ...parser.Room) (*World, map[string]string) {
	t.Helper()
	w, err := NewWorld(&parser.World{Rooms: rooms})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	messages := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) { messages[name] += string(msg) }
	return w, messages
}

// TestSpecialActorLanesRegistered pins the DP-1448 actor registrations. Every
// one of these C bodies acts before it declines, so an NPC command in its room
// reported SpecialUnsupported and was dropped; the registration is what makes
// the body run for the pet (R1/R2).
func TestSpecialActorLanesRegistered(t *testing.T) {
	for _, name := range []string{
		"carrion", "dump", "start_room", "whirlpool", "quan_lo",
		"elements_master_column", "elements_platforms", "elements_galeru_column",
		"elements_galeru_alive", "elements_guardian", "elements_minion",
		"shop_keeper",
	} {
		entry, ok := SpecRegistry[name]
		if !ok || entry.Actor == nil {
			t.Errorf("%s has no actor lane; an NPC command in its room is blocked", name)
		}
	}
}

// TestCRoomPeopleOrdersNewestFirst pins C's front-inserted people list: the
// most recent char_to_room occupant is the head (src/handler.c), for mobiles
// and players alike. The order is visible whenever a special acts on every
// occupant, so it is not a Go convenience.
func TestCRoomPeopleOrdersNewestFirst(t *testing.T) {
	w, _ := newActorRoomTestWorld(t, parser.Room{VNum: 900, Name: "Room"})
	first := NewPlayer(1, "First", 900)
	if err := w.AddPlayer(first); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	mob := newSpecProcTestMob(t, w, 900, 5)
	people := cRoomPeople(w, 900)
	if len(people) != 2 {
		t.Fatalf("people = %d, want 2", len(people))
	}
	if people[0] != Actor(mob) || people[1] != Actor(first) {
		t.Fatalf("order = [%s, %s], want the newest entrant first",
			people[0].GetName(), people[1].GetName())
	}
}

// TestSpecElementsPlatformsMovesMobiles pins the shared body's mob arm: C's
// loop walks world[ch->in_room].people and does not skip NPCs
// (src/spec_procs3.c:1012-1021), so a pet standing in 1326 is sent back to 1314
// with its own acts.
func TestSpecElementsPlatformsMovesMobiles(t *testing.T) {
	w, messages := newActorRoomTestWorld(t,
		parser.Room{VNum: 1314, Name: "Column Base"},
		parser.Room{VNum: 1326, Name: "Earth Platform"},
	)
	player := NewPlayer(1, "PlatformWatcher", 1326)
	player.SetAutoExit(false)
	if err := w.AddPlayer(player); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	pet := newSpecProcTestMob(t, w, 1326, 1)
	pet.SetLiveShortDesc("a horse")

	specElementsPlatformsMove(w, 1326)

	if got := pet.GetRoomVNum(); got != 1314 {
		t.Fatalf("pet room = %d, want 1314", got)
	}
	if got := player.GetRoomVNum(); got != 1314 {
		t.Fatalf("player room = %d, want 1314", got)
	}
	if got := messages[player.GetName()]; !strings.Contains(got, "A horse disappears in a brilliant flash of light.") {
		t.Fatalf("player missing the pet's departure act: %q", got)
	}
}

// TestSpecStartRoomMovesMobilesUpToTheImmortalStop pins both halves of C's
// body that the player lane used to miss: the walk covers mobiles, and it stops
// at the first immortal it meets (src/spec_procs.c:2209-2214).
func TestSpecStartRoomMovesMobilesUpToTheImmortalStop(t *testing.T) {
	w, _ := newActorRoomTestWorld(t,
		parser.Room{VNum: NewbieStartRoom, Name: "Burning Hut"},
		parser.Room{VNum: MortalStartRoom, Name: "Temple Altar"},
		parser.Room{VNum: 8162, Name: "Infirmary"},
	)
	immortal := NewPlayer(1, "StartImmortal", NewbieStartRoom)
	immortal.SetLevel(lvlImmort)
	if err := w.AddPlayer(immortal); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	// Newest entrant: the horse is the head of C's people list, so C moves it
	// before it reaches the immortal and returns FALSE.
	horse := newSpecProcTestMob(t, w, NewbieStartRoom, 1)
	horse.SetLiveShortDesc("a horse")

	if specStartRoomBody(w, NewbieStartRoom) {
		t.Fatal("start_room consumed the command despite an immortal in the room")
	}
	if got := horse.GetRoomVNum(); got != MortalStartRoom {
		t.Fatalf("horse room = %d, want %d (read_mobile leaves hometown 0)", got, MortalStartRoom)
	}
	if got := immortal.GetRoomVNum(); got != NewbieStartRoom {
		t.Fatalf("immortal room = %d, want %d", got, NewbieStartRoom)
	}
}

// TestSpecDumpActorClearsRoomAndDeclines pins SPECIAL(dump)
// (src/spec_procs.c:254-293) for a mobile actor: the room is cleared before the
// command is read, a non-drop command declines so the pet's command runs, and
// the drop arm stays an explicit gap because the port has no mobile do_drop.
func TestSpecDumpActorClearsRoomAndDeclines(t *testing.T) {
	w, messages := newActorRoomTestWorld(t,
		parser.Room{VNum: 8085, Name: "The Dump"},
	)
	watcher := NewPlayer(1, "DumpWatcher", 8085)
	if err := w.AddPlayer(watcher); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	pet := newSpecProcTestMob(t, w, 8085, 1)
	pet.SetLiveShortDesc("a horse")
	barrel := NewObjectInstance(&parser.Obj{
		VNum: 8000, Keywords: "barrel", ShortDesc: "a barrel", Cost: 55,
	}, 8085)
	w.AddItemToRoom(barrel, 8085)

	ctx := CharacterSpecialContext{Actor: pet}
	if got := specDumpActor(w, ctx, nil, "emote", " neighs"); got != SpecialDeclined {
		t.Fatalf("emote = %v, want SpecialDeclined", got)
	}
	if items := w.GetItemsInRoom(8085); len(items) != 0 {
		t.Fatalf("room still holds %d objects", len(items))
	}
	// C's act() CAPs the line and renders $p through OBJS.
	if got := messages[watcher.GetName()]; !strings.Contains(got, "A barrel vanishes in a puff of smoke!") {
		t.Fatalf("watcher output = %q, want the capitalised vanish line", got)
	}
	if got := specDumpActor(w, ctx, nil, "drop", " barrel"); got != SpecialUnsupported {
		t.Fatalf("drop = %v, want SpecialUnsupported (no mobile do_drop)", got)
	}
}

// TestSpecElementsMinionActorPortsC pins SPECIAL(elements_minion)
// (src/spec_procs3.c:1228-1239): every keyword pass destroys the first matching
// object in the minion's own inventory, then the special declines.
func TestSpecElementsMinionActorPortsC(t *testing.T) {
	w, messages := newActorRoomTestWorld(t,
		parser.Room{VNum: 1313, Name: "Minion Room"},
	)
	watcher := NewPlayer(1, "MinionWatcher", 1313)
	if err := w.AddPlayer(watcher); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	minion := newSpecProcTestMob(t, w, 1313, 10)
	talisman := NewObjectInstance(&parser.Obj{
		VNum: 1300, Keywords: "earth talisman", ShortDesc: "an earth talisman",
	}, -1)
	if err := w.MoveObjectToMobInventoryFront(talisman, minion); err != nil {
		t.Fatalf("MoveObjectToMobInventoryFront: %v", err)
	}
	pet := newSpecProcTestMob(t, w, 1313, 1)

	ctx := CharacterSpecialContext{Actor: pet}
	if got := specElementsMinionActor(w, ctx, minion, "emote", ""); got != SpecialDeclined {
		t.Fatalf("result = %v, want SpecialDeclined", got)
	}
	for _, obj := range minion.Inventory {
		if obj != nil && obj.VNum == 1300 {
			t.Fatal("the talisman survived the minion's scan")
		}
	}
	if got := messages[watcher.GetName()]; !strings.Contains(got, "utters the words 'eradico paratus'") {
		t.Fatalf("watcher output = %q, want the disintegrate act", got)
	}
}

// TestSpecQuanLoActorNamesTheActor pins both C messages for a pet actor: the
// gossip and the monk's speech both interpolate GET_NAME(ch)
// (src/spec_procs3.c:380-392), and the special always declines.
func TestSpecQuanLoActorNamesTheActor(t *testing.T) {
	w, messages := newActorRoomTestWorld(t,
		parser.Room{VNum: 19405, Name: "Shaolin"},
	)
	watcher := NewPlayer(1, "QuanWatcher", 19405)
	if err := w.AddPlayer(watcher); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	monk := newSpecProcTestMob(t, w, 19405, 24)
	monk.SetLiveKeywords("master shaolin quan lo")
	pet := newSpecProcTestMob(t, w, 19405, 1)
	pet.SetLiveShortDesc("a horse")

	ctx := CharacterSpecialContext{Actor: pet}
	if got := specQuanLoActor(w, ctx, monk, "look", " quan"); got != SpecialDeclined {
		t.Fatalf("look = %v, want SpecialDeclined", got)
	}
	if got := messages[watcher.GetName()]; !strings.Contains(got, "What is it you seek, a horse?") {
		t.Fatalf("watcher output = %q, want the pet's name in the monk's speech", got)
	}
}

// TestSpecWhirlpoolActorDeclinesWithoutAVictim pins the actor lane's decline:
// with no pullable player in the room C returns FALSE, so the pet's command
// continues. The pull itself is player-only (src/spec_procs2.c:256).
func TestSpecWhirlpoolActorDeclinesWithoutAVictim(t *testing.T) {
	rooms := []parser.Room{{VNum: 12200, Name: "Whirlpool"}}
	for _, vnum := range []int{4600, 4699} {
		rooms = append(rooms, parser.Room{VNum: vnum, Name: "Water"})
	}
	w, _ := newActorRoomTestWorld(t, rooms...)
	pet := newSpecProcTestMob(t, w, 12200, 1)

	ctx := CharacterSpecialContext{Actor: pet}
	if got := specWhirlpoolActor(w, ctx, nil, "emote", ""); got != SpecialDeclined {
		t.Fatalf("result = %v, want SpecialDeclined", got)
	}
}

// TestSpecCarrionActorSpawnsForTheActor pins SPECIAL(carrion) for a mobile
// actor: the stalker is loaded at the actor's room with the actor's level and
// damage roll (src/spec_procs2.c:1742-1749), and the command is consumed.
func TestSpecCarrionActorSpawnsForTheActor(t *testing.T) {
	w, _ := newActorRoomTestWorld(t,
		parser.Room{VNum: 14305, Name: "Carrion Room"},
	)
	w.mu.Lock()
	w.mobs[carrionStalkerVnum] = &parser.Mob{
		VNum: carrionStalkerVnum, Keywords: "carrion stalker",
		ShortDesc: "a carrion stalker", LongDesc: "A carrion stalker is here.",
		Level: 3, HP: parser.DiceRoll{Num: 1, Sides: 8, Plus: 10},
	}
	w.mu.Unlock()
	pet := newSpecProcTestMob(t, w, 14305, 11)
	pet.SetPosition(combat.PosStanding)

	ctx := CharacterSpecialContext{Actor: pet}
	if got := specCarrionActor(w, ctx, nil, "look", " corpse"); got != SpecialHandled {
		t.Fatalf("result = %v, want SpecialHandled", got)
	}
	var stalker *MobInstance
	for _, mob := range w.GetMobsInRoom(14305) {
		if mob.GetVNum() == carrionStalkerVnum {
			stalker = mob
		}
	}
	if stalker == nil {
		t.Fatal("carrion did not load mob 14308 at the actor's room")
	}
	if got := stalker.GetLevel(); got != pet.GetLevel() {
		t.Fatalf("stalker level = %d, want the actor's %d", got, pet.GetLevel())
	}
	if got := stalker.GetDamroll(); got != pet.GetLevel() {
		t.Fatalf("stalker damroll = %d, want the actor's level %d", got, pet.GetLevel())
	}
	// A sleeping actor with positive HP is C's first gate.
	pet.SetPosition(combat.PosSleeping)
	if got := specCarrionActor(w, ctx, nil, "look", " corpse"); got != SpecialDeclined {
		t.Fatalf("sleeping result = %v, want SpecialDeclined", got)
	}
}
