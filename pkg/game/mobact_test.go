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
