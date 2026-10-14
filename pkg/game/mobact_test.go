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
	return captureMobactDrawsWith(t, dprng.Number)
}

func captureMobactDrawsWith(t *testing.T, draw func(from, to int) int) *[]mobactDrawRange {
	t.Helper()
	draws := make([]mobactDrawRange, 0, 4)
	previous := mobactNumber
	mobactNumber = func(from, to int) int {
		draws = append(draws, mobactDrawRange{from: from, to: to})
		return draw(from, to)
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

func newMobactWanderTestWorld(t *testing.T, targetSector int, flags ...string) (*World, *MobInstance) {
	return newMobactWanderTestWorldWithTargetFlags(t, targetSector, nil, flags...)
}

func newMobactWanderTestWorldWithTargetFlags(t *testing.T, targetSector int, targetRoomFlags []string, flags ...string) (*World, *MobInstance) {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Source", Zone: 1, Exits: map[string]parser.Exit{"north": {ToRoom: 1002}}},
			{VNum: 1002, Name: "Target", Zone: 1, Sector: targetSector, Flags: targetRoomFlags},
		},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	mob := NewMob(&parser.Mob{
		VNum: 9402, Keywords: "wanderer", ShortDesc: "a wanderer",
		ActionFlags: flags, Position: 8, DefaultPos: 8, Level: 10, Race: 1,
	}, 1001)
	w.mu.Lock()
	w.activeMobs[mob.GetID()] = mob
	w.mu.Unlock()
	return w, mob
}

func pinNorthWanderDraws(t *testing.T) *[]mobactDrawRange {
	t.Helper()
	return captureMobactDrawsWith(t, func(from, to int) int {
		if from == 0 && to == 18 {
			return 0 // north: same draw gates movement and selects the exit
		}
		if from == 0 && to == 15 {
			return 1 // do not invoke the sound branch
		}
		return from
	})
}

func TestMobileActivityWaterAndFlyingSectorGates(t *testing.T) {
	tests := []struct {
		name          string
		sector        int
		affect        int
		boatInventory bool
		boatEquipment bool
		level         int
		targetFlags   []string
		wantRoom      int
		wantDraws     []mobactDrawRange
	}{
		{name: "water swim requires capability", sector: SECT_WATER_SWIM, wantRoom: 1001, wantDraws: []mobactDrawRange{{0, 18}}},
		{name: "water noswim requires capability", sector: SECT_WATER_NOSWIM, wantRoom: 1001, wantDraws: []mobactDrawRange{{0, 18}}},
		{name: "death room gate precedes swim check", sector: SECT_WATER_SWIM, targetFlags: []string{"2"}, wantRoom: 1001, wantDraws: []mobactDrawRange{{0, 18}, {0, 15}}},
		{name: "no-mob room gate precedes flight check", sector: SECT_FLYING, targetFlags: []string{"4"}, wantRoom: 1001, wantDraws: []mobactDrawRange{{0, 18}, {0, 15}}},
		{name: "waterwalk crosses swim water", sector: SECT_WATER_SWIM, affect: affWaterWalk, wantRoom: 1002, wantDraws: []mobactDrawRange{{0, 18}, {0, 15}}},
		{name: "flight crosses noswim water", sector: SECT_WATER_NOSWIM, affect: affFly, wantRoom: 1002, wantDraws: []mobactDrawRange{{0, 18}, {0, 15}}},
		{name: "inventory boat crosses water", sector: SECT_WATER_SWIM, boatInventory: true, wantRoom: 1002, wantDraws: []mobactDrawRange{{0, 18}, {0, 15}}},
		{name: "equipped boat crosses noswim water", sector: SECT_WATER_NOSWIM, boatEquipment: true, wantRoom: 1002, wantDraws: []mobactDrawRange{{0, 18}, {0, 15}}},
		{name: "immortal level crosses water", sector: SECT_WATER_SWIM, level: LVL_IMMORT, wantRoom: 1002, wantDraws: []mobactDrawRange{{0, 18}, {0, 15}}},
		{name: "flight crosses flying sector", sector: SECT_FLYING, affect: affFly, wantRoom: 1002, wantDraws: []mobactDrawRange{{0, 18}, {0, 15}}},
		{name: "waterwalk does not cross flying sector", sector: SECT_FLYING, affect: affWaterWalk, wantRoom: 1001, wantDraws: []mobactDrawRange{{0, 18}}},
		{name: "boat does not cross flying sector", sector: SECT_FLYING, boatInventory: true, wantRoom: 1001, wantDraws: []mobactDrawRange{{0, 18}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, mob := newMobactWanderTestWorldWithTargetFlags(t, tt.sector, tt.targetFlags)
			if tt.affect != 0 {
				mob.SetAffected(tt.affect)
			}
			if tt.level != 0 {
				mob.SetLevel(tt.level)
			}
			if tt.boatInventory || tt.boatEquipment {
				boat := &ObjectInstance{Prototype: &parser.Obj{VNum: 9403, TypeFlag: ITEM_BOAT}}
				if tt.boatInventory {
					mob.Inventory = append(mob.Inventory, boat)
				}
				if tt.boatEquipment {
					mob.Equipment[0] = boat
				}
			}

			draws := pinNorthWanderDraws(t)
			w.MobileActivity()
			if got := mob.GetRoom(); got != tt.wantRoom {
				t.Fatalf("mob room = %d, want %d", got, tt.wantRoom)
			}
			assertMobactDraws(t, draws, tt.wantDraws)
		})
	}
}

func TestMobileActivityBlockedSectorSkipsAggression(t *testing.T) {
	w, mob := newMobactWanderTestWorld(t, SECT_WATER_SWIM, "AGGRESSIVE")
	player := NewPlayer(1, "Visible", 1001)
	for i := range player.RaceHates {
		player.RaceHates[i] = -1
	}
	w.mu.Lock()
	w.players[player.Name] = player
	w.mu.Unlock()
	ce := &testCombatEngine{}
	w.combatEngine = ce

	draws := pinNorthWanderDraws(t)
	w.MobileActivity()
	if got := mob.GetRoom(); got != 1001 {
		t.Fatalf("blocked mob room = %d, want 1001", got)
	}
	if len(ce.starts) != 0 {
		t.Fatalf("blocked sector still dispatched aggression: %+v", ce.starts)
	}
	assertMobactDraws(t, draws, []mobactDrawRange{{0, 18}})
}

func TestMobileActivityWanderActsMatchPerformMove(t *testing.T) {
	tests := []struct {
		name       string
		sneak      bool
		affect     int
		sector     int
		wantOrigin string
		wantTarget string
	}{
		{name: "visible movement", sector: SECT_FIELD, wantOrigin: "A wanderer leaves north.\r\n", wantTarget: "A wanderer arrives from the south.\r\n"},
		{name: "flying arrival", affect: affFly, sector: SECT_FLYING, wantOrigin: "A wanderer leaves north.\r\n", wantTarget: "A wanderer flies in from the south.\r\n"},
		{name: "sneaking movement", sneak: true, sector: SECT_FIELD},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, mob := newMobactWanderTestWorld(t, tt.sector)
			if tt.affect != 0 {
				mob.SetAffected(tt.affect)
			}
			if tt.sneak {
				mob.SetAffected(affSneak)
			}
			origin := NewPlayer(1, "Origin", 1001)
			target := NewPlayer(2, "Target", 1002)
			if err := w.AddPlayer(origin); err != nil {
				t.Fatalf("AddPlayer origin: %v", err)
			}
			if err := w.AddPlayer(target); err != nil {
				t.Fatalf("AddPlayer target: %v", err)
			}
			out := captureOutput(w)
			pinNorthWanderDraws(t)

			w.MobileActivity()
			if got := outputOf(out, origin.Name); got != tt.wantOrigin {
				t.Fatalf("origin movement output = %q, want %q", got, tt.wantOrigin)
			}
			if got := outputOf(out, target.Name); got != tt.wantTarget {
				t.Fatalf("target movement output = %q, want %q", got, tt.wantTarget)
			}
			if got := mob.GetRoom(); got != 1002 {
				t.Fatalf("mob room = %d, want 1002", got)
			}
		})
	}
}

