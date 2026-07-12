package game

// Regression tests for Kimi Batch B (BRIEF-2026-07-11-kimi-batch-b.md):
//   - DP-1030: PK bookkeeping on live player death path
//   - DP-1027: Death traps kill mortal players
//   - DP-1028: Mob spell affects expire in AffectUpdate

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// ---------------------------------------------------------------------------
// DP-1030: PK bookkeeping in handlePlayerDeath
// ---------------------------------------------------------------------------

func TestHandlePlayerDeath_PKBookkeeping(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Combat Arena", Zone: 1}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	killer := NewPlayer(1, "Killer", 1001)
	killer.SetLevel(10)
	if err := w.AddPlayer(killer); err != nil {
		t.Fatalf("AddPlayer killer failed: %v", err)
	}

	victim := NewPlayer(2, "Victim", 1001)
	victim.SetLevel(10)
	victim.SetHealth(-1)
	if err := w.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer victim failed: %v", err)
	}

	w.handlePlayerDeath(victim, true, TypeHit, killer.Name)

	if killer.PKs != 1 {
		t.Errorf("expected killer PKs == 1, got %d", killer.PKs)
	}
	if victim.Deaths != 1 {
		t.Errorf("expected victim Deaths == 1, got %d", victim.Deaths)
	}
	if victim.LastDeath == 0 {
		t.Error("expected victim LastDeath to be set")
	}
	if killer.GetFlags()&(1<<uint(PlrOutlaw)) == 0 {
		t.Error("expected killer to be flagged PLR_OUTLAW")
	}
}

func TestHandlePlayerDeath_VictimAlreadyOutlawNoDoubleFlag(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Combat Arena", Zone: 1}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	killer := NewPlayer(1, "Killer", 1001)
	killer.SetLevel(10)
	if err := w.AddPlayer(killer); err != nil {
		t.Fatalf("AddPlayer killer failed: %v", err)
	}

	victim := NewPlayer(2, "Victim", 1001)
	victim.SetLevel(10)
	victim.SetPlrFlag(PlrOutlaw, true)
	victim.SetHealth(-1)
	if err := w.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer victim failed: %v", err)
	}

	w.handlePlayerDeath(victim, true, TypeHit, killer.Name)

	if killer.PKs != 1 {
		t.Errorf("expected killer PKs == 1, got %d", killer.PKs)
	}
	// Killer should NOT be flagged outlaw because the victim was already an outlaw.
	if killer.GetFlags()&(1<<uint(PlrOutlaw)) != 0 {
		t.Error("expected killer NOT to be flagged PLR_OUTLAW when victim was already outlaw")
	}
}

// ---------------------------------------------------------------------------
// DP-1027: Death traps kill mortal players
// ---------------------------------------------------------------------------

func TestMovePlayer_DeathTrapKillsMortal(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Safe Room", Zone: 1, Exits: map[string]parser.Exit{"north": {ToRoom: 1002}}},
			// ROOM_DEATH is bit 1 → bitmask value 2.
			{VNum: 1002, Name: "Death Trap", Zone: 1, Flags: []string{"2"}},
		},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	p := NewPlayer(1, "Victim", 1001)
	p.SetLevel(10)
	p.SetMove(100)
	if err := w.AddPlayer(p); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	if _, err := w.MovePlayer(p, "north"); err != nil {
		t.Fatalf("MovePlayer failed: %v", err)
	}

	if p.GetHP() > 0 {
		t.Errorf("expected player HP <= 0 after death trap, got %d", p.GetHP())
	}
}

func TestMovePlayer_DeathTrapImmortalSurvives(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Safe Room", Zone: 1, Exits: map[string]parser.Exit{"north": {ToRoom: 1002}}},
			{VNum: 1002, Name: "Death Trap", Zone: 1, Flags: []string{"2"}},
		},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	p := NewPlayer(1, "Immortal", 1001)
	p.SetLevel(LVL_IMMORT)
	p.SetMove(100)
	if err := w.AddPlayer(p); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	if _, err := w.MovePlayer(p, "north"); err != nil {
		t.Fatalf("MovePlayer failed: %v", err)
	}

	if p.GetHP() <= 0 {
		t.Errorf("expected immortal HP > 0 in death trap, got %d", p.GetHP())
	}
}

// ---------------------------------------------------------------------------
// DP-1028: Mob spell affects expire in AffectUpdate
// ---------------------------------------------------------------------------

func TestAffectUpdate_MobAffectExpires(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Affect Room", Zone: 1}},
		Mobs: []parser.Mob{{
			VNum:      1,
			Keywords:  "rat",
			ShortDesc: "a rat",
			LongDesc:  "A rat is here.",
			Level:     1,
		}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	mob, err := w.SpawnMob(1, 1001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}

	aff := &engine.Affect{
		SpellID:  1,
		Duration: 0,
		Flags:    engine.AFFPoison,
	}
	mob.AddAffect(aff)

	if _, ok := mob.CustomData["affect_1"]; !ok {
		t.Fatal("expected affect_1 key in mob CustomData before update")
	}
	if !mob.HasAffect(affPoison) {
		t.Fatal("expected AFF_POISON set before update")
	}

	w.AffectUpdate()

	if _, ok := mob.CustomData["affect_1"]; ok {
		t.Error("expected affect_1 key removed from mob CustomData after expiry")
	}
	if mob.HasAffect(affPoison) {
		t.Error("expected AFF_POISON cleared after expiry")
	}
}

func TestAffectUpdate_MobPermanentAffectPersists(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Affect Room", Zone: 1}},
		Mobs: []parser.Mob{{
			VNum:      1,
			Keywords:  "rat",
			ShortDesc: "a rat",
			LongDesc:  "A rat is here.",
			Level:     1,
		}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	mob, err := w.SpawnMob(1, 1001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}

	aff := &engine.Affect{
		SpellID:  1,
		Duration: -1,
		Flags:    engine.AFFPoison,
	}
	mob.AddAffect(aff)

	w.AffectUpdate()

	if _, ok := mob.CustomData["affect_1"]; !ok {
		t.Error("expected permanent affect_1 key to persist")
	}
	if !mob.HasAffect(affPoison) {
		t.Error("expected AFF_POISON to persist on permanent affect")
	}
}

func TestAffectUpdate_MobAffectDecrements(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Affect Room", Zone: 1}},
		Mobs: []parser.Mob{{
			VNum:      1,
			Keywords:  "rat",
			ShortDesc: "a rat",
			LongDesc:  "A rat is here.",
			Level:     1,
		}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	mob, err := w.SpawnMob(1, 1001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}

	aff := &engine.Affect{
		SpellID:  1,
		Duration: 3,
		Flags:    engine.AFFPoison,
	}
	mob.AddAffect(aff)

	w.AffectUpdate()

	stored, ok := mob.CustomData["affect_1"].(*engine.Affect)
	if !ok {
		t.Fatal("expected affect_1 to remain after decrement")
	}
	if stored.Duration != 2 {
		t.Errorf("expected duration decremented to 2, got %d", stored.Duration)
	}
	if !mob.HasAffect(affPoison) {
		t.Error("expected AFF_POISON to persist while affect is active")
	}
}
