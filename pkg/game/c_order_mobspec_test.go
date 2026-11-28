package game

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newCOrderTestWorld(t testing.TB, mobs []parser.Mob, roomVNums ...int) *World {
	t.Helper()
	rooms := make([]parser.Room, 0, len(roomVNums))
	for _, rv := range roomVNums {
		rooms = append(rooms, parser.Room{VNum: rv, Name: fmt.Sprintf("Room %d", rv), Zone: 1})
	}
	w, err := NewWorld(&parser.World{Rooms: rooms, Mobs: mobs})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	w.MessageSink = func(string, []byte) {}
	return w
}

func makeCOrderMobProto(vnum int, name, shortDesc string, level, align int) parser.Mob {
	return parser.Mob{
		VNum:       vnum,
		Keywords:   name,
		ShortDesc:  shortDesc,
		LongDesc:   shortDesc + " is standing here.",
		Level:      level,
		Alignment:  align,
		DefaultPos: 8,
		Position:   8,
	}
}

// TestCOrderFirstMatch_EntryProg verifies R5h for first-match mob selection in
// EntryProg (src/mobprog.c:296): captain of the guard (8059) must greet the
// newest-entered guard/citizen in world[room].people, 50/50 times.
func TestCOrderFirstMatch_EntryProg(t *testing.T) {
	const roomVNum = 8000
	protos := []parser.Mob{
		makeCOrderMobProto(8059, "captain", "the captain of the guard", 20, 0),
		makeCOrderMobProto(8001, "guardA", "guard Alpha", 10, 0),
		makeCOrderMobProto(8002, "guardB", "guard Beta", 10, 0),
		makeCOrderMobProto(8027, "guardC", "guard Gamma", 10, 0),
	}
	for iter := range 50 {
		w := newCOrderTestWorld(t, protos, roomVNum)
		var observed []string
		w.MessageSink = func(_ string, msg []byte) { observed = append(observed, string(msg)) }

		observer := NewPlayer(1, "Observer", roomVNum)
		if err := w.AddPlayer(observer); err != nil {
			t.Fatalf("iter %d: AddPlayer: %v", iter, err)
		}

		// Spawn in known order: Alpha (oldest), Beta (middle), Gamma (newest).
		if _, err := w.SpawnMob(8001, roomVNum); err != nil {
			t.Fatalf("iter %d: spawn 8001: %v", iter, err)
		}
		if _, err := w.SpawnMob(8002, roomVNum); err != nil {
			t.Fatalf("iter %d: spawn 8002: %v", iter, err)
		}
		if _, err := w.SpawnMob(8027, roomVNum); err != nil {
			t.Fatalf("iter %d: spawn 8027: %v", iter, err)
		}
		captain, err := w.SpawnMob(8059, roomVNum)
		if err != nil {
			t.Fatalf("iter %d: spawn 8059: %v", iter, err)
		}

		w.EntryProg(captain, roomVNum)
		joined := strings.Join(observed, "\n")
		if !strings.Contains(joined, "Guard Gamma snaps to attention") {
			t.Fatalf("iter %d: expected newest guard Gamma to salute first, got:\n%s", iter, joined)
		}
	}
}

// TestCOrderListOutput_AssassinRoster verifies R5h for list-order output in
// SPECIAL(assassin) (src/spec_procs2.c:856): listing available assassins in
// room+1 must print newest-first (C world[as_loc].people order), 50/50 times.
func TestCOrderListOutput_AssassinRoster(t *testing.T) {
	const shopRoom = 9000
	const storeRoom = 9001
	protos := []parser.Mob{
		makeCOrderMobProto(9101, "alpha", "Alpha Assassin", 10, 0),
		makeCOrderMobProto(9102, "beta", "Beta Assassin", 12, 0),
		makeCOrderMobProto(9103, "gamma", "Gamma Assassin", 14, 0),
	}
	for iter := range 50 {
		w := newCOrderTestWorld(t, protos, shopRoom, storeRoom)
		if _, err := w.SpawnMob(9101, storeRoom); err != nil {
			t.Fatalf("iter %d: spawn 9101: %v", iter, err)
		}
		if _, err := w.SpawnMob(9102, storeRoom); err != nil {
			t.Fatalf("iter %d: spawn 9102: %v", iter, err)
		}
		if _, err := w.SpawnMob(9103, storeRoom); err != nil {
			t.Fatalf("iter %d: spawn 9103: %v", iter, err)
		}

		roster := assassinRoster(w, storeRoom)
		if len(roster) != 3 {
			t.Fatalf("iter %d: want 3 assassins, got %d", iter, len(roster))
		}
		got := []string{roster[0].GetName(), roster[1].GetName(), roster[2].GetName()}
		want := []string{"Gamma Assassin", "Beta Assassin", "Alpha Assassin"}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("iter %d: assassinRoster order = %v, want %v", iter, got, want)
			}
		}
	}
}

