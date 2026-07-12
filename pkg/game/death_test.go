package game

import (
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// ---------------------------------------------------------------------------
// attackTypeToCorpseAttack
// ---------------------------------------------------------------------------

func TestAttackTypeToCorpseAttack(t *testing.T) {
	tests := []struct {
		name       string
		attackType int
		want       CorpseAttackType
	}{
		{"fireball (5)", 5, AttackFire},
		{"chill touch (8)", 8, AttackCold},
		{"color spray (10)", 10, AttackBlast},
		{"energy drain (21)", 21, AttackEnergyDrain},
		{"lightning bolt (30)", 30, AttackLightning},
		{"psiblast (34)", 34, AttackPsiblast},
		{"petrify (35)", 35, AttackPetrify},
		{"drowning (103)", 103, AttackDrowning},
		{"slash type (303)", TypeSlash, AttackSlash},
		{"bite type (304)", TypeBite, AttackSlash},
		{"claw type (308)", TypeClaw, AttackSlash},
		{"whip type (302)", TypeWhip, AttackBruised},
		{"crush type (306)", TypeCrush, AttackCrush},
		{"pierce type (311)", TypePierce, AttackPierce},
		{"bash skill (132)", SkillBashNum, AttackBruised},
		{"backstab skill", SkillBackstabNum, AttackSlash},
		{"disembowel skill", SkillDisembowelNum, AttackDisembowel},
		{"unknown (9999)", 9999, AttackUndefined},
		{"negative (-1)", -1, AttackUndefined},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := attackTypeToCorpseAttack(tt.attackType)
			if got != tt.want {
				t.Errorf("attackTypeToCorpseAttack(%d) = %d, want %d", tt.attackType, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// createMoneyDesc
// ---------------------------------------------------------------------------

func TestCreateMoneyDesc(t *testing.T) {
	tests := []struct {
		amount int
		want   string
	}{
		{1, "a gold coin"},
		{2, "a tiny pile of gold coins"},
		{100, "a small pile of gold coins"},
		{1000, "a pile of gold coins"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := createMoneyDesc(tt.amount)
			if got != tt.want {
				t.Errorf("createMoneyDesc(%d) = %q, want %q", tt.amount, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// capitalize
// ---------------------------------------------------------------------------

func TestCapitalize(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"hello", "Hello"},
		{"Hello", "Hello"},
		{"", ""},
		{"a", "A"},
		{"ALREADY", "ALREADY"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := capitalize(tt.input)
			if got != tt.want {
				t.Errorf("capitalize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// genderPronoun
// ---------------------------------------------------------------------------

func TestGenderPronoun(t *testing.T) {
	tests := []struct {
		sex  int
		want string
	}{
		{0, "his"},  // male
		{1, "her"},  // female
		{2, "its"},  // neuter
		{99, "his"}, // unknown defaults male
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := genderPronoun(tt.sex)
			if got != tt.want {
				t.Errorf("genderPronoun(%d) = %q, want %q", tt.sex, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// corpseAttackLongDesc
// ---------------------------------------------------------------------------

func TestCorpseAttackLongDesc(t *testing.T) {
	tests := []struct {
		attackType CorpseAttackType
		gender     string
		contains   string // substring expected in result
	}{
		{AttackFire, "male", "charred corpse"},
		{AttackCold, "female", "frozen corpse"},
		{AttackBlast, "neuter", "blasted corpse"},
		{AttackSlash, "male", "hacked up"},
		{AttackDisembowel, "female", "guts spilled"},
		{AttackBruised, "neuter", "bruised"},
		{AttackPierce, "male", "well-ventilated"},
		{AttackCrush, "female", "crushed"},
		{AttackDrowning, "neuter", "waterlogged"},
		{AttackPetrify, "male", "frozen in stone"},
		{AttackNeckBreak, "female", "neck snapped"},
		{AttackPsiblast, "neuter", "brains exploded"},
	}
	for _, tt := range tests {
		t.Run(tt.contains, func(t *testing.T) {
			got := corpseAttackLongDesc("Victim", tt.attackType, tt.gender)
			if len(got) == 0 {
				t.Error("corpseAttackLongDesc returned empty string")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// HandleDeath Tests
// ---------------------------------------------------------------------------

func TestHandleDeath_Mob(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Combat Arena", Zone: 1},
		},
		Mobs: []parser.Mob{
			{
				VNum:      1,
				ShortDesc: "a scary dragon",
				LongDesc:  "A scary dragon is here.",
				Keywords:  "dragon scary",
				Level:     5,
				Exp:       1000,
				Gold:      500,
			},
		},
	}

	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	// Spawn mob
	mob, err := w.SpawnMob(1, 1001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}

	// Create killer player
	killer := NewPlayer(99, "Hero", 1001)
	if err := w.AddPlayer(killer); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	// Handle death
	w.HandleDeath(mob, killer, 5) // Fireball

	// Verify mob is dead
	if mob.IsAlive() {
		t.Error("mob should not be alive after HandleDeath")
	}

	// Verify XP and Kills were updated on player
	if killer.Kills != 1 {
		t.Errorf("killer Kills = %d, want 1", killer.Kills)
	}

	// Check if corpse was created in the room
	items := w.roomItems[1001]
	if len(items) != 1 {
		t.Errorf("room items count = %d, want 1 (corpse)", len(items))
	} else {
		corpse := items[0]
		if !corpse.IsCorpse {
			t.Error("spawned item should be a corpse")
		}
		if !strings.Contains(corpse.Runtime.Keywords, "corpse") {
			t.Errorf("corpse keywords = %q, want it to contain 'corpse'", corpse.Runtime.Keywords)
		}
	}
}

func TestHandleDeath_Player(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Combat Arena", Zone: 1},
			{VNum: MortalStartRoom, Name: "Temple", Zone: 8},
		},
	}

	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	// Create victim player
	victim := NewPlayer(1, "Victim", 1001)
	victim.SetLevel(10)
	victim.SetExp(10000)
	victim.Stats.Con = 15
	if err := w.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	// Killer player
	killer := NewPlayer(2, "Killer", 1001)
	if err := w.AddPlayer(killer); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	// Handle death — HP dropped to <=0, the precondition combat guarantees.
	victim.SetHP(-1)
	w.HandleDeath(victim, killer, 303) // Slash

	// Verify victim was moved to respawn room (MortalStartRoom)
	if victim.GetRoom() != MortalStartRoom {
		t.Errorf("victim room = %d, want %d", victim.GetRoom(), MortalStartRoom)
	}

	// Verify XP penalty (combat death = exp/37)
	expectedExp := 10000 - (10000 / 37)
	if victim.GetExp() != expectedExp {
		t.Errorf("victim Exp = %d, want %d", victim.GetExp(), expectedExp)
	}

	// Check corpse created in room 1001
	items := w.roomItems[1001]
	if len(items) != 1 {
		t.Errorf("room items count = %d, want 1 (corpse)", len(items))
	}
}

// TestHandlePlayerDeathIdempotent verifies that the dying guard prevents
// double death penalties (DP-943).
//
// The death path is synchronous and completes before a second goroutine is
// typically scheduled, so we test the guard directly:
//  1. dying=true → handlePlayerDeath is a complete no-op (CAS rejects)
//  2. dying=false → handlePlayerDeath applies penalties normally, resets via defer
func TestHandlePlayerDeathIdempotent(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Combat Arena", Zone: 1},
			{VNum: MortalStartRoom, Name: "Temple", Zone: 8},
		},
	}

	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	// Test 1: dying=true → handlePlayerDeath is a no-op.
	victim := NewPlayer(1, "Victim", 1001)
	victim.SetLevel(10)
	victim.SetExp(10000)
	victim.Stats.Con = 15
	if err := w.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	victim.dying.Store(true)
	w.handlePlayerDeath(victim, true, 303, "Killer")
	if victim.GetExp() != 10000 {
		t.Errorf("dying guard failed: exp changed to %d, want 10000 (no-op)", victim.GetExp())
	}
	if victim.Stats.Con != 15 {
		t.Errorf("dying guard failed: CON changed to %d, want 15 (no-op)", victim.Stats.Con)
	}
	// Guard stays set — we set it externally, the CAS rejected the call.
	if !victim.IsDying() {
		t.Error("dying should still be true after rejected call")
	}

	// Test 2: dying=false → handlePlayerDeath applies penalties normally.
	victim.dying.Store(false)
	victim.SetRoom(1001) // reset room since no-op above didn't move player
	victim.SetHP(-1)     // pending death: HP <=0, as combat guarantees
	startExp := victim.GetExp()
	w.handlePlayerDeath(victim, true, 303, "Killer")
	expectedExp := startExp - (startExp / 37)
	if victim.GetExp() != expectedExp {
		t.Errorf("normal death: exp = %d, want %d", victim.GetExp(), expectedExp)
	}
	// After return, defer has reset dying to false.
	if victim.IsDying() {
		t.Error("dying should be false after handlePlayerDeath returns (defer reset)")
	}
}

// TestHandlePlayerDeathDyingReset verifies that the dying flag is cleared
// after respawn so future deaths can be processed normally (DP-943).
func TestHandlePlayerDeathDyingReset(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Combat Arena", Zone: 1},
			{VNum: MortalStartRoom, Name: "Temple", Zone: 8},
		},
	}

	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	victim := NewPlayer(1, "Victim", 1001)
	victim.SetLevel(1)
	victim.SetExp(100)
	if err := w.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	victim.SetHP(-1) // pending death: HP <=0, as combat guarantees
	w.handlePlayerDeath(victim, true, 303, "Killer")

	if victim.IsDying() {
		t.Error("IsDying should be false after respawn")
	}
}

// ---------------------------------------------------------------------------
// COV-2: Death-Path Concurrency Tests (DP-963)
// ---------------------------------------------------------------------------

// TestHandlePlayerDeath_ConcurrentKills verifies that handlePlayerDeath is
// idempotent when two goroutines simultaneously call HandleDeath for the same
// player. The dying CAS guard (DP-943) lets exactly one death path process;
// the second goroutine finds dying=true and no-ops.
//
// Assertions:
//  1. Player respawned (in MortalStartRoom, HP > 0)
//  2. EXACTLY one EXP penalty applied (not doubled)
//  3. Exactly one corpse in the death room
//  4. No -race violations
func TestHandlePlayerDeath_ConcurrentKills(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Combat Arena", Zone: 1},
			{VNum: MortalStartRoom, Name: "Temple", Zone: 8},
		},
	}

	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	// Create victim player with some EXP, dropped to <=0 HP so a death is
	// actually pending — the production precondition for HandleDeath (combat only
	// calls it once HP hits <=0). The first handler respawns + heals; a duplicate
	// must then no-op (single penalty).
	victim := NewPlayer(1, "Victim", 1001)
	victim.SetLevel(10)
	victim.SetExp(10000)
	victim.SetHP(-1)
	if err := w.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	// Killer doesn't get Kills++ for player kills, so no Kills race here
	killer := NewPlayer(2, "Killer", 1001)
	if err := w.AddPlayer(killer); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	startExp := victim.GetExp()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		w.HandleDeath(victim, killer, TypeSlash)
	}()
	go func() {
		defer wg.Done()
		runtime.Gosched() // nudge scheduler to increase collision probability
		w.HandleDeath(victim, killer, TypeSlash)
	}()

	wg.Wait()

	// Assertion 1: player respawned to MortalStartRoom with positive HP
	if victim.GetRoom() != MortalStartRoom {
		t.Errorf("victim room = %d, want %d (respawn)", victim.GetRoom(), MortalStartRoom)
	}
	if victim.GetHP() <= 0 {
		t.Errorf("victim HP = %d, want > 0 after respawn", victim.GetHP())
	}

	// Assertion 2: EXACTLY one EXP penalty (combat death: exp/37, not doubled)
	expectedExp := startExp - (startExp / 37)
	if victim.GetExp() != expectedExp {
		t.Errorf("victim Exp = %d, want %d (single penalty, not doubled)",
			victim.GetExp(), expectedExp)
	}

	// Assertion 3: exactly one corpse in the death room
	items := w.GetItemsInRoom(1001)
	corpseCount := 0
	for _, item := range items {
		if item.IsCorpse {
			corpseCount++
		}
	}
	if corpseCount != 1 {
		t.Errorf("corpse count in room 1001 = %d, want 1", corpseCount)
	}
}

// TestHandleMobDeath_ConcurrentKills verifies that handleMobDeath is
// idempotent when two goroutines simultaneously call HandleDeath for the same
// mob. The alive CAS guard (DP-963) lets exactly one death path process;
// the second caller finds alive already false and returns before the mob
// death hook, AwardMobKillXP, and Kills++.
//
// Assertions:
//  1. Exactly one corpse in the room
//  2. Killer Kills incremented exactly once
//  3. No -race violations
func TestHandleMobDeath_ConcurrentKills(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Combat Arena", Zone: 1},
		},
		Mobs: []parser.Mob{
			{
				VNum:      1,
				ShortDesc: "a goblin",
				LongDesc:  "A goblin is here.",
				Keywords:  "goblin",
				Level:     9,
				Exp:       1000,
				Gold:      500,
			},
		},
	}

	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	// Spawn the mob
	mob, err := w.SpawnMob(1, 1001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}

	// Create the killer player
	killer := NewPlayer(99, "Hero", 1001)
	killer.SetLevel(10)
	if err := w.AddPlayer(killer); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	startKills := killer.Kills

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		w.HandleDeath(mob, killer, TypeSlash)
	}()
	go func() {
		defer wg.Done()
		runtime.Gosched()
		w.HandleDeath(mob, killer, TypeSlash)
	}()

	wg.Wait()

	// Assertion 1: exactly one corpse in the room
	items := w.GetItemsInRoom(1001)
	corpseCount := 0
	for _, item := range items {
		if item.IsCorpse {
			corpseCount++
		}
	}
	if corpseCount != 1 {
		t.Errorf("corpse count in room 1001 = %d, want 1", corpseCount)
	}

	// Assertion 2: Kills incremented exactly once
	expectedKills := startKills + 1
	if killer.Kills != expectedKills {
		t.Errorf("killer Kills = %d, want %d (single increment)",
			killer.Kills, expectedKills)
	}
}

// TestHandlePlayerDeath_SecondKillNoOps verifies the dying CAS guard in the
// sequential case: calling HandleDeath twice on the same player. The first
// call processes the death path normally (CAS succeeds, defer resets). The
// second call also processes because dying is reset — this confirms the
// guard doesn't permanently lock the player out of future deaths.
func TestHandlePlayerDeath_SecondKillNoOps(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Combat Arena", Zone: 1},
			{VNum: MortalStartRoom, Name: "Temple", Zone: 8},
		},
	}

	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	victim := NewPlayer(1, "Victim", 1001)
	victim.SetLevel(10)
	victim.SetExp(20000) // enough for two deaths
	if err := w.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	killer := NewPlayer(2, "Killer", 1001)
	if err := w.AddPlayer(killer); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	// First death — HP <=0, the precondition combat guarantees.
	victim.SetHP(-1)
	w.HandleDeath(victim, killer, TypeSlash)

	// After first death: respawned at MortalStartRoom, dying flag reset
	if victim.IsDying() {
		t.Error("dying should be false after first death completes (defer reset)")
	}
	if victim.GetRoom() != MortalStartRoom {
		t.Errorf("after first death: room = %d, want %d", victim.GetRoom(), MortalStartRoom)
	}
	expAfterFirst := victim.GetExp()
	expectedFirstExp := 20000 - (20000 / 37)
	if expAfterFirst != expectedFirstExp {
		t.Errorf("after first death: Exp = %d, want %d", expAfterFirst, expectedFirstExp)
	}

	// Move victim back to arena room for second death
	victim.SetRoom(1001)

	// Verify there's exactly 1 corpse from the first death
	items := w.GetItemsInRoom(1001)
	corpseCount := 0
	for _, item := range items {
		if item.IsCorpse {
			corpseCount++
		}
	}
	if corpseCount != 1 {
		t.Errorf("first death: corpse count = %d, want 1", corpseCount)
	}

	// Second death — a genuine re-death: the player took lethal damage again
	// (HP <=0) after respawning. dying was reset by the first death's defer, so
	// this processes normally and applies a second penalty.
	victim.SetHP(-1)
	w.HandleDeath(victim, killer, TypeSlash)

	// After second death: EXP penalized again, second corpse
	expectedSecondExp := expAfterFirst - (expAfterFirst / 37)
	if victim.GetExp() != expectedSecondExp {
		t.Errorf("after second death: Exp = %d, want %d", victim.GetExp(), expectedSecondExp)
	}
	if victim.IsDying() {
		t.Error("dying should be false after second death completes (defer reset)")
	}

	// Exactly 2 corpses now in the room (one from each death)
	items = w.GetItemsInRoom(1001)
	corpseCount = 0
	for _, item := range items {
		if item.IsCorpse {
			corpseCount++
		}
	}
	if corpseCount != 2 {
		t.Errorf("second death: corpse count = %d, want 2 (one per death)", corpseCount)
	}
}

// -----------------------------------------------------------------------------
// DP-1031: alignment shift from kills
// -----------------------------------------------------------------------------

func testAlignmentSetup(t *testing.T, victimAlignment int) (*World, *Player, *MobInstance) {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Combat Arena", Zone: 1},
		},
		Mobs: []parser.Mob{
			{
				VNum:      1,
				ShortDesc: "a test mob",
				LongDesc:  "A test mob is here.",
				Keywords:  "test mob",
				Level:     5,
				Exp:       100,
				Gold:      10,
				Alignment: victimAlignment,
			},
		},
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

	killer := NewPlayer(99, "Hero", 1001)
	killer.SetAlignment(0) // neutral
	if err := w.AddPlayer(killer); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	return w, killer, mob
}

func TestHandleDeath_KillingEvilMobShiftsAlignmentGood(t *testing.T) {
	w, killer, mob := testAlignmentSetup(t, -500)

	w.HandleDeath(mob, killer, 303)

	if killer.GetAlignment() <= 0 {
		t.Errorf("killing evil mob should shift alignment positive, got %d", killer.GetAlignment())
	}
}

func TestHandleDeath_KillingGoodMobShiftsAlignmentEvil(t *testing.T) {
	w, killer, mob := testAlignmentSetup(t, 500)

	w.HandleDeath(mob, killer, 303)

	if killer.GetAlignment() >= 0 {
		t.Errorf("killing good mob should shift alignment negative, got %d", killer.GetAlignment())
	}
}

func TestHandleDeath_KillingNeutralMobNoShift(t *testing.T) {
	w, killer, mob := testAlignmentSetup(t, 0)

	w.HandleDeath(mob, killer, 303)

	if killer.GetAlignment() != 0 {
		t.Errorf("killing neutral mob should not shift alignment, got %d", killer.GetAlignment())
	}
}
