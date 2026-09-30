// Package db handles database persistence.
package db

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zax0rz/darkpawns/pkg/errlog"

	// modernc.org/sqlite is the pure-Go SQLite driver and the only driver this
	// package opens: the static CGO_ENABLED=0 build in DEPLOYMENT.md depends on
	// it. mattn/go-sqlite3 needed cgo, which is how the store this package
	// replaced rotted: pkg/storage (since deleted) still compiled without cgo,
	// and every call then failed at runtime with "go-sqlite3 requires cgo to
	// work. This is a stub".
	_ "modernc.org/sqlite"
)

// ErrAmbiguousPlayerName refuses legacy case-colliding rows rather than selecting an account.
var ErrAmbiguousPlayerName = errors.New("ambiguous character name; saved records require administrator review")

// DB wraps the database connection.
type DB struct {
	conn *sql.DB
	// created records that this open created the database file, which is what a
	// first boot looks like. A caller can announce it rather than leaving an
	// operator to wonder whether the service opened the database that was
	// migrated or quietly made a new empty one.
	created bool
}

// SQLDB returns the underlying *sql.DB for use by other packages (e.g., moderation).
func (db *DB) SQLDB() *sql.DB {
	return db.conn
}

// Created reports whether opening this database created the file.
func (db *DB) Created() bool {
	return db.created
}

// exec, query and queryRow are the statement choke points for the game store.
// SQLite binds positionally, so arguments are passed in the order their `?`
// markers appear. Statements must be issued one at a time: SQLite rejects
// multi-statement strings, and a single statement reports the failing piece
// precisely.
func (db *DB) exec(query string, args ...interface{}) (sql.Result, error) {
	return db.conn.Exec(query, args...)
}

func (db *DB) query(query string, args ...interface{}) (*sql.Rows, error) {
	return db.conn.Query(query, args...)
}

func (db *DB) queryRow(query string, args ...interface{}) *sql.Row {
	return db.conn.QueryRow(query, args...)
}

func (db *DB) execDDL(stmt string) error {
	_, err := db.conn.Exec(stmt)
	return err
}

// PlayerRecord represents a player in the database.
type PlayerRecord struct {
	ID          int
	Name        string
	Password    string // hashed
	Description string
	Title       string
	RoomVNum    int
	Level       int
	Exp         int
	Health      int
	MaxHealth   int
	Mana        int
	MaxMana     int
	Strength    int
	Class       int
	Race        int
	StatStr     int
	StatStrAdd  int // 18/xx for warriors
	StatInt     int
	StatWis     int
	StatDex     int
	StatCon     int
	StatCha     int
	Hunger      int
	Thirst      int
	Drunk       int
	Move        int
	MaxMove     int
	Hometown    int
	// OlcZone is C's player_special_data_saved.olc_zone. It is a scalar
	// playerfile field, so keep it as a typed column rather than hiding it in
	// the JSON inventory/equipment payloads.
	OlcZone             int
	Inventory           []byte // JSON encoded inventory
	Equipment           []byte // JSON encoded equipment
	CharacterData       []byte // JSON: the rest of the character (game.EncodeCharacterData)
	FailedLoginAttempts int
	LockedUntil         *time.Time
}

