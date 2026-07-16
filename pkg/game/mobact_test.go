package game

import (
	"reflect"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

type mobactDrawRange struct {
	from int
	to   int
}

func captureMobactDraws(t *testing.T) *[]mobactDrawRange {
	t.Helper()
	draws := make([]mobactDrawRange, 0, 4)
	previous := mobactNumber
	mobactNumber = func(from, to int) int {
		draws = append(draws, mobactDrawRange{from: from, to: to})
		return dprng.Number(from, to)
	}
	t.Cleanup(func() { mobactNumber = previous })
	return &draws
}

func newMobactDrawTestWorld() *World {
	room := &parser.Room{VNum: 1001, Name: "Draw Test Room", Zone: 1}
	snapshots := NewSnapshotManager()
	snapshots.Publish(map[int]*parser.Room{room.VNum: room})
	return &World{
		snapshots:  snapshots,
		rooms:      map[int]*parser.Room{room.VNum: room},
		mobs:       make(map[int]*parser.Mob),
		players:    make(map[string]*Player),
		activeMobs: make(map[int]*MobInstance),
		roomItems:  make(map[int][]*ObjectInstance),
	}
}

func newMobactDrawTestMob(flags ...string) *MobInstance {
	return NewMobInstance(&parser.Mob{
		VNum:        9401,
		Keywords:    "draw test mob",
		ShortDesc:   "a draw test mob",
		ActionFlags: flags,
		HP:          parser.DiceRoll{Num: 1, Sides: 1, Plus: 10},
	}, 1001)
}

func assertMobactDraws(t *testing.T, got *[]mobactDrawRange, want []mobactDrawRange) {
	t.Helper()
	if !reflect.DeepEqual(*got, want) {
		t.Fatalf("draw sequence = %+v, want %+v", *got, want)
	}
}

func assertTwoDrawStreamPosition(t *testing.T, seed uint32) {
	t.Helper()
	wantStream := dprng.New(seed)
	wantStream.Number(0, 18)
	wantStream.Number(0, 15)
	wantNext := wantStream.Next()
	if gotNext := dprng.Next(); gotNext != wantNext {
		t.Fatalf("stream after mob tick = %d, want %d (exactly two draws)", gotNext, wantNext)
	}
}

func TestMobileActivitySentinelBurnsMovementAndSoundDraws(t *testing.T) {
	w := newMobactDrawTestWorld()
	mob := newMobactDrawTestMob("sentinel")
	startRoom := mob.GetRoom()

	const seed = 1
	dprng.ResetStream(seed)
	draws := captureMobactDraws(t)
	w.mobileActivityForMob(mob)

	assertMobactDraws(t, draws, []mobactDrawRange{{0, 18}, {0, 15}})
	if got := mob.GetRoom(); got != startRoom {
		t.Fatalf("sentinel moved from %d to %d", startRoom, got)
	}
	assertTwoDrawStreamPosition(t, seed)
}

func TestMobileActivityStandingMobBurnsSameTwoDraws(t *testing.T) {
	w := newMobactDrawTestWorld()
	mob := newMobactDrawTestMob()

	const seed = 1
	dprng.ResetStream(seed)
	draws := captureMobactDraws(t)
	w.mobileActivityForMob(mob)

	assertMobactDraws(t, draws, []mobactDrawRange{{0, 18}, {0, 15}})
	assertTwoDrawStreamPosition(t, seed)
}

func TestMobileActivitySoundDrawPrecedesAggressiveDraw(t *testing.T) {
	w := newMobactDrawTestWorld()
	w.combatEngine = &testCombatEngine{}
	mob := newMobactDrawTestMob("aggressive", "sentinel")
	player := NewPlayer(1, "Sneaky", 1001)
	player.Affects |= 1 << affSneak
	w.players[player.Name] = player

	dprng.ResetStream(1)
	draws := captureMobactDraws(t)
	w.mobileActivityForMob(mob)

	assertMobactDraws(t, draws, []mobactDrawRange{{0, 18}, {0, 15}, {0, 3}})
}

func TestMobileActivityEmptyScavengerRoomSkipsScavengeDraw(t *testing.T) {
	w := newMobactDrawTestWorld()
	mob := newMobactDrawTestMob("scavenger", "sentinel")

	dprng.ResetStream(1)
	draws := captureMobactDraws(t)
	w.mobileActivityForMob(mob)

	assertMobactDraws(t, draws, []mobactDrawRange{{0, 18}, {0, 15}})
}

func TestMobileActivityScavengerDrawPrecedesMovementWhenRoomHasContents(t *testing.T) {
	w := newMobactDrawTestWorld()
	mob := newMobactDrawTestMob("scavenger", "sentinel")
	w.roomItems[mob.GetRoom()] = []*ObjectInstance{{}}

	dprng.ResetStream(1)
	draws := captureMobactDraws(t)
	w.mobileActivityForMob(mob)

	assertMobactDraws(t, draws, []mobactDrawRange{{0, 10}, {0, 18}, {0, 15}})
}

// testCombatEngine is a minimal CombatEngine implementation for mobact tests.
type testCombatEngine struct {
	starts []testCombatPair
}

type testCombatPair struct {
	attacker string
	defender string
}

func (t *testCombatEngine) StartCombat(attacker, defender combat.Combatant) error {
	t.starts = append(t.starts, testCombatPair{attacker.GetName(), defender.GetName()})
	return nil
}

func (t *testCombatEngine) IsFighting(name string) bool {
	for _, p := range t.starts {
		if p.attacker == name || p.defender == name {
			return true
		}
	}
	return false
}

func (t *testCombatEngine) GetCombatTarget(charName string) (combat.Combatant, bool) {
	return nil, false
}

func (t *testCombatEngine) reset() {
	t.starts = t.starts[:0]
}

func newMobactTestWorld(t *testing.T) (*World, *testCombatEngine) {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Room 1", Zone: 1}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	ce := &testCombatEngine{}
	w.combatEngine = ce
	return w, ce
}

func newMobactTestMob(w *World, vnum, race, alignment int, flags ...string) *MobInstance {
	proto := &parser.Mob{
		VNum:        vnum,
		Race:        race,
		Alignment:   alignment,
		ActionFlags: flags,
		Level:       1,
		HP:          parser.DiceRoll{Num: 1, Sides: 1, Plus: 10},
	}
	w.mu.Lock()
	w.mobs[vnum] = proto
	w.mu.Unlock()
	mob, _ := w.SpawnMob(vnum, 1001)
	return mob
}

func newMobactTestPlayer(w *World, name string) *Player {
	p := NewPlayer(1, name, 1001)
	w.mu.Lock()
	w.players[name] = p
	w.mu.Unlock()
	return p
}

func TestRaceHateAggression_MobAttacksHater(t *testing.T) {
	w, ce := newMobactTestWorld(t)
	mob := newMobactTestMob(w, 9001, 7, 0, "sentinel")
	p := newMobactTestPlayer(w, "Hater")
	p.RaceHates[0] = mob.Prototype.Race

	w.mobileActivityForMob(mob)

	if len(ce.starts) != 1 {
		t.Fatalf("expected 1 combat start, got %d", len(ce.starts))
	}
	if ce.starts[0].attacker != mob.GetName() || ce.starts[0].defender != p.Name {
		t.Fatalf("unexpected combat pair: %+v", ce.starts[0])
	}
}

func TestRaceHateAggression_ShopKeeperSkips(t *testing.T) {
	w, ce := newMobactTestWorld(t)
	mob := newMobactTestMob(w, 9002, 7, 0, "sentinel")
	p := newMobactTestPlayer(w, "Hater")
	p.RaceHates[0] = mob.Prototype.Race

	old := MobSpecAssign[mob.Prototype.VNum]
	MobSpecAssign[mob.Prototype.VNum] = "shop_keeper"
	defer func() {
		if old == "" {
			delete(MobSpecAssign, mob.Prototype.VNum)
		} else {
			MobSpecAssign[mob.Prototype.VNum] = old
		}
	}()

	w.mobileActivityForMob(mob)

	if len(ce.starts) != 0 {
		t.Fatalf("expected 0 combat starts for shop_keeper, got %d", len(ce.starts))
	}
}

func TestRaceHateAggression_ProtectEvilBlocks(t *testing.T) {
	w, ce := newMobactTestWorld(t)
	mob := newMobactTestMob(w, 9003, 7, 0, "sentinel") // non-evil mob
	p := newMobactTestPlayer(w, "Hater")
	p.RaceHates[0] = mob.Prototype.Race
	p.Affects |= 1 << affProtectEvil

	w.mobileActivityForMob(mob)

	if len(ce.starts) != 0 {
		t.Fatalf("expected 0 combat starts with protect evil vs non-evil mob, got %d", len(ce.starts))
	}
}

func TestRaceHateAggression_ProtectEvilEvilPasses(t *testing.T) {
	w, ce := newMobactTestWorld(t)
	mob := newMobactTestMob(w, 9004, 7, -500, "sentinel") // evil mob
	p := newMobactTestPlayer(w, "Hater")
	p.RaceHates[0] = mob.Prototype.Race
	p.Affects |= 1 << affProtectEvil

	// With a 1-in-6 bypass chance and an unseedable global RNG, run enough
	// iterations to be statistically certain of at least one bypass.
	attacks := 0
	for i := 0; i < 200; i++ {
		ce.reset()
		w.mobileActivityForMob(mob)
		if len(ce.starts) > 0 {
			attacks++
			break
		}
	}
	if attacks == 0 {
		t.Fatal("expected at least one bypass of protect evil by evil mob in 200 iterations")
	}
}

func TestDoubleSpeedHunt_CalledTwice(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Room 1", Zone: 1, Exits: map[string]parser.Exit{"north": {ToRoom: 1002}}},
			{VNum: 1002, Name: "Room 2", Zone: 1, Exits: map[string]parser.Exit{"north": {ToRoom: 1003}, "south": {ToRoom: 1001}}},
			{VNum: 1003, Name: "Room 3", Zone: 1, Exits: map[string]parser.Exit{"south": {ToRoom: 1002}}},
		},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	mob := newMobactTestMob(w, 9005, 1, 0, "hunter")
	p := newMobactTestPlayer(w, "Prey")
	p.RoomVNum = 1003
	mob.SetHunting(p.Name)

	w.mobileActivityForMob(mob)

	if mob.GetRoom() != 1003 {
		t.Fatalf("hunter mob ended in room %d, want 1003 (should have moved twice)", mob.GetRoom())
	}
}

