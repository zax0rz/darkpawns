package session

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/dprng"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/testutil"
	"golang.org/x/crypto/bcrypt"
)

// These contract tests intentionally use PostgreSQL, not MockDatabase: name
// equality, uniqueness, and persistence timing are part of this regression.
// Set DP_ENTRY_TEST_DATABASE_URL to a disposable local PostgreSQL database.
// Each test owns an isolated schema; no production data or passwords are used.
func entryDatabase(t *testing.T) *db.DB {
	t.Helper()
	dsn := os.Getenv("DP_ENTRY_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set DP_ENTRY_TEST_DATABASE_URL to run PostgreSQL entry contract tests")
	}
	u, err := url.Parse(dsn)
	if err != nil || (u.Hostname() != "127.0.0.1" && u.Hostname() != "localhost") {
		t.Fatal("entry tests require a local PostgreSQL URL")
	}
	admin, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = admin.Close() })
	schema := fmt.Sprintf("entry_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE SCHEMA " + schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec("DROP SCHEMA " + schema + " CASCADE"); err != nil {
			t.Errorf("drop isolated entry schema: %v", err)
		}
	})
	q := u.Query()
	// Use a lib/pq startup option so every pooled connection, including the
	// cleanup save path, resolves the isolated schema.
	q.Set("options", "-csearch_path="+schema)
	u.RawQuery = q.Encode()
	database, err := db.New(u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return database
}