// New opens the game store at the SQLite database the setting names, creating the
// file and the schema when it is not there yet.
//
// The setting may be a bare filesystem path or the same path with a sqlite://
// prefix. Anything else is refused by SQLitePath, including a PostgreSQL DSN: a
// unit file left over from the PostgreSQL era must fail loudly rather than be
// ignored, because being ignored would boot the server against a fresh empty
// SQLite file and look healthy while holding no characters.
func New(connString string) (*DB, error) {
	path, err := SQLitePath(connString)
	if err != nil {
		return nil, err
	}
	dir := filepath.Dir(path)
	if dir != "." {
		info, err := os.Stat(dir)
		switch {
		case errors.Is(err, os.ErrNotExist):
			return nil, fmt.Errorf("database directory %s does not exist", dir)
		case err != nil:
			return nil, fmt.Errorf("inspect database directory %s: %w", dir, err)
		case !info.IsDir():
			return nil, fmt.Errorf("database directory %s is not a directory", dir)
		}
	}

	_, statErr := os.Stat(path)
	created := errors.Is(statErr, os.ErrNotExist)
	if statErr != nil && !created {
		return nil, fmt.Errorf("inspect database file %s: %w", path, statErr)
	}
	if created {
		// Create it here rather than letting the driver report "unable to open
		// database file": a permission problem then names the file and the mode,
		// and a new database is owner-only from the start, exactly like the
		// migrated production file (0600).
		file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
		if err != nil {
			return nil, fmt.Errorf("create database file %s: %w", path, err)
		}
		if err := file.Close(); err != nil {
			return nil, fmt.Errorf("close new database file %s: %w", path, err)
		}
	}

	conn, err := sql.Open(DriverName, path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Ensure the opened handle is closed on every init failure path so that
	// repeated failed startups do not leak connection pools and underlying
	// database resources (DP-812).
	success := false
	defer func() {
		if !success {
			if cerr := conn.Close(); cerr != nil {
				slog.Error("failed to close db connection after init failure", "error", cerr)
			}
		}
	}()

	// Single connection plus WAL: SQLite allows one writer at a time, and there
	// are no explicit transactions in this package to serialize. busy_timeout
	// keeps event-driven writes from failing on a locked file.
	conn.SetMaxOpenConns(1)
	for _, pragma := range []string{
		"PRAGMA busy_timeout = 5000",
		"PRAGMA journal_mode = WAL",
	} {
		if _, err := conn.Exec(pragma); err != nil {
			return nil, fmt.Errorf("set sqlite pragma %q: %w", pragma, err)
		}
	}

	db := &DB{conn: conn, created: created}
	if err := db.createTables(); err != nil {
		return nil, fmt.Errorf("create tables: %w", err)
	}
	success = true
	return db, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}

// playersMigrationColumns are the columns added to existing installs since the
// original players schema. Applied idempotently per boot through
// AddColumnIfNotExists, which looks each column up in pragma_table_info first
// because SQLite's ALTER TABLE has no IF NOT EXISTS.
var playersMigrationColumns = []string{
	"strength INTEGER DEFAULT 10",
	"class INTEGER DEFAULT 3",
	"race INTEGER DEFAULT 0",
	"stat_str INTEGER DEFAULT 10",
	"stat_str_add INTEGER DEFAULT 0",
	"stat_int INTEGER DEFAULT 10",
	"stat_wis INTEGER DEFAULT 10",
	"stat_dex INTEGER DEFAULT 10",
	"stat_con INTEGER DEFAULT 10",
	"stat_cha INTEGER DEFAULT 10",
	"inventory JSON DEFAULT '[]'",
	"equipment JSON DEFAULT '{}'",
	"move INTEGER DEFAULT 100",
	"max_move INTEGER DEFAULT 100",
	"hunger INTEGER DEFAULT 24",
	"thirst INTEGER DEFAULT 24",
	"drunk INTEGER DEFAULT 0",
	"is_admin BOOLEAN DEFAULT false",
	"hometown INTEGER DEFAULT 0",
	"olc_zone INTEGER DEFAULT 0",
	"failed_login_attempts INTEGER DEFAULT 0",
	"locked_until TIMESTAMP",
	"description TEXT DEFAULT ''",
	"title VARCHAR(80) DEFAULT ''",
	// Everything else a character is (C's char_file_u beyond the columns
	// above), as game.EncodeCharacterData writes it. JSON, not JSONB, so the
	// record reads back byte for byte.
	"character_data JSON DEFAULT '{}'",
}

// createTables creates the game-store tables if they don't exist. It is the one
// definition of the game store's schema: a fresh install, a restart and the
// conversion bridge's destination all get their tables from here.
func (db *DB) createTables() error {
	// Statements are issued one at a time: SQLite rejects multi-statement
	// strings, and a single failing statement then names itself in the error.
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS players (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name VARCHAR(32) UNIQUE NOT NULL,
			password_hash VARCHAR(255),
			room_vnum INTEGER DEFAULT 8004,
			level INTEGER DEFAULT 1,
			exp INTEGER DEFAULT 1,
			health INTEGER DEFAULT 10,
			max_health INTEGER DEFAULT 10,
			mana INTEGER DEFAULT 100,
			max_mana INTEGER DEFAULT 100,
			move INTEGER DEFAULT 100,
			max_move INTEGER DEFAULT 100,
			strength INTEGER DEFAULT 10,
			class INTEGER DEFAULT 3,
			race INTEGER DEFAULT 0,
			stat_str INTEGER DEFAULT 10,
			stat_int INTEGER DEFAULT 10,
			stat_wis INTEGER DEFAULT 10,
			stat_dex INTEGER DEFAULT 10,
			stat_con INTEGER DEFAULT 10,
			stat_cha INTEGER DEFAULT 10,
			hunger INTEGER DEFAULT 24,
			thirst INTEGER DEFAULT 24,
			drunk INTEGER DEFAULT 0,
			hometown INTEGER DEFAULT 0,
			olc_zone INTEGER DEFAULT 0,
			inventory JSON DEFAULT '[]',
			equipment JSON DEFAULT '{}',
			description TEXT DEFAULT '',
			title VARCHAR(80) DEFAULT '',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		// C find_name uses str_cmp: character identity is case-insensitive.
		// Existing colliding rows must be resolved explicitly before this migration.
		`CREATE UNIQUE INDEX IF NOT EXISTS players_name_folded_key ON players (lower(name))`,
		`CREATE INDEX IF NOT EXISTS idx_players_name ON players(name)`,
		`CREATE INDEX IF NOT EXISTS idx_players_locked_until ON players(locked_until)`,
	}

	for _, stmt := range stmts[:1] {
		if err := db.execDDL(stmt); err != nil {
			return err
		}
	}

	// Add new columns to existing installs before the indexes that reference
	// them: locked_until and friends arrive via these migrations, and an index
	// built on a not-yet-added column fails on both dialects.
	for _, col := range playersMigrationColumns {
		if err := db.addColumnIfNotExists("players", col); err != nil {
			return fmt.Errorf("add players column %s: %w", strings.Fields(col)[0], err)
		}
	}

	for _, stmt := range stmts[1:] {
		if err := db.execDDL(stmt); err != nil {
			return err
		}
	}

	return nil
}

// addColumnIfNotExists applies one migration column to the players table
// idempotently. Verified idempotent across repeat runs.
func (db *DB) addColumnIfNotExists(table, columnDef string) error {
	return AddColumnIfNotExists(db.conn, table, columnDef)
}

// GetPlayer retrieves a player by name. Returns nil, nil if not found.
func (db *DB) GetPlayer(name string) (*PlayerRecord, error) {
	query := `
		SELECT id, name, COALESCE(password_hash,''), room_vnum, level, exp,
		       health, max_health, mana, max_mana, move, max_move, strength,
		       class, race, stat_str, stat_str_add, stat_int, stat_wis, stat_dex, stat_con, stat_cha,
		       hunger, thirst, drunk, hometown, COALESCE(olc_zone, 0),
		       inventory, equipment, COALESCE(character_data, '{}'),
		       COALESCE(failed_login_attempts, 0), locked_until, COALESCE(description, ''), COALESCE(title, ''), COUNT(*) OVER ()
		FROM players WHERE lower(name) = lower(?)
	`
	var p PlayerRecord
	var matches int
	var lockedUntil sql.NullTime
	err := db.queryRow(query, name).Scan(
		&p.ID, &p.Name, &p.Password, &p.RoomVNum, &p.Level, &p.Exp,
		&p.Health, &p.MaxHealth, &p.Mana, &p.MaxMana, &p.Move, &p.MaxMove, &p.Strength,
		&p.Class, &p.Race, &p.StatStr, &p.StatStrAdd, &p.StatInt, &p.StatWis, &p.StatDex, &p.StatCon, &p.StatCha,
		&p.Hunger, &p.Thirst, &p.Drunk, &p.Hometown, &p.OlcZone,
		&p.Inventory, &p.Equipment, &p.CharacterData,
		&p.FailedLoginAttempts, &lockedUntil, &p.Description, &p.Title, &matches,
	)
	if lockedUntil.Valid {
		p.LockedUntil = &lockedUntil.Time
	}
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if matches != 1 {
		return nil, ErrAmbiguousPlayerName
	}
	return &p, nil
}

// ListPlayerNames returns all registered player names sorted alphabetically.
// Source: C player_table iteration in do_gen_ps SCMD_PLAYER_LIST.
func (db *DB) ListPlayerNames() ([]string, error) {
	rows, err := db.query(`SELECT name FROM players ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer errlog.Close(rows, "close player-name listing rows")
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

// CountPlayers returns the number of persisted player rows. Used by the
// first-player-is-God bootstrap (init_char, db.c:3016) to detect a fresh MUD:
// when the player store is empty, the first character created is crowned God.
func (db *DB) CountPlayers() (int, error) {
	var n int
	if err := db.queryRow(`SELECT COUNT(*) FROM players`).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

// CreatePlayer inserts a new player record.
func (db *DB) CreatePlayer(p *PlayerRecord) error {
	query := `
		INSERT INTO players
		  (name, password_hash, room_vnum, level, exp, health, max_health, mana, max_mana, move, max_move, strength,
		   class, race, stat_str, stat_str_add, stat_int, stat_wis, stat_dex, stat_con, stat_cha,
		   hunger, thirst, drunk, hometown,
		   olc_zone, inventory, equipment, description, title, character_data)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		RETURNING id
	`
	return db.queryRow(
		query,
		p.Name, p.Password, p.RoomVNum, p.Level, p.Exp, p.Health, p.MaxHealth, p.Mana, p.MaxMana, p.Move, p.MaxMove, p.Strength,
		p.Class, p.Race, p.StatStr, p.StatStrAdd, p.StatInt, p.StatWis, p.StatDex, p.StatCon, p.StatCha,
		p.Hunger, p.Thirst, p.Drunk, p.Hometown,
		p.OlcZone, p.Inventory, p.Equipment, p.Description, p.Title, characterDataOrEmpty(p.CharacterData),
	).Scan(&p.ID)
}

// UpdatePassword updates a player's password hash.
func (db *DB) UpdatePassword(playerID int, hash string) error {
	_, err := db.exec(`UPDATE players SET password_hash = ? WHERE id = ?`, hash, playerID)
	return err
}

// UpdateDescription updates the description displayed when another player looks at this character.
func (db *DB) UpdateDescription(playerID int, description string) error {
	_, err := db.exec(`UPDATE players SET description = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, description, playerID)
	return err
}

// DeletePlayer permanently removes a player record.
func (db *DB) DeletePlayer(playerID int) error {
	_, err := db.exec(`DELETE FROM players WHERE id = ?`, playerID)
	return err
}

// GetAccountLockout returns the current failed-login attempt count and any
// active lockout deadline for the named account.
func (db *DB) GetAccountLockout(name string) (int, *time.Time, error) {
	query := `SELECT COALESCE(failed_login_attempts, 0), locked_until FROM players WHERE lower(name) = lower(?)`
	var attempts int
	var lockedUntil sql.NullTime
	err := db.queryRow(query, name).Scan(&attempts, &lockedUntil)
	if err == sql.ErrNoRows {
		return 0, nil, nil
	}
	if err != nil {
		return 0, nil, err
	}
	if lockedUntil.Valid {
		return attempts, &lockedUntil.Time, nil
	}
	return attempts, nil, nil
}

// RecordLoginFailure increments the failed-login counter for a player and,
// if the threshold is reached, sets locked_until to lockoutDuration from now.
// It returns true when this failure caused the account to become locked.
func (db *DB) RecordLoginFailure(name string, threshold int, lockoutDuration time.Duration) (bool, error) {
	// The lockout deadline is computed in Go rather than in SQL: the
	// PostgreSQL spelling (NOW() + ?::interval) has no SQLite equivalent,
	// and one statement that runs on both dialects beats two. Placeholders
	// are numbered in order of appearance because SQLite binds positionally.
	lockoutUntil := time.Now().Add(lockoutDuration)
	query := `
		UPDATE players
		SET failed_login_attempts = failed_login_attempts + 1,
		    locked_until = CASE
		        WHEN failed_login_attempts + 1 >= ? THEN ?
		        ELSE locked_until
		    END
		WHERE lower(name) = lower(?)
		RETURNING failed_login_attempts, locked_until
	`
	var attempts int
	var lockedUntil sql.NullTime
	err := db.queryRow(query, threshold, lockoutUntil, name).Scan(&attempts, &lockedUntil)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return lockedUntil.Valid && attempts >= threshold, nil
}

// RecordLoginSuccess clears failed-login state for a player.
func (db *DB) RecordLoginSuccess(name string) error {
	_, err := db.exec(
		`UPDATE players SET failed_login_attempts = 0, locked_until = NULL WHERE lower(name) = lower(?)`,
		name,
	)
	return err
}

// Exec runs a raw SQL query against the database.
// Used for operations not covered by the typed methods. The query is passed
// through exactly as given: callers own the dialect syntax, just like before
// the SQLite support landed.
func (db *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return db.conn.Exec(query, args...)
}

// SavePlayer persists a player's current state.
func playerUpdate(p *PlayerRecord) (string, []any) {
	query := `
		UPDATE players SET
		  room_vnum=?, level=?, exp=?, health=?, max_health=?,
		  mana=?, max_mana=?, move=?, max_move=?, strength=?,
		  class=?, race=?,
		  stat_str=?, stat_str_add=?, stat_int=?, stat_wis=?, stat_dex=?, stat_con=?, stat_cha=?,
		  hunger=?, thirst=?, drunk=?, hometown=?, olc_zone=?,
		  inventory=?, equipment=?, description=?, title=?, character_data=?, updated_at=CURRENT_TIMESTAMP
		WHERE id=?
	`
	args := []any{
		p.RoomVNum, p.Level, p.Exp, p.Health, p.MaxHealth,
		p.Mana, p.MaxMana, p.Move, p.MaxMove, p.Strength,
		p.Class, p.Race,
		p.StatStr, p.StatStrAdd, p.StatInt, p.StatWis, p.StatDex, p.StatCon, p.StatCha,
		p.Hunger, p.Thirst, p.Drunk, p.Hometown, p.OlcZone,
		p.Inventory, p.Equipment, p.Description, p.Title, characterDataOrEmpty(p.CharacterData), p.ID,
	}
	return query, args
}

func (db *DB) SavePlayer(p *PlayerRecord) error {
	query, args := playerUpdate(p)
	_, err := db.exec(query, args...)
	return err
}

// ErrPlayerRecordChanged prevents an offline editor from overwriting a save
// made after its read. No timestamp or new save-format field is required.
var ErrPlayerRecordChanged = errors.New("player record changed during offline edit")

// SavePlayerIfCurrent performs an atomic compare-and-save against every
// field SavePlayer replaces. Session-owned OLC and all other untouched values
// are retained by the caller's conversion, and a concurrent newer save wins.
func SavePlayerIfCurrent(store GameStore, updated, original *PlayerRecord) error {
	query, args := playerUpdate(updated)
	_, oldArgs := playerUpdate(original)
	if updated.Name != original.Name {
		// C set's name field updates the player index (act.wizard.c:2889-2895).
		query = strings.Replace(query, "UPDATE players SET", "UPDATE players SET name=?,", 1)
		args = append([]any{updated.Name}, args...)
	}
	query += " AND name IS ?"
	args = append(args, original.Name)
	columns := []string{"room_vnum", "level", "exp", "health", "max_health", "mana", "max_mana", "move", "max_move", "strength", "class", "race", "stat_str", "stat_str_add", "stat_int", "stat_wis", "stat_dex", "stat_con", "stat_cha", "hunger", "thirst", "drunk", "hometown", "olc_zone", "inventory", "equipment", "description", "title", "character_data"}
	for i, column := range columns {
		expression := column
		switch column {
		case "olc_zone":
			expression = "COALESCE(olc_zone,0)"
		case "description", "title":
			expression = "COALESCE(" + column + ",'')"
		case "character_data":
			expression = "COALESCE(character_data,'{}')"
		}
		query += " AND " + expression + " IS ?"
		args = append(args, oldArgs[i])
	}
	result, err := store.Exec(query, args...)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count != 1 {
		return ErrPlayerRecordChanged
	}
	return nil
}

// characterDataOrEmpty stores an absent record as the column's default.
func characterDataOrEmpty(data []byte) []byte {
	if len(data) == 0 {
		return []byte("{}")
	}
	return data
}
