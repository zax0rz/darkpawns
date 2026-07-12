// Package db handles database persistence.
package db

import (
	"database/sql"
	"time"
)

// Database defines the fully comprehensive interface for DB persistence in Dark Pawns.
// It matches all public operations exposed by db.DB, ensuring complete decoupling
// from raw database connection pools and enabling full mock implementations in tests.
type Database interface {
	Close() error

	// Player Persistence
	GetPlayer(name string) (*PlayerRecord, error)
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

	// Decision Capture Telemetry
	EnsureDecisionLogPartitions() error
	NewDecisionLogWriter() *DecisionLogWriter

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
