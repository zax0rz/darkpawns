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
	ID       int
	Name     string
	Password string // hashed
	RoomVNum int
	Level    int
	Exp      int
	Health   int
	MaxHealth int
	Mana     int
	MaxMana  int
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
		password_hash VARCHAR(255), -- nullable for Phase 1
		room_vnum INTEGER DEFAULT 3001,
		level INTEGER DEFAULT 1,
		exp INTEGER DEFAULT 0,
		health INTEGER DEFAULT 100,
		max_health INTEGER DEFAULT 100,
		mana INTEGER DEFAULT 100,
		max_mana INTEGER DEFAULT 100,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_players_name ON players(name);
	`

	_, err := db.conn.Exec(query)
	return err
}

// GetPlayer retrieves a player by name.
func (db *DB) GetPlayer(name string) (*PlayerRecord, error) {
	query := `
		SELECT id, name, password_hash, room_vnum, level, exp, health, max_health, mana, max_mana
		FROM players
		WHERE name = $1
	`

	var p PlayerRecord
	err := db.conn.QueryRow(query, name).Scan(
		&p.ID, &p.Name, &p.Password, &p.RoomVNum, &p.Level, &p.Exp,
		&p.Health, &p.MaxHealth, &p.Mana, &p.MaxMana,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &p, nil
}

// CreatePlayer creates a new player.
func (db *DB) CreatePlayer(name string) (*PlayerRecord, error) {
	query := `
		INSERT INTO players (name, room_vnum)
		VALUES ($1, 3001)
		RETURNING id, name, room_vnum, level, exp, health, max_health, mana, max_mana
	`

	var p PlayerRecord
	err := db.conn.QueryRow(query, name).Scan(
		&p.ID, &p.Name, &p.RoomVNum, &p.Level, &p.Exp,
		&p.Health, &p.MaxHealth, &p.Mana, &p.MaxMana,
	)
	if err != nil {
		return nil, fmt.Errorf("create player: %w", err)
	}

	return &p, nil
}

// SavePlayer updates a player's state.
func (db *DB) SavePlayer(p *PlayerRecord) error {
	query := `
		UPDATE players
		SET room_vnum = $1, level = $2, exp = $3, health = $4, max_health = $5, mana = $6, max_mana = $7, updated_at = CURRENT_TIMESTAMP
		WHERE id = $8
	`

	_, err := db.conn.Exec(query,
		p.RoomVNum, p.Level, p.Exp, p.Health, p.MaxHealth, p.Mana, p.MaxMana, p.ID,
	)
	return err
}

// GetOrCreatePlayer gets a player or creates if not exists.
func (db *DB) GetOrCreatePlayer(name string) (*PlayerRecord, error) {
	p, err := db.GetPlayer(name)
	if err != nil {
		return nil, err
	}
	if p != nil {
		return p, nil
	}
	return db.CreatePlayer(name)
}