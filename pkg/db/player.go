// Package db handles database persistence.
package db

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/zax0rz/darkpawns/pkg/errlog"

	_ "github.com/lib/pq"
	// modernc.org/sqlite is the pure-Go SQLite driver: the static
	// CGO_ENABLED=0 build in DEPLOYMENT.md depends on it. mattn/go-sqlite3
	// needed cgo, which is how the store this package replaced rotted:
	// pkg/storage (since deleted) still compiled without cgo, and every call
	// then failed at runtime with "go-sqlite3 requires cgo to work. This is a
	// stub".
	_ "modernc.org/sqlite"
)

// ErrAmbiguousPlayerName refuses legacy case-colliding rows rather than selecting an account.
var ErrAmbiguousPlayerName = errors.New("ambiguous character name; saved records require administrator review")

// DB wraps the database connection.
type DB struct {
	conn    *sql.DB
	dialect Dialect
}

// SQLDB returns the underlying *sql.DB for use by other packages (e.g., moderation).
func (db *DB) SQLDB() *sql.DB {
	return db.conn
}

// Dialect returns the SQL flavour the connection was opened for. A bare *sql.DB
// cannot answer that, and packages sharing this handle (moderation) must not
// guess: guessing is what put "no such function: NOW" in the boot log.
func (db *DB) Dialect() Dialect {
	return db.dialect
}

// exec, query and queryRow are the statement choke points for the game store:
// bind prepares the statement and its arguments for the connection's dialect.
// DDL goes through execDDL instead, which additionally translates the schema.
func (db *DB) exec(query string, args ...interface{}) (sql.Result, error) {
	query, args = db.bind(query, args)
	return db.conn.Exec(query, args...)
}

func (db *DB) query(query string, args ...interface{}) (*sql.Rows, error) {
	query, args = db.bind(query, args)
	return db.conn.Query(query, args...)
}

func (db *DB) queryRow(query string, args ...interface{}) *sql.Row {
	query, args = db.bind(query, args)
	return db.conn.QueryRow(query, args...)
}

// bind prepares a statement for the connection's dialect and returns the
// argument list it must be executed with: repeated placeholders are expanded
// to one marker per argument (see expandRepeatedPlaceholders), then $N markers
// are made positional for SQLite.
func (db *DB) bind(query string, args []interface{}) (string, []interface{}) {
	query, args = expandRepeatedPlaceholders(query, args)
	return db.dialect.Rebind(query), args
}

// execDDL runs one schema statement, translating PostgreSQL-only syntax when
// the connection is SQLite. Statements must be issued one at a time: SQLite
// rejects multi-statement strings, and single statements report the failing
// piece precisely.
func (db *DB) execDDL(stmt string) error {
	_, err := db.conn.Exec(db.dialect.DDL(stmt))
	return err
}

// PlayerRecord represents a player in the database.
type PlayerRecord struct {
	ID                  int
	Name                string
	Password            string // hashed
	Description         string
	Title               string
	RoomVNum            int
	Level               int
	Exp                 int
	Health              int
	MaxHealth           int
	Mana                int
	MaxMana             int
	Strength            int
	Class               int
	Race                int
	StatStr             int
	StatStrAdd          int // 18/xx for warriors
	StatInt             int
	StatWis             int
	StatDex             int
	StatCon             int
	StatCha             int
	Hunger              int
	Thirst              int
	Drunk               int
	Move                int
	MaxMove             int
	Hometown            int
	Inventory           []byte // JSON encoded inventory
	Equipment           []byte // JSON encoded equipment
	FailedLoginAttempts int
	LockedUntil         *time.Time
}