// TestCOrderFirstMatch_SpecRescuer verifies R5h for SPECIAL(rescuer)
// (src/spec_procs2.c:535): when three allies are fighting a player, the
// rescuer must pick the newest arrival in world[room].people, 50/50 times.
func TestCOrderFirstMatch_SpecRescuer(t *testing.T) {
	const roomVNum = 9100
	protos := []parser.Mob{
		makeCOrderMobProto(9200, "rescuer", "a noble rescuer", 20, 0),
		makeCOrderMobProto(9201, "allyA", "Ally Alpha", 10, 0),
		makeCOrderMobProto(9202, "allyB", "Ally Beta", 10, 0),
		makeCOrderMobProto(9203, "allyC", "Ally Gamma", 10, 0),
	}
	for iter := range 50 {
		w := newCOrderTestWorld(t, protos, roomVNum)
		player := NewPlayer(1, "Foe", roomVNum)
		if err := w.AddPlayer(player); err != nil {
			t.Fatalf("iter %d: AddPlayer: %v", iter, err)
		}

		allyA, _ := w.SpawnMob(9201, roomVNum)
		allyB, _ := w.SpawnMob(9202, roomVNum)
		allyC, _ := w.SpawnMob(9203, roomVNum)
		rescuer, _ := w.SpawnMob(9200, roomVNum)
		allyA.SetFightingBody(player)
		allyB.SetFightingBody(player)
		allyC.SetFightingBody(player)

		var rescuedArg string
		w.CharacterCommandDirectExecFunc = func(_ combat.Combatant, cmd, arg string, _ int) CharacterCommandResult {
			if cmd == "rescue" {
				rescuedArg = arg
			}
			return CharacterCommandHandled
		}

		if !specRescuer(w, nil, rescuer, "", "") {
			t.Fatalf("iter %d: specRescuer returned false", iter)
		}
		if rescuedArg != "Ally Gamma" {
			t.Fatalf("iter %d: specRescuer rescued %q, want newest arrival %q", iter, rescuedArg, "Ally Gamma")
		}
	}
}

// TestCOrderPerItemDraw_UndeadKnight verifies R5h for per-item RNG draws in
// specUndeadKnight (src/spec_procs.c:1192,1244): with three hated mobs in the
// room and a fixed RNG seed, the same mob must be chosen in C's newest-first
// order 50/50 times.
func TestCOrderPerItemDraw_UndeadKnight(t *testing.T) {
	const roomVNum = 9200
	const hatedVNum = 11471
	// Find a seed where the first number(0,2) is non-zero and the second is 0,
	// so the middle (2nd newest) mob is selected iff order is deterministic.
	var chosenSeed uint32
	for seed := uint32(1); seed < 1000; seed++ {
		dprng.ResetStream(seed)
		if number(0, 2) != 0 && number(0, 2) == 0 {
			chosenSeed = seed
			break
		}
	}
	if chosenSeed == 0 {
		t.Fatal("failed to find seed for 2nd-item hit")
	}

	protos := []parser.Mob{
		makeCOrderMobProto(11470, "black", "Black Knight", 20, -500),
		makeCOrderMobProto(hatedVNum, "red", "Red Knight", 20, -500),
	}
	for iter := range 50 {
		w := newCOrderTestWorld(t, protos, roomVNum)
		oldCB := combat.GetCallbacks()
		ce := combat.NewCombatEngine()
		ce.SetCallbacks(w.WireCombatCallbacks())
		w.SetCombatEngine(ce)

		m1, _ := w.SpawnMob(hatedVNum, roomVNum)
		m2, _ := w.SpawnMob(hatedVNum, roomVNum)
		m3, _ := w.SpawnMob(hatedVNum, roomVNum)
		m1.prototype.Store(&parser.Mob{VNum: hatedVNum, Keywords: "red1", ShortDesc: "Red One"})
		m2.prototype.Store(&parser.Mob{VNum: hatedVNum, Keywords: "red2", ShortDesc: "Red Two"})
		m3.prototype.Store(&parser.Mob{VNum: hatedVNum, Keywords: "red3", ShortDesc: "Red Three"})
		knight, _ := w.SpawnMob(11470, roomVNum)

		dprng.ResetStream(chosenSeed)
		ok := specUndeadKnight(w, knight, "", hatedVNum, "$n yells!", func() {})
		ce.Stop()
		combat.SetCallbacks(oldCB)
		if !ok {
			t.Fatalf("iter %d: specUndeadKnight returned false", iter)
		}
		// Newest-first order is m3, m2, m1; draw 1 != 0 skips m3, draw 2 == 0 hits m2.
		if knight.GetFightingBody() != m2 {
			t.Fatalf("iter %d: target = %v, want middle arrival m2 (%q)", iter, knight.GetFightingBody(), m2.GetName())
		}
	}
}

