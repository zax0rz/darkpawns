package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

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