// New creates a new database connection. The DSN scheme selects the dialect:
// postgres:// (or postgresql://) uses PostgreSQL; sqlite://, a bare file path
// or :memory: uses embedded SQLite.
func New(connString string) (*DB, error) {
	dialect, dsn, err := SplitDSN(connString)
	if err != nil {
		return nil, err
	}
	driver := "postgres"
	if dialect == DialectSQLite {
		driver = "sqlite"
	}
	conn, err := sql.Open(driver, dsn)
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

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	if dialect == DialectSQLite {
		// Single connection plus WAL: SQLite allows one writer at a time, and
		// there are no explicit transactions in this package to serialize.
		// busy_timeout keeps event-driven writes from failing on a locked file.
		conn.SetMaxOpenConns(1)
		for _, pragma := range []string{
			"PRAGMA busy_timeout = 5000",
			"PRAGMA journal_mode = WAL",
		} {
			if _, err := conn.Exec(pragma); err != nil {
				return nil, fmt.Errorf("set sqlite pragma %q: %w", pragma, err)
			}
		}
	} else {
		// Configure connection pool to avoid exhausting pg max_connections under load.
		// Values mirror deployment guidance; override via env vars if needed.
		conn.SetMaxOpenConns(getEnvInt("DB_MAX_OPEN_CONNS", 25))
		conn.SetMaxIdleConns(getEnvInt("DB_MAX_IDLE_CONNS", 5))
		conn.SetConnMaxLifetime(time.Duration(getEnvInt("DB_CONN_MAX_LIFETIME_SECONDS", 300)) * time.Second)
	}

	db := &DB{conn: conn, dialect: dialect}
	if err := db.createTables(); err != nil {
		return nil, fmt.Errorf("create tables: %w", err)
	}
	if err := db.InitNarrativeMemory(); err != nil {
		return nil, fmt.Errorf("init narrative memory: %w", err)
	}

	// Decision capture tables (DP-213). These are PostgreSQL-only (PARTITION
	// BY RANGE, pg_tables catalog queries, TEXT[]), so SQLite deployments skip
	// them entirely; the ResearchStore interface is never satisfied there.
	if dialect == DialectPostgres {
		if err := db.createDecisionLogTables(); err != nil {
			return nil, fmt.Errorf("create decision log tables: %w", err)
		}
		// Bootstrap the current/next-month partitions. decision_log and
		// combat_log are PARTITION BY RANGE(ts) parents; without a matching
		// partition every INSERT fails ("no partition of relation ... found
		// for row"). Fail loudly here rather than only warning, since decision
		// capture is unusable otherwise.
		if err := db.EnsureDecisionLogPartitions(); err != nil {
			return nil, fmt.Errorf("ensure decision log partitions: %w", err)
		}
	}

	success = true
	return db, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}

// getEnvInt reads an integer environment variable, falling back to defaultValue.
func getEnvInt(name string, defaultValue int) int {
	v := os.Getenv(name)
	if v == "" {
		return defaultValue
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultValue
	}
	return n
}

// playersMigrationColumns are the columns added to existing installs since
// the original players schema. Applied idempotently per boot: PostgreSQL via
// ADD COLUMN IF NOT EXISTS, SQLite via a pragma_table_info guard.
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
	"failed_login_attempts INTEGER DEFAULT 0",
	"locked_until TIMESTAMPTZ",
	"description TEXT DEFAULT ''",
	"title VARCHAR(80) DEFAULT ''",
}

