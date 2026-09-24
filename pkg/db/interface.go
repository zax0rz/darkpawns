// Package db handles database persistence.
package db

import (
	"database/sql"
	"time"
)

// GameStore is the player-facing persistence surface. It is the seam the game
// server compiles against; *DB satisfies it on both PostgreSQL and SQLite.
type GameStore interface {
	Close() error

	// Player Persistence
	GetPlayer(name string) (*PlayerRecord, error)
	ListPlayerNames() ([]string, error)
	CountPlayers() (int, error)
	CreatePlayer(p *PlayerRecord) error
	SavePlayer(p *PlayerRecord) error
	UpdatePassword(playerID int, hash string) error
	UpdateDescription(playerID int, description string) error
	DeletePlayer(playerID int) error
	GetAccountLockout(name string) (failedAttempts int, lockedUntil *time.Time, err error)
	RecordLoginFailure(name string, threshold int, lockoutDuration time.Duration) (bool, error)
	RecordLoginSuccess(name string) error
	Exec(query string, args ...interface{}) (sql.Result, error)
}

var _ GameStore = (*DB)(nil)
