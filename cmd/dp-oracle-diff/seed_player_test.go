package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/internal/oraclediff"
	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"

	"golang.org/x/crypto/bcrypt"
)

const seedPlayerScenario = `[fixture]
quiet-mobs
seed-player

[setup:oracle]
Testor
testpassword
<ENTER>
1

[setup:port]
Testor
testpassword
<ENTER>
1

[probe]
gold
`

func parseSeedPlayerScenario(t *testing.T, body string) oraclediff.Scenario {
	t.Helper()
	scenario, err := oraclediff.ParseScenario("seed-player-probe", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	return scenario
}

// The seed-player fixture must force the port's real SQLite store: the twin
// lives there, and without a store the login would fall through to creation.
func TestSeedPlayerFixtureForcesPlayerStore(t *testing.T) {
	scenario := parseSeedPlayerScenario(t, seedPlayerScenario)
	if !scenario.SeedPlayer {
		t.Fatal("seed-player fixture did not set Scenario.SeedPlayer")
	}
	if !needsPlayerStore(&scenario) {
		t.Fatal("seed-player scenario does not request the player store")
	}
}

// seed-player and empty-players are contradictory: one logs an existing
// player in, the other demands a fresh world on both sides.
func TestSeedPlayerRejectsEmptyPlayers(t *testing.T) {
	body := strings.Replace(seedPlayerScenario, "seed-player", "seed-player\nempty-players", 1)
	_, err := oraclediff.ParseScenario("seed-empty-probe", strings.NewReader(body))
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("expected mutually-exclusive error, got %v", err)
	}
}

// The twin must be C's shipped first player (lib/etc/players record 0):
// init_char's first-player block (BootstrapFirstPlayerGod ports db.c:3014-3074)
// plus the creation rolls the shipped record kept. Without the store seeded
// this way the port cannot log Testor in, and every seed-player scenario
// diverges — the red control in the PR description demonstrates exactly that.
func TestPrepareGoReloginDBSeedsTestorTwin(t *testing.T) {
	url, err := prepareGoReloginDB(t.TempDir(), false, true)
	if err != nil {
		t.Fatal(err)
	}
	store, err := db.New(url)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	record, err := store.GetPlayer("Testor")
	if err != nil {
		t.Fatalf("seeded store lookup failed: %v", err)
	}
	if record == nil {
		t.Fatal("seeded store has no Testor")
	}
	if record.ID != 1 {
		t.Fatalf("Testor twin got ID %d, want 1 (the house-control owner alignment)", record.ID)
	}
	if record.Level != game.LVL_IMPL || record.Class != game.ClassWarrior || record.Race != game.RaceHuman {
		t.Fatalf("twin identity drifted: level=%d class=%d race=%d", record.Level, record.Class, record.Race)
	}
	if record.Exp != 7000000 || record.MaxHealth != 500 || record.MaxMana != 100 || record.MaxMove != 82 {
		t.Fatalf("twin points drifted: exp=%d max_hit=%d max_mana=%d max_move=%d",
			record.Exp, record.MaxHealth, record.MaxMana, record.MaxMove)
	}
	if record.StatStr != 17 || record.StatInt != 9 || record.StatWis != 13 ||
		record.StatDex != 16 || record.StatCon != 13 || record.StatCha != 8 {
		t.Fatalf("twin stats drifted: %+v", []int{record.StatStr, record.StatInt, record.StatWis, record.StatDex, record.StatCon, record.StatCha})
	}
	if record.RoomVNum != 1204 {
		t.Fatalf("twin room=%d, want 1204 (immort_start_room, src/config.c:149)", record.RoomVNum)
	}
	if record.Hometown != 1 {
		t.Fatalf("twin hometown=%d, want 1", record.Hometown)
	}
	// Height and weight live in the CharacterData payload (save.go's
	// char_player_data mirror), not in a column.
	var charData struct {
		Height int `json:"height"`
		Weight int `json:"weight"`
	}
	if err := json.Unmarshal(record.CharacterData, &charData); err != nil {
		t.Fatalf("decode twin CharacterData: %v", err)
	}
	if charData.Height != 175 || charData.Weight != 159 {
		t.Fatalf("twin creation rolls drifted: height=%d weight=%d", charData.Height, charData.Weight)
	}
	if record.Hunger != -1 || record.Thirst != -1 || record.Drunk != -1 {
		t.Fatalf("twin conditions drifted: %d/%d/%d, want -1/-1/-1 (immortal)", record.Hunger, record.Thirst, record.Drunk)
	}
	// The shipped record's bad_pws is 1: both engines print the LOGIN FAILURES
	// line at login (src/interpreter.c:1929-1937).
	if record.FailedLoginAttempts != 1 {
		t.Fatalf("twin failed_login_attempts=%d, want 1", record.FailedLoginAttempts)
	}
	if bcrypt.CompareHashAndPassword([]byte(record.Password), []byte("testpassword")) != nil {
		t.Fatal("twin password does not verify against testpassword")
	}

	// Only the twin is seeded: the placeholder exists to stop a fresh creation
	// being crowned, and the seeded Testor already does that.
	placeholder, err := store.GetPlayer("Zzplaceholder")
	if err != nil {
		t.Fatalf("placeholder lookup failed: %v", err)
	}
	if placeholder != nil {
		t.Fatal("seed-player store also seeded Zzplaceholder")
	}
}

// Default off: without the fixture the placeholder path is exactly what it
// was before seed-player existed.
func TestPrepareGoReloginDBDefaultSeedsPlaceholder(t *testing.T) {
	url, err := prepareGoReloginDB(t.TempDir(), false, false)
	if err != nil {
		t.Fatal(err)
	}
	store, err := db.New(url)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	placeholder, err := store.GetPlayer("Zzplaceholder")
	if err != nil {
		t.Fatalf("default relogin store lookup failed: %v", err)
	}
	if placeholder == nil {
		t.Fatal("default relogin store lost its placeholder")
	}
	testor, err := store.GetPlayer("Testor")
	if err != nil {
		t.Fatalf("default relogin store lookup failed: %v", err)
	}
	if testor != nil {
		t.Fatal("default relogin store seeded Testor without the fixture")
	}
}