// createTables creates the game-store tables if they don't exist.
func (db *DB) createTables() error {
	// Statements are issued one at a time: SQLite rejects multi-statement
	// strings, and a single failing statement then names itself in the error.
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS players (
			id SERIAL PRIMARY KEY,
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
			inventory JSON DEFAULT '[]',
			equipment JSON DEFAULT '{}',
			description TEXT DEFAULT '',
			title VARCHAR(80) DEFAULT '',
			created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
		)`,
		// C find_name uses str_cmp: character identity is case-insensitive.
		// Existing colliding rows must be resolved explicitly before this migration.
		`CREATE UNIQUE INDEX IF NOT EXISTS players_name_folded_key ON players (lower(name))`,
		`CREATE INDEX IF NOT EXISTS idx_players_name ON players(name)`,
		`CREATE INDEX IF NOT EXISTS idx_players_locked_until ON players(locked_until)`,

		`CREATE TABLE IF NOT EXISTS agent_keys (
			id             SERIAL PRIMARY KEY,
			character_name VARCHAR(64) NOT NULL,
			key_hash       VARCHAR(64) NOT NULL UNIQUE,
			created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			revoked        BOOLEAN NOT NULL DEFAULT FALSE
		)`,
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

	// Installs that predate the TIMESTAMPTZ spelling above keep their naive
	// columns (ADD COLUMN IF NOT EXISTS finds them present and does nothing),
	// so they are converted here, after both tables exist.
	if err := db.migrateNaiveTimestamps(); err != nil {
		return fmt.Errorf("migrate naive timestamps: %w", err)
	}

	// Same for the JSON columns: an install that already has them keeps its
	// JSONB type, so ADD COLUMN IF NOT EXISTS cannot repair them either.
	if err := db.migrateCanonicalizingJSON(); err != nil {
		return fmt.Errorf("migrate JSON columns: %w", err)
	}

	return nil
}

// addColumnIfNotExists applies one migration column to the players table
// idempotently, through the shared guard in dialect.go. Verified idempotent
// across repeat runs on both dialects.
func (db *DB) addColumnIfNotExists(table, columnDef string) error {
	return AddColumnIfNotExists(db.conn, db.dialect, table, columnDef)
}

// gameStoreTimestamptzColumns are the game-store columns authored as naive
// TIMESTAMP before this pass. Fresh installs get TIMESTAMPTZ straight from the
// DDL in createTables, but a database created by an older build still holds
// naive columns, and ADD COLUMN IF NOT EXISTS cannot repair them: the column is
// already there, so the re-authored definition is never applied. They are
// converted in place, once, by convertNaiveTimestamps.
//
// Entries are "table.column" so the list reads as the migration log it is.
// Both halves are package constants, never user input.
var gameStoreTimestamptzColumns = []string{
	"players.locked_until",
	"players.created_at",
	"players.updated_at",
	"agent_keys.created_at",
}

// migrateNaiveTimestamps converts the game store's own timestamp columns on
// PostgreSQL, where a zone-aware type is available and the columns were
// authored naive by every build up to this one.
//
// SQLite skips it entirely. SQLite has no zone-aware type: the DDL translation
// folds TIMESTAMPTZ back to TIMESTAMP there (dialect.go), so there is nothing
// to convert to, and no information_schema to ask either.
func (db *DB) migrateNaiveTimestamps() error {
	if db.dialect != DialectPostgres {
		return nil
	}
	return db.convertNaiveTimestamps(gameStoreTimestamptzColumns)
}

// convertNaiveTimestamps rewrites each table.column from timestamp (without
// time zone) to timestamptz through convertColumns.
//
// The stored value is a wall clock with no zone attached: those columns were
// written from Go time.Time values whose offset PostgreSQL drops on the way
// into a naive column. It is re-read in the database session's zone
// (AT TIME ZONE current_setting('TimeZone')) — the zone it was written in for
// the localhost provisioning DEPLOYMENT.md documents, and a named zone, so a
// summer value keeps its summer offset. The wall clock therefore reads back
// the way it already did in that zone, a lockout still in flight stays in
// force instead of expiring on deploy, and every row written afterwards is an
// exact instant. A row written while the database ran in a different zone is
// read in the current one: the writing zone is not recorded anywhere, so no
// conversion can recover it. See DEPLOYMENT.md.
func (db *DB) convertNaiveTimestamps(columns []string) error {
	return db.convertColumns(columns, "timestamp without time zone", "TIMESTAMPTZ", func(column string) string {
		return column + ` AT TIME ZONE current_setting('TimeZone')`
	})
}

// gameStoreJSONColumns are the game-store columns that were authored JSONB.
// JSONB canonicalizes its input — it re-spaces, re-orders object keys and drops
// duplicate ones — so a record read back did not equal the bytes that were
// written. A fresh install gets json straight from the DDL in createTables, but
// a database created by an older build still holds jsonb, and ADD COLUMN IF NOT
// EXISTS cannot repair it: the column is already there. They are converted in
// place, once, by convertColumns. json stores the exact input text and still
// validates it on write, which is what makes save -> load byte-identical.
var gameStoreJSONColumns = []string{
	"players.inventory",
	"players.equipment",
}

// migrateCanonicalizingJSON converts the JSON columns above on PostgreSQL,
// where the columns were authored canonicalizing and there is an
// information_schema to ask.
//
// SQLite skips it entirely: the DDL translation already stores these columns as
// text, so there is nothing to convert, exactly as migrateNaiveTimestamps skips
// there.
func (db *DB) migrateCanonicalizingJSON() error {
	if db.dialect != DialectPostgres {
		return nil
	}
	return db.convertColumns(gameStoreJSONColumns, "jsonb", "JSON", func(column string) string {
		return column + "::json"
	})
}

// convertColumns rewrites each table.column from fromType to wantType, one
// statement at a time, with the caller supplying the USING expression.
//
// The information_schema guard is correctness, not an optimisation. On an
// already-converted column the USING expression is not a no-op: it re-reads
// the stored value in the session's zone and the ALTER writes that back, so
// without the guard every boot would shift the data by the server's offset
// again. Skipping columns that are already the target type is what makes a
// second boot harmless, and it is also what makes an interrupted conversion
// resumable: the columns already rewritten are skipped and the rest are
// converted on the next boot, because each ALTER TABLE is atomic on its own.
func (db *DB) convertColumns(columns []string, fromType, wantType string, using func(column string) string) error {
	for _, target := range columns {
		table, column, ok := strings.Cut(target, ".")
		if !ok {
			return fmt.Errorf("column target %q is not table.column", target)
		}

		var dataType string
		err := db.queryRow(
			`SELECT data_type FROM information_schema.columns
			  WHERE table_schema = current_schema() AND table_name = $1 AND column_name = $2`,
			table, column,
		).Scan(&dataType)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			// Not in this install's schema: an older build may predate the
			// column entirely, in which case there is nothing to convert.
			continue
		case err != nil:
			return fmt.Errorf("read type of %s: %w", target, err)
		case dataType != fromType:
			// Already converted, or a type this migration does not own.
			continue
		}

		if _, err := db.exec(
			`ALTER TABLE ` + table + ` ALTER COLUMN ` + column +
				` TYPE ` + wantType + ` USING ` + using(column),
		); err != nil {
			return fmt.Errorf("convert %s to %s: %w", target, wantType, err)
		}
		slog.Info("converted game-store column to "+strings.ToLower(wantType), "column", target)
	}
	return nil
}

// CreateAgentKey generates a new agent API key for the given character.
// Returns the raw key (shown once — never stored) and its DB row id.
func (db *DB) CreateAgentKey(characterName string) (rawKey string, id int64, err error) {
	// Generate 32 random bytes → 64 hex chars
	buf := make([]byte, 32)
	if _, err = rand.Read(buf); err != nil {
		return "", 0, fmt.Errorf("generate key: %w", err)
	}
	rawKey = "dp_" + hex.EncodeToString(buf)

	// SHA-256 hash — only the hash is stored
	h := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(h[:])

	err = db.queryRow(
		`INSERT INTO agent_keys (character_name, key_hash) VALUES ($1, $2) RETURNING id`,
		characterName, keyHash,
	).Scan(&id)
	if err != nil {
		return "", 0, fmt.Errorf("insert agent key: %w", err)
	}
	return rawKey, id, nil
}

// ValidateAgentKey hashes rawKey and looks it up in agent_keys.
// Returns the associated character name and row id if the key is valid and not revoked.
func (db *DB) ValidateAgentKey(rawKey string) (characterName string, keyID int64, valid bool) {
	// Reject default/example keys for security
	if rawKey == "br3nd4-69-ag3nt-k3y-d3f4ult" ||
		strings.Contains(rawKey, "example") ||
		strings.Contains(rawKey, "test") ||
		strings.Contains(rawKey, "REPLACE_WITH") {
		return "", 0, false
	}

	h := sha256.Sum256([]byte(rawKey))
	keyHash := hex.EncodeToString(h[:])

	err := db.queryRow(
		`SELECT id, character_name FROM agent_keys WHERE key_hash = $1 AND revoked = FALSE`,
		keyHash,
	).Scan(&keyID, &characterName)
	if err != nil {
		return "", 0, false
	}
	return characterName, keyID, true
}

// GetPlayer retrieves a player by name. Returns nil, nil if not found.
func (db *DB) GetPlayer(name string) (*PlayerRecord, error) {
	query := `
		SELECT id, name, COALESCE(password_hash,''), room_vnum, level, exp,
		       health, max_health, mana, max_mana, move, max_move, strength,
		       class, race, stat_str, stat_str_add, stat_int, stat_wis, stat_dex, stat_con, stat_cha,
		       hunger, thirst, drunk, hometown,
		       inventory, equipment,
		       COALESCE(failed_login_attempts, 0), locked_until, COALESCE(description, ''), COALESCE(title, ''), COUNT(*) OVER ()
		FROM players WHERE lower(name) = lower($1)
	`
	var p PlayerRecord
	var matches int
	var lockedUntil sql.NullTime
	err := db.queryRow(query, name).Scan(
		&p.ID, &p.Name, &p.Password, &p.RoomVNum, &p.Level, &p.Exp,
		&p.Health, &p.MaxHealth, &p.Mana, &p.MaxMana, &p.Move, &p.MaxMove, &p.Strength,
		&p.Class, &p.Race, &p.StatStr, &p.StatStrAdd, &p.StatInt, &p.StatWis, &p.StatDex, &p.StatCon, &p.StatCha,
		&p.Hunger, &p.Thirst, &p.Drunk, &p.Hometown,
		&p.Inventory, &p.Equipment,
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
		   inventory, equipment, description, title)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29)
		RETURNING id
	`
	return db.queryRow(
		query,
		p.Name, p.Password, p.RoomVNum, p.Level, p.Exp, p.Health, p.MaxHealth, p.Mana, p.MaxMana, p.Move, p.MaxMove, p.Strength,
		p.Class, p.Race, p.StatStr, p.StatStrAdd, p.StatInt, p.StatWis, p.StatDex, p.StatCon, p.StatCha,
		p.Hunger, p.Thirst, p.Drunk, p.Hometown,
		p.Inventory, p.Equipment, p.Description, p.Title,
	).Scan(&p.ID)
}