// -----------------------------------------------------------------------------
// DP-1034: mob aggression visibility / NOHASSLE / sneak / peaceful-room gates
// -----------------------------------------------------------------------------

func TestAggressive_CannotSeePlayer_Skips(t *testing.T) {
	w, ce := newMobactTestWorld(t)
	mob := newMobactTestMob(w, 9101, 1, 0, "aggressive", "sentinel")
	p := newMobactTestPlayer(w, "Invisible")
	p.Affects |= 1 << affInvisible

	w.mobileActivityForMob(mob)

	if len(ce.starts) != 0 {
		t.Fatalf("expected 0 combat starts for invisible player, got %d", len(ce.starts))
	}
}

func TestAggressive_NohasslePlayer_Skips(t *testing.T) {
	w, ce := newMobactTestWorld(t)
	mob := newMobactTestMob(w, 9102, 1, 0, "aggressive", "sentinel")
	p := newMobactTestPlayer(w, "NoHassle")
	p.SetPlrFlag(PrfNohassle, true)

	w.mobileActivityForMob(mob)

	if len(ce.starts) != 0 {
		t.Fatalf("expected 0 combat starts for NOHASSLE player, got %d", len(ce.starts))
	}
}

func TestAggressive_SneakPlayer_AttacksLessOften(t *testing.T) {
	w, ce := newMobactTestWorld(t)
	mob := newMobactTestMob(w, 9103, 1, 0, "aggressive", "sentinel")
	p := newMobactTestPlayer(w, "Sneaky")
	p.Affects |= 1 << affSneak

	sneakAttacks := 0
	const iterations = 200
	for i := 0; i < iterations; i++ {
		ce.reset()
		w.mobileActivityForMob(mob)
		if len(ce.starts) > 0 {
			sneakAttacks++
		}
	}

	// Non-sneaking player should be attacked every iteration.
	p2 := newMobactTestPlayer(w, "Loud")
	p2.RoomVNum = 1001
	plainAttacks := 0
	for i := 0; i < iterations; i++ {
		ce.reset()
		w.mobileActivityForMob(mob)
		if len(ce.starts) > 0 {
			plainAttacks++
		}
	}

	if plainAttacks != iterations {
		t.Fatalf("expected non-sneaking player attacked every iteration, got %d/%d", plainAttacks, iterations)
	}
	if sneakAttacks >= plainAttacks {
		t.Fatalf("expected sneaking player attacked less often than non-sneaking, got %d vs %d", sneakAttacks, plainAttacks)
	}
}

