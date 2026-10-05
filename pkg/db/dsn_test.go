package db

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// The runtime database setting, and what New does with it. The point of these
// tests is that a misconfiguration fails loudly: the failure mode that matters is
// a server that boots happily against a fresh empty SQLite file because the
// setting it was given was ignored.

func TestSQLitePathResolvesAPath(t *testing.T) {
	accepted := map[string]string{
		"/var/lib/darkpawns/darkpawns.db":     "/var/lib/darkpawns/darkpawns.db",
		"sqlite:///var/lib/darkpawns/dark.db": "/var/lib/darkpawns/dark.db",
		"sqlite:/var/lib/darkpawns/dark.db":   "/var/lib/darkpawns/dark.db",
		"data/darkpawns.db":                   "data/darkpawns.db",
		"  /tmp/dark.db  ":                    "/tmp/dark.db",
		"/tmp/../tmp/dark.db":                 "/tmp/dark.db",
	}
	for setting, want := range accepted {
		got, err := SQLitePath(setting)
		if err != nil {
			t.Errorf("SQLitePath(%q): %v", setting, err)
			continue
		}
		if got != want {
			t.Errorf("SQLitePath(%q) = %q, want %q", setting, got, want)
		}
	}
}

func TestSQLitePathRefusesPostgres(t *testing.T) {
	for _, setting := range []string{
		"postgres://darkpawns:secret@localhost:5432/darkpawns?sslmode=disable",
		"postgresql:///darkpawns?host=/var/run/postgresql",
		"POSTGRES://HOST/DB",
	} {
		_, err := SQLitePath(setting)
		if err == nil {
			t.Fatalf("SQLitePath(%q) accepted a PostgreSQL DSN", setting)
		}
		if !errors.Is(err, ErrPostgresDSN) {
			t.Errorf("SQLitePath(%q) = %v, want ErrPostgresDSN", setting, err)
		}
		if !strings.Contains(err.Error(), "tools/db-migrate") {
			t.Errorf("refusal does not say what to do about it: %v", err)
		}
		if strings.Contains(err.Error(), "secret") {
			t.Errorf("refusal leaks the password it was given: %v", err)
		}
	}
}

func TestSQLitePathRefusesAnEmptyOrInMemorySetting(t *testing.T) {
	for _, setting := range []string{"", "   ", "sqlite://", ":memory:", "file::memory:?cache=shared"} {
		if _, err := SQLitePath(setting); err == nil {
			t.Errorf("SQLitePath(%q) was accepted", setting)
		}
	}
}

func TestNewRefusesAPostgresDSNWithoutCreatingAnything(t *testing.T) {
	dir := t.TempDir()
	// A DSN whose path component would be a plausible file if it were taken
	// literally, so a bug that strips the scheme and opens it would be visible.
	setting := "postgres://user:pw@localhost:5432/" + filepath.Join(dir, "darkpawns.db")
	database, err := New(setting)
	if err == nil {
		_ = database.Close()
		t.Fatal("New accepted a PostgreSQL DSN")
	}
	if !errors.Is(err, ErrPostgresDSN) {
		t.Errorf("New(%q) = %v, want ErrPostgresDSN", setting, err)
	}
	entries, readErr := os.ReadDir(dir)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Errorf("the refused configuration created files: %v", entries)
	}
}

func TestNewRefusesAnEmptySetting(t *testing.T) {
	if database, err := New(""); err == nil {
		_ = database.Close()
		t.Fatal("New(\"\") silently opened a database; an unconfigured store must fail")
	}
}

func TestNewRefusesAMissingParentDirectory(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "absent", "darkpawns.db")
	_, err := New(missing)
	if err == nil {
		t.Fatal("New created a database under a directory that does not exist")
	}
	if !strings.Contains(err.Error(), filepath.Dir(missing)) {
		t.Errorf("error does not name the missing directory: %v", err)
	}
}

func TestNewRefusesAnUnwritableDirectory(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: a directory mode cannot deny the write this checks")
	}
	dir := filepath.Join(t.TempDir(), "readonly")
	if err := os.Mkdir(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	_, err := New(filepath.Join(dir, "darkpawns.db"))
	if err == nil {
		t.Fatal("New created a database in a directory it cannot write")
	}
	if !strings.Contains(err.Error(), "create database file") {
		t.Errorf("error does not say the file could not be created: %v", err)
	}
}

func TestNewReportsWhetherItCreatedTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "darkpawns.db")

	fresh := openGameStore(t, path)
	if !fresh.Created() {
		t.Error("a first boot did not report creating the database file")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("the database file was not created: %v", err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("a new database file is mode %o, want 600", mode)
	}
	if err := fresh.Close(); err != nil {
		t.Fatal(err)
	}

	reopened := openGameStore(t, path)
	if reopened.Created() {
		t.Error("reopening an existing database reported creating it")
	}
}

