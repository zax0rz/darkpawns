package game

// Regression tests for Kimi Batch B (BRIEF-2026-07-11-kimi-batch-b.md):
//   - DP-1030: PK bookkeeping on live player death path
//   - DP-1027: Death traps kill mortal players
//   - DP-1028: Mob spell affects expire in AffectUpdate

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
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
// DP-1027: Death traps extract mortal players (corpse-less, penalty-free)
// ---------------------------------------------------------------------------

func TestMovePlayer_DeathTrapExtractsMortal(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Safe Room", Zone: 1, Exits: map[string]parser.Exit{"north": {ToRoom: 1002}}},
			// ROOM_DEATH is bit 1 → bitmask value 2.
			{VNum: 1002, Name: "Death Trap", Zone: 1, Flags: []string{"2"}},
			// Respawn destination for mortals.
			{VNum: MortalStartRoom, Name: "Temple", Zone: 1},
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
	p.SetExp(50000)
	if err := w.AddPlayer(p); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	if _, err := w.MovePlayer(p, "north"); err != nil {
		t.Fatalf("MovePlayer failed: %v", err)
	}

	if p.GetRoom() != MortalStartRoom {
		t.Errorf("expected player extracted to MortalStartRoom (%d), got %d", MortalStartRoom, p.GetRoom())
	}
	if p.GetHP() <= 0 {
		t.Errorf("expected player HP > 0 after DT respawn, got %d", p.GetHP())
	}
	if p.GetHP() != p.GetMaxHP() {
		t.Errorf("expected player at full HP after DT respawn, got %d/%d", p.GetHP(), p.GetMaxHP())
	}
	if p.GetPosition() != combat.PosStanding {
		t.Errorf("expected player standing after DT respawn, got position %d", p.GetPosition())
	}
	if p.GetExp() != 50000 {
		t.Errorf("expected penalty-free DT (exp unchanged), got %d", p.GetExp())
	}
	if items := w.GetItemsInRoom(1002); len(items) != 0 {
		t.Errorf("expected no corpse in DT room, got %d item(s)", len(items))
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
	if p.GetRoom() != 1002 {
		t.Errorf("expected immortal to remain in DT room, got room %d", p.GetRoom())
	}
}

func TestMovePlayer_DeathTrapKeepsInventory(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Safe Room", Zone: 1, Exits: map[string]parser.Exit{"north": {ToRoom: 1002}}},
			{VNum: 1002, Name: "Death Trap", Zone: 1, Flags: []string{"2"}},
			{VNum: MortalStartRoom, Name: "Temple", Zone: 1},
		},
		Objs: []parser.Obj{{
			VNum:      1,
			Keywords:  "keepsake trinket",
			ShortDesc: "a shiny keepsake",
			LongDesc:  "A shiny keepsake lies here.",
		}},
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

	keepsake := NewObjectInstance(&parsed.Objs[0], -1)
	if err := w.MoveObjectToPlayerInventory(keepsake, p); err != nil {
		t.Fatalf("MoveObjectToPlayerInventory failed: %v", err)
	}

	if _, err := w.MovePlayer(p, "north"); err != nil {
		t.Fatalf("MovePlayer failed: %v", err)
	}

	if p.GetRoom() != MortalStartRoom {
		t.Errorf("expected player extracted to MortalStartRoom (%d), got %d", MortalStartRoom, p.GetRoom())
	}
	found, ok := p.Inventory.FindItem("keepsake")
	if !ok || found == nil {
		t.Error("expected player to keep keepsake in inventory after DT respawn")
	}
	if items := w.GetItemsInRoom(1002); len(items) != 0 {
		t.Errorf("expected no dropped corpse/items in DT room, got %d item(s)", len(items))
	}
}

func TestMovePlayer_DeathTrapMountDismounts(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Safe Room", Zone: 1, Exits: map[string]parser.Exit{"north": {ToRoom: 1002}}},
			{VNum: 1002, Name: "Death Trap", Zone: 1, Flags: []string{"2"}},
			{VNum: MortalStartRoom, Name: "Temple", Zone: 1},
		},
		Mobs: []parser.Mob{{
			VNum:      1,
			Keywords:  "pony mount",
			ShortDesc: "a gentle pony",
			LongDesc:  "A gentle pony stands here.",
			Level:     1,
		}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	p := NewPlayer(1, "Rider", 1001)
	p.SetLevel(10)
	p.SetMove(100)
	if err := w.AddPlayer(p); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	mount, err := w.SpawnMob(1, 1001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}

	// Mount the player on the mob.
	mount.SetMountRider(p.GetName())
	p.MountName = mount.GetName()
	p.SetAffect(affMounted, true)
	p.SetFollowing(mount.GetShortDesc())

	if _, err := w.MovePlayer(p, "north"); err != nil {
		t.Fatalf("MovePlayer failed: %v", err)
	}

	if p.IsMounted() {
		t.Error("expected player to be dismounted after DT respawn")
	}
	if p.GetRoom() != MortalStartRoom {
		t.Errorf("expected player extracted to MortalStartRoom (%d), got %d", MortalStartRoom, p.GetRoom())
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