func TestAggressive_PlayerAttacked(t *testing.T) {
	w, ce := newMobactTestWorld(t)
	mob := newMobactTestMob(w, 9104, 1, 0, "aggressive", "sentinel")
	p := newMobactTestPlayer(w, "Victim")

	w.mobileActivityForMob(mob)

	if len(ce.starts) != 1 {
		t.Fatalf("expected 1 combat start, got %d", len(ce.starts))
	}
	if ce.starts[0].attacker != mob.GetName() || ce.starts[0].defender != p.Name {
		t.Fatalf("unexpected combat pair: %+v", ce.starts[0])
	}
}

func TestMemory_CannotSeePlayer_Skips(t *testing.T) {
	w, ce := newMobactTestWorld(t)
	mob := newMobactTestMob(w, 9201, 1, 0, "memory", "sentinel")
	p := newMobactTestPlayer(w, "Forgotten")
	mob.Memory = []string{p.Name}
	p.Affects |= 1 << affInvisible

	w.mobileActivityForMob(mob)

	if len(ce.starts) != 0 {
		t.Fatalf("expected 0 memory combat starts for invisible player, got %d", len(ce.starts))
	}
}

func TestMemory_NohasslePlayer_Skips(t *testing.T) {
	w, ce := newMobactTestWorld(t)
	mob := newMobactTestMob(w, 9202, 1, 0, "memory", "sentinel")
	p := newMobactTestPlayer(w, "HassleFree")
	mob.Memory = []string{p.Name}
	p.SetPlrFlag(PrfNohassle, true)

	w.mobileActivityForMob(mob)

	if len(ce.starts) != 0 {
		t.Fatalf("expected 0 memory combat starts for NOHASSLE player, got %d", len(ce.starts))
	}
}