// UpdatePassword updates a player's password hash.
func (db *DB) UpdatePassword(playerID int, hash string) error {
	_, err := db.exec(`UPDATE players SET password_hash = $1 WHERE id = $2`, hash, playerID)
	return err
}

// UpdateDescription updates the description displayed when another player looks at this character.
func (db *DB) UpdateDescription(playerID int, description string) error {
	_, err := db.exec(`UPDATE players SET description = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`, description, playerID)
	return err
}

// DeletePlayer permanently removes a player record.
func (db *DB) DeletePlayer(playerID int) error {
	_, err := db.exec(`DELETE FROM players WHERE id = $1`, playerID)
	return err
}

// GetAccountLockout returns the current failed-login attempt count and any
// active lockout deadline for the named account.
func (db *DB) GetAccountLockout(name string) (int, *time.Time, error) {
	query := `SELECT COALESCE(failed_login_attempts, 0), locked_until FROM players WHERE lower(name) = lower($1)`
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
	// PostgreSQL spelling (NOW() + $3::interval) has no SQLite equivalent,
	// and one statement that runs on both dialects beats two. Placeholders
	// are numbered in order of appearance because SQLite binds positionally.
	lockoutUntil := time.Now().Add(lockoutDuration)
	query := `
		UPDATE players
		SET failed_login_attempts = failed_login_attempts + 1,
		    locked_until = CASE
		        WHEN failed_login_attempts + 1 >= $1 THEN $2
		        ELSE locked_until
		    END
		WHERE lower(name) = lower($3)
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
		`UPDATE players SET failed_login_attempts = 0, locked_until = NULL WHERE lower(name) = lower($1)`,
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
func (db *DB) SavePlayer(p *PlayerRecord) error {
	query := `
		UPDATE players SET
		  room_vnum=$1, level=$2, exp=$3, health=$4, max_health=$5,
		  mana=$6, max_mana=$7, move=$8, max_move=$9, strength=$10,
		  class=$11, race=$12,
		  stat_str=$13, stat_str_add=$14, stat_int=$15, stat_wis=$16, stat_dex=$17, stat_con=$18, stat_cha=$19,
		  hunger=$20, thirst=$21, drunk=$22, hometown=$23,
		  inventory=$24, equipment=$25, description=$26, title=$27, updated_at=CURRENT_TIMESTAMP
		WHERE id=$28
	`
	_, err := db.exec(
		query,
		p.RoomVNum, p.Level, p.Exp, p.Health, p.MaxHealth,
		p.Mana, p.MaxMana, p.Move, p.MaxMove, p.Strength,
		p.Class, p.Race,
		p.StatStr, p.StatStrAdd, p.StatInt, p.StatWis, p.StatDex, p.StatCon, p.StatCha,
		p.Hunger, p.Thirst, p.Drunk, p.Hometown,
		p.Inventory, p.Equipment, p.Description, p.Title, p.ID,
	)
	return err
}
