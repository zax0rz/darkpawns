// Package db handles database persistence.
package db

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// DB wraps the database connection.
type DB struct {
	conn *sql.DB
}

// SQLDB returns the underlying *sql.DB for use by other packages (e.g., moderation).
func (db *DB) SQLDB() *sql.DB {
	return db.conn
}

// PlayerRecord represents a player in the database.
type PlayerRecord struct {
	ID                  int
	Name                string
	Password            string // hashed
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
	Inventory           []byte // JSONB encoded inventory
	Equipment           []byte // JSONB encoded equipment
	FailedLoginAttempts int
	LockedUntil         *time.Time
}

// New creates a new database connection.
func New(connString string) (*DB, error) {
	conn, err := sql.Open("postgres", connString)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	// Configure connection pool to avoid exhausting pg max_connections under load.
	// Values mirror deployment guidance; override via env vars if needed.
	conn.SetMaxOpenConns(getEnvInt("DB_MAX_OPEN_CONNS", 25))
	conn.SetMaxIdleConns(getEnvInt("DB_MAX_IDLE_CONNS", 5))
	conn.SetConnMaxLifetime(time.Duration(getEnvInt("DB_CONN_MAX_LIFETIME_SECONDS", 300)) * time.Second)

	db := &DB{conn: conn}
	if err := db.createTables(); err != nil {
		return nil, fmt.Errorf("create tables: %w", err)
	}
	if err := db.InitNarrativeMemory(); err != nil {
		return nil, fmt.Errorf("init narrative memory: %w", err)
	}

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

// createTables creates the necessary tables if they don't exist.
func (db *DB) createTables() error {
	query := `
	CREATE TABLE IF NOT EXISTS players (
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
		inventory JSONB DEFAULT '[]',
		equipment JSONB DEFAULT '{}',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	-- Add new columns to existing installs
	ALTER TABLE players ADD COLUMN IF NOT EXISTS strength INTEGER DEFAULT 10;
	ALTER TABLE players ADD COLUMN IF NOT EXISTS class INTEGER DEFAULT 3;
	ALTER TABLE players ADD COLUMN IF NOT EXISTS race INTEGER DEFAULT 0;
	ALTER TABLE players ADD COLUMN IF NOT EXISTS stat_str INTEGER DEFAULT 10;
	ALTER TABLE players ADD COLUMN IF NOT EXISTS stat_str_add INTEGER DEFAULT 0;
	ALTER TABLE players ADD COLUMN IF NOT EXISTS stat_int INTEGER DEFAULT 10;
	ALTER TABLE players ADD COLUMN IF NOT EXISTS stat_wis INTEGER DEFAULT 10;
	ALTER TABLE players ADD COLUMN IF NOT EXISTS stat_dex INTEGER DEFAULT 10;
	ALTER TABLE players ADD COLUMN IF NOT EXISTS stat_con INTEGER DEFAULT 10;
	ALTER TABLE players ADD COLUMN IF NOT EXISTS stat_cha INTEGER DEFAULT 10;
	ALTER TABLE players ADD COLUMN IF NOT EXISTS inventory JSONB DEFAULT '[]';
	ALTER TABLE players ADD COLUMN IF NOT EXISTS equipment JSONB DEFAULT '{}';
	ALTER TABLE players ADD COLUMN IF NOT EXISTS move INTEGER DEFAULT 100;
	ALTER TABLE players ADD COLUMN IF NOT EXISTS max_move INTEGER DEFAULT 100;
	ALTER TABLE players ADD COLUMN IF NOT EXISTS hunger INTEGER DEFAULT 24;
	ALTER TABLE players ADD COLUMN IF NOT EXISTS thirst INTEGER DEFAULT 24;
	ALTER TABLE players ADD COLUMN IF NOT EXISTS drunk INTEGER DEFAULT 0;
	ALTER TABLE players ADD COLUMN IF NOT EXISTS is_admin BOOLEAN DEFAULT false;
	ALTER TABLE players ADD COLUMN IF NOT EXISTS hometown INTEGER DEFAULT 0;
	ALTER TABLE players ADD COLUMN IF NOT EXISTS failed_login_attempts INTEGER DEFAULT 0;
	ALTER TABLE players ADD COLUMN IF NOT EXISTS locked_until TIMESTAMP;

	CREATE INDEX IF NOT EXISTS idx_players_name ON players(name);
	CREATE INDEX IF NOT EXISTS idx_players_locked_until ON players(locked_until);

	CREATE TABLE IF NOT EXISTS agent_keys (
		id             SERIAL PRIMARY KEY,
		character_name VARCHAR(64) NOT NULL,
		key_hash       VARCHAR(64) NOT NULL UNIQUE,
		created_at     TIMESTAMP NOT NULL DEFAULT NOW(),
		revoked        BOOLEAN NOT NULL DEFAULT FALSE
	);
	`

	_, err := db.conn.Exec(query)
	if err != nil {
		return err
	}

	// Decision capture tables (DP-213)
	if err := db.createDecisionLogTables(); err != nil {
		return fmt.Errorf("create decision log tables: %w", err)
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

	err = db.conn.QueryRow(
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

	err := db.conn.QueryRow(
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
		       COALESCE(failed_login_attempts, 0), locked_until
		FROM players WHERE name = $1
	`
	var p PlayerRecord
	var lockedUntil sql.NullTime
	err := db.conn.QueryRow(query, name).Scan(
		&p.ID, &p.Name, &p.Password, &p.RoomVNum, &p.Level, &p.Exp,
		&p.Health, &p.MaxHealth, &p.Mana, &p.MaxMana, &p.Move, &p.MaxMove, &p.Strength,
		&p.Class, &p.Race, &p.StatStr, &p.StatStrAdd, &p.StatInt, &p.StatWis, &p.StatDex, &p.StatCon, &p.StatCha,
		&p.Hunger, &p.Thirst, &p.Drunk, &p.Hometown,
		&p.Inventory, &p.Equipment,
		&p.FailedLoginAttempts, &lockedUntil,
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
	return &p, nil
}

// CreatePlayer inserts a new player record.
func (db *DB) CreatePlayer(p *PlayerRecord) error {
	query := `
		INSERT INTO players
		  (name, password_hash, room_vnum, level, exp, health, max_health, mana, max_mana, move, max_move, strength,
		   class, race, stat_str, stat_str_add, stat_int, stat_wis, stat_dex, stat_con, stat_cha,
		   hunger, thirst, drunk, hometown,
		   inventory, equipment)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27)
		RETURNING id
	`
	return db.conn.QueryRow(
		query,
		p.Name, p.Password, p.RoomVNum, p.Level, p.Exp, p.Health, p.MaxHealth, p.Mana, p.MaxMana, p.Move, p.MaxMove, p.Strength,
		p.Class, p.Race, p.StatStr, p.StatStrAdd, p.StatInt, p.StatWis, p.StatDex, p.StatCon, p.StatCha,
		p.Hunger, p.Thirst, p.Drunk, p.Hometown,
		p.Inventory, p.Equipment,
	).Scan(&p.ID)
}

// UpdatePassword updates a player's password hash.
func (db *DB) UpdatePassword(playerID int, hash string) error {
	_, err := db.conn.Exec(`UPDATE players SET password_hash = $1 WHERE id = $2`, hash, playerID)
	return err
}

// GetAccountLockout returns the current failed-login attempt count and any
// active lockout deadline for the named account.
func (db *DB) GetAccountLockout(name string) (int, *time.Time, error) {
	query := `SELECT COALESCE(failed_login_attempts, 0), locked_until FROM players WHERE name = $1`
	var attempts int
	var lockedUntil sql.NullTime
	err := db.conn.QueryRow(query, name).Scan(&attempts, &lockedUntil)
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
	query := `
		UPDATE players
		SET failed_login_attempts = failed_login_attempts + 1,
		    locked_until = CASE
		        WHEN failed_login_attempts + 1 >= $2 THEN NOW() + $3::interval
		        ELSE locked_until
		    END
		WHERE name = $1
		RETURNING failed_login_attempts, locked_until
	`
	var attempts int
	var lockedUntil sql.NullTime
	err := db.conn.QueryRow(query, name, threshold, lockoutDuration.Minutes()).Scan(&attempts, &lockedUntil)
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
	_, err := db.conn.Exec(
		`UPDATE players SET failed_login_attempts = 0, locked_until = NULL WHERE name = $1`,
		name,
	)
	return err
}

// Exec runs a raw SQL query against the database.
// Used for operations not covered by the typed methods.
func (db *DB) Exec(query string, args ...interface{}) (interface{}, error) {
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
		  inventory=$24, equipment=$25, updated_at=CURRENT_TIMESTAMP
		WHERE id=$26
	`
	_, err := db.conn.Exec(
		query,
		p.RoomVNum, p.Level, p.Exp, p.Health, p.MaxHealth,
		p.Mana, p.MaxMana, p.Move, p.MaxMove, p.Strength,
		p.Class, p.Race,
		p.StatStr, p.StatStrAdd, p.StatInt, p.StatWis, p.StatDex, p.StatCon, p.StatCha,
		p.Hunger, p.Thirst, p.Drunk, p.Hometown,
		p.Inventory, p.Equipment, p.ID,
	)
	return err
}