func TestMemory_PlayerAttacked(t *testing.T) {
	w, ce := newMobactTestWorld(t)
	mob := newMobactTestMob(w, 9203, 1, 0, "memory", "sentinel")
	p := newMobactTestPlayer(w, "Remembered")
	mob.Memory = []string{p.Name}

	w.mobileActivityForMob(mob)

	if len(ce.starts) != 1 {
		t.Fatalf("expected 1 memory combat start, got %d", len(ce.starts))
	}
	if ce.starts[0].attacker != mob.GetName() || ce.starts[0].defender != p.Name {
		t.Fatalf("unexpected combat pair: %+v", ce.starts[0])
	}
}

func TestAggr24_CannotSeePlayer_Skips(t *testing.T) {
	w, ce := newMobactTestWorld(t)
	mob := newMobactTestMob(w, 9301, 1, 0, "aggr24", "sentinel")
	p := newMobactTestPlayer(w, "Invisible24")
	p.Level = 24
	p.Affects |= 1 << affInvisible

	w.mobileActivityForMob(mob)

	if len(ce.starts) != 0 {
		t.Fatalf("expected 0 AGGR24 combat starts for invisible player, got %d", len(ce.starts))
	}
}

func TestAggr24_NohasslePlayer_Skips(t *testing.T) {
	w, ce := newMobactTestWorld(t)
	mob := newMobactTestMob(w, 9302, 1, 0, "aggr24", "sentinel")
	p := newMobactTestPlayer(w, "NoHassle24")
	p.Level = 24
	p.SetPlrFlag(PrfNohassle, true)

	w.mobileActivityForMob(mob)

	if len(ce.starts) != 0 {
		t.Fatalf("expected 0 AGGR24 combat starts for NOHASSLE player, got %d", len(ce.starts))
	}
}

func TestAggr24_PeacefulRoom_Skips(t *testing.T) {
	w, ce := newMobactTestWorld(t)
	w.rooms[1001].Flags = append(w.rooms[1001].Flags, "peaceful")
	mob := newMobactTestMob(w, 9303, 1, 0, "aggr24", "sentinel")
	p := newMobactTestPlayer(w, "Peaceful24")
	p.Level = 24

	w.mobileActivityForMob(mob)

	if len(ce.starts) != 0 {
		t.Fatalf("expected 0 AGGR24 combat starts in peaceful room, got %d", len(ce.starts))
	}
}

func TestAggr24_PlayerAttacked(t *testing.T) {
	w, ce := newMobactTestWorld(t)
	mob := newMobactTestMob(w, 9304, 1, 0, "aggr24", "sentinel")
	p := newMobactTestPlayer(w, "Victim24")
	p.Level = 24

	w.mobileActivityForMob(mob)

	if len(ce.starts) != 1 {
		t.Fatalf("expected 1 AGGR24 combat start, got %d", len(ce.starts))
	}
	if ce.starts[0].attacker != mob.GetName() || ce.starts[0].defender != p.Name {
		t.Fatalf("unexpected combat pair: %+v", ce.starts[0])
	}
}
