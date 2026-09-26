package dbmigrate

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/moderation"
	"github.com/zax0rz/darkpawns/pkg/testutil"
)

// TestApplicationOpensAndMutatesMigratedDatabase is the proof that database
// equality is not the whole story: the migrated file is opened by the real game
// store, the real moderation manager and the real character codec, and every
// path the game uses at login and at save is exercised against it.
//
// The fixture is built by the application's own encoders, so nothing here
// depends on this package agreeing with itself about a JSON shape.
func TestApplicationOpensAndMutatesMigratedDatabase(t *testing.T) {
	sourceDSN, sourceConn := newSourceSchema(t)
	seedSource(t, sourceDSN)

	world := testutil.NewTestWorld()
	t.Cleanup(world.StopAITicker)
	player := testutil.NewTestPlayer("Zax", game.ClassWarrior, game.RaceHuman)
	player.Level = 34
	player.Exp = 1234567
	player.Gold = 4242
	player.BankGold = 99
	player.Alignment = -250
	player.Practices = 7
	player.PlayedDuration = 3600

	// A real hash, real inventory and real equipment, encoded by the application.
	hash, err := bcrypt.GenerateFromPassword([]byte("hunter2"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	// Capacity is what the restore path sets before it fills an inventory, so the
	// fixture sets it the same way rather than guessing at a limit. RestoreItem
	// reports overflow (not success) and appends the item either way.
	player.Stats.Str = 18
	player.Stats.Dex = 12
	player.Inventory.SetCapacity(player.Stats.Str, player.Stats.StrAdd, player.Stats.Dex, player.Level)
	club, err := world.SpawnObject(8023, -1)
	if err != nil {
		t.Fatalf("spawn club: %v", err)
	}
	if player.Inventory.RestoreItem(club) {
		t.Fatal("the fixture inventory overflowed")
	}
	tunic, err := world.SpawnObject(8019, -1)
	if err != nil {
		t.Fatalf("spawn tunic: %v", err)
	}
	player.Equipment.Slots[game.SlotWield] = tunic
	tunic.Location = game.LocEquippedPlayer(player.Name, game.SlotWield)

	encoded, err := db.PlayerToRecord(player, nil)
	if err != nil {
		t.Fatalf("encode fixture player: %v", err)
	}
	if _, err := sourceConn.Exec(`
		UPDATE players SET password_hash = $1, inventory = $2, equipment = $3, character_data = $4
		WHERE id = 7`, string(hash), string(encoded.Inventory), string(encoded.Equipment), string(encoded.CharacterData)); err != nil {
		t.Fatalf("install app-shaped fixture: %v", err)
	}

	destination := filepath.Join(t.TempDir(), "darkpawns.db")
	receipt, err := Run(context.Background(), migrationOptions(sourceDSN, destination))
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if !receipt.OK {
		t.Fatalf("receipt = %s", receipt.Render())
	}

	database, err := db.New("sqlite://" + destination)
	if err != nil {
		t.Fatalf("the application could not open the migrated database: %v", err)
	}

	loaded, err := database.GetPlayer("Zax")
	if err != nil {
		t.Fatalf("load migrated player: %v", err)
	}

	// Password verification through the real hash.
	if err := bcrypt.CompareHashAndPassword([]byte(loaded.Password), []byte("hunter2")); err != nil {
		t.Errorf("the migrated password hash does not verify: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(loaded.Password), []byte("wrong")); err == nil {
		t.Error("the migrated password hash accepts a wrong password")
	}
	// C's find_name is case-insensitive, and the migrated index must keep it so.
	sameCharacter, err := database.GetPlayer("zax")
	if err != nil {
		t.Fatalf("case-insensitive lookup: %v", err)
	}
	if sameCharacter.ID != loaded.ID {
		t.Errorf("case-insensitive lookup returned id %d, want %d", sameCharacter.ID, loaded.ID)
	}

	// Room and load room selection, inventory, equipment and character data.
	restored, err := db.RecordToPlayer(loaded, world)
	if err != nil {
		t.Fatalf("restore migrated player: %v", err)
	}
	if restored.GetRoom() != 8004 {
		t.Errorf("restored room = %d, want 8004", restored.GetRoom())
	}
	if restored.GetLoadRoom() != 8004 {
		t.Errorf("restored load room = %d, want 8004", restored.GetLoadRoom())
	}
	if len(restored.Inventory.Items) != 1 || restored.Inventory.Items[0].GetVNum() != 8023 {
		t.Errorf("restored inventory = %v, want the club", restored.Inventory.Items)
	}
	if wielded, ok := restored.Equipment.Slots[game.SlotWield]; !ok || wielded == nil || wielded.GetVNum() != 8019 {
		t.Errorf("restored wield slot = %v, want the tunic", restored.Equipment.Slots[game.SlotWield])
	}
	if restored.Gold != 4242 || restored.BankGold != 99 || restored.Alignment != -250 ||
		restored.Practices != 7 || restored.PlayedDuration != 3600 {
		t.Errorf("character_data did not restore: gold=%d bank=%d align=%d practices=%d played=%d",
			restored.Gold, restored.BankGold, restored.Alignment, restored.Practices, restored.PlayedDuration)
	}

	// The negative room sentinel survives into the application too.
	roamer, err := database.GetPlayer("roamer")
	if err != nil {
		t.Fatalf("load roamer: %v", err)
	}
	if roamer.RoomVNum != -1 {
		t.Errorf("roamer room = %d, want the -1 sentinel", roamer.RoomVNum)
	}
	if roamer.Password != "" {
		t.Errorf("roamer password should still be NULL/empty, got %q", roamer.Password)
	}

	// Saving again writes through the migrated schema.
	loaded.Exp = 7654321
	loaded.Title = "renamed after migration"
	if err := database.SavePlayer(loaded); err != nil {
		t.Fatalf("save migrated player: %v", err)
	}

	// A new character is allocated above every migrated id.
	newcomer := &db.PlayerRecord{Name: "newcomer", Inventory: []byte("[]"), Equipment: []byte("{}"), CharacterData: []byte("{}")}
	if err := database.CreatePlayer(newcomer); err != nil {
		t.Fatalf("create player after migration: %v", err)
	}
	if newcomer.ID <= 42 {
		t.Errorf("new player id = %d, want above the migrated maximum 42", newcomer.ID)
	}

	// Lockout state migrated and still behaves.
	attempts, lockedUntil, err := database.GetAccountLockout("Zax")
	if err != nil {
		t.Fatalf("lockout read: %v", err)
	}
	if attempts != 3 {
		t.Errorf("failed_login_attempts = %d, want 3", attempts)
	}
	if lockedUntil == nil || !lockedUntil.After(time.Now()) {
		t.Errorf("locked_until = %v, want a lockout still in force", lockedUntil)
	}
	if _, err := database.RecordLoginFailure("newcomer", 3, time.Hour); err != nil {
		t.Fatalf("record login failure: %v", err)
	}
	if err := database.RecordLoginSuccess("newcomer"); err != nil {
		t.Fatalf("record login success: %v", err)
	}

	// Moderation reads and writes against the migrated tables.
	manager := moderation.NewManager(database.SQLDB(), database.Dialect())
	defer manager.Close()
	reports, err := manager.ListReports()
	if err != nil {
		t.Fatalf("list migrated reports: %v", err)
	}
	if len(reports) != 2 {
		t.Errorf("migrated reports = %d, want 2", len(reports))
	}
	filters := manager.GetWordFilters()
	if len(filters) != 2 {
		t.Errorf("migrated word filters = %d, want 2", len(filters))
	}
	// The third return value means "blocked", not "matched": a censor filter
	// rewrites the message and leaves the boolean false. Both migrated filters are
	// exercised, the literal one and the regex one.
	filtered, action, _ := manager.CheckMessage("tester", "this contains badword here")
	if action != moderation.FilterActionCensor {
		t.Errorf("migrated censor filter produced action %q", action)
	}
	if strings.Contains(strings.ToLower(filtered), "badword") {
		t.Errorf("migrated censor filter did not censor: %q", filtered)
	}
	blocked, blockAction, wasBlocked := manager.CheckMessage("tester", "I hate such speech")
	if !wasBlocked || blockAction != moderation.FilterActionBlock || blocked != "" {
		t.Errorf("migrated regex block filter: blocked=%t action=%q message=%q", wasBlocked, blockAction, blocked)
	}

	if err := manager.AddReport(moderation.AbuseReport{
		Reporter: "tester", Target: "roamer", ReportType: moderation.ReportTypeSpam,
		Description: "after migration", Timestamp: time.Now(), Status: "pending",
	}); err != nil {
		t.Fatalf("add report after migration: %v", err)
	}
	if err := manager.AddWordFilter("(?i)after.*migration", true, "warn", "system"); err != nil {
		t.Fatalf("add word filter after migration: %v", err)
	}
	future := time.Now().Add(2 * time.Hour)
	if err := manager.AddPenalty(moderation.PlayerPenalty{
		PlayerName: "roamer", PenaltyType: moderation.ActionMute, IssuedAt: time.Now(),
		ExpiresAt: &future, Reason: "after migration", IssuedBy: "Zax",
	}); err != nil {
		t.Fatalf("add penalty after migration: %v", err)
	}
	if !manager.IsMuted("roamer") {
		t.Error("a penalty written after the migration is not in force")
	}
	reports, err = manager.ListReports()
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 3 {
		t.Errorf("reports after write = %d, want 3", len(reports))
	}
	for _, report := range reports {
		if report.Reporter == "tester" && report.ID <= 9 {
			t.Errorf("new report id = %d, want above the migrated maximum 9", report.ID)
		}
	}

	// Clean close and reopen: everything written is still there.
	manager.Close()
	if err := database.Close(); err != nil {
		t.Fatalf("close migrated database: %v", err)
	}
	reopened, err := db.New("sqlite://" + destination)
	if err != nil {
		t.Fatalf("reopen migrated database: %v", err)
	}
	defer func() { _ = reopened.Close() }()
	again, err := reopened.GetPlayer("Zax")
	if err != nil {
		t.Fatalf("reload after reopen: %v", err)
	}
	if again.Exp != 7654321 || again.Title != "renamed after migration" {
		t.Errorf("post-reopen player = exp %d title %q", again.Exp, again.Title)
	}
	count, err := reopened.CountPlayers()
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Errorf("players after reopen = %d, want 3 (two migrated, one created)", count)
	}

	// The reopened file is still a valid database, not a fresh empty one.
	var integrity string
	if err := reopened.SQLDB().QueryRow(`PRAGMA integrity_check`).Scan(&integrity); err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(integrity, "ok") {
		t.Errorf("integrity after reopen = %q", integrity)
	}
}

// TestMigratedDatabaseIsNotAFreshInstall guards the specific failure the
// cutover document warns about: a service that starts against a path it created
// by accident looks healthy until the first login finds no characters.
func TestMigratedDatabaseIsNotAFreshInstall(t *testing.T) {
	sourceDSN, sourceConn := newSourceSchema(t)
	seedSource(t, sourceDSN)
	destination := filepath.Join(t.TempDir(), "darkpawns.db")
	if _, err := Run(context.Background(), migrationOptions(sourceDSN, destination)); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	fresh := filepath.Join(t.TempDir(), "fresh.db")
	freshDatabase, err := db.New("sqlite://" + fresh)
	if err != nil {
		t.Fatal(err)
	}
	migratedDatabase, err := db.New("sqlite://" + destination)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = migratedDatabase.Close() }()

	freshCount, err := freshDatabase.CountPlayers()
	if err != nil {
		t.Fatal(err)
	}
	migratedCount, err := migratedDatabase.CountPlayers()
	if err != nil {
		t.Fatal(err)
	}
	if err := freshDatabase.Close(); err != nil {
		t.Fatal(err)
	}

	want := int(tableRowCount(t, sourceConn, "players"))
	if migratedCount != want {
		t.Errorf("migrated database has %d players, want %d", migratedCount, want)
	}
	if freshCount != 0 {
		t.Errorf("a fresh install has %d players, want 0", freshCount)
	}
	if migratedCount == freshCount {
		t.Error("the migrated database is indistinguishable from a fresh install")
	}
	var seeded int
	if err := migratedDatabase.SQLDB().QueryRow(
		`SELECT COUNT(*) FROM word_filters`).Scan(&seeded); err != nil {
		t.Fatal(err)
	}
	if seeded == 0 {
		t.Error("the migrated database lost its moderation rows")
	}
}