func entrySeed(t *testing.T, database *db.DB, name string) *db.PlayerRecord {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("oraclepass"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	p := &db.PlayerRecord{
		Name: name, Password: string(hash), RoomVNum: game.MortalStartRoom,
		Level: 1, Health: 20, MaxHealth: 20, Mana: 20, MaxMana: 20,
		Move: 100, MaxMove: 100, Class: game.ClassThief, Race: game.RaceKender,
		StatStr: 10, StatInt: 10, StatWis: 10, StatDex: 10, StatCon: 10, StatCha: 10,
		Inventory: []byte("[]"), Equipment: []byte("{}"),
	}
	if err := database.CreatePlayer(p); err != nil {
		t.Fatal(err)
	}
	return p
}

func entrySession(t *testing.T, database db.Database) *Session {
	t.Helper()
	t.Setenv("JWT_SECRET", "entry-test-jwt-secret-at-least-32-characters")
	world := testutil.NewTestWorld()
	t.Cleanup(world.StopAITicker)
	return makeCharSession(t, newTestManager(t, world, database))
}

type entryCountingDatabase struct {
	db.Database
	creates int
}

func (d *entryCountingDatabase) CreatePlayer(p *db.PlayerRecord) error {
	d.creates++
	return d.Database.CreatePlayer(p)
}

type entryFaultDatabase struct {
	db.Database
	getErr        error
	countErr      error
	createErr     error
	saveErr       error
	saveFailsOnce bool
}

func (d *entryFaultDatabase) GetPlayer(name string) (*db.PlayerRecord, error) {
	if d.getErr != nil {
		return nil, d.getErr
	}
	return d.Database.GetPlayer(name)
}

func (d *entryFaultDatabase) CountPlayers() (int, error) {
	if d.countErr != nil {
		return 0, d.countErr
	}
	return d.Database.CountPlayers()
}

func (d *entryFaultDatabase) CreatePlayer(p *db.PlayerRecord) error {
	if d.createErr != nil {
		return d.createErr
	}
	return d.Database.CreatePlayer(p)
}

func (d *entryFaultDatabase) SavePlayer(p *db.PlayerRecord) error {
	if d.saveErr != nil {
		err := d.saveErr
		if d.saveFailsOnce {
			d.saveErr = nil
		}
		return err
	}
	return d.Database.SavePlayer(p)
}

func entryInput(s *Session, choice string) error {
	data, err := json.Marshal(CharInputData{Choice: choice})
	if err != nil {
		return err
	}
	msg, err := json.Marshal(ClientMessage{Type: MsgCharInput, Data: data})
	if err != nil {
		return err
	}
	return s.handleMessage(msg)
}

func driveEntryToStats(t *testing.T, s *Session, name string) {
	t.Helper()
	s.startNewCharFlow(name)
	for _, line := range []string{"Y", "oraclepass", "oraclepass", "N", "M", "K", "T", "K"} {
		if err := entryInput(s, line); err != nil {
			t.Fatal(err)
		}
	}
}

// C: load_char -> find_name -> str_cmp (interpreter.c, db.c, utils.c).
func TestEntryIdentityCaseInsensitive(t *testing.T) {
	database := entryDatabase(t)
	want := entrySeed(t, database, "Aiko")
	for _, name := range []string{"Aiko", "aiko", "AIKO"} {
		got, err := database.GetPlayer(name)
		if err != nil {
			t.Fatal(err)
		}
		if got == nil || got.ID != want.ID {
			t.Errorf("lookup %q did not resolve saved identity %d", name, want.ID)
		}
	}
}

func TestEntryLookupFailureFailsClosed(t *testing.T) {
	database := entryDatabase(t)
	fault := &entryFaultDatabase{Database: database, getErr: errors.New("lookup unavailable")}
	s := entrySession(t, fault)
	if err := s.handleLogin(loginMsg("Aiko", "")); err != nil {
		t.Fatal(err)
	}
	if !s.SendClosed() || s.player != nil || s.authenticated || s.charCreating {
		t.Fatal("lookup failure left an entry candidate or open session")
	}
	if count, err := database.CountPlayers(); err != nil || count != 0 {
		t.Fatalf("lookup failure changed player count: count=%d err=%v", count, err)
	}
}

func TestEntryCountFailureFailsClosed(t *testing.T) {
	database := entryDatabase(t)
	fault := &entryFaultDatabase{Database: database, countErr: errors.New("count unavailable")}
	s := entrySession(t, fault)
	driveEntryToStats(t, s, "Newhero")
	if err := entryInput(s, "Y"); err != nil {
		t.Fatal(err)
	}
	if !s.SendClosed() || s.player != nil || s.authenticated || s.menuActive {
		t.Fatal("count failure left an entry candidate or open menu")
	}
	if count, err := database.CountPlayers(); err != nil || count != 0 {
		t.Fatalf("count failure changed player count: count=%d err=%v", count, err)
	}
}

func TestEntryEntrySaveFailureFailsClosed(t *testing.T) {
	database := entryDatabase(t)
	entrySeed(t, database, "Founder")
	fault := &entryFaultDatabase{Database: database, saveErr: errors.New("entry save unavailable")}
	s := entrySession(t, fault)
	driveEntryToStats(t, s, "Newhero")
	if err := entryInput(s, "Y"); err != nil {
		t.Fatal(err)
	}
	if err := entryInput(s, ""); err != nil {
		t.Fatal(err)
	}
	if err := entryInput(s, "1"); err != nil {
		t.Fatal(err)
	}
	if !s.SendClosed() || s.player != nil || s.authenticated || s.menuActive {
		t.Fatal("entry-save failure left an enterable candidate")
	}
	if _, present := s.manager.world.GetPlayer("Newhero"); present {
		t.Fatal("entry-save failure admitted the character to the world")
	}
	stored, err := database.GetPlayer("Newhero")
	if err != nil || stored == nil || stored.Level != 0 {
		t.Fatalf("entry-save failure changed persisted candidate: stored=%+v err=%v", stored, err)
	}
}

func TestEntryTransientSaveFailureDoesNotPersistAbortedBootstrap(t *testing.T) {
	database := entryDatabase(t)
	entrySeed(t, database, "Founder")
	fault := &entryFaultDatabase{Database: database, saveErr: errors.New("entry save unavailable"), saveFailsOnce: true}
	s := entrySession(t, fault)
	driveEntryToStats(t, s, "Newhero")
	if err := entryInput(s, "Y"); err != nil {
		t.Fatal(err)
	}
	if err := entryInput(s, ""); err != nil {
		t.Fatal(err)
	}
	if err := entryInput(s, "1"); err != nil {
		t.Fatal(err)
	}
	if !s.SendClosed() || s.player != nil || s.authenticated || s.menuActive {
		t.Fatal("entry-save failure left an enterable candidate")
	}
	if _, present := s.manager.world.GetPlayer("Newhero"); present {
		t.Fatal("entry-save failure admitted the character to the world")
	}
	stored, err := database.GetPlayer("Newhero")
	if err != nil || stored == nil || stored.Level != 0 {
		t.Fatalf("entry-save failure changed persisted candidate: stored=%+v err=%v", stored, err)
	}
}

// C: CON_NAME_CNFRM N returns to CON_GET_NAME, including load_char.
func TestEntryNameRetryLoadsExisting(t *testing.T) {
	database := entryDatabase(t)
	entrySeed(t, database, "Aiko")
	s := entrySession(t, database)
	if err := s.handleLogin(loginMsg("Freshname", "")); err != nil {
		t.Fatal(err)
	}
	_ = drainMsg(t, s)
	sendCharInput(t, s, "N")
	_ = drainMsg(t, s)
	sendCharInput(t, s, "Aiko")
	_, prompt := unmarshalCharCreate(t, drainMsg(t, s))
	if prompt.Prompt != "Password: " || !prompt.Secret {
		t.Errorf("retry into saved name: prompt=%q secret=%v; want existing-password prompt", prompt.Prompt, prompt.Secret)
	}
}

// C: CON_ROLLABL2 Y calls init_char and save_char before displaying the MOTD.
func TestEntryAcceptedStatsPersistBeforeMenu(t *testing.T) {
	database := entryDatabase(t)
	entrySeed(t, database, "Founder") // Exercise the mortal path.
	s := entrySession(t, database)
	s.startNewCharFlow("Newhero")
	for _, line := range []string{"Y", "oraclepass", "oraclepass", "N", "M", "K", "T", "K", "Y"} {
		sendCharInput(t, s, line)
	}
	p, err := database.GetPlayer("Newhero")
	if err != nil {
		t.Fatal(err)
	}
	if p == nil {
		t.Fatal("accepted stats reached MOTD without persisting the character")
	}
	if _, present := s.manager.world.GetPlayer("Newhero"); present {
		t.Fatal("character entered the world before menu option 1")
	}
}

// The corrected Aiko journey resolves the saved identity immediately. The
// obsolete creation keystrokes remain in the recorded pre-fix RED evidence.
func TestEntryAikoResolvesExistingWithoutCreation(t *testing.T) {
	database := entryDatabase(t)
	want := entrySeed(t, database, "Aiko")
	counted := &entryCountingDatabase{Database: database}
	s := entrySession(t, counted)
	if err := s.handleLogin(loginMsg("aiko", "")); err != nil {
		t.Fatal(err)
	}
	_, prompt := unmarshalCharCreate(t, drainMsg(t, s))
	if prompt.Prompt != "Password: " || !prompt.Secret {
		t.Fatalf("existing identity received %q", prompt.Prompt)
	}
	for _, line := range []string{"oraclepass", "", "1"} {
		if err := entryInput(s, line); err != nil {
			t.Fatal(err)
		}
	}
	if counted.creates != 0 || s.player.ID != want.ID || !s.authenticated {
		t.Fatalf("returning identity: creates=%d ID=%d authenticated=%v", counted.creates, s.player.ID, s.authenticated)
	}
	if _, ok := s.manager.GetSession("AIKO"); !ok {
		t.Fatal("online session lookup lost case-insensitive identity")
	}
	if _, ok := s.manager.world.GetPlayer("aiko"); !ok {
		t.Fatal("world lookup lost case-insensitive identity")
	}
}

// A second creator claims the name after confirmation but before acceptance.
// PostgreSQL, not a mocked constraint, rejects the losing insert.
func TestEntryAikoCollisionDoesNotLeaveEnterableCandidate(t *testing.T) {
	database := entryDatabase(t)
	counted := &entryCountingDatabase{Database: database}
	s := entrySession(t, counted)
	s.startNewCharFlow("aiko")
	for _, line := range []string{"Y", "oraclepass", "oraclepass", "Y", "M", "K", "T", "K"} {
		if err := entryInput(s, line); err != nil {
			t.Fatal(err)
		}
	}
	winner := entrySeed(t, database, "Aiko")
	if err := entryInput(s, "Y"); err != nil {
		t.Fatal(err)
	}
	if !s.SendClosed() || s.player != nil || s.authenticated || s.menuActive {
		t.Fatal("losing insert retained an enterable or authenticated candidate")
	}
	if err := entryInput(s, "1"); err == nil {
		t.Fatal("closed failed creator accepted menu input")
	}
	if _, present := s.manager.world.GetPlayer("Aiko"); present || counted.creates != 1 {
		t.Fatalf("failed creator entered/retried: present=%v creates=%d", present, counted.creates)
	}
	stored, err := database.GetPlayer("aiko")
	if err != nil || stored == nil || stored.ID != winner.ID || stored.Password != winner.Password {
		t.Fatal("losing creator changed the winning identity")
	}
}

func TestEntryMenuDisconnectResumesSavedCharacter(t *testing.T) {
	database := entryDatabase(t)
	entrySeed(t, database, "Founder")
	s := entrySession(t, database)
	s.startNewCharFlow("Newhero")
	for _, line := range []string{"Y", "oraclepass", "oraclepass", "N", "M", "K", "T", "K", "Y"} {
		if err := entryInput(s, line); err != nil {
			t.Fatal(err)
		}
	}
	want := s.player.ID
	stats := s.player.Stats
	s.CloseSend() // No world entry or explicit save/quit is needed.
	returning := makeCharSession(t, s.manager)
	if err := returning.handleLogin(loginMsg("NEWHERO", "oraclepass")); err != nil {
		t.Fatal(err)
	}
	if returning.player == nil || returning.player.Level != 0 || returning.player.ID != want || returning.player.Stats != stats {
		t.Fatal("menu reconnect failed to restore the unfinished level-zero character")
	}
	for _, line := range []string{"", "1"} {
		if err := entryInput(returning, line); err != nil {
			t.Fatal(err)
		}
	}
	if returning.player.Level != 1 || returning.player.ID != want || returning.player.GetRoom() != game.NewbieStartRoom {
		t.Fatal("reconnected character did not perform first entry exactly once")
	}
	stored, err := database.GetPlayer("Newhero")
	if err != nil || stored == nil || stored.Level != 1 {
		t.Fatal("first-entry level was not persisted")
	}
}

func TestEntryDatabasePersistsGodAndMortalEntry(t *testing.T) {
	database := entryDatabase(t)
	t.Setenv("DP_FRESH_MUD", "")

	god := entrySession(t, database)
	driveEntryToStats(t, god, "Freshgod")
	if err := entryInput(god, "Y"); err != nil {
		t.Fatal(err)
	}
	godLevel := -1
	if god.player != nil {
		godLevel = god.player.GetLevel()
	}
	if godLevel < game.LVL_IMMORT {
		t.Fatalf("first persisted character level = %d, want God", godLevel)
	}
	if err := entryInput(god, ""); err != nil {
		t.Fatal(err)
	}
	if err := entryInput(god, "1"); err != nil {
		t.Fatal(err)
	}
	godRecord, err := database.GetPlayer("FRESHGOD")
	if err != nil || godRecord == nil || godRecord.Level < game.LVL_IMMORT {
		t.Fatalf("God acceptance/entry was not persisted: record=%+v err=%v", godRecord, err)
	}

	mortal := entrySession(t, database)
	driveEntryToStats(t, mortal, "Freshmortal")
	if err := entryInput(mortal, "Y"); err != nil {
		t.Fatal(err)
	}
	mortalLevel := -1
	if mortal.player != nil {
		mortalLevel = mortal.player.GetLevel()
	}
	if mortalLevel != 0 {
		t.Fatalf("mortal accepted-stat level = %d, want level zero before menu entry", mortalLevel)
	}
	if err := entryInput(mortal, ""); err != nil {
		t.Fatal(err)
	}
	if err := entryInput(mortal, "1"); err != nil {
		t.Fatal(err)
	}
	mortalRecord, err := database.GetPlayer("FRESHMORTAL")
	if err != nil || mortalRecord == nil || mortalRecord.Level != 1 {
		t.Fatalf("mortal first entry was not persisted: record=%+v err=%v", mortalRecord, err)
	}
	if godRecord.ID == mortalRecord.ID {
		t.Fatal("God and mortal persistence reused one player ID")
	}
}

func TestEntryPasswordRetries(t *testing.T) {
	database := entryDatabase(t)
	entrySeed(t, database, "Aiko")
	s := entrySession(t, database)
	if err := s.handleLogin(loginMsg("aiko", "")); err != nil {
		t.Fatal(err)
	}
	_ = drainMsg(t, s)
	for attempt := 1; attempt <= 3; attempt++ {
		if err := entryInput(s, "wrongpass"); err != nil {
			t.Fatal(err)
		}
		_, prompt := unmarshalCharCreate(t, drainMsg(t, s))
		want := "Wrong password.\r\nPassword: "
		if attempt == 3 {
			want = "Wrong password... disconnecting.\r\n"
		}
		if prompt.Prompt != want || s.SendClosed() != (attempt == 3) || s.player != nil || s.authenticated {
			t.Fatalf("attempt %d: %q, closed=%v", attempt, prompt.Prompt, s.SendClosed())
		}
	}
}

func TestEntryConcurrentCaseVariantsCannotCreateTwoRows(t *testing.T) {
	database := entryDatabase(t)
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, name := range []string{"Aiko", "aiko"} {
		wg.Go(func() {
			results <- database.CreatePlayer(&db.PlayerRecord{Name: name, Inventory: []byte("[]"), Equipment: []byte("{}")})
		})
	}
	wg.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		} else if !isUniqueConstraintError(err) {
			t.Fatalf("unexpected insert failure: %v", err)
		}
	}
	if successes != 1 {
		t.Fatalf("successful case-variant inserts = %d, want 1", successes)
	}
}

