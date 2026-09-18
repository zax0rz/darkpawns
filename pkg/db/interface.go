// Package db handles database persistence.
package db

import (
	"database/sql"
	"time"
)

// GameStore is the player-facing persistence surface: players, agent keys and
// the AI narrative memory layer. It is the seam the game server compiles
// against; *DB satisfies it on both PostgreSQL and SQLite.
//
// The research corpus (decision_log, combat_log) is deliberately not here:
// those tables stay on PostgreSQL and are reached through ResearchStore, so a
// self-hosted SQLite deployment never sees player-typed input.
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

	// Agent Authorization & Keys
	CreateAgentKey(characterName string) (rawKey string, id int64, err error)
	ValidateAgentKey(rawKey string) (characterName string, keyID int64, valid bool)

	// AI Narrative Memory Layer
	InitNarrativeMemory() error
	WriteNarrativeMemory(m *NarrativeMemory) (int64, error)
	BootstrapMemories(agentName string, limit int) ([]*NarrativeMemory, error)
	RecentMemories(agentName, sessionID string) ([]*NarrativeMemory, error)
	SocialEventMemories(socialEventID string) ([]*NarrativeMemory, error)
	WriteSessionSummary(agentName, sessionID, summary string, eventCount int, start, end time.Time) error
	GetSessionSummaries(agentName string, limit int) ([]string, error)
	DecayStaleMemories(cutoffDays int) (decayed, pruned int, err error)
}

// ResearchStore is the decision-capture telemetry surface. It stays
// PostgreSQL-only: decision_log.raw_input is the literal line a player typed,
// and the corpus tables use PostgreSQL-only features (PARTITION BY RANGE,
// pg_tables catalog queries, TEXT[]).
type ResearchStore interface {
	EnsureDecisionLogPartitions() error
	NewDecisionLogWriter() *DecisionLogWriter
}

// *DB satisfies both halves; the split exists so callers can depend on only
// the surface they need and a SQLite deployment never has to satisfy the
// PostgreSQL-only research methods.
var (
	_ GameStore     = (*DB)(nil)
	_ ResearchStore = (*DB)(nil)
)
