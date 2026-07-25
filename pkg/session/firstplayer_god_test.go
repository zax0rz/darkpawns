package session

import (
	"os"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/testutil"
)

// TestShouldCrownFirstPlayer_HarnessLatch — under DP_FRESH_MUD, the in-process
// latch crowns exactly ONE character per boot. The first call returns true and
// flips the latch; every subsequent call returns false.
func TestShouldCrownFirstPlayer_HarnessLatch(t *testing.T) {
	t.Setenv("DP_FRESH_MUD", "1")
	m := makeTestManager(t)
	m.hasDB = false // harness runs with a dead DB; the env path doesn't touch it

	if !m.shouldCrownFirstPlayer() {
		t.Fatal("first shouldCrownFirstPlayer under DP_FRESH_MUD should crown (latch unset)")
	}
	// The latch now sticks: every subsequent char is an ordinary mortal.
	for i := 0; i < 3; i++ {
		if m.shouldCrownFirstPlayer() {
			t.Fatalf("shouldCrownFirstPlayer crowned a second time (call %d); latch must stick", i+2)
		}
	}
}

// TestShouldCrownFirstPlayer_NotFreshIsMortal — the regression guard: with
// DP_FRESH_MUD UNSET and no DB (the harness default for existing scenarios like
// combat-death), the first char is an ordinary mortal. This is what keeps
// existing scenarios from wrongly crowning their primary actor.
func TestShouldCrownFirstPlayer_NotFreshIsMortal(t *testing.T) {
	os.Unsetenv("DP_FRESH_MUD")
	m := makeTestManager(t)
	m.hasDB = false

	if m.shouldCrownFirstPlayer() {
		t.Fatal("shouldCrownFirstPlayer should be false with DP_FRESH_MUD unset and no DB (ordinary mortal)")
	}
}

// TestCompleteCharCreation_FreshGodThenMortal — end-to-end: under DP_FRESH_MUD,
// the first created character is crowned God (LVL_IMPL, all skills 100, conds
// -1); the second is an ordinary level-1 mortal of its class (backstab 10 for a
// thief, conditions not -1). The latch is consumed exactly once.
func TestCompleteCharCreation_FreshGodThenMortal(t *testing.T) {
	t.Setenv("DP_FRESH_MUD", "1")
	m := makeTestManager(t)

	createChar := func(name string) *game.Player {
		t.Helper()
		s := makeTestSession(t, m, name, 1001, false)
		s.charName = name
		s.charClass = game.ClassThief
		s.charRace = game.RaceHuman
		s.charSex = 1
		s.charStats = game.CharStats{Str: 15, Dex: 12, Con: 14, Int: 10, Wis: 11, Cha: 9}
		if err := s.completeCharCreation(); err != nil {
			t.Fatalf("completeCharCreation %s: %v", name, err)
		}
		if s.player == nil {
			t.Fatalf("completeCharCreation %s: player is nil", name)
		}
		return s.player
	}

	god := createChar("Alpha")
	if got := god.GetLevel(); got != game.LVL_IMPL {
		t.Errorf("first char level = %d, want LVL_IMPL (%d)", got, game.LVL_IMPL)
	}
	if got := god.GetExp(); got != 7000000 {
		t.Errorf("first char exp = %d, want 7000000", got)
	}
	if got := god.GetSkill("backstab"); got != 100 {
		t.Errorf("God backstab = %d, want 100", got)
	}
	if got := god.GetSkill("kick"); got != 100 {
		t.Errorf("God kick = %d, want 100", got)
	}
	if god.Hunger != -1 || god.Thirst != -1 {
		t.Errorf("God conditions = hunger %d thirst %d, want -1/-1", god.Hunger, god.Thirst)
	}
	if got := god.GetRoom(); got != game.ImmortStartRoom {
		t.Errorf("God room = %d, want ImmortStartRoom (%d)", got, game.ImmortStartRoom)
	}

	// Second char: ordinary mortal thief. GiveStartingSkills runs (backstab 10),
	// level 1, conditions 24/36 (not -1).
	mortal := createChar("Beta")
	if got := mortal.GetLevel(); got != 1 {
		t.Errorf("second char level = %d, want 1 (mortal)", got)
	}
	if got := mortal.GetSkill("backstab"); got != 10 {
		t.Errorf("mortal thief backstab = %d, want 10 (GiveStartingSkills)", got)
	}
	if got := mortal.GetSkill("kick"); got != 0 {
		t.Errorf("mortal kick = %d, want 0 (no God grant)", got)
	}
	if mortal.Hunger == -1 {
		t.Error("mortal hunger = -1, want a finite value (not God)")
	}
	// Mortal with hometown=0 (default) ends at MortalStartRoom after birth transition.
	if got := mortal.GetRoom(); got != game.MortalStartRoom {
		t.Errorf("mortal room = %d, want MortalStartRoom (%d)", got, game.MortalStartRoom)
	}
}

// TestCompleteCharCreation_GodRoomWithDB — the first-player God's live room and
// persisted RoomVNum must both be ImmortStartRoom (1204). No 8099, no
// NewbieHometownRoom anywhere on the God path (DP-1205).
func TestCompleteCharCreation_GodRoomWithDB(t *testing.T) {
	t.Setenv("DP_FRESH_MUD", "1")
	database := testutil.NewMockDatabase()
	world := testutil.NewTestWorld()
	t.Cleanup(world.StopAITicker)
	m := newTestManager(t, world, database)

	s := makeCharSession(t, m)
	s.charCreating = true
	s.charName = "TestGod"
	s.charClass = game.ClassWarrior
	s.charRace = game.RaceHuman
	s.charSex = 1
	s.charHometown = 1
	s.charPassword = "hashed_pw"
	s.charStats = game.CharStats{Str: 18, Dex: 18, Con: 18, Int: 18, Wis: 18, Cha: 18}

	if err := s.completeCharCreation(); err != nil {
		t.Fatalf("completeCharCreation: %v", err)
	}

	if got := s.player.GetLevel(); got != game.LVL_IMPL {
		t.Fatalf("God level = %d, want LVL_IMPL (%d)", got, game.LVL_IMPL)
	}

	// Live room must be ImmortStartRoom.
	if got := s.player.GetRoom(); got != game.ImmortStartRoom {
		t.Errorf("God live room = %d, want ImmortStartRoom (%d)", got, game.ImmortStartRoom)
	}

	// Persisted RoomVNum must also be ImmortStartRoom.
	record, err := database.GetPlayer("TestGod")
	if err != nil {
		t.Fatalf("GetPlayer: %v", err)
	}
	if record == nil {
		t.Fatal("created player record not found")
	}
	if record.RoomVNum != game.ImmortStartRoom {
		t.Errorf("God persisted room = %d, want ImmortStartRoom (%d)", record.RoomVNum, game.ImmortStartRoom)
	}

	// The Burning Hut (8099) must not appear anywhere on the God path.
	if record.RoomVNum == game.NewbieStartRoom {
		t.Error("God persisted room is NewbieStartRoom (8099); should be ImmortStartRoom")
	}
}