// TestCOrderFirstMatch_CityguardCombatants verifies R5h for mixed player+mob
// world[room].people ordering in cityguardRoomCombatants (src/spec_procs.c:785,800).
func TestCOrderFirstMatch_CityguardCombatants(t *testing.T) {
	const roomVNum = 9300
	protos := []parser.Mob{
		makeCOrderMobProto(9301, "mobA", "Mob Alpha", 10, -500),
		makeCOrderMobProto(9302, "mobB", "Mob Beta", 10, -500),
	}
	for iter := range 50 {
		w := newCOrderTestWorld(t, protos, roomVNum)
		mA, _ := w.SpawnMob(9301, roomVNum)
		p := NewPlayer(1, "MiddlePC", roomVNum)
		if err := w.AddPlayer(p); err != nil {
			t.Fatalf("iter %d: AddPlayer: %v", iter, err)
		}
		mB, _ := w.SpawnMob(9302, roomVNum)

		got := cityguardRoomCombatants(w, roomVNum)
		if len(got) != 3 {
			t.Fatalf("iter %d: want 3 combatants, got %d", iter, len(got))
		}
		if got[0] != mB || got[1] != p || got[2] != mA {
			t.Fatalf("iter %d: cityguardRoomCombatants = [%v %v %v], want [mB p mA]", iter, got[0], got[1], got[2])
		}
	}
}

func benchmarkRoomMobLookup(b *testing.B, totalMobs, roomMobs int, ordered bool) {
	protos := []parser.Mob{makeCOrderMobProto(100, "mob", "a test mob", 10, 0)}
	w := newCOrderTestWorld(b, protos, 1, 2)
	for i := range totalMobs {
		rv := 2
		if i < roomMobs {
			rv = 1
		}
		if _, err := w.SpawnMob(100, rv); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		if ordered {
			_ = w.COrderedRoomMobs(1)
		} else {
			_ = w.GetMobsInRoom(1)
		}
	}
}

func BenchmarkGetMobsInRoom_300Total_1InRoom(b *testing.B) { benchmarkRoomMobLookup(b, 300, 1, false) }

func BenchmarkCOrderedRoomMobs_300Total_1InRoom(b *testing.B) {
	benchmarkRoomMobLookup(b, 300, 1, true)
}

func BenchmarkGetMobsInRoom_300Total_5InRoom(b *testing.B) { benchmarkRoomMobLookup(b, 300, 5, false) }

func BenchmarkCOrderedRoomMobs_300Total_5InRoom(b *testing.B) {
	benchmarkRoomMobLookup(b, 300, 5, true)
}

func BenchmarkGetMobsInRoom_300Total_20InRoom(b *testing.B) {
	benchmarkRoomMobLookup(b, 300, 20, false)
}

func BenchmarkCOrderedRoomMobs_300Total_20InRoom(b *testing.B) {
	benchmarkRoomMobLookup(b, 300, 20, true)
}