func TestEntryConcurrentCaseVariantSessionsCannotCreateTwoRows(t *testing.T) {
	database := entryDatabase(t)
	t.Setenv("DP_FRESH_MUD", "")
	s1 := entrySession(t, database)
	s2 := entrySession(t, database)
	driveEntryToStats(t, s1, "Aiko")
	driveEntryToStats(t, s2, "aiko")

	var wg sync.WaitGroup
	for _, s := range []*Session{s1, s2} {
		wg.Add(1)
		go func(current *Session) {
			defer wg.Done()
			if err := entryInput(current, "Y"); err != nil {
				t.Errorf("concurrent stats acceptance: %v", err)
			}
		}(s)
	}
	wg.Wait()

	count, err := database.CountPlayers()
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("concurrent case-variant sessions created %d rows, want 1", count)
	}
	winners := 0
	for _, s := range []*Session{s1, s2} {
		if s.creationSaved && s.authenticated && s.player != nil {
			winners++
			continue
		}
		if !s.SendClosed() || s.player != nil || s.authenticated {
			t.Fatal("losing concurrent creator remained registered or enterable")
		}
		if err := entryInput(s, "1"); err == nil {
			t.Fatal("closed losing creator accepted menu input")
		}
	}
	if winners != 1 {
		t.Fatalf("concurrent case-variant session winners = %d, want 1", winners)
	}
}