func newMobactDrawTestMob(flags ...string) *MobInstance {
	return NewMob(&parser.Mob{
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

func TestMobileActivityWoundedCharmedMobKeepsCActivity(t *testing.T) {
	w := newMobactDrawTestWorld()
	mob := newMobactDrawTestMob("sentinel", "charmed")
	mob.SetHealth(1)

	draws := captureMobactDraws(t)
	w.mobileActivityForMob(mob)

	assertMobactDraws(t, draws, []mobactDrawRange{{0, 18}, {0, 15}})
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

type extractionPulseRecorder struct {
	world *World
	mob   *MobInstance
	calls []string
}

func (r *extractionPulseRecorder) ForgetFailures() {}

func (r *extractionPulseRecorder) RunScript(_ *ScriptContext, _ string, trigger string) (bool, error) {
	r.calls = append(r.calls, trigger)
	if trigger == "onpulse_all" {
		r.world.ExtractMob(r.mob)
	}
	return false, nil
}

func TestMobileActivityCheckDeadStopsAfterPendingExtchar(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Pulse room"}},
		Mobs: []parser.Mob{{
			VNum: 9402, Keywords: "pulse guard", ShortDesc: "a pulse guard",
			ScriptName: "pulse.lua", LuaFunctions: 64 | 128, // MS_ONPULSE_ALL | MS_ONPULSE_PC.
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	w.StopAITicker()
	t.Cleanup(w.StopAITicker)
	mob, err := w.SpawnMobQuiet(9402, 1001)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.AddPlayer(NewPlayer(1, "PulseObserver", 1001)); err != nil {
		t.Fatal(err)
	}
	recorder := &extractionPulseRecorder{world: w, mob: mob}
	previous := ScriptEngine
	ScriptEngine = recorder
	t.Cleanup(func() { ScriptEngine = previous })

	w.mobileActivityForMob(mob)
	if len(recorder.calls) != 1 || recorder.calls[0] != "onpulse_all" {
		t.Fatalf("pulse script calls = %v, want only onpulse_all after check_dead", recorder.calls)
	}
	if _, ok := w.GetMobByID(mob.GetID()); !ok {
		t.Fatal("check_dead extraction removed mob before the pending pass")
	}
	w.ExtractPendingChars()
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

func TestMobileActivityScavengerUsesCanGetVisibility(t *testing.T) {
	for _, tc := range []struct {
		name        string
		dark        bool
		invisible   bool
		detect      bool
		infravision bool
		wantPickup  bool
	}{
		{name: "invisible", invisible: true},
		{name: "detect invisible", invisible: true, detect: true, wantPickup: true},
		{name: "dark", dark: true},
		{name: "infravision", dark: true, infravision: true, wantPickup: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := newMobactDrawTestWorld()
			if tc.dark {
				setRoomFlagBit(w.GetRoomInWorld(1001), 0)
			}
			mob := newMobactDrawTestMob("scavenger", "sentinel")
			mob.Str, mob.Dex = 18, 18
			mob.CopyBaseAttributes()
			if tc.detect {
				mob.SetAffected(affDetectInvisible)
			}
			if tc.infravision {
				mob.SetAffected(affInfravision)
			}
			obj := NewObjectInstance(&parser.Obj{
				VNum: 9402, Keywords: "probe gem", ShortDesc: "a probe gem",
				WearFlags: [4]int{1}, Weight: 1, Cost: 10,
			}, 1001)
			if tc.invisible {
				obj.SetExtraFlag(0, extraFlagInvisible)
			}
			w.roomItems[1001] = []*ObjectInstance{obj}
			previous := mobactNumber
			mobactNumber = func(from, to int) int { return 0 }
			t.Cleanup(func() { mobactNumber = previous })

			w.mobileActivityForMob(mob)

			if got := len(mob.Inventory); (got == 1) != tc.wantPickup {
				t.Fatalf("carried count=%d want pickup=%t", got, tc.wantPickup)
			}
			if got := len(w.roomItems[1001]); (got == 0) != tc.wantPickup {
				t.Fatalf("room item count=%d want pickup=%t", got, tc.wantPickup)
			}
		})
	}
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

func (t *testCombatEngine) IsFighting(body combat.Combatant) bool {
	for _, p := range t.starts {
		if p.attacker == body.GetName() || p.defender == body.GetName() {
			return true
		}
	}
	return false
}

func (t *testCombatEngine) GetCombatTarget(body combat.Combatant) (combat.Combatant, bool) {
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
	p.RaceHates[0] = mob.Proto().Race

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
	p.RaceHates[0] = mob.Proto().Race

	old := MobSpecAssign[mob.Proto().VNum]
	MobSpecAssign[mob.Proto().VNum] = "shop_keeper"
	defer func() {
		if old == "" {
			delete(MobSpecAssign, mob.Proto().VNum)
		} else {
			MobSpecAssign[mob.Proto().VNum] = old
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
	p.RaceHates[0] = mob.Proto().Race
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
	p.RaceHates[0] = mob.Proto().Race
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

func TestAggr24MobileTargetVisibilityGateUsesGlobalPulse(t *testing.T) {
	for _, tt := range []struct {
		name          string
		targetAffects int
		wantStarts    int
	}{
		{name: "invisible NPC target is skipped", targetAffects: affInvisible},
		{name: "visible NPC target is selected", wantStarts: 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w, ce := newMobactTestWorld(t)
			attacker := newMobactTestMob(w, 9310, 1, 0, "aggr24", "aggressive")
			attacker.SetLevel(33)
			target := newMobactTestMob(w, 9311, 1, 0)
			target.SetLevel(10)
			if tt.targetAffects != 0 {
				target.SetAffected(tt.targetAffects)
			}

			draws := pinNorthWanderDraws(t)
			w.MobileActivity()

			if len(ce.starts) != tt.wantStarts {
				t.Fatalf("AGGR24 combat starts = %d, want %d; starts=%+v", len(ce.starts), tt.wantStarts, ce.starts)
			}
			if tt.wantStarts == 1 && (ce.starts[0].attacker != attacker.GetName() || ce.starts[0].defender != target.GetName()) {
				t.Fatalf("unexpected combat pair: %+v", ce.starts[0])
			}
			wantDraws := []mobactDrawRange{{0, 18}, {0, 15}, {0, 18}, {0, 15}}
			if !reflect.DeepEqual(*draws, wantDraws) {
				t.Fatalf("mobile pulse draws = %v, want %v", *draws, wantDraws)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// DP-1435: mobile_activity room acts
//
// Every case below drives the real MobileActivity pulse and reads the room
// bytes through the message sink, so a reverted fix fails on the exact C
// string (R1/R4/R5h).
// ---------------------------------------------------------------------------

// newMobactSpeakingPlayer registers a player that can receive act() output:
// Player.SendMessage needs the world reference AddPlayer installs.
func newMobactSpeakingPlayer(t *testing.T, w *World, name string) *Player {
	t.Helper()
	p := NewPlayer(1, name, 1001)
	if err := w.AddPlayer(p); err != nil {
		t.Fatalf("AddPlayer %s: %v", name, err)
	}
	return p
}

// TestMobileActivityMemoryWarningActs pins C's two memory-warning room acts
// (src/mobact.c:273-278). The speak arm needs can_speak(ch); the growl arm is
// its silent counterpart; both sit inside one ROOM_PEACEFUL gate, and the
// attack that follows is unconditional.
func TestMobileActivityMemoryWarningActs(t *testing.T) {
	tests := []struct {
		name       string
		race       int
		peaceful   bool
		wantOutput string
		wantStarts int
	}{
		{
			name:       "speaking mobile exclaims",
			race:       1, // RACE_NPCHUMAN — in intelligent_races[] (src/constants.c:353)
			wantOutput: "'Hey!  You're the fiend that attacked me!!!', exclaims a memory mob.\r\n",
			wantStarts: 1,
		},
		{
			name:       "silent mobile growls",
			race:       12, // not in intelligent_races[]
			wantOutput: "You hear a low growl in the back of a memory mob's throat.\r\n",
			wantStarts: 1,
		},
		{
			name:       "peaceful room suppresses both warning acts but still attacks",
			race:       1,
			peaceful:   true,
			wantStarts: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, ce := newMobactTestWorld(t)
			if tt.peaceful {
				w.rooms[1001].Flags = append(w.rooms[1001].Flags, "peaceful")
			}
			mob := newMobactTestMob(w, 9320, tt.race, 0, "memory", "sentinel")
			mob.Proto().ShortDesc = "a memory mob"
			remembered := newMobactSpeakingPlayer(t, w, "Remembered")
			observer := newMobactSpeakingPlayer(t, w, "Observer")
			mob.Memory = []string{remembered.Name}

			out := captureOutput(w)
			pinNorthWanderDraws(t)
			w.MobileActivity()

			if got := outputOf(out, observer.Name); got != tt.wantOutput {
				t.Fatalf("observer output = %q, want %q", got, tt.wantOutput)
			}
			if len(ce.starts) != tt.wantStarts {
				t.Fatalf("memory combat starts = %d, want %d; starts=%+v", len(ce.starts), tt.wantStarts, ce.starts)
			}
		})
	}
}

// TestMobileActivityHelperJumpsToAid pins C's helper room act and its CAN_SEE
// gate (src/mobact.c:292-301). The helper joins the first mobile it can see
// fighting a player; an invisible player is not a valid cause.
func TestMobileActivityHelperJumpsToAid(t *testing.T) {
	tests := []struct {
		name        string
		playerInvis bool
		wantOutput  string
		wantStarts  int
	}{
		{
			name:       "visible player is defended",
			wantOutput: "A helper mob jumps to the aid of a foe mob!\r\n",
			wantStarts: 1,
		},
		{
			name:        "invisible player is not defended",
			playerInvis: true,
			wantStarts:  0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, ce := newMobactTestWorld(t)
			helper := newMobactTestMob(w, 9330, 1, 0, "helper", "sentinel")
			helper.Proto().ShortDesc = "a helper mob"
			foe := newMobactTestMob(w, 9331, 1, 0, "sentinel")
			foe.Proto().ShortDesc = "a foe mob"
			player := newMobactSpeakingPlayer(t, w, "Bystander")
			if tt.playerInvis {
				player.Affects |= 1 << affInvisible
			}
			observer := newMobactSpeakingPlayer(t, w, "Observer")
			foe.SetFightingBody(player)

			out := captureOutput(w)
			pinNorthWanderDraws(t)
			w.MobileActivity()

			if got := outputOf(out, observer.Name); got != tt.wantOutput {
				t.Fatalf("observer output = %q, want %q", got, tt.wantOutput)
			}
			if len(ce.starts) != tt.wantStarts {
				t.Fatalf("helper combat starts = %d, want %d; starts=%+v", len(ce.starts), tt.wantStarts, ce.starts)
			}
			if tt.wantStarts == 1 && (ce.starts[0].attacker != helper.GetName() || ce.starts[0].defender != player.Name) {
				t.Fatalf("unexpected combat pair: %+v", ce.starts[0])
			}
		})
	}
}

// TestMobileActivityAggr24WeakerMobileGrin pins the AGGR24 + AGGRESSIVE
// full-health grin (src/mobact.c:332-334) at C's point, immediately before
// hit(). The observer carries NOHASSLE so the ordinary aggressive and AGGR24
// player branches leave it alone.
func TestMobileActivityAggr24WeakerMobileGrin(t *testing.T) {
	w, ce := newMobactTestWorld(t)
	attacker := newMobactTestMob(w, 9340, 1, 0, "aggr24", "aggressive", "sentinel")
	attacker.Proto().ShortDesc = "an aggr24 mob"
	attacker.SetLevel(33)
	target := newMobactTestMob(w, 9341, 1, 0, "sentinel")
	target.Proto().ShortDesc = "a weaker mob"
	target.SetLevel(10)
	observer := newMobactSpeakingPlayer(t, w, "Observer")
	observer.SetPlrFlag(PrfNohassle, true)

	out := captureOutput(w)
	pinNorthWanderDraws(t)
	w.MobileActivity()

	want := "An aggr24 mob grins evilly.\r\n"
	if got := outputOf(out, observer.Name); got != want {
		t.Fatalf("observer output = %q, want %q", got, want)
	}
	if len(ce.starts) != 1 {
		t.Fatalf("AGGR24 weaker-mobile combat starts = %d, want 1; starts=%+v", len(ce.starts), ce.starts)
	}
	if ce.starts[0].attacker != attacker.GetName() || ce.starts[0].defender != target.GetName() {
		t.Fatalf("unexpected combat pair: %+v", ce.starts[0])
	}
}

// TestMobileActivityAggr24PlayerRoomActs pins the three AGGR24 player-branch
// room acts in C's order (src/mobact.c:312-320): the grin, then the two-line
// speak pair gated on can_speak.
func TestMobileActivityAggr24PlayerRoomActs(t *testing.T) {
	w, ce := newMobactTestWorld(t)
	mob := newMobactTestMob(w, 9350, 1, 0, "aggr24", "sentinel")
	mob.Proto().ShortDesc = "a dark pawn"
	victim := newMobactSpeakingPlayer(t, w, "Knight")
	victim.Level = 24
	observer := newMobactSpeakingPlayer(t, w, "Observer")

	out := captureOutput(w)
	pinNorthWanderDraws(t)
	w.MobileActivity()

	want := "A dark pawn grins evilly.\r\n" +
		"'Another dark pawn trying to be a knight, eh?', asks a dark pawn.\r\n" +
		"A dark pawn says, 'I can fix that.'\n\r\r\n"
	if got := outputOf(out, observer.Name); got != want {
		t.Fatalf("observer output = %q, want %q", got, want)
	}
	if len(ce.starts) != 1 {
		t.Fatalf("AGGR24 player combat starts = %d, want 1; starts=%+v", len(ce.starts), ce.starts)
	}
	if ce.starts[0].attacker != mob.GetName() || ce.starts[0].defender != victim.Name {
		t.Fatalf("unexpected combat pair: %+v", ce.starts[0])
	}
}

// TestMobileActivityCharmedHelperAssistsMaster pins the charmed-master arm
// (src/mobact.c:287-291 → do_assist, src/act.offensive.c:92-95). The master
// hears "$N assists you!" and the room sees "$n assists $N."; both of C's
// TO_CHAR lines to the assisting mobile are dropped, because a mobile has no
// descriptor.
func TestMobileActivityCharmedHelperAssistsMaster(t *testing.T) {
	w, ce := newMobactTestWorld(t)
	helper := newMobactTestMob(w, 9360, 1, 0, "helper", "sentinel")
	helper.Proto().ShortDesc = "a helper mob"
	helper.SetAffected(affCharm)
	master := newMobactSpeakingPlayer(t, w, "Master")
	observer := newMobactSpeakingPlayer(t, w, "Observer")
	foe := newMobactTestMob(w, 9361, 1, 0, "sentinel")
	foe.Proto().ShortDesc = "a foe mob"
	helper.SetFollowing(master.Name)
	master.SetFightingBody(foe)
	foe.SetFightingBody(master)

	out := captureOutput(w)
	pinNorthWanderDraws(t)
	w.MobileActivity()

	if got, want := outputOf(out, master.Name), "A helper mob assists you!\r\n"; got != want {
		t.Fatalf("master output = %q, want %q", got, want)
	}
	if got, want := outputOf(out, observer.Name), "A helper mob assists Master.\r\n"; got != want {
		t.Fatalf("observer output = %q, want %q", got, want)
	}
	if len(ce.starts) != 1 {
		t.Fatalf("charmed assist combat starts = %d, want 1; starts=%+v", len(ce.starts), ce.starts)
	}
	if ce.starts[0].attacker != helper.GetName() || ce.starts[0].defender != foe.GetName() {
		t.Fatalf("unexpected combat pair: %+v", ce.starts[0])
	}
}

// TestMobAssistGuards pins do_assist's mobile-side entry gates: a mobile that
// is already fighting, mounted, or looking at a helpee with no attacker in
// sight does nothing (src/act.offensive.c:58-89).
func TestMobAssistGuards(t *testing.T) {
	t.Run("already fighting", func(t *testing.T) {
		w, ce := newMobactTestWorld(t)
		helper := newMobactTestMob(w, 9370, 1, 0, "sentinel")
		master := newMobactSpeakingPlayer(t, w, "Master")
		foe := newMobactTestMob(w, 9371, 1, 0, "sentinel")
		helper.SetFightingBody(foe)
		master.SetFightingBody(foe)
		foe.SetFightingBody(master)

		w.mobAssist(helper, master.Name)
		if len(ce.starts) != 0 {
			t.Fatalf("fighting helper still assisted: %+v", ce.starts)
		}
	})
	t.Run("mounted", func(t *testing.T) {
		w, ce := newMobactTestWorld(t)
		helper := newMobactTestMob(w, 9372, 1, 0, "sentinel")
		helper.SetAffected(affMount)
		master := newMobactSpeakingPlayer(t, w, "Master")
		foe := newMobactTestMob(w, 9373, 1, 0, "sentinel")
		master.SetFightingBody(foe)
		foe.SetFightingBody(master)

		w.mobAssist(helper, master.Name)
		if len(ce.starts) != 0 {
			t.Fatalf("mounted helper still assisted: %+v", ce.starts)
		}
	})
	t.Run("master not visible", func(t *testing.T) {
		// C resolves the helpee with get_char_room_vis(ch, GET_NAME(master))
		// (src/act.offensive.c:72): an invisible master is NOPERSON to a
		// mobile without detect invisibility, so nobody is assisted.
		w, ce := newMobactTestWorld(t)
		helper := newMobactTestMob(w, 9376, 1, 0, "sentinel")
		master := newMobactSpeakingPlayer(t, w, "Master")
		master.SetAffect(affInvisible, true)
		foe := newMobactTestMob(w, 9377, 1, 0, "sentinel")
		master.SetFightingBody(foe)
		foe.SetFightingBody(master)

		w.mobAssist(helper, master.Name)
		if len(ce.starts) != 0 {
			t.Fatalf("helper assisted a master it cannot see: %+v", ce.starts)
		}
	})
	t.Run("nobody fighting the helpee", func(t *testing.T) {
		w, ce := newMobactTestWorld(t)
		helper := newMobactTestMob(w, 9374, 1, 0, "sentinel")
		master := newMobactSpeakingPlayer(t, w, "Master")

		w.mobAssist(helper, master.Name)
		if len(ce.starts) != 0 {
			t.Fatalf("helper assisted with no opponent: %+v", ce.starts)
		}
	})
}
