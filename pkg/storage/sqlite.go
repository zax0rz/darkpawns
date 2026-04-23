package storage

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/zax0rz/darkpawns/pkg/game"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteBackend implements Backend using SQLite for persistence.
type SQLiteBackend struct {
	db *sql.DB
}

// NewSQLiteBackend opens (or creates) a SQLite database at the given path.
func NewSQLiteBackend(dbPath string) (*SQLiteBackend, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(1)   // SQLite WAL still prefers single writer
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	b := &SQLiteBackend{db: db}
	if err := b.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate sqlite: %w", err)
	}

	slog.Info("sqlite storage initialized", "path", dbPath)
	return b, nil
}

// migrate creates tables if they don't exist.
func (b *SQLiteBackend) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS players (
		name       TEXT PRIMARY KEY,
		data       TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS world_state (
		id         INTEGER PRIMARY KEY,
		data       TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_players_updated ON players(updated_at);
	`
	_, err := b.db.Exec(schema)
	return err
}

// Save persists a player's state as JSON in SQLite.
func (b *SQLiteBackend) Save(player *game.Player) error {
	data, err := game.SerializePlayer(player)
	if err != nil {
		return fmt.Errorf("serialize player: %w", err)
	}

	_, err = b.db.Exec(
		`INSERT INTO players (name, data, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT(name) DO UPDATE SET data = excluded.data, updated_at = CURRENT_TIMESTAMP`,
		player.Name, data,
	)
	if err != nil {
		return fmt.Errorf("save player to sqlite: %w", err)
	}
	return nil
}

// Load retrieves a player by name from SQLite.
func (b *SQLiteBackend) Load(name string) (*game.Player, error) {
	var data string
	err := b.db.QueryRow("SELECT data FROM players WHERE name = ?", name).Scan(&data)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("player not found: %s", name)
	}
	if err != nil {
		return nil, fmt.Errorf("load player from sqlite: %w", err)
	}

	return game.DeserializePlayer(data)
}

// Delete removes a player from SQLite.
func (b *SQLiteBackend) Delete(name string) error {
	_, err := b.db.Exec("DELETE FROM players WHERE name = ?", name)
	if err != nil {
		return fmt.Errorf("delete player from sqlite: %w", err)
	}
	return nil
}

// Exists checks whether a player exists in SQLite.
func (b *SQLiteBackend) Exists(name string) (bool, error) {
	var count int
	err := b.db.QueryRow("SELECT COUNT(1) FROM players WHERE name = ?", name).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check player exists in sqlite: %w", err)
	}
	return count > 0, nil
}

// List returns all saved player names.
func (b *SQLiteBackend) List() ([]string, error) {
	rows, err := b.db.Query("SELECT name FROM players ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("list players from sqlite: %w", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan player name: %w", err)
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

// SaveWorld persists world state as JSON.
func (b *SQLiteBackend) SaveWorld(w *game.World) error {
	data, err := game.SerializeWorld(w)
	if err != nil {
		return fmt.Errorf("serialize world: %w", err)
	}

	_, err = b.db.Exec(
		`INSERT INTO world_state (id, data, updated_at) VALUES (1, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT(id) DO UPDATE SET data = excluded.data, updated_at = CURRENT_TIMESTAMP`,
		data,
	)
	if err != nil {
		return fmt.Errorf("save world to sqlite: %w", err)
	}
	return nil
}

// LoadWorld retrieves world state.
func (b *SQLiteBackend) LoadWorld() (*game.World, error) {
	var data string
	err := b.db.QueryRow("SELECT data FROM world_state WHERE id = 1").Scan(&data)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no saved world state")
	}
	if err != nil {
		return nil, fmt.Errorf("load world from sqlite: %w", err)
	}

	return game.DeserializeWorld(data)
}

// Close releases the database connection.
func (b *SQLiteBackend) Close() error {
	return b.db.Close()
}

// compile-time interface checks
var _ PlayerStore = (*SQLiteBackend)(nil)
var _ WorldStore = (*SQLiteBackend)(nil)