// TestUniqueFoldedNameIsEnforcedByTheDatabase pins the constraint character
// identity depends on: C's find_name compares with str_cmp, so two rows whose
// names differ only in case are one character, and the schema refuses the second.
func TestUniqueFoldedNameIsEnforcedByTheDatabase(t *testing.T) {
	database := openGameStore(t, filepath.Join(t.TempDir(), "darkpawns.db"))
	first := &PlayerRecord{Name: "Zax", Inventory: []byte("[]"), Equipment: []byte("{}")}
	if err := database.CreatePlayer(first); err != nil {
		t.Fatalf("create the first player: %v", err)
	}
	collision := &PlayerRecord{Name: "zAx", Inventory: []byte("[]"), Equipment: []byte("{}")}
	if err := database.CreatePlayer(collision); err == nil {
		t.Fatal("a name differing only in case was accepted")
	}
	if _, err := database.GetPlayer("zax"); err != nil {
		t.Fatalf("case-insensitive lookup failed: %v", err)
	}
}

// TestGameStoreSchemaShape locks the schema a fresh install creates, because the
// migrated production database was created by the same statements (through the
// dialect translator this build no longer has). A change here is a change to the
// shape production runs on, so it has to be deliberate.
func TestGameStoreSchemaShape(t *testing.T) {
	database := openGameStore(t, filepath.Join(t.TempDir(), "darkpawns.db"))

	wantTypes := map[string]string{
		"id":                    "INTEGER",
		"name":                  "VARCHAR(32)",
		"password_hash":         "VARCHAR(255)",
		"room_vnum":             "INTEGER",
		"level":                 "INTEGER",
		"inventory":             "JSON",
		"equipment":             "JSON",
		"character_data":        "JSON",
		"created_at":            "TIMESTAMP",
		"updated_at":            "TIMESTAMP",
		"locked_until":          "TIMESTAMP",
		"is_admin":              "BOOLEAN",
		"failed_login_attempts": "INTEGER",
		"stat_str_add":          "INTEGER",
		"olc_zone":              "INTEGER",
		"last_logon":            "INTEGER",
	}
	rows, err := database.conn.Query(`SELECT name, type FROM pragma_table_info('players')`)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = rows.Close() }()
	got := map[string]string{}
	for rows.Next() {
		var name, declared string
		if err := rows.Scan(&name, &declared); err != nil {
			t.Fatal(err)
		}
		got[name] = declared
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	for column, want := range wantTypes {
		if got[column] != want {
			t.Errorf("players.%s is declared %q, want %q", column, got[column], want)
		}
	}
	if len(got) != 38 {
		t.Errorf("players has %d columns, want 38", len(got))
	}

	// The generated key is an AUTOINCREMENT column, not a bare INTEGER primary
	// key: only AUTOINCREMENT keeps a sequence above the highest id ever
	// inserted, which is what makes a deleted character's id unreusable.
	var ddl string
	if err := database.conn.QueryRow(
		`SELECT sql FROM sqlite_master WHERE type='table' AND name='players'`).Scan(&ddl); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ddl, "INTEGER PRIMARY KEY AUTOINCREMENT") {
		t.Errorf("players id is not AUTOINCREMENT:\n%s", ddl)
	}
	if strings.Contains(ddl, "SERIAL") || strings.Contains(ddl, "TIMESTAMPTZ") {
		t.Errorf("players DDL still carries a PostgreSQL type name:\n%s", ddl)
	}
}

// TestConcurrentSavesSerialise exercises the single-writer contract: the store
// opens one connection and sets a busy timeout, so concurrent saves queue instead
// of failing with "database is locked".
func TestConcurrentSavesSerialise(t *testing.T) {
	database := openGameStore(t, filepath.Join(t.TempDir(), "darkpawns.db"))
	player := &PlayerRecord{Name: "Writers", Inventory: []byte("[]"), Equipment: []byte("{}"), Level: 1}
	if err := database.CreatePlayer(player); err != nil {
		t.Fatalf("create player: %v", err)
	}

	const writers = 8
	const writesEach = 12
	var wg sync.WaitGroup
	errs := make(chan error, writers)
	for writer := 0; writer < writers; writer++ {
		wg.Add(1)
		go func(offset int) {
			defer wg.Done()
			for i := 0; i < writesEach; i++ {
				record, err := database.GetPlayer("Writers")
				if err != nil {
					errs <- err
					return
				}
				if record == nil {
					errs <- errors.New("player vanished during concurrent writes")
					return
				}
				record.Level = 1 + ((offset + i) % 40)
				if err := database.SavePlayer(record); err != nil {
					errs <- err
					return
				}
			}
		}(writer)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent save failed: %v", err)
	}

	final, err := database.GetPlayer("Writers")
	if err != nil || final == nil {
		t.Fatalf("read back after concurrent saves: %v", err)
	}
	if final.Level < 1 || final.Level > 40 {
		t.Errorf("level after concurrent saves = %d, want a value one writer set", final.Level)
	}
}

