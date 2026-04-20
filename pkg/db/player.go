// Package db handles database persistence.
package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

// DB wraps the database connection.
type DB struct {
	conn *sql.DB
}

// PlayerRecord represents a player in the database.
type PlayerRecord struct {
	ID        int
	Name      string
	Password  string // hashed
	RoomVNum  int
	Level     int
	Exp       int
	Health    int
	MaxHealth int
	Mana      int
	MaxMana   int
	Strength  int
	Class     int
	Race      int
	StatStr    int
	StatStrAdd int // 18/xx for warriors
	StatInt    int
	StatWis    int
	StatDex    int
	StatCon    int
	StatCha    int
	Inventory  []byte // JSONB encoded inventory
	Equipment  []byte // JSONB encoded equipment
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

	db := &DB{conn: conn}
	if err := db.createTables(); err != nil {
		return nil, fmt.Errorf("create tables: %w", err)
	}

	return db, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
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
		strength INTEGER DEFAULT 10,
		class INTEGER DEFAULT 3,
		race INTEGER DEFAULT 0,
		stat_str INTEGER DEFAULT 10,
		stat_int INTEGER DEFAULT 10,
		stat_wis INTEGER DEFAULT 10,
		stat_dex INTEGER DEFAULT 10,
		stat_con INTEGER DEFAULT 10,
		stat_cha INTEGER DEFAULT 10,
		inventory JSONB DEFAULT '[]',
		equipment JSONB DEFAULT '{}',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	-- Add new columns to existing installs
	ALTER TABLE players ADD COLUMN IF NOT EXISTS class INTEGER DEFAULT 3;
	ALTER TABLE players ADD COLUMN IF NOT EXISTS race INTEGER DEFAULT 0;
	ALTER TABLE players ADD COLUMN IF NOT EXISTS stat_str INTEGER DEFAULT 10;
	ALTER TABLE players ADD COLUMN IF NOT EXISTS stat_str_add INTEGER DEFAULT 0;
	ALTER TABLE players ADD COLUMN IF NOT EXISTS stat_int INTEGER DEFAULT 10;
	ALTER TABLE players ADD COLUMN IF NOT EXISTS stat_wis INTEGER DEFAULT 10;
	ALTER TABLE players ADD COLUMN IF NOT EXISTS stat_dex INTEGER DEFAULT 10;
	ALTER TABLE players ADD COLUMN IF NOT EXISTS stat_con INTEGER DEFAULT 10;
	ALTER TABLE players ADD COLUMN IF NOT EXISTS stat_cha INTEGER DEFAULT 10;

	CREATE INDEX IF NOT EXISTS idx_players_name ON players(name);
	`

	_, err := db.conn.Exec(query)
	return err
}

// GetPlayer retrieves a player by name. Returns nil, nil if not found.
func (db *DB) GetPlayer(name string) (*PlayerRecord, error) {
	query := `
		SELECT id, name, COALESCE(password_hash,''), room_vnum, level, exp,
		       health, max_health, mana, max_mana, strength,
		       class, race, stat_str, stat_str_add, stat_int, stat_wis, stat_dex, stat_con, stat_cha,
		       inventory, equipment
		FROM players WHERE name = $1
	`
	var p PlayerRecord
	err := db.conn.QueryRow(query, name).Scan(
		&p.ID, &p.Name, &p.Password, &p.RoomVNum, &p.Level, &p.Exp,
		&p.Health, &p.MaxHealth, &p.Mana, &p.MaxMana, &p.Strength,
		&p.Class, &p.Race, &p.StatStr, &p.StatStrAdd, &p.StatInt, &p.StatWis, &p.StatDex, &p.StatCon, &p.StatCha,
		&p.Inventory, &p.Equipment,
	)
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
		  (name, room_vnum, level, exp, health, max_health, mana, max_mana, strength,
		   class, race, stat_str, stat_str_add, stat_int, stat_wis, stat_dex, stat_con, stat_cha,
		   inventory, equipment)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)
		RETURNING id
	`
	return db.conn.QueryRow(query,
		p.Name, p.RoomVNum, p.Level, p.Exp, p.Health, p.MaxHealth, p.Mana, p.MaxMana, p.Strength,
		p.Class, p.Race, p.StatStr, p.StatStrAdd, p.StatInt, p.StatWis, p.StatDex, p.StatCon, p.StatCha,
		p.Inventory, p.Equipment,
	).Scan(&p.ID)
}

// SavePlayer persists a player's current state.
func (db *DB) SavePlayer(p *PlayerRecord) error {
	query := `
		UPDATE players SET
		  room_vnum=$1, level=$2, exp=$3, health=$4, max_health=$5,
		  mana=$6, max_mana=$7, strength=$8,
		  class=$9, race=$10,
		  stat_str=$11, stat_str_add=$12, stat_int=$13, stat_wis=$14, stat_dex=$15, stat_con=$16, stat_cha=$17,
		  inventory=$18, equipment=$19, updated_at=CURRENT_TIMESTAMP
		WHERE id=$20
	`
	_, err := db.conn.Exec(query,
		p.RoomVNum, p.Level, p.Exp, p.Health, p.MaxHealth,
		p.Mana, p.MaxMana, p.Strength,
		p.Class, p.Race,
		p.StatStr, p.StatStrAdd, p.StatInt, p.StatWis, p.StatDex, p.StatCon, p.StatCha,
		p.Inventory, p.Equipment, p.ID,
	)
	return err
}