func TestEntryLegacyCollisionIsAmbiguous(t *testing.T) {
	database := entryDatabase(t)
	// Reproduce a pre-migration database without changing real records.
	if _, err := database.Exec("DROP INDEX players_name_folded_key"); err != nil {
		t.Fatal(err)
	}
	entrySeed(t, database, "Aiko")
	entrySeed(t, database, "aiko")
	if _, err := database.GetPlayer("Aiko"); !errors.Is(err, db.ErrAmbiguousPlayerName) {
		t.Fatalf("ambiguous identity lookup = %v", err)
	}
	s := entrySession(t, database)
	if err := s.handleLogin(loginMsg("aiko", "oraclepass")); err != nil {
		t.Fatal(err)
	}
	if !s.SendClosed() || s.player != nil || s.authenticated {
		t.Fatal("ambiguous identity admitted a character")
	}
}

// R3: init_char consumes the two body draws at acceptance; do_start draws
// only on mortal first entry. Reloading saved records must consume no draws.
func TestEntryCreationDrawBoundaries(t *testing.T) {
	for _, god := range []bool{false, true} {
		t.Run(fmt.Sprint(god), func(t *testing.T) {
			t.Setenv("DP_CLOCK", "1")
			t.Setenv("DP_FRESH_MUD", "")
			if god {
				t.Setenv("DP_FRESH_MUD", "1")
			}
			m := makeTestManager(t)
			s := makeCharSession(t, m)
			s.charName, s.charClass, s.charRace = "Drawtest", game.ClassThief, game.RaceHuman
			s.charStats = game.CharStats{Str: 15, Dex: 12, Con: 14, Int: 10, Wis: 11, Cha: 9}
			dprng.ResetStream(123)
			expected := dprng.New(123)
			wantWeight, wantHeight := expected.Number(120, 180), expected.Number(160, 200)
			if err := s.persistAcceptedCharacter(); err != nil {
				t.Fatal(err)
			}
			if s.player.Weight != wantWeight || s.player.Height != wantHeight || dprng.Next() != expected.Next() {
				t.Fatal("init_char body values/draw count diverged")
			}
			if !god {
				expected.Number(7, 13)
				expected.Number(1, 4)
			}
			if err := s.completeCharCreation(); err != nil {
				t.Fatal(err)
			}
			if dprng.Next() != expected.Next() {
				t.Fatal("first entry draw count diverged")
			}
			record, err := db.PlayerToRecord(s.player, nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := db.RecordToPlayer(record, m.world); err != nil {
				t.Fatal(err)
			}
			if dprng.Next() != expected.Next() {
				t.Fatal("loading a saved character consumed RNG draws")
			}
		})
	}
}