// TestReopenPersistsCharacters is the restart contract at the store level: a
// character written, a close and a reopen, and the character is still there with
// its id, so the game after a restart is the game before it.
func TestReopenPersistsCharacters(t *testing.T) {
	path := filepath.Join(t.TempDir(), "darkpawns.db")

	first := openGameStore(t, path)
	player := &PlayerRecord{Name: "Persisted", Inventory: []byte("[]"), Equipment: []byte("{}"), Level: 7, Exp: 4242}
	if err := first.CreatePlayer(player); err != nil {
		t.Fatalf("create player: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}

	second := openGameStore(t, path)
	reloaded, err := second.GetPlayer("Persisted")
	if err != nil || reloaded == nil {
		t.Fatalf("reload after reopen: %v", err)
	}
	if reloaded.ID != player.ID || reloaded.Level != 7 || reloaded.Exp != 4242 {
		t.Errorf("reloaded = id %d level %d exp %d, want id %d level 7 exp 4242",
			reloaded.ID, reloaded.Level, reloaded.Exp, player.ID)
	}

	// The next id is above the existing one, so a restart cannot reuse an id.
	third := &PlayerRecord{Name: "AfterRestart", Inventory: []byte("[]"), Equipment: []byte("{}")}
	if err := second.CreatePlayer(third); err != nil {
		t.Fatalf("create player after reopen: %v", err)
	}
	if third.ID <= player.ID {
		t.Errorf("id after reopen = %d, want above %d", third.ID, player.ID)
	}
}

// TestSQLitePathRefusesUriSettingsOtherThanSQLite pins the rule that a URI is not
// a filename. Every one of these used to be taken literally, so a wrong DSN
// produced a file named after it in the working directory ("mysql:/host/db")
// and a server that booted happily against an empty database.
func TestSQLitePathRefusesUriSettingsOtherThanSQLite(t *testing.T) {
	cases := map[string]string{
		"mysql://user:pw@localhost:3306/darkpawns": "mysql",
		"https://example.com/darkpawns.db":         "https",
		"redis://localhost:6379/0":                 "redis",
		"sqlite+foo://localhost/darkpawns.db":      "sqlite+foo",
		"foo:bar":                                  "foo",
		"file:/var/lib/darkpawns/darkpawns.db":     "file",
		"FTP://host/darkpawns.db":                  "FTP",
	}
	for setting, scheme := range cases {
		path, err := SQLitePath(setting)
		if err == nil {
			t.Errorf("SQLitePath(%q) = %q, want a refusal naming scheme %q", setting, path, scheme)
			continue
		}
		if path != "" {
			t.Errorf("SQLitePath(%q) returned %q alongside an error", setting, path)
		}
		if !errors.Is(err, ErrUnsupportedScheme) {
			t.Errorf("SQLitePath(%q) = %v, want ErrUnsupportedScheme", setting, err)
		}
		if !strings.Contains(err.Error(), scheme) {
			t.Errorf("refusal for %q does not name the scheme %q: %v", setting, scheme, err)
		}
		if strings.Contains(err.Error(), "pw") {
			t.Errorf("refusal for %q leaks the credentials it was handed: %v", setting, err)
		}
		// A refusal must still say what is supported, or an operator's next move
		// is a guess.
		if !strings.Contains(err.Error(), "sqlite://") {
			t.Errorf("refusal for %q does not say what is supported: %v", setting, err)
		}
	}
}

// TestUriSettingsAreRefusedWithoutCreatingFiles is the consequence that matters,
// and the reason the rule exists: New() must not leave a file named after a URI
// anywhere, least of all in the working directory of a server that then looks
// healthy. The test runs each case from a private directory so a relative
// creation cannot hide outside it.
func TestUriSettingsAreRefusedWithoutCreatingFiles(t *testing.T) {
	for _, setting := range []string{
		"mysql://user:pw@localhost:3306/darkpawns",
		"redis://localhost:6379/0",
		"https://example.com/darkpawns.db",
		"sqlite+foo://localhost/darkpawns.db",
		"foo:bar",
		":memory:",
		"postgres://user:pw@localhost:5432/darkpawns",
	} {
		t.Run(setting, func(t *testing.T) {
			dir := t.TempDir()
			t.Chdir(dir)
			database, err := New(setting)
			if err == nil {
				_ = database.Close()
				t.Fatalf("New(%q) was accepted", setting)
			}
			entries, readErr := os.ReadDir(dir)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if len(entries) != 0 {
				t.Errorf("New(%q) left files behind: %v", setting, entries)
			}
		})
	}
}

// TestSQLitePathTreatsASingleLetterSchemeAsADriveLetter documents the one
// deliberate exception: C:/data/darkpawns.db is a Windows path, not an
// unsupported URI.
func TestSQLitePathTreatsASingleLetterSchemeAsADriveLetter(t *testing.T) {
	for _, setting := range []string{"C:/data/darkpawns.db", `C:\\data\\darkpawns.db`} {
		path, err := SQLitePath(setting)
		if err != nil {
			t.Errorf("SQLitePath(%q) was refused: %v", setting, err)
			continue
		}
		if !strings.Contains(path, "data") {
			t.Errorf("SQLitePath(%q) = %q, want the path itself", setting, path)
		}
	}
}
